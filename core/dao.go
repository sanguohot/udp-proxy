package core

import (
	"github.com/astaxie/beego/logs"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"fmt"
	"errors"
	"strings"
)

var (
	unknownDevMap map[string]TblNeNa
	defaultSvcName string
)

type TblNe struct {
	ProductSns string
	DomainUuid int
	DomainName string
}

type TblNeNa struct {
	Alias string
	DomainUuid int
}

type TblSys struct {
	SvcName string
	DefaultFlag int
	Uuid string
}

type TblDomain struct {
	Uuid string
	Name string
	SysUuid string
}

func (TblSys) TableName() string {
	return "tbl_sys"
}

func (TblNeNa) TableName() string {
	return "tbl_ne_na"
}

func (TblDomain) TableName() string {
	return "tbl_domain"
}

func (TblNe) TableName() string {
	return "tbl_ne"
}

func InitAllUnknownDevMap()   {
	var list []TblNeNa
	result := DB_DEV.Find(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return
	}
	unknownDevMap = make(map[string]TblNeNa)
	for _,value  :=range list{
		sn := strings.ToLower(value.Alias)
		unknownDevMap[sn] = value
	}
}

func GetAllDevList() ([]TblNe,error)  {
	var list []TblNe
	result := DB_DEV.Find(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil,result.Error
	}
	if  result.RecordNotFound() {
		logs.Error("设备表没有任何设备")
		return nil,errors.New("查询结果为空")
	}

	return list,nil
}

func InitDevBackendMap()  {
	logs.Info("正在初始化hashmap,可能耗时比较久,请耐心等候...")
	if defaultSvcName == ""{
		logs.Error("找不到默认服务，直接返回")
		return
	}
	devList,err := GetAllDevList()
	if err != nil {
		return
	}
	domainMap,err := GetAllDomainMap()
	if err != nil {
		return
	}
	sysMap,err := GetAllSysMapAndResolveSvcAddr()
	if err != nil {
		return
	}

	for _,value  :=range devList{
		sn := strings.ToLower(value.ProductSns)
		devBackendMap[sn] = defaultSvcName
		domain, ok := domainMap[value.DomainName]
		if !ok {
			continue
		}
		sys,ok := sysMap[domain.SysUuid]
		if !ok {
			continue
		}
		devBackendMap[sn] = sys.SvcName
		//logs.Info(value.ProductSns,devBackendMap[value.ProductSns])
	}
	logs.Info("hashmap初始化完成,key：设备序列号，value：服务域名")
}
func GetAllDomainMap() (map[string]TblDomain,error)   {
	var list []TblDomain
	result := DB_SVC.Find(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil,result.Error
	}
	logs.Info("全部域列表",list)
	var m = make(map[string]TblDomain)
	for _,value  :=range list{
		m[value.Name] = value
	}

	return m,nil
}

func GetAllSysMapAndResolveSvcAddr() (map[string]TblSys,error)   {
	var list []TblSys
	result := DB_SVC.Find(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil,result.Error
	}
	logs.Info("全部服务列表",list)
	var m = make(map[string]TblSys)
	for _,value  :=range list{
		m[value.Uuid] = value
		ResolveAndSetUdpAddrFromAddr(fmt.Sprintf("%s:%d",value.SvcName,OpenConfig.DefaultSvcPort))
	}

	return m,nil
}

func GetDevBySn(ProductSn string) *TblNe {
	var t TblNe
	record:=DB_DEV.Where("product_sns = ?", ProductSn).First(&t)
	if  record.RecordNotFound() {
		logs.Error("找不到设备",ProductSn)
		return nil
	}
	if record.Error!=nil {
		logs.Error(record.Error)
		return nil
	}
	return &t
}

func GetDefaultSvcName() string {
	if defaultSvcName != "" {
		return defaultSvcName
	}
	var t TblSys
	record:=DB_SVC.Where("default_flag = ?",1).First(&t)
	if  record.RecordNotFound() {
		logs.Error("找不到默认服务")
		return ""
	}
	if record.Error!=nil {
		logs.Error(record.Error)
		return ""
	}
	defaultSvcName = t.SvcName
	return t.SvcName
}
func GetDomainByDomainName(name string) (*TblDomain,error) {
	var t TblDomain
	record:=DB_SVC.Where("name = ?", name).First(&t)
	if  record.RecordNotFound() {
		logs.Error("找不到域",name)
		return nil,errors.New("domain not found")
	}
	if record.Error!=nil {
		logs.Error(record.Error)
		return nil,record.Error
	}
	return &t,nil
}

func GetSysByDomainName(domainName string) (*TblSys,error) {
	var t TblSys
	domain,err := GetDomainByDomainName(domainName)
	if err != nil {
		return nil,err
	}
	record:=DB_SVC.Where("uuid = ?", domain.SysUuid).First(&t)
	if  record.RecordNotFound() {
		logs.Error("找不到服务",domain.SysUuid)
		return nil,errors.New("sys not found")
	}
	if record.Error!=nil {
		logs.Error(record.Error)
		return nil,record.Error
	}
	return &t,nil
}
func GetSvcNameBySn(sn string) string  {
	dev:=GetDevBySn(sn)
	if dev == nil {
		return ""
	}
	domain:=dev.DomainName
	if domain == "" {
		logs.Error("设备域名非法",dev)
		return ""
	}
	sys,err := GetSysByDomainName(domain)
	if err == nil {
		return ""
	}
	return sys.SvcName
}

func InitDao()  {
	GetDefaultSvcName()
}

func main() {
	logs.SetLogger(logs.AdapterConsole)
	InitDb()
	logs.Info(GetSvcNameBySn("da00-0040-8800-0218"))
}