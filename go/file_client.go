package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"

	"go/model"
	"go/proto"

	"github.com/BurntSushi/toml"
	"github.com/mreiferson/go-options"
)

var (
	flagSet = flag.NewFlagSet("udp transfer file", flag.ExitOnError)
	config  = flagSet.String("config", "", "config file")

	// client
	send_path   = flagSet.String("send-path", "", "path of sending file")
	recv_path   = flagSet.String("recv-path", "", "path of recving file")
	filename    = flagSet.String("filename", "", "file that will send")
	listen_ip   = flagSet.String("listen-ip", "127.0.0.1", "listen ip addr")
	listen_port = flagSet.Int("listen-port", 12345, "listen port")
	// server
	serveraddr = flagSet.String("serveraddr", "127.0.0.1:12345", "server address")
)

type Options struct {
	SendPath   string `flag:"send-path"`
	RecvPath   string `flag:"recv-path"`
	Filename   string `flag:"filename"`
	ListenIP   string `flag:"listen-ip"`
	ListenPort int    `flag:"listen-port"`

	ServerAddr string `flag:"serveraddr"`
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

	realFile := filepath.Join(opts.SendPath, opts.Filename)
	fp, err := os.Open(realFile)
	if err != nil {
		log.Fatalf("ERROR:failed to open file %s - %s", realFile, err.Error())
	}
	defer fp.Close()

	fileinfo, err := os.Stat(realFile)
	if err != nil {
		log.Fatalf("ERROR:failed to stat file %s - %s", realFile, err.Error())
	}

	md5Ctx := md5.New()
	if _, err := io.Copy(md5Ctx, fp); err != nil {
		fmt.Println(err)
	}
	filemd5 := hex.EncodeToString(md5Ctx.Sum(nil))

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP(opts.ListenIP),
		Port: opts.ListenPort,
	})
	if err != nil {
		fmt.Println("listen failed", err)
		return
	}
	defer conn.Close()

	serveraddr, _ := net.ResolveUDPAddr("udp", opts.ServerAddr)

	header := model.ConnectData{
		Status:      proto.SYN,
		Filename:    opts.Filename,
		FilePath:    opts.RecvPath,
		FilePackets: int((fileinfo.Size() + 1024 - 1) / 1024),
		FileMD5:     filemd5,
	}

	rdata, _ := json.Marshal(header)

	conn.WriteToUDP([]byte(rdata), serveraddr)

	// wait for ack
	data := make([]byte, 1024)
	dataCount, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		panic(err)
	}
	var ackData model.ReturnData
	json.Unmarshal(data[:dataCount], &ackData)

	filedata := model.FileData{
		Filename: opts.Filename,
		FilePath: opts.RecvPath,
	}

	buf := make([]byte, 1024)
	fp.Seek(0, 0)
	r := bufio.NewReader(fp)
	for i := 0; i < header.FilePackets; i++ {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		if 0 == n {
			break
		}
		filedata.Body = buf[:n]
		filedata.PacketIndex = i
		filedata.FileOffset = i * 1024
		filedata.Status = proto.BLOCK
		retdata, _ := json.Marshal(filedata)
		conn.WriteToUDP(retdata, remoteAddr)
	}
}
