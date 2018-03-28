// Package forward contains a UDP packet forwarder.
package core

import (
	"net"
	dao "udp-proxy/mysql"
	"github.com/astaxie/beego/logs"
	"fmt"
	"regexp"
)

type Backend struct {
	sn string
	svc string
}

const REGISTER_MSG_LEN = 1308
const REGISTER_MSG_SN_OFFSET = 12
const REGISTER_MSG_SN_LEN = 8
const REGISTER_MSG_HEAD_LEN = 2
var v  = make(map[string]string)

func IsValidSn(sn string)bool  {
	re:= regexp.MustCompile("[a-fA-F0-9]{4}(-)[a-fA-F0-9]{4}(-)[a-fA-F0-9]{4}(-)[a-fA-F0-9]{4}")
	if result:=re.FindString(sn);result!="" {
		return true
	}
	return false
}

func GetDstFromOffset(stream []byte) *net.UDPAddr  {
	var dstAddr string
	sn := FindSnByOffset(stream)
	if sn == "" {
		logs.Error("找不到合法的设备序列号,直接返回")
		return nil
	}
	backend := FindBackendBySn(sn)
	if backend != nil {
		dstAddr = fmt.Sprintf("%s:%d", backend.svc, defaultDstPort)
	}else {
		dstAddr = fmt.Sprintf("%s:%d", dao.GetDefaultSvcName(), defaultDstPort)
	}
	if dstAddr == "" {
		logs.Error("找不到服务")
		return nil
	}

	dst, err := net.ResolveUDPAddr("udp", dstAddr)
	if err != nil {
		logs.Error(err)
		return nil
	}
	return dst
}

//从报文中解析出sn，要求该报文是为注册报文
func FindSnByOffset(stream []byte)  string{
	//hexStr := fmt.Sprintf("%x", stream)
	length := len(stream)
	//这里需要再偏移两个字节
	if length != REGISTER_MSG_LEN+REGISTER_MSG_HEAD_LEN {
		logs.Error("非法注册报文长度",length,fmt.Sprintf("%x", stream))
		return ""
	}

	snArr := stream[REGISTER_MSG_HEAD_LEN+REGISTER_MSG_SN_OFFSET : REGISTER_MSG_HEAD_LEN+REGISTER_MSG_SN_OFFSET+REGISTER_MSG_SN_LEN]
	sn0 := snArr[0:2]
	sn1 := snArr[2:4]
	sn2 := snArr[4:6]
	sn3 := snArr[6:8]
	sn := fmt.Sprintf("%x-%x-%x-%x", sn0,sn1,sn2,sn3)
	logs.Info("获取到原始设备序列号",sn)
	//这里校验非常必要，因为有可能每个字节超出合法值
	if !IsValidSn(sn) {
		return ""
	}
	return sn
}

func FindBackendBySn(sn string)*Backend {
	svc, ok := v[sn]
	if !ok {
		//进行mysql查询更新map
		go FindAndUpdateBackendFromDb(sn)
		return nil
	}
	return &Backend{sn, svc}
}

func FindAndUpdateBackendFromDb(sn string)  {
	backend := dao.GetSvcNameBySn(sn)
	if backend != "" {
		v[sn] = backend
		logs.Info("从数据库更新设备成功",sn,backend)
		return
	}
}