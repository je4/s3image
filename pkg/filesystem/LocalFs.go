package filesystem

import (
	"fmt"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
)

type LocalFs struct {
	basepath string
	logger   *logging.Logger
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func FolderExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func NewLocalFs(basepath string, logger *logging.Logger) (*LocalFs, error) {
	if !FolderExists(basepath) {
		return nil, fmt.Errorf("path %v does not exists", basepath)
	}
	return &LocalFs{basepath: basepath, logger: logger}, nil
}

func (lfs *LocalFs) Protocol() string {
	return "file://"
}

func (lfs *LocalFs) String() string {
	return lfs.basepath
}

func (lfs *LocalFs) FileStat(folder, name string, opts FileStatOptions) (os.FileInfo, error) {
	path := filepath.Join(folder, name)
	return os.Stat(filepath.Join(lfs.basepath, path))
}

func (lfs *LocalFs) FileExists(folder, name string) (bool, error) {
	path := filepath.Join(folder, name)
	return FileExists(filepath.Join(lfs.basepath, path)), nil
}

func (lfs *LocalFs) FolderExists(folder string) (bool, error) {
	return FolderExists(filepath.Join(lfs.basepath, folder)), nil
}

func (lfs *LocalFs) FolderCreate(folder string, opts FolderCreateOptions) error {
	path := filepath.Join(lfs.basepath, folder)
	if FolderExists(path) {
		return nil
	}
	lfs.logger.Debugf("create folder %v", path)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return errors.Wrapf(err, "cannot create folder %v", path)
	}
	return nil
}

func (lfs *LocalFs) FileGet(folder, name string, opts FileGetOptions) ([]byte, error) {
	path := filepath.Join(folder, name)
	data, err := ioutil.ReadFile(filepath.Join(lfs.basepath, path))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read file %v", path)
	}
	return data, nil
}

func (lfs *LocalFs) FilePut(folder, name string, data []byte, opts FilePutOptions) error {
	if err := lfs.FolderCreate(folder, FolderCreateOptions{}); err != nil {
		return errors.Wrapf(err, "cannot create folder %v", folder)
	}
	path := filepath.Join(lfs.basepath, filepath.Join(folder, name))
	lfs.logger.Debugf("writing data to: %v", path)
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return errors.Wrapf(err, "cannot write data to %v", path)
	}
	return nil
}

func (lfs *LocalFs) FileWrite(folder, name string, r io.Reader, size int64, opts FilePutOptions) error {
	if err := lfs.FolderCreate(folder, FolderCreateOptions{}); err != nil {
		return errors.Wrapf(err, "cannot create folder %v", folder)
	}
	path := filepath.Join(folder, name)
	file, err := os.OpenFile(filepath.Join(lfs.basepath, path), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "cannot open file %v", path)
	}
	defer file.Close()
	if size == -1 {
		if _, err := io.Copy(file, r); err != nil {
			return errors.Wrapf(err, "cannot write to file %v", path)
		}
	} else {
		if _, err := io.CopyN(file, r, size); err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				return errors.Wrapf(err, "cannot write to file %v", path)
			}
		}
	}
	return nil
}

func (lfs *LocalFs) FileRead(folder, name string, w io.Writer, size int64, opts FileGetOptions) error {
	path := filepath.Join(folder, name)
	file, err := os.OpenFile(filepath.Join(lfs.basepath, path), os.O_RDONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "cannot open file %v", path)
	}
	defer file.Close()
	if size == -1 {
		if _, err := io.Copy(w, file); err != nil {
			return errors.Wrapf(err, "cannot read from %v/%v", path, name)
		}
	} else {
		if _, err := io.CopyN(w, file, size); err != nil {
			return errors.Wrapf(err, "cannot read from %v/%v", path, name)
		}
	}
	return nil
}

func (lfs *LocalFs) FileOpenRead(folder, name string, opts FileGetOptions) (io.ReadCloser, string, error) {
	path := filepath.Join(folder, name)
	file, err := os.OpenFile(filepath.Join(lfs.basepath, path), os.O_RDONLY, 0644)
	if err != nil {
		return nil, "", errors.Wrapf(err, "cannot open file %v", path)
	}
	// todo: detect mime type
	return file, "application/octet-stream", nil
}

func (lfs *LocalFs) FileList(folder, name string) ([]fs.DirEntry, error) {
	path := filepath.Join(folder, name)
	fullpath := filepath.Join(lfs.basepath, path)
	de, err := os.ReadDir(fullpath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read %s", fullpath)
	}
	var list = []fs.DirEntry{}
	for _, e := range de {
		fp := filepath.Join(path, e.Name())
		fi, err := os.Stat(filepath.Join(lfs.basepath, fp))
		if err != nil {
			return nil, errors.Wrapf(err, "cannot stat %s", fp)
		}
		lde := LocalDirEntry{FileInfo: fi, bucket: path}
		list = append(list, lde)
	}
	return list, nil
}
