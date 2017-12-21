package sftp

import (
	"crypto/md5"
	"io"
	"os"
)

type filer struct {
	filename string
	md5      string // 16 bytes
	filesize int64
}

// return a new filer pointer
func NewFiler(filename string) (*filer, error) {
	// get file size
	info, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	// calculate file md5
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	h := md5.New()
	if _, err = io.Copy(h, fp); err != nil {
		return nil, err
	}
	return &filer{
		filename: filename,
		md5:      string(h.Sum(nil)),
		filesize: info.Size(),
	}, nil
}
