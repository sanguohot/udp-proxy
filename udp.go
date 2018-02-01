package main
import (
"fmt"
"net"
"os"
)
func read(conn *net.UDPConn) {
	for {
		data := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			fmt.Printf("error during read: %s", err)
		}
		fmt.Printf("receive %s from <%s>\n", data[:n], remoteAddr)
		//go conn.WriteToUDP(data, remoteAddr)
	}
}
func main() {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 2023}
	go func() {
		listener, err := net.ListenUDP("udp", addr)
		if err != nil {
			fmt.Println(err)
			return
		}
		go read(listener)
	}()
	//确保程序不退出
	os.Stdin.Read(make([]byte,1))
}