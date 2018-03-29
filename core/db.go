package core

import (
	"time"
	"github.com/astaxie/beego/logs"
	"github.com/jinzhu/gorm"
	"fmt"
)

var (
	DB_DEV *gorm.DB
	DB_SVC *gorm.DB
)

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

func InitDb()  {
	logs.SetLogger(logs.AdapterConsole)
	config := OpenConfig.Mysql
	ip := GetIpByDnsLookup(config.MysqlHost)
	if ip == "" {
		time.Sleep(60*time.Second)
		InitDb()
		return
	}
	path := fmt.Sprintf("%s:%s@tcp(%s:%s)/",config.MysqlUser,config.MysqlPass,ip,config.MysqlPort)
	devPath := fmt.Sprintf("%s%s",path,config.MysqlDbAll)
	svcPath := fmt.Sprintf("%s%s",path,config.MysqlDb)
	//DB_DEV = InitConnection("root:MTIzNDU2@tcp(47.96.145.70:3306)/dmcld-v1-all")
	//DB_SVC = InitConnection("root:MTIzNDU2@tcp(47.96.145.70:3306)/dmcloud-v1")
	DB_DEV = InitConnection(devPath)
	DB_SVC = InitConnection(svcPath)
}
