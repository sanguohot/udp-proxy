package main

import (
	core "udp-proxy/core"
	"github.com/astaxie/beego/logs"
	dao "udp-proxy/mysql"
	"net/http"
	"fmt"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Hello world!")
}

func main() {
	// Forward(src, dst). It's asynchronous.
	logs.SetLogger(logs.AdapterConsole)
	dao.InitDao()
	//core.InitDev()
	src := "0.0.0.0:4042"
	_, err := core.Forward(src, core.DefaultTimeout)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Info("UDP转发代理初始化成功,正在监听",src)
	//确保程序不退出
	//os.Stdin.Read(make([]byte,1))
	http.HandleFunc("/", helloHandler)
	logs.Info("Http listening on port 8888")
	httpError := http.ListenAndServe(":8888", nil)
	if httpError != nil {
		logs.Error(httpError)
	}
}