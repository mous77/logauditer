package logmining

import (
	in "logauditer/internal"
	"path/filepath"
	"regexp"

	log "github.com/laik/logger"
)

// file : 10.10.2.104_2018-12-04_DianXin-route.log => host = 10.10.2.104

type file struct {
	name           string
	closeCh        chan struct{}
	runtimeOptions *in.RuntimeOptions
	tail           *Tailfollower
	lastPosition   *LastPosition
	dw             *DBWrite
}

func newFile(runtimeOptions *in.RuntimeOptions, lastPosition *LastPosition, dw *DBWrite) (*file, error) {
	log.Debug("runtimeops = %#v\n", runtimeOptions)
	f := &file{
		name:           lastPosition.Name,
		runtimeOptions: runtimeOptions,
		lastPosition:   lastPosition,
		dw:             dw,
	}
	f.closeCh = make(chan struct{})

	tail, err := NewTailfollower(f.lastPosition)
	if err != nil {
		log.Error("new tail follower error:%s last position %#v\n", err, f.lastPosition)
		return nil, err
	}
	f.tail = tail

	var host string
	var date string
	_, fileName := filepath.Split(f.name)

	log.Debug("host regexp %s\n", f.runtimeOptions.Host)
	hostReg, err := regexp.Compile(runtimeOptions.Host)
	if err != nil {
		log.Error("compile host regexp error:(%s)\n", err)
		return nil, err
	}

	if hlist := hostReg.FindAllString(fileName, -1); len(hlist) >= 1 {
		host = hlist[0]
	}

	log.Debug("logDate regexp %s\n", f.runtimeOptions.LogDate)

	logDateReg, err := regexp.Compile(runtimeOptions.LogDate)
	if err != nil {
		log.Error("compile logDate regexp error:(%s)\n", err)
		return nil, err
	}
	if dlist := logDateReg.FindAllString(fileName, -1); len(dlist) >= 1 {
		date = dlist[0]
	} else {
		log.Debug("parse logDate list %#v\n", dlist)
	}

	go func() {
		for out := range f.tailing() {
			log.Debug(`
parse: (%s)
filename: (%s)
host: (%s)
date: (%s)

`, out, fileName, host, date)
			if err := dw.Write(host, date, out); err != nil {
				log.Error("dw write error %s\n", err)
			}
		}
	}()

	return f, nil
}

func (f *file) tailing() chan []byte {
	buf := make(chan []byte)
	go func() {
		defer close(buf)
		logParts := in.NewLogParts()
		for {
			select {
			case line, ok := <-f.tail.Lines():
				if !ok {
					return
				}
				lineData := line.Bytes()
				resp := in.NewResponse()
				var _err error
				in.VisitLogsAudit2(
					logParts,
					lineData,
					resp,
					f.runtimeOptions,
					&_err,
				)
				if resp.Err == nil && _err == nil {
					buf <- []byte(resp.Data)
				} else {
					log.Error("%s\n", _err)
				}
				f.lastPosition.Offset = f.tail.Offset()
				log.Debug("file (%s) offset (%d) data (%s)\n", f.name, f.lastPosition.Offset, lineData)
			case <-f.closeCh:
				log.Debug("close file tail %s\n", f.name)
				f.tail.Close()
				return
			}
		}
	}()
	return buf
}

func (f *file) close() {
	f.closeCh <- struct{}{}
}
