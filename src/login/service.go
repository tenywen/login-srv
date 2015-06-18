package main

import (
	"bytes"
	"crypto/md5"
	"db"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"services"
	"services/proto"
	"strings"
	"time"

	"golang.org/x/net/context"
)

import (
	pb "proto"
	. "types"

	log "github.com/GameGophers/nsq-logger"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

	LOGIN_TYPE_UUID = iota
	LOGIN_TYPE_WEIXIN
	LOGIN_TYPE_ALIPAY
	LOGIN_TYPE_SINA
)

type server struct {
	db *mgo.Database
}

func (s *server) init() {

	// read mongodb host
	mongodb_url := DEFAULT_MONGODB_URL
	if env := os.Getenv(ENV_MONGODB_URL); env != "" {
		mongodb_url = env
	}

	// start connection to mongodb
	sess, err := mgo.Dial(mongodb_url)
	if err != nil {
		log.Critical(err)
		os.Exit(-1)
	}
	// database is provided in url
	s.db = sess.DB("")
}

//------------------------------------------------------- user login
func (s *server) Login(ctx context.Context, in *pb.User_LoginInfo) (*pb.User_LoginResp, error) {
	uuid := strings.ToUpper(in.Uuid)
	login_type := in.LoginType
	if uuid == "" {
		return nil, errors.New("require uuid")
	}
	auth := Auth{}
	switch login_type {
	case LOGIN_TYPE_UUID:
		err := s.db.C(TABLE_AUTH).Find(bson.M{"uuid": uuid, "login_type": login_type}).One(&auth)
		if err != nil {
			if err != mgo.ErrNotFound {
				return nil, err
			}
			//insert a new user? or do it on agent service.
			uid := s.next_uid()
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, uid)
			auth = Auth{
				Id:         uid,
				Uuid:       uuid,
				Host:       in.Host,
				LoginType:  int8(login_type),
				Cert:       fmt.Sprintf("%x", md5.Sum(buf.Bytes())),
				CreateTime: time.Now().Unix(),
			}
			log.Info("registe new user : %+v", auth)
			// save to redis db.
			db.Redis.Set(TABLE_AUTH, auth.Id, auth)
		}
	case LOGIN_TYPE_WEIXIN:
		fallthrough
	case LOGIN_TYPE_ALIPAY:
		fallthrough
	case LOGIN_TYPE_SINA:
		return nil, errors.New("not support yet")
	}

	return &pb.User_LoginResp{Uid: auth.Id}, nil
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
