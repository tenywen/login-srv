package main

import (
	"errors"
	"fmt"
	"os"
	"services"
	"strings"
	"time"

	"golang.org/x/net/context"
	"gopkg.in/vmihailenco/msgpack.v2"
)

import (
	pb "proto"

	log "github.com/GameGophers/nsq-logger"
	"github.com/fzzy/radix/redis"
)

const (
	SERVICE            = "[LOGIN-SRV]"
	DEFAULT_ETCD       = "http://127.0.0.1:2379"
	DEFAULT_REDIS_HOST = "127.0.0.1:6379"
	ENV_REDIS_HOST     = "REDIS_HOST"
)

const (
	FSM_KEY  = "FSM:%s"
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
}

type Stats struct {
	Uid    int32
	Host   string
	Status int8
	TS     int64
}

type Auth struct {
	Id            int32  //User Id
	Uuid          string //UUID
	Domain        string //所在服务器
	LastLoginTime int64  //最后登录时间
	LoginType     int8   //登录方式
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
}

//------------------------------------------------------- user login
func (s *server) Login(ctx context.Context, in *pb.User_Login) (*pb.User_Auth, error) {
	uuid := strings.ToUpper(in.Uuid)
	host := in.Host
	login_type := in.LoginType

	if uuid == "" {
		return nil, errors.New("require uuid")
	}

	// get lock from etcd or use redis watch.
	fsm_key := fmt.Sprintf(FSM_KEY, uuid)
	auth_key := fmt.Sprintf("auth:%s", uuid)
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

	user_stats := &Stats{}
	if bin != nil {
		err = msgpack.Unmarshal(bin, user_stats)
		if err != nil {
			log.Critical(err)
			return nil, nil
		}
	} else {
		//read from auth.
		auth := &Auth{}
		bin, err := s.redis_client.Cmd("get", auth_key).Bytes()
		if err != nil {
			log.Critical(err)
			return nil, nil
		}
		if bin == nil {
			log.Critical("uuid not registe")
			return nil, errors.New("uuid not registe")
		}
		err = msgpack.Unmarshal(bin, auth)
		if err != nil {
			log.Critical(err)
			return nil, nil
		}
		user_stats.Uid = auth.Id
		user_stats.Status = UNKNOWN
		user_stats.Host = host
	}

	switch user_stats.Status {
	case OFF_FREE, UNKNOWN:
		user_stats.Status = ON_FREE
	}
	// save to redis db.
	s.redis_client.Cmd("set", fsm_key, msgpack.Marshal(user_stats))
	s.redis_client.Cmd("set", auth_key, msgpack.Marshal(auth))

	return &pb.User_Auth{Uid: user_stats.Uid, Stats: int32(user_stats.Status)}, nil
}

//------------------------------------------------------- user logout
func (s *server) Logout(ctx context.Context, in *pb.User_Uid) (*pb.User_Auth, error) {
	uid := in.Uid
	if uid == 0 {
		return nil, errors.New("require uid")
	}

	// get lock from etcd or use redis watch.
	fsm_key := fmt.Sprintf(FSM_KEY, uuid)
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
	user_stats := &Stats{}
	err = msgpack.Unmarshal(bin, user_stats)
	if err != nil {
		log.Critical(err)
		return nil, nil
	}
	switch user_stats {
	case ON_FREE:
		user_stats.Status = OFF_FREE
	}

	return &pb.User_Auth{Uid: user_stats.Uid, Stats: user_stats.Stats}, nil

}

//------------------------------------------------------- user login
func (s *server) Registe(ctx context.Context, in *pb.User_Login) (*pb.User_Auth, error) {
	//registe user
	uid := s.next_uid(uuid)
	auth = &Auth{
		Id:            uid,
		Uuid:          uuid,
		Domain:        "",
		LastLoginTime: time.Now().Unix(),
		LoginType:     login_type,
	}
	return &pb.User_Auth{}, nil

}

func (s *server) next_uid(uuid string) (uid int32) {
	srv, err := services.GetService(services.SERVICE_SNOWFLAKE)
	uid, err := srv.Next(context.Background(), &pb.Snowflake_Key{Name: SEQS_UID})
	if err != nil {
		log.Critical(err)
		return
	}
}
