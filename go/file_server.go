package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	fpath "path/filepath"

	"go/handler"
	"go/model"
	"go/proto"

	"github.com/BurntSushi/toml"
	"github.com/mreiferson/go-options"
)

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
		if err != nil {
			fmt.Println("read data failed!", err)
			continue
		}

		go doFunc(remoteAddr, data[:dataCount])
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
		return result, err
	}
	defer fp.Close()
	body, err := ioutil.ReadAll(fp)
	if err != nil {
		return result, err
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

func doFunc(remoteAddr *net.UDPAddr, data []byte) {
	var conndata model.ConnectData
	json.Unmarshal(data, &conndata)
	filename := conndata.Filename
	filepath := conndata.FilePath
	filemd5 := conndata.FileMD5
	if filename == "" || filepath == "" {
		return
	} else {
		udpclient := &handler.UdpClient{}
		udpclient.SetAddr(remoteAddr.String())
		udpclient.Connect()

		realfile := fpath.Join(filepath, filename)
		isExist, _ := isFileExist(realfile)
		if isExist {
			result, err := isFileSame(realfile, filemd5)
			if err != nil {
				return
			}
			if result {
				rdata := model.ReturnData{Filename: conndata.Filename, Status: proto.EXIST}
				retData, _ := json.Marshal(rdata)
				udpclient.SendData([]byte(retData))
				fmt.Println("result:", true)
			} else {
				rdata := model.ReturnData{Filename: conndata.Filename, Status: proto.ACK}
				retData, _ := json.Marshal(rdata)
				udpclient.SendData([]byte(retData))
				fileclient := handler.FileClient{
					UC:          udpclient,
					Filename:    conndata.Filename,
					FileMD5:     conndata.FileMD5,
					FilePackets: conndata.FilePackets,
					FilePath:    conndata.FilePath,
				}
				fileclient.SetIsFile(true)
				result := fileclient.Recv()
				fmt.Println("result:", result)
			}
		} else {
			rdata := model.ReturnData{Filename: conndata.Filename, Status: proto.ACK}
			retData, _ := json.Marshal(rdata)
			udpclient.SendData([]byte(retData))
			fileclient := handler.FileClient{
				UC:          udpclient,
				Filename:    conndata.Filename,
				FileMD5:     conndata.FileMD5,
				FilePackets: conndata.FilePackets,
				FilePath:    conndata.FilePath,
			}
			result := fileclient.Recv()
			fmt.Println("result:", result)
		}
	}
}
