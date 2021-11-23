package filesystem

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"strings"
)

type S3Fs struct {
	s3       *minio.Client
	endpoint string
}

func NewS3Fs(Endpoint string,
	AccessKeyId string,
	SecretAccessKey string,
	UseSSL bool) (*S3Fs, error) {
	// connect to S3 / Minio
	s3, err := minio.New(Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(AccessKeyId, SecretAccessKey, ""),
		Secure: UseSSL,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to s3 instance")
	}
	return &S3Fs{s3: s3, endpoint: Endpoint}, nil
}

func (fs *S3Fs) Protocol() string {
	return fmt.Sprintf("s3://%s", fs.endpoint)
}

func (fs *S3Fs) String() string {
	return fmt.Sprintf(fs.s3.EndpointURL().String())
}

func (fs *S3Fs) FileList(folder, name string) ([]os.DirEntry, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	objectCh := fs.s3.ListObjects(ctx, folder, minio.ListObjectsOptions{
		Prefix:    name,
		Recursive: true,
	})
	var result = []os.DirEntry{}
	var subDirs = []string{}
	for object := range objectCh {
		subPath := strings.Trim(strings.TrimPrefix(object.Key, name), "/")
		if subPath == "" {
			continue
		}
		if strings.ContainsRune(subPath, '/') {
			subDir := subPath[:strings.IndexByte(subPath, '/')]
			found := false
			for _, d := range subDirs {
				if d == subDir {
					found = true
					break
				}
			}
			if !found {
				subDirs = append(subDirs, subDir)
			}
			continue
		}
		result = append(result, S3DirEntry{
			S3FileInfo{
				bucket: folder,
				name:   name,
				info:   object,
			},
		})
	}
	for _, d := range subDirs {
		subDir := fmt.Sprintf("%s/%s", name, d)
		result = append(result, DummyDirEntry{
			name:     fmt.Sprintf("%s/%s", folder, subDir),
			isDir:    true,
			fileMode: 0,
		})
	}
	return result, nil
}

func (fs *S3Fs) FileStat(folder, name string, opts FileStatOptions) (os.FileInfo, error) {
	sinfo, err := fs.s3.StatObject(context.Background(), folder, name, minio.StatObjectOptions{})
	if err != nil {
		// no file no error
		s3Err, ok := err.(minio.ErrorResponse)
		if ok {
			if s3Err.StatusCode == http.StatusNotFound {
				return nil, &NotFoundError{err: err}
			}
		}
		return nil, errors.Wrapf(err, "cannot get file info for %v/%v", folder, name)
	}
	return NewS3FileInfo(folder, name, sinfo), nil
}

func (fs *S3Fs) FileExists(folder, name string) (bool, error) {
	_, err := fs.FileStat(folder, name, FileStatOptions{})
	if err != nil {
		// no file no error
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "cannot get file info for %v/%v", folder, name)
	}
	return true, nil
}

func (fs *S3Fs) FolderExists(folder string) (bool, error) {
	found, err := fs.s3.BucketExists(context.Background(), folder)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get check for folder %v", folder)
	}
	return found, nil
}

func (fs *S3Fs) FolderCreate(folder string, opts FolderCreateOptions) error {
	if err := fs.s3.MakeBucket(context.Background(), folder, minio.MakeBucketOptions{ObjectLocking: opts.ObjectLocking}); err != nil {
		return errors.Wrapf(err, "cannot create bucket %s", folder)
	}
	return nil
}

func (fs *S3Fs) FileGet(folder, name string, opts FileGetOptions) ([]byte, error) {
	object, err := fs.s3.GetObject(context.Background(), folder, name, minio.GetObjectOptions{VersionID: opts.VersionID})
	if err != nil {
		// no file no error
		s3Err, ok := err.(minio.ErrorResponse)
		if ok {
			if s3Err.StatusCode == http.StatusNotFound {
				return nil, &NotFoundError{err: s3Err}
			}
		}
		return nil, errors.Wrapf(err, "cannot get file info for %v/%v", folder, name)
	}

	var b = &bytes.Buffer{}
	if _, err := io.Copy(b, object); err != nil {
		return nil, errors.Wrapf(err, "cannot copy data from %v/%v", folder, name)
	}
	return b.Bytes(), nil
}

func (fs *S3Fs) FilePut(folder, name string, data []byte, opts FilePutOptions) error {
	if _, err := fs.s3.PutObject(
		context.Background(),
		folder,
		name,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: opts.ContentType, Progress: opts.Progress},
	); err != nil {
		return errors.Wrapf(err, "cannot put %v/%v", folder, name)
	}
	return nil
}

func (fs *S3Fs) FileWrite(folder, name string, r io.Reader, size int64, opts FilePutOptions) error {
	if _, err := fs.s3.PutObject(
		context.Background(),
		folder,
		name,
		r,
		size,
		minio.PutObjectOptions{ContentType: opts.ContentType, Progress: opts.Progress},
	); err != nil {
		return errors.Wrapf(err, "cannot put %v/%v", folder, name)
	}
	return nil
}

func (fs *S3Fs) FileRead(folder, name string, w io.Writer, size int64, opts FileGetOptions) error {
	object, err := fs.s3.GetObject(
		context.Background(),
		folder,
		name,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return errors.Wrapf(err, "cannot get object %v/%v", folder, name)
	}
	defer object.Close()
	if size == -1 {
		if _, err := io.Copy(w, object); err != nil {
			return errors.Wrapf(err, "cannot read from obect %v/%v", folder, name)
		}
	} else {
		if _, err := io.CopyN(w, object, size); err != nil {
			if err != io.ErrUnexpectedEOF && err != io.EOF {
				return errors.Wrapf(err, "cannot read from obect %v/%v", folder, name)
			}
		}
	}
	return nil
}

func (fs *S3Fs) FileOpenRead(folder, name string, opts FileGetOptions) (io.ReadCloser, string, error) {
	object, err := fs.s3.GetObject(
		context.Background(),
		folder,
		name,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, "", errors.Wrapf(err, "cannot get object %v/%v", folder, name)
	}
	oinfo, err := object.Stat()
	if err != nil {
		return nil, "", errors.Wrapf(err, "cannot stat object %v/%v", folder, name)
	}
	return object, oinfo.ContentType, nil
}
