package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	fpath "path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mreiferson/go-options"
)

// max concurrent 50 files
var connectionChannel chan []byte = make(chan []byte, 50)
var dataChannel chan []byte = make(chan []byte, 100000)

var (
	flagSet = flag.NewFlagSet("udp transfer file", flag.ExitOnError)
	config  = flagSet.String("config", "", "config file")

	listen_ip   = flagSet.String("listen-ip", "127.0.0.1", "listen ip addr")
	listen_port = flagSet.Int("listen-port", 12345, "listen port")
)

type Options struct {
	ListenIP   string `flag:"listen-ip"`
	ListenPort int    `flag:"listen-port"`
}

type ConnectData struct {
	Filename string
	FilePath string
	FileMD5  string
}

type FileData struct {
	FilePackets int
	PacketIndex int
	FileOffset  int
	Filename    string
	FilePath    string
	PacketMD5   string
	Data        string
}

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

func main() {
	flagSet.Parse(os.Args[1:])
	var cfg map[string]interface{}
	if *config != "" {
		_, err := toml.DecodeFile(*config, &cfg)
		if err != nil {
			log.Fatalf("ERROR:failed to load config file %s - %s", *config, err.Error())

		}
	}
	opts := &Options{}
	options.Resolve(opts, flagSet, cfg)

	// create listen socket
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP(opts.ListenIP),
		Port: opts.ListenPort,
	})
	if err != nil {
		fmt.Println("listen failed", err)
		return
	}
	defer socket.Close()

	for {
		// read data
		data := make([]byte, 4096)
		dataCount, remoteAddr, err := socket.ReadFromUDP(data)
		fmt.Println(remoteAddr)
		if err != nil {
			fmt.Println("read data failed!", err)
			continue
		}

		go sendFunc(remoteAddr)
		// put connection to channel
		connectionChannel <- data[:dataCount]
	}
}

func isFileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func isFileSame(filename string, filemd5 string) (bool, error) {
	result := false
	fp, err := os.Open(filename)
	if err != nil {
		panic("File not found")
	}
	defer fp.Close()
	body, err := ioutil.ReadAll(fp)
	if err != nil {
		panic("Read file error")
	}
	md5Ctx := md5.New()
	md5Ctx.Write(body)
	checkmd5 := md5Ctx.Sum(nil)
	if hex.EncodeToString(checkmd5) == filemd5 {
		result = true
	} else {
		result = false
	}
	return result, err
}

func sendUdp(remoteAddr *net.UDPAddr) {
	udpclient := &UdpClient{}
	udpclient.SetAddr(remoteAddr.String())
	udpclient.Connect()
	udpclient.SendData([]byte("Hello world!"))
}

func sendFunc(remoteAddr *net.UDPAddr) {
	for {
		data := <-connectionChannel

		sendUdp(remoteAddr)
		var conndata ConnectData
		json.Unmarshal(data, &conndata)
		filename := conndata.Filename
		filepath := conndata.FilePath
		filemd5 := conndata.FileMD5
		if filename == "" || filepath == "" {
			continue
		} else {
			realfile := fpath.Join(filepath, filename)
			isExist, _ := isFileExist(realfile)
			if isExist {
				result, err := isFileSame(realfile, filemd5)
				if err != nil {
					panic("open file error")
				}
				if result {

				} else {

				}
			}
		}
	}
}
