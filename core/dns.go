// Package forward contains a UDP packet forwarder.
package core

import (
	"net"
	"github.com/astaxie/beego/logs"
)

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

func main()  {
	GetIpByDnsLookup("baidu.com")
}