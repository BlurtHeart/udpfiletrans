package sftp

import (
	"crypto/md5"
	"io"
	"os"
)

const (
	ackNPermit = uint16(1) // File not allowed to read or write
	ackNExist  = uint16(2) // File not exists
	ackSame    = uint16(3) // File exists
	ackNSame   = uint16(4) // File not same, used for transmission resuming at break-points
	ackDir     = uint16(5) // assigned name is a directory
)

const (
	stateSync     = uint8(0) // sync
	stateNO       = uint8(1) // do not transfer
	stateYES      = uint8(2) // transfer
	stateComplete = uint8(8) // transfer complete
)

type Filer struct {
	Filename   string `json:"filename"`    // file name
	MD5        string `json:"md5"`         // md5 value of file, take 16 bytes
	FileSize   int64  `json:"filesize"`    // file size
	StartIndex int64  `json:"start_index"` // start index for read or write
	FileMode   uint32 `json:"file_mode"`   // file mode
	ACK        uint16 `json:"ack"`         // ack code for request
	State      uint8  `json:"state"`       // tranfer state
}

// return a new Filer pointer
func NewFiler(filename string) (*Filer, error) {
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
	return &Filer{
		Filename: filename,
		MD5:      string(h.Sum(nil)),
		FileSize: info.Size(),
		FileMode: uint32(info.Mode().Perm()),
		State:    stateSync,
	}, nil
}

func isSameFiler(local Filer, remote Filer) bool {
	return local.Filename == remote.Filename && local.FileSize == remote.FileSize && local.MD5 == remote.MD5
}

func isHalfFiler(local Filer, remote Filer) bool {
	if local.Filename != remote.Filename || local.FileSize < remote.FileSize {
		return false
	}
	fp, err := os.Open(local.Filename)
	if err != nil {
		return false
	}
	defer fp.Close()
	h := md5.New()
	n, err := io.CopyN(h, fp, remote.FileSize)
	if err != nil || n != remote.FileSize {
		return false
	}
	if remote.MD5 == string(h.Sum(nil)) {
		return true
	} else {
		return false
	}
}
