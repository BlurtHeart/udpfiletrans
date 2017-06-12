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

	send_path  = flagSet.String("send-path", "", "path of sending file")
	recv_path  = flagSet.String("recv-path", "", "path of recving file")
	filename   = flagSet.String("filename", "", "file that will send")
	serveraddr = flagSet.String("serveraddr", "127.0.0.1:12345", "server address")
)

type Options struct {
	SendPath   string `flag:"send-path"`
	RecvPath   string `flag:"recv-path"`
	Filename   string `flag:"filename"`
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

	r := bufio.NewReader(fp)

	saddr, err := net.ResolveUDPAddr("udp", opts.ServerAddr)
	conn, err := net.DialUDP("udp", nil, saddr)
	if err != nil {
		log.Fatalf("ERROR:failed to establish udp socket: %s", err.Error())
	}
	defer conn.Close()

	header := model.ConnectData{
		Status:      proto.SYN,
		Filename:    opts.Filename,
		FilePath:    opts.RecvPath,
		FilePackets: int((fileinfo.Size() + 1024 - 1) / 1024),
		FileMD5:     filemd5,
	}

	rdata, _ := json.Marshal(header)
	conn.Write([]byte(rdata))

	// wait for ack
	data := make([]byte, 1024)
	fmt.Println(conn.LocalAddr())
	fmt.Println(conn.RemoteAddr())
	dataCount, remoteAddr, err := conn.ReadFromUDP(data)
	if err != nil {
		panic(err)
	}
	fmt.Println("read over")
	var ackData model.ReturnData
	json.Unmarshal(data[:dataCount], &ackData)
	fmt.Println(ackData)

	conn2, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Fatalf("ERROR:failed to establish udp socket: %s", err.Error())
	}
	defer conn2.Close()

	filedata := model.FileData{
		Filename: opts.Filename,
		FilePath: opts.RecvPath,
	}
	buf := make([]byte, 1024)
	for i := 0; i < header.FilePackets; i++ {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		if 0 == n {
			break
		}
		filedata.Body = string(buf[:n])
		filedata.PacketIndex = i
		filedata.FileOffset = i * 1024
		retdata, _ := json.Marshal(filedata)
		conn2.Write([]byte(retdata))
	}
}
