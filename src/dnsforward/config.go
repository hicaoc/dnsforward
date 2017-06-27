package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	//	"fmt"

	"bytes"
	"strconv"
)

type config struct {
	localudpport      int
	remotednsaddr1    string
	remotednsaddr2    string
	localdnsaddr1     string
	localdnsaddr2     string
	ednssubnet        []byte
	cache             bool
	connpoolsize      int
	localdomainlist   string
	outsidedomainlist string
	localdomain       *regexp.Regexp
	outsidedomain     *regexp.Regexp
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

	//	log.Println("read config file dnsforward.ini ......")

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
		case "remotednsaddr1":
			c.remotednsaddr1 = s[1]
		case "remotednsaddr2":
			c.remotednsaddr2 = s[1]
		case "localdnsaddr1":
			c.localdnsaddr1 = s[1]
		case "localdnsaddr2":
			c.localdnsaddr2 = s[1]
		case "connpoolsize":
			c.connpoolsize = strtoint(s[1])
		case "ednssubnet":
			c.ednssubnet = striptosubnet(s[1])
		case "cache":
			if string(s[1]) == "true" {
				c.cache = true
			}
		case "localdomainlist":
			c.localdomainlist = s[1]
			if conf.localdnsaddr1 != "" {
				var err error
				if c.localdomain, err = conf.readdomainlist(s[1]); err != nil {
					log.Println("Read localdomainlist err:", err)
				}
			}

		case "outsidedomainlist":
			c.outsidedomainlist = s[1]
			var err error
			if c.outsidedomain, _ = conf.readdomainlist(s[1]); err != nil {
				log.Println("Read outsidedomainlist err:", err)
			}

		}

	}
	if c.localudpport == 0 {
		c.localudpport = 53
	}

	if c.connpoolsize == 0 {
		c.connpoolsize = 5
	}

	if c.remotednsaddr1 == "" {
		log.Println("config err,not found remotednsaddr1 config ,\n\tusage: remotednsaddr1=8.8.8.8:53")
		os.Exit(1)
	}

	log.Println("Read dnsforward conf file ok ")
}

func (c *config) readdomainlist(s string) (*regexp.Regexp, error) {

	log.Println("read  domainlist  file ", s, " ......")

	f, err := os.Open(s)
	if err != nil {
		log.Println("open ", s, " err!")
		os.Exit(1)
	}
	defer f.Close()

	rd := bufio.NewReader(f)

	strbuffer := bytes.Buffer{}

	for {

		line, err := rd.ReadBytes('\n') //以'\n'为结束符读入一行
		//	fmt.Println("str1:", line)

		if err != nil || io.EOF == err {
			//	log.Println("read basinfo file error :", err)
			break
		}

		if bytes.HasPrefix(line, []byte("#")) {
			continue
		}

		strbuffer.Write(line)

	}

	log.Println("read domainlist  file ", s, " ok ")

	return regexp.Compile(strings.TrimRight(strings.Replace(strbuffer.String(), "\n", "|", -1), "|"))

	//	fmt.Println("str2:", str)

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
