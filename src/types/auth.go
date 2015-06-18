package types

type Auth struct {
	Id         int32  //User Id
	Uuid       string //UUID
	Cert       string //证书
	Host       string //所在服务器
	LoginType  int8   //登录方式
	CreateTime int64  //创建时间
}
