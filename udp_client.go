package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

var raddr = flag.String("raddr", "server02.dmcld.com:12022", "remote server address")

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
}

func listen(conn *net.UDPConn)  {
	for {
		buf := make([]byte, 1024)
		rn, rmAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Printf("<<<  %d bytes received from: %v, data: %s\n", rn, rmAddr, string(buf[:rn]))
		}
	}
}
func main() {
	// 解析dns
	remoteAddr, err := net.ResolveUDPAddr("udp", *raddr)
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	// 设置本地地址
	localAddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 0,
	}

	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	// 出错退出
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	defer conn.Close()

	// 开始发送消息
	_, err = conn.Write([]byte("hello world"))
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(">>> Packet sent to: ", *raddr)
	}
	// 开始监听服务器返回消息
	go listen(conn)
	// 确保程序不退出
	os.Stdin.Read(make([]byte,1))
}