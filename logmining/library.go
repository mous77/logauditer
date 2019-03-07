package logmining

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

const (
	stored        = "data/cache"
	binary_suffix = ".bin"
)

type LibraryCache map[string]LastPosition

type library interface {
	Get(string) []byte
	Set(string, string)
}

func Marshal(l *LastPosition) error {
	fn := filepath.Join(stored, l.Name+binary_suffix)
	ff, err := os.OpenFile(fn, os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0655)
	if err != nil {
		return err
	}
	defer ff.Close()
	return binary.Write(ff, binary.BigEndian, l)
}

func Unmarshal(name string) (*LastPosition, error) {
	fn := filepath.Join(stored, name+binary_suffix)
	ff, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer ff.Close()
	var l LastPosition

	if err = binary.Read(ff, binary.BigEndian, &l); err != nil {
		return nil, err
	}
	return &l, nil
}
