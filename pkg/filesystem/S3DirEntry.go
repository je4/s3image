package filesystem

import (
	"os"
)

type S3DirEntry struct {
	S3FileInfo
}

func (sdi S3DirEntry) Name() string {
	return sdi.S3FileInfo.Name()
}
func (sdi S3DirEntry) IsDir() bool {
	return sdi.S3FileInfo.IsDir()
}
func (sdi S3DirEntry) Type() os.FileMode {
	return 0
}
func (sdi S3DirEntry) Info() (os.FileInfo, error) {
	return &sdi.S3FileInfo, nil
}
