package main

import (
	"crypto/md5"
	"hash"
	"db"
	"errors"
	"services"
	"services/proto"
	"strings"

	"golang.org/x/net/context"
)

import (
	pb "proto"
	log "github.com/GameGophers/libs/nsq-logger"
)

const (
	SERVICE             = "[LOGIN-SRV]"
	DEFAULT_ETCD        = "http://127.0.0.1:2379"
	DEFAULT_MONGODB_URL = "mongodb://127.0.0.1/mydb"
	ENV_MONGODB_URL     = "MONGODB_URL"
)

const (
	TABLE_AUTH = "auth"
	SEQS_UID   = "uid"

	LOGIN_TYPE_UUID 	= iota 
	LOGIN_TYPE_FACEBOOK
)

var h hash.Hash

type server struct {}

func (s *server) init() {
	h = md5.New()
}

//------------------------------------------------------- user login
func (s *server) Login(ctx context.Context, in *pb.User_LoginInfo) (*pb.User_LoginResp, error) {
	uuid := strings.ToUpper(in.Uuid)
	login_type := in.LoginType
	if uuid == "" {
		return nil, errors.New("require uuid")
	}
	cert := ""
	// TODO 根据登录类型进行第三方验证
	switch login_type {
	case LOGIN_TYPE_UUID:  // 不需要验证
		// md5(uuid) -> cert 
		cert = string(h.Sum([]byte(uuid)))
	// TODO
	case LOGIN_TYPE_FACEBOOK: // facebook  
	default:
		log.Error("login type error","logintype:",login_type,"uuid:",uuid) 
		return nil,errors.New("login_type error")
	}
	new_user := false
	id,exist := db.IsExist(cert,login_type) 
	if !exist {
		new_user = true
		id = s.next_uid()
		db.New(id,uuid,cert,in.Host,login_type)
	}
	
	return &pb.User_LoginResp{Uid: id, NewUser: new_user}, nil
}

func (s *server) next_uid() int32 {
	c, err := services.GetService(services.SERVICE_SNOWFLAKE)
	service, _ := c.(proto.SnowflakeServiceClient)
	uid, err := service.Next(context.Background(), &proto.Snowflake_Key{Name: SEQS_UID})
	if err != nil {
		log.Critical(err)
		return 0
	}
	return int32(uid.Value)
}
