// Package forward contains a UDP packet forwarder.
package core

import (
	"net"
	"github.com/astaxie/beego/logs"
)

var svcMap  = make(map[string]*net.UDPAddr)

func GetIpByDnsLookup(host string) string {
	logs.SetLogger(logs.AdapterConsole)
	hosts, err := net.LookupHost(host)
	if err != nil {
		logs.Error(err)
		return ""
	}
	logs.Info(hosts)
	return hosts[0]
}

func GetUdpAddrFromAddr(addr string) *net.UDPAddr {
	var err error
	udpAddr, ok := svcMap[addr]
	if ok {
		return udpAddr
	}
	logs.Error("hash map 找不到地址",addr,"进行dns查询")
	udpAddr, err = net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logs.Error(err)
		return nil
	}
	svcMap[addr] = udpAddr
	return udpAddr
}

func main()  {
	GetIpByDnsLookup("baidu.com")
}