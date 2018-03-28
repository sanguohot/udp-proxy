package mysql

import (
	"time"
	"github.com/astaxie/beego/logs"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"fmt"
)

var DB_DEV *gorm.DB
var DB_SVC *gorm.DB

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

func GetAllDev() map[string]TblNe  {
	var list []TblNe
	result := DB_DEV.Select(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil
	}
	if  result.RecordNotFound() {
		logs.Error("设备表没有任何设备")
		return nil
	}
	var m = make(map[string]TblNe)
	for _,value  :=range list{
		m[value.ProductSns] = value
	}

	return m
}

func GetAllDomain() map[string]TblDomain  {
	var list []TblDomain
	result := DB_SVC.Where("uuid is not null").Select(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil
	}
	if  result.RecordNotFound() {
		logs.Error("域表没有任何域")
		return nil
	}
	fmt.Println(list)
	var m = make(map[string]TblDomain)
	for _,value  :=range list{
		m[value.Name] = value
	}

	return m
}

func GetAllSys() map[string]TblSys  {
	var list []TblSys
	result := DB_SVC.Select(&list)
	if result.Error!=nil {
		logs.Error(result.Error)
		return nil
	}
	if  result.RecordNotFound() {
		logs.Error("sys表没有任何sys")
		return nil
	}
	var m = make(map[string]TblSys)
	for _,value  :=range list{
		m[value.Uuid] = value
	}

	return m
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

func InitConnection(path string) * gorm.DB{
	var err error
	var db *gorm.DB
	db, err = gorm.Open("mysql", path+"?charset=utf8&parseTime=True&loc=Local")
	db.LogMode(true)
	db.DB().Ping()
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	if  err!=nil{
		logs.Error(err)
		time.Sleep(60*time.Second)
		InitConnection(path)
		return nil
	}

	logs.Info("初始化mysql连接成功",path)
	return db
}

func InitDao()  {
	DB_DEV = InitConnection("root:MTIzNDU2@tcp(47.96.145.70:3306)/dmcld-v1-all")
	DB_SVC = InitConnection("root:MTIzNDU2@tcp(47.96.145.70:3306)/dmcloud-v1")
}

func main() {
	logs.SetLogger(logs.AdapterConsole)
	InitDao()
	logs.Info(GetSvcNameBySn("da00-0040-8800-0218"))
}