// Package forward contains a UDP packet forwarder.
package core

import (
	"net"
	"github.com/astaxie/beego/logs"
	"fmt"
	"regexp"
	"strings"
	"errors"
)

type Backend struct {
	sn string
	svc string
}

const (
	defaultDstPort = 12022
	REGISTER_MSG_LEN = 1308
	REGISTER_MSG_SN_OFFSET = 12
	REGISTER_MSG_SN_LEN = 8
	REGISTER_MSG_HEAD_LEN = 0
)

var (
	devBackendMap  = make(map[string]string)
	notFoundMap = make(map[string]int)
)

func IsValidSn(sn string)bool  {
	re:= regexp.MustCompile("[a-fA-F0-9]{4}(-)[a-fA-F0-9]{4}(-)[a-fA-F0-9]{4}(-)[a-fA-F0-9]{4}")
	if result:=re.FindString(sn);result!="" {
		return true
	}
	return false
}

func GetDstAndSnFromOffset(stream []byte) (*net.UDPAddr,string,error)  {
	var dstHost string
	sn := FindSnByOffset(stream)
	if sn == "" {
		return nil,"",errors.New("sn not found")
	}
	backend := FindBackendBySn(sn)
	if backend == nil {
		return nil,"",errors.New("backend not found")
	}
	if backend.svc == "" {
		//这里先发往默认服务,后续再优化
		dstHost = GetDefaultSvcName()
	}else {
		dstHost = backend.svc
	}
	if dstHost == "" {
		logs.Error("找不到服务")
		return nil,"",errors.New("svc not found")
	}
	dst,err := GetUdpAddrFromAddr(fmt.Sprintf("%s:%d", dstHost,defaultDstPort))
	if err != nil {
		return nil,"",err
	}
	//未知设备的原因导致打印太多
	//logs.Info("目标地址",dst,sn,len(stream))
	return dst,sn,nil
}

//从报文中解析出sn，要求该报文是为注册报文
func FindSnByOffset(stream []byte)  string{
	//hexStr := fmt.Sprintf("%x", stream)
	length := len(stream)
	if length<REGISTER_MSG_SN_LEN+REGISTER_MSG_SN_OFFSET {
		logs.Error("非法报文长度",length,fmt.Sprintf("%x", stream))
		return ""
	}
	snArr := stream[REGISTER_MSG_HEAD_LEN+REGISTER_MSG_SN_OFFSET : REGISTER_MSG_HEAD_LEN+REGISTER_MSG_SN_OFFSET+REGISTER_MSG_SN_LEN]
	sn0 := snArr[0:2]
	sn1 := snArr[2:4]
	sn2 := snArr[4:6]
	sn3 := snArr[6:8]
	sn := fmt.Sprintf("%x-%x-%x-%x", sn0,sn1,sn2,sn3)
	sn = strings.ToLower(sn)
	//这里校验非常必要，因为有可能每个字节超出合法值
	if !IsValidSn(sn) {
		logs.Error("非法设备序列号",sn)
		return ""
	}
	return sn
}

func FindBackendBySn(sn string)*Backend {
	svc, ok := devBackendMap[sn]
	if !ok {
		//未知设备冲击
		//进行mysql查询更新map
		//go FindAndUpdateBackendFromDb(sn)
		return nil
	}
	return &Backend{sn, svc}
}

func FindAndUpdateBackendFromDb(sn string)  {
	_,ok := unknownDevMap[sn]
	if ok {
		return
	}
	backend := GetSvcNameBySn(sn)
	if backend != "" {
		devBackendMap[sn] = backend
		logs.Info("从数据库更新设备成功",sn,backend)
		return
	}
	//未知设备冲击
	backend = GetDefaultSvcName()
	if backend!="" {
		devBackendMap[sn] = backend
		logs.Info("找不到服务名，设置为默认服务名",sn,backend)
	}
	return
}

func InitDev()  {
	logs.SetLogger(logs.AdapterConsole)
	//domainMap := dao.GetAllDomain()
	//sysMap := dao.GetAllSys()
	//devBackendMap := dao.GetAllDev()
	//logs.Info(domainMap,sysMap,devBackendMap)
}