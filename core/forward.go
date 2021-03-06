package core

import (
	"github.com/astaxie/beego/logs"
	"net"
	"sync"
	"time"
)

const bufferSize = 1024 * 60

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

// 默认超时时间，超时就拆除连接
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

	//forwarder.client = &net.UDPAddr{
	//	IP:   forwarder.src.IP,
	//	//IP: net.IPv4zero,
	//	Port: 0,
	//	Zone: forwarder.src.Zone,
	//}

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
		if n > 0{
			go f.handle(buf[:n], addr)
		}
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
	addrString := addr.String()
	conn, found := f.connections[addrString]
	f.connectionsMutex.RUnlock()
	//logs.Info("是否找到连接",found,"连接对象为",conn)
	if !found {
		dst,sn,err := GetDstAndSnFromOffset(data)
		if err != nil {
			logs.Error(err)
			return
		}
		var isNewConn bool = false
		f.connectionsMutex.Lock()
		//高并发下需要再次判断有没有已经设置hash，已经存在只更新时间
		if _, found := f.connections[addrString]; !found {
			err = f.CreateConnectionAndSaveToMap(sn, addr, dst)
			if err!=nil {
				f.connectionsMutex.Unlock()
				return
			}
			isNewConn = true;
		}else {
			f.UpdateActiveTimeInSyncLock(addrString)
		}
		f.connectionsMutex.Unlock()
		f.connectCallback(addrString)
		f.connectionsMutex.RLock()
		conn, _ := f.connections[addrString]
		f.connectionsMutex.RUnlock()
		if isNewConn {
			go f.ListenServerMsg(conn.udp,addr,dst,sn)
		}
		//_,err = conn.udp.WriteTo(data, dst)
		//因为是已经建立的连接，无需指定目的地址
		_,err = conn.udp.Write(data)
		if err!=nil {
			logs.Error(err)
		}
		return
	}

	go f.UpdateActiveTime(addr)
	//_,err := conn.udp.WriteTo(data, conn.dst)
	//因为是已经建立的连接，无需指定目的地址
	_,err := conn.udp.Write(data)
	if err!=nil {
		logs.Error(err)
	}
}

func (f *Forwarder) CreateConnectionAndSaveToMap(sn string, src *net.UDPAddr, dst *net.UDPAddr) error  {
	dstString := dst.String()
	srcString := src.String()
	logs.Info("已知设备创建新的连接",sn,srcString,">>>",dstString,"已连接的设备数",len(f.connections))
	//返回unconnected连接
	//udpConn, err := net.ListenUDP("udp", f.client)
	//if err != nil {
	//	logs.Error("udp-forwader: failed to dial:", err)
	//	return
	//}
	//返回connected连接
	udpConn, err := net.DialUDP("udp", nil, dst)
	if err != nil {
		logs.Error("udp-forwader: failed to dial:", err)
		return err
	}
	f.connections[srcString] = connection{
		udp:        udpConn,
		lastActive: time.Now(),
		dst:        dst,
		sn:			sn,
	}
	return  nil
}

func (f *Forwarder) ListenServerMsg(udpConn *net.UDPConn,src *net.UDPAddr, dst *net.UDPAddr, sn string) {
	dstString := dst.String()
	srcString := src.String()
	var doCircle bool = true
	//logs.Info("开始循环监听服务器报文",dstString,">>",srcString,sn)
	for doCircle {
		buf := make([]byte, bufferSize)
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			logs.Error(err,"关闭连接",srcString,">>>",dstString,sn)
			doCircle = false;
			f.connectionsMutex.Lock()
			udpConn.Close()
			delete(f.connections, srcString)
			f.connectionsMutex.Unlock()
			return
		}
		if n>0 {
			go func(data []byte, conn *net.UDPConn, addr *net.UDPAddr) {
				f.listenerConn.WriteTo(data, addr)
			}(buf[:n], udpConn, src)
		}
	}
}

//注意该函数需要锁下执行
func (f *Forwarder) UpdateActiveTimeInSyncLock(addrString string) {
	connWrapper := f.connections[addrString]
	connWrapper.lastActive = time.Now()
	f.connections[addrString] = connWrapper
}

func (f *Forwarder) UpdateActiveTime(src *net.UDPAddr) {
	srcString := src.String()
	//shouldChangeTime := false
	//f.connectionsMutex.RLock()
	//if _, found := f.connections[addrString]; found {
	//	if f.connections[addrString].lastActive.After(
	//		time.Now().Add(-f.timeout)) {
	//		shouldChangeTime = true
	//		//logs.Info("更新超时时间",f.connections[addrString].sn,addrString,f.connections[addrString].lastActive,data)
	//	}
	//}
	//f.connectionsMutex.RUnlock()
	//
	//if shouldChangeTime {
	//	f.connectionsMutex.Lock()
	//	// Make sure it still exists
	//	if _, found := f.connections[addrString]; found {
	//		f.UpdateActiveTimeInSyncLock(addrString)
	//	}
	//	f.connectionsMutex.Unlock()
	//}
	f.connectionsMutex.Lock()
	// Make sure it still exists
	if _, found := f.connections[srcString]; found {
		f.UpdateActiveTimeInSyncLock(srcString)
	}
	f.connectionsMutex.Unlock()
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
