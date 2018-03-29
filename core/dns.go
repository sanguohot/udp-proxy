// Package forward contains a UDP packet forwarder.
package core

import (
	"net"
	"github.com/astaxie/beego/logs"
)

var svcMap  = make(map[string]*net.UDPAddr)

func GetIpByDnsLookup(host string) string {
	logs.SetLogger(logs.AdapterConsole)
	ips, err := net.LookupHost(host)
	if err != nil {
		logs.Error(err)
		return ""
	}
	logs.Info(host,"===>",ips)
	return ips[0]
}

func GetUdpAddrFromAddr(addr string) (*net.UDPAddr,error) {
	udpAddr, ok := svcMap[addr]
	if ok {
		return udpAddr,nil
	}
	logs.Error("hash map 找不到地址",addr,"进行dns查询")
	udpAddr,err := ResolveAndSetUdpAddrToAddr(addr)
	if err != nil {
		return nil,err
	}
	return udpAddr,nil
}

func ResolveAndSetUdpAddrToAddr(addr string) (*net.UDPAddr,error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logs.Error(err,addr)
		return nil,err
	}
	svcMap[addr] = udpAddr
	return udpAddr,nil
}

//func main()  {
//	GetIpByDnsLookup("baidu.com")
//}