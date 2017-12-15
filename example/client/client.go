package main

import (
	"fmt"
	"log"
	"os"

	"github.com/blurty/sftp"
)

func main() {
	path := "./test.txt"
	c, err := sftp.NewClient("127.0.0.1:22345")
	checkErrorOnExit(err)
	file, err := os.Open(path)
	checkErrorOnExit(err)
	rf, err := c.Send("test.txt", "octet")
	checkErrorOnExit(err)
	n, err := rf.ReadFrom(file)
	checkErrorOnExit(err)
	fmt.Printf("%d bytes sent\n", n)
}

func checkErrorOnExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
