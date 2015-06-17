package types

type Auth struct {
	Id            int32  //User Id
	Uuid          string //UUID
	Cert          string //证书
	Domain        string //所在服务器
	LastLoginTime int64  //最后登录时间
	LoginType     int8   //登录方式
}
