package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
)

type dnsforward struct {
	//	dnsrequestchan chan []byte
	//dnsresponschan chan []byte
	conns []map[uint32]*net.UDPAddr //transaction ID
}

var dns = &dnsforward{}
var hash = NewHashRing(50)

func main() {

	conf.init()
	connforwards.init(conf.remoteaddr)
	connforwards2.init(conf.remoteaddr2)

	nodeWeight := make(map[string]int)
	nodeWeight["node1"] = 1
	nodeWeight["node2"] = 1
	nodeWeight["node3"] = 1
	//	vitualSpots := 100
	//	hash := NewHashRing(vitualSpots)

	//	dns.dnsrequestchan = make(chan []byte, 100)
	//	dns.dnsresponschan = make(chan []byte, 100)
	dns.conns = make([]map[uint32]*net.UDPAddr, connforwards.connsnum)
	for i := range connforwards.connchans {
		dns.conns[i] = make(map[uint32]*net.UDPAddr)
	}
	//	go dns.forwardudp()
	dns.dnsudp()
	//hashtest()

}

func hashtest() {
	// virtualSpots means virtual spots created by each node
	nodeWeight := make(map[string]int)
	nodeWeight["node1"] = 1
	nodeWeight["node2"] = 1
	nodeWeight["node3"] = 1
	vitualSpots := 100
	hash := NewHashRing(vitualSpots)

	//add nodes
	hash.AddNodes(nodeWeight)

	//remove node
	//	hash.RemoveNode("node3")

	//add node
	hash.AddNode("node4", 1)
	hash.AddNode("node5", 1)
	hash.AddNode("node6", 1)
	hash.AddNode("node6", 1)

	//get key's node
	node := hash.GetNode("192.168.0.75")

	fmt.Println("node:", node)
	node = hash.GetNode("192.168.0.74")
	fmt.Println("node:", node)
	node = hash.GetNode("192.168.0.73")
	fmt.Println("node:", node)
	node = hash.GetNode("192.168.0.72")
	node = hash.GetNode("192.168.0.71")
	fmt.Println("node:", node)
	node = hash.GetNode("192.168.0.71")

	fmt.Println("node:", node)

}

func (d *dnsforward) dnsudp() {
	// 创建监听
	log.Println("DNS forword UDP Listening:", conf.localudpport)
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: conf.localudpport,
	})
	if err != nil {
		log.Println("UDP监听失败!", err)
		return
	}
	defer socket.Close()

	for i := range connforwards.connchans {
		go func(i int) {
			for {
				//	log.Println("respone send to client :", i)
				senddata := <-connforwards.connchans[i].dnsresponschan
				//	log.Println("respone send to client :", senddata)
				if c, ok := d.conns[i][bytestoInt16LE(senddata[0:2])]; ok {
					_, err = socket.WriteToUDP(senddata, c)
					delete(d.conns[i], bytestoInt16LE(senddata[0:2]))
					if err != nil {
						fmt.Println("发送数据失败!", err)
						//	return
					}
				}
			}
		}(i)
	}

	for {
		// 读取数据
		data := make([]byte, 4096)
		read, remoteAddr, err := socket.ReadFromUDP(data)
		if err != nil {
			log.Println("UDP读取数据失败!", err)
			continue
		}

		//	d.conns[bytestoInt16LE(data[0:2])] = remoteAddr

		node := hash.GetNode(remoteAddr.IP.String())
		i, err := strconv.ParseInt(node, 10, 32)
		if err != nil {
			panic(err)
		}

		d.conns[i][bytestoInt16LE(data[0:2])] = remoteAddr

		requestchan := connforwards.connchans[i].dnsrequestchan
		requestchan2 := connforwards2.connchans[i].dnsrequestchan
		//	hash.AddNode(remoteAddr.IP.String(), 1)

		domain := data[12:read]
		//	fmt.Println("domain:", domain)
		dns := ""
		lenght := 12
		for domain[0] != 0 {
			lenght = lenght + int(domain[0]) + 1
			dns = dns + string(domain[1:domain[0]+1]) + "."
			domain = domain[domain[0]+1:]

		}
		// fmt.Println("DNS query:", read, remoteAddr.String(),
		// 	"id:", bytestoInt16LE(data[0:2]),
		// 	"query:", dns,
		// )

		//	fmt.Println("type:", data[lenght+2])
		if data[lenght+2] != 1 {
			requestchan <- data[0:read]
			requestchan2 <- data[0:read]
			continue
		}

		addrec := []byte{0x00, 0x00, 0x29, 0x10, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x0b, 0x00, 0x08, 0x00, 0x07,
			0x00, 0x01, 0x18, 0x00}
		addrec = append(addrec, conf.ednssubnet...)

		if bytes.Equal(data[10:12], []byte{0x00, 0x00}) {

			data[11] = 0x01
			newdata := append(data[0:read], addrec...)
			requestchan <- newdata
			requestchan2 <- newdata
			// fmt.Println("dns ....", read, remoteAddr.String(),
			// 	"id:", bytestoInt16LE(data[0:2]),
			// 	"query:", dns,
			// )

		} else {
			if bytes.Equal(data[10:12], []byte{0x00, 0x01}) {
				newdata := append(data[0:lenght+5], addrec...)

				// fmt.Println("dns ....", read, remoteAddr.String(),
				// 	"id:", bytestoInt16LE(data[0:2]),
				// 	"query:", dns,
				// )
				requestchan <- newdata
				requestchan2 <- newdata
			}
		}
	}

}
