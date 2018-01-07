package sftp

import (
	"fmt"
	"net"
	"time"
)

type Client struct {
	addr    *net.UDPAddr
	timeout time.Duration
	retries int
	backoff backoffFunc
	blksize int
	tsize   bool
}

func NewClient(addr string) (*Client, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolving address %s: %v", addr, err)
	}
	return &Client{
		addr:    a,
		timeout: defaultTimeout,
		retries: defaultRetries,
	}, nil
}

// recv file from server
// return error if failed, otherwise, return nil
func (c *Client) RecvFile(filename string) error { return nil }

// send file to server
// return error if failed, otherwise, return nil
func (c *Client) SendFile(filename string) error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		return err
	}
	flr, err := NewFiler(filename)
	if err != nil {
		return err
	}
	s := &sender{
		receive: make([]byte, datagramLength),
		conn:    conn,
		retry:   &backoff{handler: c.backoff},
		timeout: c.timeout,
		retries: c.retries,
		addr:    c.addr,
		file:    flr,
		opcode:  opWRQ,
	}
	s.setBlockNum(0)
	s.setBlockSize(0)
	err = s.shakeHands()
	if err != nil {
		return err
	}
	// begin send file block
	err = s.sendContents()
	if err != nil {
		return err
	}
	return nil
}

// SetTimeout sets maximum time client waits for single network round-trip to succeed.
// Default is 5 seconds.
func (c *Client) SetTimeout(t time.Duration) {
	if t <= 0 {
		c.timeout = defaultTimeout
	}
	c.timeout = t
}

// SetRetries sets maximum number of attempts client made to transmit a packet.
// Default is 5 attempts.
func (c *Client) SetRetries(count int) {
	if count < 1 {
		c.retries = defaultRetries
	}
	c.retries = count
}

// SetBackoff sets a user provided function that is called to provide a
// backoff duration prior to retransmitting an unacknowledged packet.
func (c *Client) SetBackoff(h backoffFunc) {
	c.backoff = h
}
