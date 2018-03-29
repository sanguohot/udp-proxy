package core

import (
	"io/ioutil"
	"encoding/json"
	"github.com/astaxie/beego/logs"
)
const CONFIG_PATH = "/etc/config/config"

type Config struct {
	Mysql Mysql `json:"mysql,omitempty"`
}
type Mysql struct {
	MysqlHost string  `json:"host,omitempty"`
	MysqlUser string  `json:"user,omitempty"`
	MysqlPass string  `json:"pass,omitempty"`
	MysqlPort string  `json:"port,omitempty"`
	MysqlDb string  `json:"db,omitempty"`
	MysqlDbAll string  `json:"db_all,omitempty"`
}
var OpenConfig Config
func InitConfig() {
	logs.SetLogger(logs.AdapterConsole)
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