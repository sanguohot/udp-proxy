package main

import (
	"github.com/1lann/udp-forward"
	"os"
)

func main() {
	// Forward(src, dst). It's asynchronous.
	_, err := forward.Forward("0.0.0.0:2022", "k8s-test-master.dmcld.com:20220", forward.DefaultTimeout)
	if err != nil {
		panic(err)
	}

	// Do something...
	//确保程序不退出
	os.Stdin.Read(make([]byte,1))
}