package core

import (
	"github.com/astaxie/beego/logs"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"fmt"
	"errors"
)

type TblNe struct {
	ProductSns string
	DomainUuid int
	DomainName string
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

func (TblDomain) TableName() string {
	return "tbl_domain"
}

func (TblNe) TableName() string {
	return "tbl_ne"
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
	defaultSvc := GetDefaultSvcName()
	if defaultSvc == ""{
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
	sysMap,err := GetAllSysMap()
	if err != nil {
		return
	}

	for _,value  :=range devList{
		devBackendMap[value.ProductSns] = defaultSvc
		domain, ok := domainMap[value.DomainName]
		if !ok {
			continue
		}
		sys,ok := sysMap[domain.SysUuid]
		if !ok {
			continue
		}
		devBackendMap[value.ProductSns] = sys.SvcName
		logs.Info(value.ProductSns,devBackendMap[value.ProductSns])
	}
	logs.Info("hashmap初始化完成")
}
func GetAllDomainMap() (map[string]TblDomain,error)   {
	var list []TblDomain
	result := DB_SVC.Find(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil,result.Error
	}
	if  result.RecordNotFound() {
		logs.Error("域表没有任何域")
		return nil,errors.New("查询结果为空")
	}
	fmt.Println(list)
	var m = make(map[string]TblDomain)
	for _,value  :=range list{
		m[value.Name] = value
	}

	return m,nil
}

func GetAllSysMap() (map[string]TblSys,error)   {
	var list []TblSys
	result := DB_SVC.Find(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil,result.Error
	}
	if  result.RecordNotFound() {
		logs.Error("sys表没有任何sys")
		return nil,errors.New("查询结果为空")
	}
	var m = make(map[string]TblSys)
	for _,value  :=range list{
		m[value.Uuid] = value
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
	return t.SvcName
}
func GetDomainByDomainName(name string) *TblDomain {
	var t TblDomain
	record:=DB_SVC.Where("name = ?", name).First(&t)
	if  record.RecordNotFound() {
		logs.Error("找不到域",name)
		return nil
	}
	if record.Error!=nil {
		logs.Error(record.Error)
		return nil
	}
	return &t
}

func GetSysByDomainName(domainName string) *TblSys {
	var t TblSys
	domain := GetDomainByDomainName(domainName)
	if domain == nil {
		return nil
	}
	record:=DB_SVC.Where("uuid = ?", domain.SysUuid).First(&t)
	if  record.RecordNotFound() {
		logs.Error("找不到服务",domain.SysUuid)
		return nil
	}
	if record.Error!=nil {
		logs.Error(record.Error)
		return nil
	}
	return &t
}
func GetSvcNameBySn(sn string)string  {
	dev:=GetDevBySn(sn)
	if dev == nil {
		return ""
	}
	domain:=dev.DomainName
	if domain == "" {
		logs.Error("设备域名非法",dev)
		return ""
	}
	sys := GetSysByDomainName(domain)
	if sys == nil {
		return ""
	}
	return sys.SvcName
}

func InitDao()  {
	InitDevBackendMap()
}

func main() {
	logs.SetLogger(logs.AdapterConsole)
	InitDb()
	logs.Info(GetSvcNameBySn("da00-0040-8800-0218"))
}