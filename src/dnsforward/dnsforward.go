package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go get github.com/miekg/dns

type domainitem struct {
	tracactionid []byte
	domainname   string
	querytype    byte
	ttl          int
	createtime   int64
	packet       []byte
	querylenght  int
}

type dnsforward struct {
	domainmap map[string]*domainitem
	lock      sync.RWMutex
	lockconn  sync.RWMutex
	conns     []map[uint32]*net.UDPAddr //transaction ID
}

var dns = &dnsforward{}
var hash = NewHashRing(50)

func main() {

	conf.init()
	connforwards1.init(conf.remotednsaddr1)

	if conf.remotednsaddr2 != "" {
		connforwards2.init(conf.remotednsaddr2)
	}

	if conf.localdnsaddr1 != "" {
		connlocal1.init(conf.localdnsaddr1)
	}

	if conf.localdnsaddr2 != "" {
		connlocal2.init(conf.localdnsaddr2)
	}

	dns.domainmap = make(map[string]*domainitem)

	dns.conns = make([]map[uint32]*net.UDPAddr, conf.connpoolsize)
	for i := range connforwards1.connchans {
		dns.conns[i] = make(map[uint32]*net.UDPAddr)
	}

	dns.dnsudp()

}

/*
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
*/

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

	for i := range connforwards1.connchans {
		go d.reciverespons(i, connforwards1, socket)

		if conf.remotednsaddr2 != "" {
			go d.reciverespons(i, connforwards2, socket)
		}

		if conf.localdnsaddr1 != "" {
			go d.reciverespons(i, connlocal1, socket)
		}

		if conf.localdnsaddr2 != "" {
			go d.reciverespons(i, connlocal2, socket)
		}
	}

	var requestchan1, requestchan2, requestchan3, requestchan4 chan []byte
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
		d.lockconn.Lock()
		d.conns[i][bytestoInt16LE(data[0:2])] = remoteAddr
		d.lockconn.Unlock()

		requestchan1 = connforwards1.connchans[i].dnsrequestchan
		if conf.remotednsaddr2 != "" {
			requestchan2 = connforwards2.connchans[i].dnsrequestchan
		}
		if conf.localdnsaddr1 != "" {
			requestchan3 = connlocal1.connchans[i].dnsrequestchan
		}
		if conf.localdnsaddr2 != "" {
			requestchan4 = connlocal2.connchans[i].dnsrequestchan
		}
		//	hash.AddNode(remoteAddr.IP.String(), 1)

		domain := d.getdomain(data[:read])
		d.lock.RLock()
		if dns, ok := d.domainmap[domain.domainname]; conf.cache && domain.querytype == 1 && ok {
			d.lock.RUnlock()
			timeinter := time.Now().Unix() - dns.createtime
			if timeinter < int64(dns.ttl) {
				dns.ttl = dns.ttl - int(timeinter)
				if dns.ttl < 0 {
					d.lock.Lock()
					delete(d.domainmap, domain.domainname)
					d.lock.Unlock()
				}

				copy(dns.packet, domain.tracactionid)
				_, err = socket.WriteToUDP(dns.packet, remoteAddr)
				fmt.Println("cache respone:", dns.domainname)
				if err != nil {
					fmt.Println("send dns packet error :", err)
					//	return
				}

				continue

			}
		} else {
			d.lock.RUnlock()
		}

		if conf.localdnsaddr1 != "" || conf.localdnsaddr2 != "" {

			if conf.localdomain.MatchString(domain.domainname) {
				fmt.Println("match local domain:", domain.domainname)
				if conf.localdnsaddr1 != "" {
					requestchan3 <- data[0:read]
				}
				if conf.localdnsaddr2 != "" {
					requestchan4 <- data[0:read]
				}
				continue
			}
		}

		if domain.querytype != 1 {
			requestchan1 <- data[0:read]
			if conf.remotednsaddr2 != "" {
				requestchan2 <- data[0:read]
			}
			continue
		}

		addrec := []byte{0x00, 0x00, 0x29, 0x10, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x0b, 0x00, 0x08, 0x00, 0x07,
			0x00, 0x01, 0x18, 0x00}
		addrec = append(addrec, conf.ednssubnet...)

		if bytes.Equal(data[10:12], []byte{0x00, 0x00}) {

			data[11] = 0x01
			newdata := append(data[0:read], addrec...)
			requestchan1 <- newdata
			if conf.remotednsaddr2 != "" {
				requestchan2 <- newdata
			}
			// fmt.Println("UDP Radiuscast ....", read, remoteAddr.String(),
			// 	"id:", bytestoInt16LE(data[0:2]),
			// 	"query:", dns,
			// )

		} else {
			if bytes.Equal(data[10:12], []byte{0x00, 0x01}) {
				newdata := append(data[0:domain.querylenght], addrec...)

				// fmt.Println("UDP Radiuscast ....", read, remoteAddr.String(),
				// 	"id:", bytestoInt16LE(data[0:2]),
				// 	"query:", dns,
				// )
				if conf.remotednsaddr2 != "" {
					requestchan1 <- newdata
				}
				requestchan2 <- newdata
			}
		}
	}

}

func (d *dnsforward) getdomain(packet []byte) *domainitem {

	domain := &domainitem{}
	domain.createtime = time.Now().Unix()
	domain.packet = packet
	domain.ttl = 500

	//fmt.Println("packet:", packet)
	dname := packet[12:]
	//	fmt.Println("domain:", domain)
	dns := ""
	lenght := 12
	for dname[0] != 0 {
		lenght = lenght + int(dname[0]) + 1
		dns = dns + string(dname[1:dname[0]+1]) + "."
		dname = dname[dname[0]+1:]

	}

	//	fmt.Println("domain name:", dns, lenght)
	domain.querylenght = lenght + 5
	domain.domainname = strings.TrimSuffix(dns, ".")
	domain.querytype = packet[lenght+2]
	domain.tracactionid = packet[0:2]

	return domain
}

func (d *dnsforward) reciverespons(i int, connpool *conns, conn *net.UDPConn) {
	for {
		//	log.Println("1 respone send to client :", i)
		senddata := <-connpool.connchans[i].dnsresponschan
		domain := d.getdomain(senddata)
		if domain.querytype == 1 {
			d.lock.Lock()
			d.domainmap[domain.domainname] = domain
			d.lock.Unlock()
		}
		d.lockconn.RLock()
		if c, ok := d.conns[i][bytestoInt16LE(senddata[0:2])]; ok {
			d.lockconn.RUnlock()
			_, err := conn.WriteToUDP(senddata, c)
			d.lockconn.Lock()
			delete(d.conns[i], bytestoInt16LE(senddata[0:2]))
			d.lockconn.Unlock()
			if err != nil {
				fmt.Println("1 发送数据失败!", err)
				//	return
			}
		} else {
			d.lockconn.RUnlock()
		}
	}
}
