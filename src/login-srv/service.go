package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"services"
	"strings"
	"time"

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
	DEFAULT_REDIS_HOST  = "127.0.0.1:6379"
	DEFAULT_MONGODB_URL = "mongodb://127.0.0.1/mydb"
	ENV_REDIS_HOST      = "REDIS_HOST"
	ENV_MONGODB_URL     = "MONGODB_URL"
)

const (
	FSM_KEY  = "fsm:%s"
	AUTH_KEY = "auth:%s"
	SEQS_UID = "uid"
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
	Uid    int32
	Host   string
	Status int8
	TS     int64
}

func (s *server) init() {
	// read redis host
	redis_host := DEFAULT_REDIS_HOST
	if env := os.Getenv(ENV_REDIS_HOST); env != "" {
		redis_host = env
	}
	// start connection to redis
	client, err := redis.Dial("tcp", redis_host)
	if err != nil {
		log.Critical(err)
		os.Exit(-1)
	}

	s.redis_client = client

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

	new_user = false
	if uuid == "" {
		return nil, errors.New("require uuid")
	}
	auth := Auth{}
	err = c.Find(bson.M{"uuid": uuid, "login_type": login_type}).One(&auth)
	if err != nil {
		if err != mgo.NotFound {
			return nil, errors.New("query user err: %v", err)
		}
		//TODO new user
		uid := s.next_uid()
		new_user = true
		auth = Auth{
			Id:            uid,
			Uuid:          uuid,
			Host:          in.Host,
			LoginType:     login_type,
			Cert:          fmt.Sprintf("%x", md5.Sum(uuid)),
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

	// get lock from etcd or use redis watch.
	fsm_key := fmt.Sprintf(FSM_KEY, auth.Id)
	auth_key := fmt.Sprintf(AUTH_KEY, auth.Id)
	lock := etcdmutex.Lock(fsm_key)
	defer m.Unlock()

	if lock == nil {
		log.Critical("cann't lock user: %v", fsm_key)
		return nil, nil
	}
	bin, err := s.redis_client.Cmd("get", fsm_key).Bytes()
	if err != nil {
		log.Critical(err)
		return nil, nil
	}

	user_fsm := FSM{}
	if bin != nil {
		err = msgpack.Unmarshal(bin, &user_fsm)
		if err != nil {
			log.Critical(err)
			return nil, nil
		}
	} else {
		//create new user fsm
		user_fsm = FSM{
			Uid:    auth.Id,
			Status: ON_FREE,
			Host:   host,
			TS:     time.Now().Unix(),
		}
	}

	switch user_fsm.Status {
	case OFF_FREE, UNKNOWN:
		user_fsm.Status = ON_FREE
	}
	// save to redis db.
	s.redis_client.Cmd("set", fsm_key, msgpack.Marshal(user_fsm))
	s.redis_client.Cmd("set", auth_key, msgpack.Marshal(auth))
	//TODO send dirty key to bgsave

	return &pb.User_Auth{Uid: user_fsm.Uid, Stats: int32(user_fsm.Status), NewUser: new_user}, nil
}

//------------------------------------------------------- user logout
func (s *server) Logout(ctx context.Context, in *pb.User_Uid) (*pb.User_Auth, error) {
	uid := in.Uid
	if uid == 0 {
		return nil, errors.New("require uid")
	}

	// get lock from etcd or use redis watch.
	fsm_key := fmt.Sprintf(FSM_KEY, uid)
	l := etcdmutex.Lock(fsm_key)
	defer m.Unlock()
	if l == nil {
		log.Critical("cann't lock user: %v", fsm_key)
		return nil, nil
	}
	bin, err := s.redis_client.Cmd("get", fsm_key).Bytes()
	if err != nil {
		log.Critical(err)
		return nil, nil
	}
	user_fsm := FSM{}
	err = msgpack.Unmarshal(bin, user_fsm)
	if err != nil {
		log.Critical(err)
		return nil, nil
	}
	switch user_fsm.Status {
	case ON_FREE:
		user_fsm.Status = OFF_FREE
	}

	return &pb.User_Auth{Uid: user_fsm.Uid, Stats: user_fsm.Stats}, nil
}

func (s *server) next_uid() (uid int32) {
	srv, err := services.GetService(services.SERVICE_SNOWFLAKE)
	uid, err := srv.Next(context.Background(), &pb.Snowflake_Key{Name: SEQS_UID})
	if err != nil {
		log.Critical(err)
		return
	}
}
