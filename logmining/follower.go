package logmining

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/globalsign/mgo/bson"

	"github.com/fsnotify/fsnotify"
)

const (
	bufSize  = 4 * 1024
	peekSize = 1024
)

type Line struct {
	bytes     []byte
	discarded int
}

func (l *Line) Bytes() []byte {
	return l.bytes
}

func (l *Line) String() string {
	return string(l.bytes)
}

func (l *Line) Discarded() int {
	return l.discarded
}

type LastPosition struct {
	Name   string `bson:"_id" json:"_id"`
	Offset int64  `bson:"offset" json:"offset"`
	Whence int    `bson:"whence" json:"whence"`
	Reopen bool   `bson:"reopen" json:"reopen"`
}

func (l *LastPosition) Query() bson.M {
	return bson.M{"_id": l.Name}
}

func (l *LastPosition) String() string {
	b, err := json.Marshal(l)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// excerpt from github.com/papertrail/go-tail/follower.go
type Tailfollower struct {
	once     sync.Once
	file     *os.File
	filename string
	lines    chan Line
	err      error
	config   *LastPosition
	reader   *bufio.Reader
	watcher  *fsnotify.Watcher
	offset   int64
	closeCh  chan struct{}
}

func NewTailfollower(cfg *LastPosition) (*Tailfollower, error) {
	t := &Tailfollower{
		filename: cfg.Name,
		lines:    make(chan Line),
		config:   cfg,
		closeCh:  make(chan struct{}),
	}

	err := t.reopen()
	if err != nil {
		return nil, err
	}

	go t.once.Do(t.run)

	return t, nil
}

func (t *Tailfollower) Lines() chan Line {
	return t.lines
}

func (t *Tailfollower) Err() error {
	return t.err
}

func (t *Tailfollower) Close() {
	t.closeCh <- struct{}{}
}

func (t *Tailfollower) run() {
	t.close(t.follow())
}

func (t *Tailfollower) Offset() int64 {
	return t.offset
}

func (t *Tailfollower) follow() error {
	_, err := t.file.Seek(t.config.Offset, t.config.Whence)
	if err != nil {
		return err
	}

	var (
		eventChan = make(chan fsnotify.Event)
		errChan   = make(chan error, 1)
	)

	t.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer t.watcher.Close()
	go t.watchFileEvents(eventChan, errChan)

	t.watcher.Add(t.filename)

	for {
		for {
			// discard leading NUL bytes
			var discarded int

			for {
				b, _ := t.reader.Peek(peekSize)
				i := bytes.LastIndexByte(b, '\x00')

				if i > 0 {
					n, _ := t.reader.Discard(i + 1)
					discarded += n
				}

				if i+1 < peekSize {
					break
				}
			}

			s, err := t.reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				return err
			}
			if err == io.EOF {
				l := len(s)

				t.offset, err = t.file.Seek(-int64(l), io.SeekCurrent)
				if err != nil {
					return err
				}

				t.reader.Reset(t.file)
				break
			}

			t.sendLine(s, discarded)
		}

		// we're now at EOF, so wait for changes
		select {
		case evt := <-eventChan:
			switch evt.Op {

			// as soon as something is written, go back and read until EOF.
			case fsnotify.Chmod:
				fallthrough

			case fsnotify.Write:
				fi, err := t.file.Stat()
				if err != nil {
					if !os.IsNotExist(err) {
						return err
					}

					// it's possible that an unlink can cause fsnotify.Chmod,
					// so attempt to rewatch if the file is missing
					if err := t.rewatch(); err != nil {
						return err
					}

					continue
				}

				// file was truncated, seek to the beginning
				if t.offset > fi.Size() {
					t.offset, err = t.file.Seek(0, io.SeekStart)
					if err != nil {
						return err
					}

					t.reader.Reset(t.file)
				}

				continue

			default:
				if !t.config.Reopen {
					return nil
				}

				if err := t.rewatch(); err != nil {
					return err
				}

				continue
			}

		// any errors that come from fsnotify
		case err := <-errChan:
			return err

		// a request to stop
		case <-t.closeCh:
			t.watcher.Remove(t.filename)
			return nil

		case <-time.After(10 * time.Second):
			fi1, err := t.file.Stat()
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			fi2, err := os.Stat(t.filename)
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			if os.SameFile(fi1, fi2) {
				continue
			}

			if err := t.rewatch(); err != nil {
				return err
			}

			continue
		}
	}
}

func (t *Tailfollower) rewatch() error {
	t.watcher.Remove(t.filename)
	if err := t.reopen(); err != nil {
		return err
	}

	t.watcher.Add(t.filename)
	return nil
}

func (t *Tailfollower) reopen() error {
	if t.file != nil {
		t.file.Close()
		t.file = nil
	}

	file, err := os.Open(t.filename)
	if err != nil {
		return err
	}

	t.file = file
	t.reader = bufio.NewReaderSize(t.file, bufSize)

	return nil
}

func (t *Tailfollower) close(err error) {
	t.err = err

	if t.file != nil {
		t.file.Close()
	}

	close(t.lines)
}

func (t *Tailfollower) sendLine(l []byte, d int) {
	t.lines <- Line{l[:len(l)-1], d}
}

func (t *Tailfollower) watchFileEvents(eventChan chan fsnotify.Event, errChan chan error) {
	for {
		select {
		case evt, ok := <-t.watcher.Events:
			if !ok {
				return
			}

			// debounce write events, but send all others
			switch evt.Op {
			case fsnotify.Write:
				select {
				case eventChan <- evt:
				default:
				}

			default:
				select {
				case eventChan <- evt:
				case err := <-t.watcher.Errors:
					errChan <- err
					return
				}
			}

		// die on a file watching error
		case err := <-t.watcher.Errors:
			errChan <- err
			return
		}
	}
}
