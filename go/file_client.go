package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	flagSet = flag.NewFlagSet()
	config  = flagSet.String("config", "", "config file")

	send_path = flagSet.String("send-path", "", "path of sending file")
	recv_path = flagSet.String("recv-path", "", "path of recving file")
	filename  = flagSet.String("filename", "", "file that will send")
)

func main() {
	flag.Int(name, value, usage)
}
