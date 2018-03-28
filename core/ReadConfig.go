package core

import (
	"io/ioutil"
	"encoding/json"
	"github.com/astaxie/beego/logs"
)
const CONFIG_PATH = "../etc/config"

type Config struct {
	LogConfig LogConfig `json:"log,omitempty"`
	Mysql Mysql `json:"mysql,omitempty"`
	DefaultServer string `json:"defaultserver,omitempty"`
	Timeout int64  `rotation:"timeout,omitempty"`
}
type LogConfig struct {
	RootPath string  `json:"path,omitempty"`
	MaxAge int64  `json:"maxAge,omitempty"`
	RotationTime int64  `rotation:"rotation,omitempty"`
}
type Mysql struct {
	MysqlName string  `json:"name,omitempty"`
	MysqlUser string  `json:"user,omitempty"`
	MysqlPass string  `json:"pass,omitempty"`
	MysqlPort string  `json:"port,omitempty"`
	MysqlDbName string  `json:"dbname,omitempty"`
}
var OpenConfig Config
func InitConfig()  {
	logs.Info("初始化配置文件",CONFIG_PATH)
	dat, err := ioutil.ReadFile(CONFIG_PATH)
	if err != nil {
		logs.Error(err)
		return
	}
	err=json.Unmarshal(dat,&OpenConfig)
	if err != nil {
		logs.Error(err)
		return
	}
	logs.Info("初始化配置文件成功",OpenConfig)
}
//func main()  {
//	InitConfig()
//}