package main

import (
	"crypto/md5"
	"db"
	"errors"
	"etcdmutex"
	"fmt"
	"os"
	"services"
	"services/proto"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"golang.org/x/net/context"
)

import (
	pb "proto"
	. "types"

	log "github.com/GameGophers/nsq-logger"
	"github.com/fzzy/radix/redis"
)

const (
	SERVICE             = "[LOGIN-SRV]"
	DEFAULT_ETCD        = "http://127.0.0.1:2379"
	DEFAULT_MONGODB_URL = "mongodb://127.0.0.1/mydb"
	ENV_MONGODB_URL     = "MONGODB_URL"
)

const (
	TABLE_FSM  = "fsm"
	TABLE_AUTH = "auth"
	SEQS_UID   = "uid"
)

//------------------------------------------------ 状态机定义
const (
	UNKNOWN = int8(iota)
	OFF_FREE
	ON_FREE
)

type server struct {
	redis_client *redis.Client
	db           *mgo.Database
}

type FSM struct {
	Uid    int64
	Host   string
	Status int8
	TS     int64
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
func (s *server) Login(ctx context.Context, in *pb.User_Login) (*pb.User_Auth, error) {
	uuid := strings.ToUpper(in.Uuid)
	login_type := in.LoginType

	new_user := false
	if uuid == "" {
		return nil, errors.New("require uuid")
	}
	auth := Auth{}
	err := s.db.C(TABLE_AUTH).Find(bson.M{"uuid": uuid, "login_type": login_type}).One(&auth)
	if err != nil {
		if err != mgo.ErrNotFound {
			return nil, err
		}
		//TODO new user
		uid := s.next_uid()
		new_user = true
		auth = Auth{
			Id:            uid,
			Uuid:          uuid,
			Host:          in.Host,
			LoginType:     int8(login_type),
			Cert:          fmt.Sprintf("%x", md5.Sum([]byte(uuid))),
			ClientVersion: in.ClientVersion,
			Lang:          in.Lang,
			Appid:         in.Appid,
			OsVersion:     in.OsVersion,
			DeviceName:    in.DeviceName,
			DeviceId:      in.DeviceId,
			LoginIp:       in.LoginIp,
		}
		log.Info("registe new user : %v, %v", uuid, uid)
	}

	// get lock from etcd
	lock := etcdmutex.Lock(fmt.Sprintf("%v-%v", TABLE_FSM, auth.Id))
	defer lock.Unlock()

	if lock == nil {
		log.Critical("cann't lock user: %v, %v", TABLE_FSM, auth.Id)
		return nil, nil
	}
	user_fsm := FSM{}
	err = db.Client.Get(TABLE_FSM, auth.Id, &user_fsm)
	if err != nil {
		log.Critical(err)
		return nil, err
	} else {
		//create new user fsm
		user_fsm = FSM{
			Uid:    auth.Id,
			Status: ON_FREE,
			Host:   in.Host,
			TS:     time.Now().Unix(),
		}
	}

	switch user_fsm.Status {
	case OFF_FREE, UNKNOWN:
		user_fsm.Status = ON_FREE
	}
	// save to redis db.
	db.Client.Set(TABLE_FSM, auth.Id, user_fsm)
	db.Client.Set(TABLE_AUTH, auth.Id, auth)

	return &pb.User_Auth{Uid: user_fsm.Uid, Stats: int32(user_fsm.Status), NewUser: new_user}, nil
}

//------------------------------------------------------- user logout
func (s *server) Logout(ctx context.Context, in *pb.User_Uid) (*pb.User_Auth, error) {
	uid := in.Uid
	if uid == 0 {
		return nil, errors.New("require uid")
	}

	// get lock from etcd
	m := etcdmutex.Lock(fmt.Sprintf("%v-%v", TABLE_FSM, uid))
	defer m.Unlock()
	if m == nil {
		log.Critical("cann't lock user: %v", uid)
		return nil, nil
	}
	user_fsm := FSM{}
	err := db.Client.Get(TABLE_FSM, uid, &user_fsm)
	if err != nil {
		log.Critical(err)
		return nil, err
	}
	switch user_fsm.Status {
	case ON_FREE:
		user_fsm.Status = OFF_FREE
	}

	return &pb.User_Auth{Uid: user_fsm.Uid, Stats: int32(user_fsm.Status)}, nil
}

func (s *server) next_uid() int64 {
	c, err := services.GetService(services.SERVICE_SNOWFLAKE)
	service, _ := c.(proto.SnowflakeServiceClient)
	uid, err := service.Next(context.Background(), &proto.Snowflake_Key{Name: SEQS_UID})
	if err != nil {
		log.Critical(err)
		return 0
	}
	return uid.Value
}
