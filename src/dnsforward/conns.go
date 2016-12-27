package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

type connchan struct {
	dnsrequestchan chan []byte
	dnsresponschan chan []byte
}

type conns struct {

	//connlist  []net.Conn
	connchans []*connchan
}

var connforwards1 = &conns{}
var connforwards2 = &conns{}
var connlocal1 = &conns{}
var connlocal2 = &conns{}

func (c *conns) init(remoteaddr string) {
	//	c.connsnum = conf.connpoolsize
	//	c.connlist = make([]net.Conn, c.connsnum)
	c.connchans = make([]*connchan, conf.connpoolsize)

	for i := range c.connchans {
		id := strconv.Itoa(i)
		hash.AddNode(id, 1)
		c.connchans[i] = &connchan{}
		c.connchans[i].dnsrequestchan = make(chan []byte, 100)
		c.connchans[i].dnsresponschan = make(chan []byte, 100)
		go c.forwardudp(c.connchans[i], remoteaddr, i)
	}
}

// MurMurHash算法 :https://github.com/spaolacci/murmur3
// func (c *conns) hashStr(key string) uint32 {
// 	return crc32.ChecksumIEEE([]byte(key))
// }

func (c *conns) forwardudp(connchan *connchan, remoteadr string, chanid int) {
	// 创建监听
	conn, err := net.Dial("udp", remoteadr)
	defer conn.Close()
	if err != nil {
		os.Exit(1)
	}
	log.Println("goruner:", chanid, conn.LocalAddr(), conn.RemoteAddr())

	go func() {
		for {
			msg := <-connchan.dnsrequestchan
			//	fmt.Println("query2:", msg)
			conn.Write(msg)
			//conn.Write(<-connchan.dnsrequestchan)
		}
	}()

	for {
		var dnsrespons = make([]byte, 2048)
		read, _ := conn.Read(dnsrespons)
		//	fmt.Println("respons is+++", read, dnsrespons[0:read])
		if read == 0 {
			log.Println("query error，dns server respons timeout", chanid, conn.RemoteAddr())
			continue
		}
		connchan.dnsresponschan <- dnsrespons[:read]
		//	fmt.Println("respons is---", chanid, read, conn.RemoteAddr(), dnsrespons[0:read])
	}
}

func bytestoInt16LE(data []byte) uint32 {
	var x uint16
	binary.Read(bytes.NewBuffer(data), binary.BigEndian, &x)
	return uint32(x)
}
