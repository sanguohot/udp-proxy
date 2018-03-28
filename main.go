package main

import (
	forward "udp-proxy/core"
	"github.com/astaxie/beego/logs"
	"os"
	dao "udp-proxy/mysql"
)

func main() {
	// Forward(src, dst). It's asynchronous.
	logs.SetLogger(logs.AdapterConsole)
	dao.InitDao()
	src := "0.0.0.0:4042"
	_, err := forward.Forward(src, forward.DefaultTimeout)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Info("UDP转发代理初始化成功,正在监听",src)
	//确保程序不退出
	os.Stdin.Read(make([]byte,1))
}