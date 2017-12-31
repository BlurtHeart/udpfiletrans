package sftp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	defaultBlockNum  = 5
	maxBlockNum      = 10
	defaultBlockSize = 512
	minBlockSize     = 512
	maxBlockSize     = 65464
)

type blocker struct {
	id    int
	data  []byte // data in block
	retry *backoff
	timer *time.Timer
}

var (
	noPermissionError   = errors.New("no permission to read or write file")
	fileNotExistError   = errors.New("file does not exist")
	fileNotSame         = errors.New("file not same")
	serverInternalError = errors.New("server internal error occurred")
)

type sender struct {
	blockNum     int // block numbers for sending while waiting for ack
	blocks       []blocker
	blockStatus  map[int]bool // status for each block, false for unused and true for used
	mux          sync.Mutex
	conn         *net.UDPConn
	addr         *net.UDPAddr
	localIP      net.IP
	receive      []byte
	receivedSize int
	tid          int
	retry        *backoff
	timeout      time.Duration
	retries      int
	mode         string
	opts         options
	file         *Filer
	opcode       uint16
}

func (s *sender) setBlockNum(num int) {
	if num < 1 || num > maxBlockNum {
		num = defaultBlockNum
	}
	if s.blockNum > 0 {
		for k, _ := range s.blockStatus {
			delete(s.blockStatus, k)
		}
	}
	s.blockNum = num
	s.blocks = make([]blocker, num)
	for i := 0; i < num; i++ {
		s.blockStatus[i] = false
	}
}

func (s *sender) setBlockSize(blksize int) {
	if blksize < minBlockSize || blksize > maxBlockSize {
		blksize = defaultBlockSize
	}
	for i := 0; i < s.blockNum; i++ {
		s.blocks[i].data = make([]byte, blksize+4)
	}
}

func (s *sender) SetSize(n int64) {
	if s.opts != nil {
		if _, ok := s.opts["tsize"]; ok {
			s.opts["tsize"] = strconv.FormatInt(n, 10)
		}
	}
}

func (s *sender) shakeHands() error {
	info, err := json.Marshal(s.file)
	if err != nil {
		return err
	}
	n := packRQ(s.blocks[0].data, s.opcode, info, s.opts)
	s.retry.reset()
	for {
		s.sendDatagram(s.blocks[0].data[:n])
		err := s.recvDatagram()
		if err != nil {
			if s.retry.count() < s.retries {
				s.retry.backoff()
				continue
			} else {
				return err
			}
		}
		// begin parse hand-shake package
		res, err := parsePacket(s.receive[:s.receivedSize])
		if err != nil {
			return err
		}
		if s.opcode == opRRQ {
			if ackData, ok := res.(pRRQ); ok {
				// check filename, filesize, filemd5, result
				// if filename equals, but other not equal, return error
				fdata, opts, err := unpackRRQ(ackData)
				if err != nil {
					return err
				}
				s.dealOpts(opts)
				var fi Filer
				err = json.Unmarshal(fdata, &fi)
				if err != nil {
					return err
				}
				if fi.Filename != s.file.Filename {
					return serverInternalError
				}
				switch fi.ACK {
				case ackNPermit:
					s.file.State = stateNO
					return noPermissionError
				case ackNExist:
					s.file.State = stateNO
					return fileNotExistError
				case ackNSame:
					s.file.State = stateNO
					info, _ = json.Marshal(s.file)
					n = packRQ(s.blocks[0].data, s.opcode, info, nil)
					s.sendDatagram(s.blocks[0].data[:n])
					return fileNotSame
				case ackSame:
					s.file.State = stateYES
					info, _ = json.Marshal(s.file)
					n = packRQ(s.blocks[0].data, s.opcode, info, nil)
					s.sendDatagram(s.blocks[0].data[:n])
					return nil
				}
			} else {
				s.retry.backoff()
				continue
			}
		} else if s.opcode == opWRQ {
			if ackData, ok := res.(pWRQ); ok {
				// check filename, filesize, filemd5, result
				// if filename equals, but other not equal, return error
				fdata, opts, err := unpackWRQ(ackData)
				if err != nil {
					return err
				}
				s.dealOpts(opts)
				var fi Filer
				err = json.Unmarshal(fdata, &fi)
				if err != nil {
					return err
				}
				if fi.Filename != s.file.Filename {
					return serverInternalError
				}
				switch fi.ACK {
				case ackNPermit:
					s.file.State = stateNO
					return noPermissionError
				case ackNExist:
					s.file.State = stateYES
					return nil
				case ackNSame:
					s.file.State = stateYES
					if isHalfFiler(*s.file, fi) {
						s.file.StartIndex = fi.FileSize + 1
					} else {
						s.file.StartIndex = 0
					}
					info, _ = json.Marshal(s.file)
					n = packRQ(s.blocks[0].data, s.opcode, info, nil)
					s.sendDatagram(s.blocks[0].data[:n])
					return nil
				case ackSame:
					s.file.State = stateComplete
					return nil
				}
			} else {
				s.retry.backoff()
				continue
			}
		} else {
			return serverInternalError
		}
		return nil
	}
}

func (s *sender) dealOpts(opts options) {}

func (s *sender) sendRQ() error {
	info, err := json.Marshal(s.file)
	if err != nil {
		return err
	}
	n := packRQ(s.blocks[0].data, s.opcode, info, s.opts)
	s.sendDatagram(s.blocks[0].data[:n])
	return nil
}

func (s *sender) sendOptions() error {
	for name, value := range s.opts {
		if name == "blksize" {
			blksize, err := strconv.Atoi(value)
			if err != nil {
				delete(s.opts, name)
				continue
			}
			s.setBlockSize(blksize)
		} else if name == "tsize" {
			if value != "0" {
				s.opts["tsize"] = value
			} else {
				delete(s.opts, name)
			}
		} else {
			delete(s.opts, name)
		}
	}
	if len(s.opts) > 0 {
		m := packOACK(s.blocks[0].data, s.opts)
		s.retry.reset()
		for {
			err := s.sendDatagram(s.blocks[0].data[:m])
			if err != nil {
				return err
			}
			err = s.recvDatagram()
			if err == nil {
				p, err := parsePacket(s.receive[:s.receivedSize])
				if err == nil {
					if pack, ok := p.(pOACK); ok {
						opts, err := unpackOACK(pack)
						if err != nil {
							s.abort(err)
							return err
						}
						for name, value := range opts {
							if name == "blksize" {
								blksize, err := strconv.Atoi(value)
								if err != nil {
									delete(s.opts, name)
									continue
								}
								s.setBlockSize(blksize)
							}
						}
						return nil
					}
				}
			}
			if s.retry.count() < s.retries {
				s.retry.backoff()
				continue
			}
			return err
		}
	}
	return nil
}

// read file from r, and write data to server
func (s *sender) ReadFrom(r io.Reader) (int64, error) {
	return 0, nil
}

func (s *sender) recvDatagram() error {
	err := s.conn.SetReadDeadline(time.Now().Add(s.timeout))
	if err != nil {
		return err
	}
	n, addr, err := s.conn.ReadFromUDP(s.receive)
	if err != nil {
		return err
	}
	if !addr.IP.Equal(s.addr.IP) || (s.tid != 0 && addr.Port != s.tid) {
		return fmt.Errorf("datagram not wanted")
	}
	s.tid = addr.Port
	s.receivedSize = n
	return nil
}

func (s *sender) sendDatagram(data []byte) error {
	_, err := s.conn.WriteToUDP(data, s.addr)
	return err
}

func (s *sender) abort(err error) error {
	if s.conn == nil {
		return nil
	}
	defer func() {
		s.conn.Close()
		s.conn = nil
	}()
	n := packERROR(s.blocks[0].data, 1, err.Error())
	_, err = s.conn.WriteToUDP(s.blocks[0].data[:n], s.addr)
	return err
}
