package filesystem

import (
	"github.com/pkg/errors"
	"os"
)

type DummyDirEntry struct {
	name     string
	isDir    bool
	fileMode os.FileMode
}

func NewDummyDirEntry(name string) *DummyDirEntry {
	return &DummyDirEntry{
		name:     name,
		isDir:    true,
		fileMode: 0,
	}
}

func (sdi DummyDirEntry) Name() string {
	return sdi.name
}
func (sdi DummyDirEntry) IsDir() bool {
	return sdi.isDir
}
func (sdi DummyDirEntry) Type() os.FileMode {
	return sdi.fileMode
}
func (sdi DummyDirEntry) Info() (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}
