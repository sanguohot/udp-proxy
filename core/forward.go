package core

import (
	"github.com/astaxie/beego/logs"
	"net"
	"sync"
	"time"
)

const bufferSize = 1024 * 32

type connection struct {
	udp        *net.UDPConn
	lastActive time.Time
	dst        *net.UDPAddr
	sn		   string
}

// 转发服务对象
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

// 默认超时时间是根据设备1分钟没有心跳就通讯失败设置的，超时就拆除连接
var DefaultTimeout = time.Minute * 1

// 转发服务监听设备端的连接，根据设备序列号转发往目标simserver
// 含建立连接和超时断开连接回调
// 实现反向穿透、透明传输、多设备连接
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
		logs.Info("已知设备创建新的连接",sn,addr.String(),"已连接的设备数",len(f.connections))
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
			sn:			sn,
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
			time.Now().Add(-f.timeout)) {
			shouldChangeTime = true
			//logs.Info("更新超时时间",f.connections[addr.String()].sn,addr.String(),f.connections[addr.String()].lastActive,data)
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

// 当监听client连接时触发
func (f *Forwarder) OnConnect(callback func(addr string)) {
	f.connectCallback = callback
}

// 当拆除client连接时触发
func (f *Forwarder) OnDisconnect(callback func(addr string)) {
	f.disconnectCallback = callback
}

// 返回IP:port格式的已连接地址数组
func (f *Forwarder) Connected() []string {
	f.connectionsMutex.Lock()
	results := make([]string, 0, len(f.connections))
	for key, _ := range f.connections {
		results = append(results, key)
	}
	return results
}
