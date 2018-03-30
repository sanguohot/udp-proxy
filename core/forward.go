// Package forward contains a UDP packet forwarder.
package core

import (
	"github.com/astaxie/beego/logs"
	"net"
	"sync"
	"time"
)

const bufferSize = 1024 * 4

type connection struct {
	udp        *net.UDPConn
	lastActive time.Time
	dst        *net.UDPAddr
}

// Forwarder represents a UDP packet forwarder.
type Forwarder struct {
	src          *net.UDPAddr
	dst          *net.UDPAddr
	client       *net.UDPAddr
	listenerConn *net.UDPConn

	connections      map[string]connection
	connectionsMutex *sync.RWMutex

	connectCallback    func(addr string)
	disconnectCallback func(addr string)

	timeout time.Duration

	closed bool
}

// DefaultTimeout is the default timeout period of inactivity for convenience
// sake. It is equivelant to 5 minutes.
var DefaultTimeout = time.Minute * 2

// Forward forwards UDP packets from the src address to the dst address, with a
// timeout to "disconnect" clients after the timeout period of inactivity. It
// implements a reverse NAT and thus supports multiple seperate users. Forward
// is also asynchronous.
func Forward(src string, timeout time.Duration) (*Forwarder, error) {
	logs.SetLogger(logs.AdapterConsole)

	forwarder := new(Forwarder)
	forwarder.connectCallback = func(addr string) {}
	forwarder.disconnectCallback = func(addr string) {}
	forwarder.connectionsMutex = new(sync.RWMutex)
	forwarder.connections = make(map[string]connection)
	forwarder.timeout = timeout

	var err error
	forwarder.src, err = net.ResolveUDPAddr("udp", src)
	if err != nil {
		return nil, err
	}

	forwarder.client = &net.UDPAddr{
		IP:   forwarder.src.IP,
		Port: 0,
		Zone: forwarder.src.Zone,
	}

	forwarder.listenerConn, err = net.ListenUDP("udp", forwarder.src)
	if err != nil {
		return nil, err
	}

	go forwarder.janitor()
	go forwarder.run()

	return forwarder, nil
}

func (f *Forwarder) run() {
	for {
		buf := make([]byte, bufferSize)
		n, addr, err := f.listenerConn.ReadFromUDP(buf)
		if err != nil {
			logs.Error(err,"读取客户端报文出错，直接返回")
			return
		}
		//logs.Info("收到报文",addr)
		go f.handle(buf[:n], addr)
	}
}

func (f *Forwarder) janitor() {
	for !f.closed {
		time.Sleep(f.timeout)
		var keysToDelete []string

		f.connectionsMutex.RLock()
		for k, conn := range f.connections {
			if conn.lastActive.Before(time.Now().Add(-f.timeout)) {
				keysToDelete = append(keysToDelete, k)
			}
		}
		f.connectionsMutex.RUnlock()

		f.connectionsMutex.Lock()
		for _, k := range keysToDelete {
			f.connections[k].udp.Close()
			delete(f.connections, k)
		}
		f.connectionsMutex.Unlock()

		for _, k := range keysToDelete {
			f.disconnectCallback(k)
		}
	}
}

func (f *Forwarder) handle(data []byte, addr *net.UDPAddr) {
	f.connectionsMutex.RLock()
	conn, found := f.connections[addr.String()]
	f.connectionsMutex.RUnlock()
	//logs.Info("是否找到连接",found,"连接对象为",conn)
	if !found {
		dst,sn,err := GetDstAndSnFromOffset(data)
		if err != nil {
			return
		}

		conn, err := net.ListenUDP("udp", f.client)
		if err != nil {
			logs.Error("udp-forwader: failed to dial:", err)
			return
		}

		f.connectionsMutex.Lock()
		f.connections[addr.String()] = connection{
			udp:        conn,
			lastActive: time.Now(),
			dst:        dst,
		}
		f.connectionsMutex.Unlock()

		f.connectCallback(addr.String())

		conn.WriteTo(data, dst)
		//if err != nil {
		//	logs.Error(err,"新建路的连接往服务器转发报文失败,直接返回")
		//	return
		//}

		for conn != nil {
			buf := make([]byte, bufferSize)
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				logs.Error(err,"即将关闭连接，并清除hashmap记录",sn)
				f.connectionsMutex.Lock()
				conn.Close()
				delete(f.connections, addr.String())
				conn = nil
				f.connectionsMutex.Unlock()
				return
			}

			go func(data []byte, conn *net.UDPConn, addr *net.UDPAddr) {
				f.listenerConn.WriteTo(data, addr)
			}(buf[:n], conn, addr)
		}

		return
	}

	conn.udp.WriteTo(data, conn.dst)
	//if err != nil {
	//	logs.Error(err,"hashmap保存的连接往服务器转发报文失败，直接返回")
	//	return
	//}

	shouldChangeTime := false
	f.connectionsMutex.RLock()
	//if _, found := f.connections[addr.String()]; found {
	//	if f.connections[addr.String()].lastActive.Before(
	//		time.Now().Add(f.timeout / 4)) {
	//		shouldChangeTime = true
	//	}
	//}
	if _, found := f.connections[addr.String()]; found {
		if f.connections[addr.String()].lastActive.After(
			time.Now().Add(-f.timeout/2)) {
			shouldChangeTime = true
		}
	}
	f.connectionsMutex.RUnlock()

	if shouldChangeTime {
		f.connectionsMutex.Lock()
		// Make sure it still exists
		if _, found := f.connections[addr.String()]; found {
			connWrapper := f.connections[addr.String()]
			connWrapper.lastActive = time.Now()
			f.connections[addr.String()] = connWrapper
		}
		f.connectionsMutex.Unlock()
	}
}

func (f *Forwarder) Close() {
	f.connectionsMutex.Lock()
	f.closed = true
	for _, conn := range f.connections {
		conn.udp.Close()
	}
	f.listenerConn.Close()
	f.connectionsMutex.Unlock()
}

// OnConnect can be called with a callback function to be called whenever a
// new client connects.
func (f *Forwarder) OnConnect(callback func(addr string)) {
	f.connectCallback = callback
}

// OnConnect can be called with a callback function to be called whenever a
// new client disconnects (after 5 minutes of inactivity).
func (f *Forwarder) OnDisconnect(callback func(addr string)) {
	f.disconnectCallback = callback
}

// Connected returns the list of connected clients in IP:port form.
func (f *Forwarder) Connected() []string {
	f.connectionsMutex.Lock()
	results := make([]string, 0, len(f.connections))
	for key, _ := range f.connections {
		results = append(results, key)
	}
	return results
}
