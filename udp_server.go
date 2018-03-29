package main
import (
"fmt"
"net"
"os"
"time"
)

func write(conn *net.UDPConn,remoteAddr *net.UDPAddr,data []byte)  {
	//收到什么返回什么
	conn.WriteToUDP(data,remoteAddr)
	sleep := 1000 * time.Millisecond
	time.Sleep(sleep)
	buf := []byte("after sleep "+sleep.String())
	fmt.Printf("send [%s] to <%s>\n", buf, remoteAddr)
	conn.WriteToUDP(buf,remoteAddr)
}
func read(conn *net.UDPConn) {
	for {
		data := make([]byte, 4096)
		n, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			fmt.Printf("error during read: %s", err)
		}
		fmt.Printf("receive [%s] from <%s>\n", data[:n], remoteAddr)
		go write(conn, remoteAddr, data)
	}
}
func main() {
	addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 2022}
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