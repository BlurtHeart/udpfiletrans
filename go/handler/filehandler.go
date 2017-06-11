package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	fpath "path/filepath"
	"strconv"
	"strings"
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
}

func (fc FileClient) SetIsFile(status bool) {
	fc.isfile = status
}
func (fc FileClient) write() {

}

func (fc FileClient) Recv() bool {
	// open file
	if !fc.isfile {
		_, err := os.Stat(fc.FilePath)
		b := err == nil || os.IsExist(err)
		if !b {
			if err = os.Mkdir(fc.FilePath, 0755); err != nil {
				return false
			}
		}
		realfile := fpath.Join(fc.FilePath, fc.Filename)
		fc.fp, err = os.OpenFile(realfile, os.O_RDWR|os.O_CREATE, 0655)
		if err != nil {
			return false
		}
		defer fc.fp.Close()
		fc.isfile = true
	} else {
		realfile := fpath.Join(fc.FilePath, fc.Filename)
		var err error
		fc.fp, err = os.OpenFile(realfile, os.O_RDWR, 0655)
		if err != nil {
			return false
		}
		defer fc.fp.Close()
	}

	// store received packet index
	var receivedPackets map[int]bool
	receivedPackets = make(map[int]bool)
	// recv file
	isFinished := false
	for {
		for i := 0; i < fc.FilePackets; i++ {
			if _, ok := receivedPackets[i]; ok {
				if i == fc.FilePackets-1 {
					isFinished = true
				}
			} else {
				break
			}
		}
		fc.UC.SetReadDeadLine()
		data, dataCount := fc.UC.RecvData()
		if dataCount != 0 {
			var fdata model.FileData
			json.Unmarshal(data, &fdata)
			if fdata.Filename == fc.Filename {
				receivedPackets[fdata.PacketIndex] = true
				fmt.Println(receivedPackets)
				_, err := fc.fp.WriteAt([]byte(fdata.Data), int64(fdata.FileOffset))
				if err != nil {
					rdata := model.ReturnData{Filename: fdata.Filename, PacketIndex: fdata.PacketIndex, Status: proto.BLOCKNOTCORRENT}
					retdata, _ := json.Marshal(rdata)
					fc.UC.SendData([]byte(retdata))
				}
			}
		} else {
			fmt.Println("timeout")
		}
		// break
	}
	return isFinished
}
