package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	//	"fmt"

	"strconv"
)

type config struct {
	localudpport int
	remoteaddr   string
	ednssubnet   []byte
}

var conf = &config{}

func (c *config) init() {

	conf.readconffile()
	//	go c.cronread()

}

func (c *config) cronread() {
	for {
		time.Sleep(time.Minute * 5)
		c.readconffile()

	}
}

func (c *config) readconffile() {

	log.Println("read config file dnsforward.ini ......")

	f, err := os.Open("./dnsforward.ini")
	if err != nil {
		log.Println("打开dnsforward.ini配置文件错误:", err)
		os.Exit(1)
	}
	defer f.Close()

	rd := bufio.NewReader(f)

	for {

		line, err := rd.ReadString('\n') //以'\n'为结束符读入一行

		if err != nil || io.EOF == err || line == ".\n" {
			//	log.Println("read basinfo file error :", err)
			break
		}

		s := strings.Split(strings.TrimSuffix(line, "\n"), "=")

		switch s[0] {
		case "localudpport":
			c.localudpport = strtoint(s[1])
		case "remoteaddr":
			c.remoteaddr = s[1]
		case "ednssubnet":

			c.ednssubnet = striptosubnet(s[1])

		}

	}
	log.Println("Read dnsforward conf file ok ", c.localudpport, c.remoteaddr, c.ednssubnet)
}

func strtoint(a string) int {
	i, err := strconv.Atoi(a)
	if err != nil {
		log.Println("字符串转换成整数失败", i, err)
	}
	return i
}

func striptosubnet(str string) []byte {

	log.Println("")
	_, i, err := net.ParseCIDR(str)
	if err != nil {
		log.Println("subnet format err, usage: 192.168.0.0/24", err)
	}

	//	fmt.Println("subnet:", ipp, []byte(i.IP), i.Mask, i.Network(), i.String())
	return i.IP[:3]

}
