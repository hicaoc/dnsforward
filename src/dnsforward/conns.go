package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
)

type connchan struct {
	dnsrequestchan chan []byte
	dnsresponschan chan []byte
}

type conns struct {
	connsnum int
	//connlist  []net.Conn
	connchans []*connchan
}

var connforwards = &conns{}

func (c *conns) init() {
	c.connsnum = 10
	//	c.connlist = make([]net.Conn, c.connsnum)
	c.connchans = make([]*connchan, c.connsnum)

	for i := range c.connchans {
		id := strconv.Itoa(i)
		hash.AddNode(id, 1)
		c.connchans[i] = &connchan{}
		c.connchans[i].dnsrequestchan = make(chan []byte, 100)
		c.connchans[i].dnsresponschan = make(chan []byte, 100)
		go c.forwardudp(c.connchans[i])
	}
}

// MurMurHash算法 :https://github.com/spaolacci/murmur3
// func (c *conns) hashStr(key string) uint32 {
// 	return crc32.ChecksumIEEE([]byte(key))
// }

func (c *conns) forwardudp(connchan *connchan) {
	// 创建监听
	conn, err := net.Dial("udp", conf.remoteaddr)
	defer conn.Close()
	if err != nil {
		os.Exit(1)
	}
	log.Println("localaddr:", conn.LocalAddr().String())

	go func() {
		for {
			//	msg := <-connchan.dnsrequestchan
			conn.Write(<-connchan.dnsrequestchan)
		}
	}()

	for {
		var dnsrespons = make([]byte, 2048)
		read, _ := conn.Read(dnsrespons)
		//	log.Println("respons is", read, dnsrespons[0:read])
		connchan.dnsresponschan <- dnsrespons[:read]
		//	fmt.Println("msg is", remote, dnsrespons)
	}
}

func bytestoInt16LE(data []byte) uint32 {
	var x uint16
	binary.Read(bytes.NewBuffer(data), binary.BigEndian, &x)
	return uint32(x)
}
