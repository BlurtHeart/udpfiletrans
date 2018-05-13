package main

import (
	"fmt"
	"log"

	"github.com/blurty/sftp"
)

func main() {
	path := "./test.txt"
	client, err := sftp.NewClient("127.0.0.1:22345")
	if err != nil {
		log.Fatal(err)
	}
	sentBytes, err := client.SendFile(path)
	fmt.Printf("send file %s bytes:%d, err:%v", path, sentBytes, err)
}
