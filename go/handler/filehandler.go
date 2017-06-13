package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	fpath "path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go/model"
	"go/proto"
)

// object udpclient
type UdpClient struct {
	ServerIP   string
	ServerPort int
	ServerAddr string
	Conn       net.Conn
}

func (uc *UdpClient) SetIP(ip string) {
	uc.ServerIP = ip
	uc.ServerAddr = uc.ServerIP + ":" + string(uc.ServerPort)
}

func (uc *UdpClient) SetPort(port int) {
	uc.ServerPort = port
	uc.ServerAddr = uc.ServerIP + ":" + string(uc.ServerPort)
}
func (uc *UdpClient) SetAddr(addr string) error {
	arr := strings.Split(addr, ":")
	if len(arr) != 2 {
		return errors.New("addr not correct")
	}
	uc.ServerIP = arr[0]
	uc.ServerPort, _ = strconv.Atoi(arr[1])
	uc.ServerAddr = addr
	return nil
}
func (uc *UdpClient) Connect() error {
	var err error
	uc.Conn, err = net.Dial("udp", uc.ServerAddr)
	return err
}
func (uc *UdpClient) SetReadDeadLine() error {
	t := time.Now()
	err := uc.Conn.SetReadDeadline(t.Add(time.Duration(1 * time.Second)))
	return err
}
func (uc UdpClient) SendData(data []byte) {
	uc.Conn.Write(data)
}
func (uc UdpClient) RecvData() ([]byte, int) {
	data := make([]byte, 4096)
	dataCount, _ := uc.Conn.Read(data)
	return data[:dataCount], dataCount
}

type FileClient struct {
	fp          *os.File
	isfile      bool
	FilePath    string
	Filename    string
	FileMD5     string
	FilePackets int
	UC          *UdpClient
	channel     chan []byte
	isOver      bool
	recvTimeout bool
}

func (fc *FileClient) SetIsFile(status bool) {
	fc.isfile = status
}

func (fc *FileClient) preCheck() bool {
	if !fc.isfile {
		_, err := os.Stat(fc.FilePath)
		b := err == nil || os.IsExist(err)
		if !b {
			fmt.Println(fc.FilePath)
			if err = os.MkdirAll(fc.FilePath, 0755); err != nil {
				return false
			}
		}
		realfile := fpath.Join(fc.FilePath, fc.Filename)
		fc.fp, err = os.OpenFile(realfile, os.O_RDWR|os.O_CREATE, 0655)
		if err != nil {
			return false
		}
		fc.isfile = true
	} else {
		realfile := fpath.Join(fc.FilePath, fc.Filename)
		var err error
		fc.fp, err = os.OpenFile(realfile, os.O_RDWR, 0655)
		if err != nil {
			return false
		}
	}
	return true
}

func (fc *FileClient) handle() bool {
	// store received packet index
	var receivedPackets map[int]bool
	receivedPackets = make(map[int]bool)

	isFinished := false
	defer fc.fp.Close()

	for {
		if len(receivedPackets) == fc.FilePackets {
			isFinished = true
			fc.isOver = true
			break
		}

		select {
		case data := <-fc.channel:
			var fdata model.FileData
			json.Unmarshal(data, &fdata)
			if fdata.Filename == fc.Filename {
				rdata := model.ReturnData{
					Filename:    fdata.Filename,
					PacketIndex: fdata.PacketIndex,
				}
				_, err := fc.fp.WriteAt([]byte(fdata.Body), int64(fdata.FileOffset))
				fmt.Println("write data:", fdata.PacketIndex)
				if err != nil {
					fmt.Println(err)
					rdata.Status = proto.BLOCKNOTCORRENT
				} else {
					receivedPackets[fdata.PacketIndex] = true
					rdata.Status = proto.BLOCKACK
				}
				retdata, _ := json.Marshal(rdata)
				fc.UC.SendData([]byte(retdata))
			}
		case <-time.After(time.Second):
			if fc.recvTimeout {
				fc.isOver = true
				break
			} else {
				time.Sleep(time.Second)
			}
		}
		if fc.isOver == true {
			break
		}
	}
	// check md5
	if isFinished {
		md5Ctx := md5.New()
		if _, err := io.Copy(md5Ctx, fc.fp); err != nil {
			isFinished = false
			fmt.Println(err)
		}
		checkmd5 := md5Ctx.Sum(nil)
		if !(hex.EncodeToString(checkmd5) == fc.FileMD5) {
			isFinished = false
		}
	}
	return isFinished
}

func (fc *FileClient) Recv() bool {
	checkResult := fc.preCheck()
	if !checkResult {
		return false
	}

	fc.channel = make(chan []byte, 10000)
	defer close(fc.channel)

	var result bool

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		result = fc.handle()
		wg.Done()
	}()

	for {
		if fc.isOver {
			break
		}
		fc.UC.SetReadDeadLine()
		data, dataCount := fc.UC.RecvData()
		if dataCount != 0 {
			fc.channel <- data
			fc.recvTimeout = false
		} else {
			fc.recvTimeout = true
		}
	}
	wg.Wait()
	return result
}
