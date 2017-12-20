package sftp

import (
	"crypto/md5"
	"io"
	"os"
)

type file struct {
	filename string
	md5      string // 16 bytes
	filesize int64
}

// return if file exists and file size name
func (f file) checkFileSize() bool {
	info, err := os.Stat(f.filename)
	if err != nil {
		return false
	}
	if info.Size() == f.filesize {
		return true
	}
	return false
}

// return if file md5 is same
func (f file) checkFileMd5() bool {
	fp, err := os.Open(f.filename)
	if err != nil {
		return false
	}
	defer fp.Close()
	h := md5.New()
	if _, err = io.Copy(h, fp); err != nil {
		return false
	}
	if string(h.Sum(nil)) == f.md5 {
		return true
	}
	return false
}
