package filesystem

import (
	"os"
	"path/filepath"
)

type LocalDirEntry struct {
	os.FileInfo
	bucket string
}

func (lde LocalDirEntry) Name() string {
	return filepath.Join(lde.bucket, lde.FileInfo.Name())
}
func (lde LocalDirEntry) IsDir() bool {
	return lde.FileInfo.IsDir()
}
func (lde LocalDirEntry) Type() os.FileMode {
	return lde.FileInfo.Mode()
}
func (lde LocalDirEntry) Info() (os.FileInfo, error) {
	return lde.FileInfo, nil
}
