package core

func Init()  {
	//注意以下按顺序执行
	InitConfig()
	InitDb()
	InitDao()
	InitDev()
}