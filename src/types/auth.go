package types

type Auth struct {
	Id            int64  //User Id
	Uuid          string //UUID
	Cert          string //证书
	Host          string //所在服务器
	LoginType     int8   //登录方式
	ClientVersion string //客户端版本
	Lang          string //语言
	Appid         string //appid
	OsVersion     string //操作系统版本
	DeviceName    string //设备名
	DeviceId      string //设备号Token
	LoginIp       string //IP
	LastLoginTime int64  //最后登录时间
}
