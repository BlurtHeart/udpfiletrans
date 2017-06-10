package handler

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
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

func (uc UdpClient) SendData(data []byte) {
	uc.Conn.Write(data)
}

func (uc UdpClient) RecvData() ([]byte, int) {
	data := make([]byte, 4096)
	dataCount, _ := uc.Conn.Read(data)
	return data[:dataCount], dataCount
}

type FileClient struct {
	Filename string
	FileMD5  string
	UC       *UdpClient
}

func (fc FileClient) Recv() {
	for {
		data, dataCount := fc.UC.RecvData()
		fmt.Println(data, dataCount)
		break
	}
}
