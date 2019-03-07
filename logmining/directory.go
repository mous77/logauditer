package logmining

import (
	"fmt"
	"io"
	"io/ioutil"
	"logauditer/dbapi"
	in "logauditer/internal"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/globalsign/mgo/bson"
	log "github.com/laik/logger"
)

type FileType uint8

const (
	ROOT          = 0
	FILE FileType = iota
	DIR
	UNKNOW
)
const LIBDB = "audit_lib"

type Directory struct {
	name           string
	runtimeOptions *in.RuntimeOptions
	level          int
	filePattern    *regexp.Regexp

	watcher *fsnotify.Watcher

	fileMap map[string]*file
	dirMap  map[string]*Directory

	closeCh chan struct{}

	mu sync.Mutex

	libcoll string

	persists *dbapi.StorageParts
}

func NewDirectory(runtimeOptions *in.RuntimeOptions, level int, persists *dbapi.StorageParts, rule string) (*Directory, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("cloud not initialize watcher.\n")
	}

	if _, err := os.Stat(runtimeOptions.Dir); os.IsNotExist(err) {
		err = os.MkdirAll(runtimeOptions.Dir, 0655)
		if err != nil {
			return nil, err
		}
	}

	err = watcher.Add(runtimeOptions.Dir)
	if err != nil {
		return nil, err
	}

	dir := &Directory{
		name:           runtimeOptions.Dir,
		level:          level,
		runtimeOptions: runtimeOptions,
		watcher:        watcher,
		mu:             sync.Mutex{},
		persists:       persists,
		libcoll:        rule,
	}
	dir.fileMap = make(map[string]*file)
	dir.dirMap = make(map[string]*Directory)
	dir.closeCh = make(chan struct{})

	filePattern, err := regexp.Compile(dir.runtimeOptions.FilePattern)
	if err != nil {
		return nil, err
	}
	dir.filePattern = filePattern

	go dir.track()
	dir.start()
	dir.monitorCurrentDateFile()
	dir.async2second()

	return dir, nil
}

func (d *Directory) Close() {
	d.closeCh <- struct{}{}
}

func (d *Directory) async2second() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ticker.C:
				r := new([]LastPosition)
				d.asyncFlush(r)
			}
		}
	}()
}

func (d *Directory) asyncFlush(r *[]LastPosition) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, f := range d.fileMap {
		var _err error
		dbapi.AccessDatabase(d.persists, LIBDB, d.libcoll, bson.M{"_id": f.name},
			*f.lastPosition,
			dbapi.SET,
			dbapi.KV,
			&_err,
		)
	}
	for _, _d := range d.dirMap {
		_d.asyncFlush(r)
	}
}

func (d *Directory) List(r *[]string) {
	for _, v := range d.fileMap {
		info := "name:" + v.name + "," + "offset" + ":" + fmt.Sprintf("%d", v.lastPosition.Offset)
		*r = append(*r, info)
	}
	for _, _d := range d.dirMap {
		_d.List(r)
	}
}

func (d *Directory) addDir(dir string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	cloneOps := d.runtimeOptions.Clone()
	cloneOps.Dir = dir
	directory, err := NewDirectory(cloneOps, d.level+1, d.persists, d.libcoll)
	if err != nil {
		return err
	}
	d.dirMap[dir] = directory
	log.Debug("add directory %s\n", dir)
	return nil
}

func (d *Directory) removeDir(dir string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	log.Debug("remove directory %s\n", dir)
	if childD, ok := d.dirMap[dir]; ok {
		for f, _ := range childD.fileMap {
			childD.removeFile(f)
		}
		childD.removeDir(childD.name)
		childD.Close()
		delete(d.dirMap, dir)
	}
}

func (d *Directory) addFile(f string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	lastp := &LastPosition{}
	var _err error
	dbapi.AccessDatabase(d.persists,
		LIBDB,
		d.libcoll, //每个规则的collection 独立保存
		bson.M{"_id": f},
		lastp,
		dbapi.GET,
		dbapi.KV,
		&_err,
	)
	if _err != nil || lastp.Offset == 0 {
		lastp.Name = f
		lastp.Offset = 0
		lastp.Whence = io.SeekStart
		lastp.Reopen = true
		log.Debug("find rule (%s.%s) (key:%s) not found.\n", LIBDB, d.libcoll, f)
		var _err1 error
		dbapi.AccessDatabase(d.persists, LIBDB, d.libcoll, bson.M{"_id": f}, lastp, dbapi.SET, dbapi.KV, &_err1)
		if _err1 != nil {
			return _err1
		}
	} else {
		lastp.Whence = io.SeekCurrent
		lastp.Reopen = true
	}
	file, err := newFile(d.runtimeOptions, lastp, NewDBWrite(d.persists))
	if err != nil {
		return err
	}
	d.fileMap[f] = file
	log.Debug("add file (%s).\n", f)
	return nil
}

func (d *Directory) addEventFile(f string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	lastp := &LastPosition{
		Name:   f,
		Offset: 0,
		Whence: io.SeekStart,
		Reopen: true,
	}
	var _err error
	dbapi.AccessDatabase(d.persists, LIBDB, d.libcoll, bson.M{"_id": f}, lastp, dbapi.SET, dbapi.KV, &_err)
	if _err != nil {
		return _err
	}

	file, err := newFile(d.runtimeOptions, lastp, NewDBWrite(d.persists))
	if err != nil {
		return err
	}
	d.fileMap[f] = file
	log.Debug("add event file (%s).\n", f)
	return nil
}

func (d *Directory) removeFile(f string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	log.Debug("remove file %s\n", f)
	ff, ok := d.fileMap[f]
	if !ok {
		return
	}
	go ff.close() //异步可关闭
	var _err error
	dbapi.AccessDatabase(d.persists, LIBDB, d.libcoll, bson.M{"_id": f}, nil, dbapi.DEL, dbapi.KV, &_err)
	if _err != nil {
		log.Error("%s\n", _err)
	}
	delete(d.fileMap, f)
}

func (d *Directory) monitorCurrentDateFile() {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				for fn, _ := range d.fileMap {
					if !d.isExpired(fn) {
						continue
					}
					d.removeFile(fn)
				}
			}
		}
	}()
}

func (d *Directory) isExpired(fn string) bool {
	timeLayout := "2006 01 02" //
	source := time.Unix(time.Now().Unix(), 0).Format(timeLayout)
	curDate := strings.Replace(source, " ", "-", 2)
	getDate, _ := regexp.Compile("\\d+-\\d+-\\d+")
	_, name := filepath.Split(fn)
	date := getDate.FindString(name)
	if date == curDate {
		return false
	}
	return true
}

func (d *Directory) start() {
	ffinfos, err := ioutil.ReadDir(d.name)
	if err != nil {
		log.Error("read dir error:%s.\n", err)
		return
	}

	for _, finfo := range ffinfos {
		ff := finfo.Name()
		newff := filepath.Join(d.name, ff)

		if finfo.IsDir() {
			if err := d.addDir(newff); err != nil {
				log.Error("add child dir error:%s.\n", err)
			}
			continue
		}
		if !d.filePattern.MatchString(ff) || d.isExpired(ff) {
			log.Debug("file (%s) not match define rule or is expried file.\n", newff)
			continue
		}
		if err := d.addFile(newff); err != nil {
			log.Error("add child file tail follower error:%s.\n", err)
		}
	}
}

func (d *Directory) track() {
	for {
		select {
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}

			switch event.Op {
			case fsnotify.Create:
				switch ftype(event.Name) {
				case FILE:
					_, fileName := filepath.Split(event.Name)
					if !d.filePattern.MatchString(fileName) || d.isExpired(fileName) {
						log.Debug("file (%s) not match define rule or is expired file.\n", event.Name)
						break
					}
					d.addEventFile(event.Name)
				case DIR:
					d.addDir(event.Name)
				}
			case fsnotify.Remove:
				if _, ok := d.fileMap[event.Name]; ok {
					log.Debug("remove file op %s.\n", event.Name)
					d.removeFile(event.Name)
				} else {
					log.Debug("remove dir op %s.\n", event.Name)
					d.removeDir(event.Name)
				}
			default:
			}

		case <-d.closeCh:
			log.Debug("recevier stop directory (%s).\n", d.name)
			var _err error
			for _fn, _f := range d.fileMap {
				(*_f).close()
				lastp := _f.lastPosition
				dbapi.AccessDatabase(d.persists, LIBDB, d.libcoll, bson.M{"_id": _fn}, lastp, dbapi.SET, dbapi.KV, &_err)
				if _err != nil {
					log.Error("close file save record error: (%s)\n", _err)
				}
				log.Debug("graceful close (%s) save lastposition.\n", _fn)
			}

			for _, _d := range d.dirMap {
				(*_d).Close()
			}
			if err := d.watcher.Close(); err != nil {
				log.Error("close watch dir (%s) error:%s.\n", d.name, err)
			}
			return
		}
	}
}

func ftype(fn string) FileType {
	info, e := os.Stat(fn)
	if e != nil {
		return UNKNOW
	}
	if !info.IsDir() {
		return FILE
	}
	return DIR
}
