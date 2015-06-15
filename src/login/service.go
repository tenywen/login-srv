package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/vmihailenco/msgpack.v2"

	"golang.org/x/net/context"
)

import (
	pb "proto"

	log "github.com/GameGophers/nsq-logger"
	"github.com/fzzy/radix/redis"
)

const (
	SERVICE            = "[LOGIN]"
	DEFAULT_ETCD       = "http://127.0.0.1:2379"
	DEFAULT_REDIS_HOST = "127.0.0.1:6379"
	ENV_REDIS_HOST     = "REDIS_HOST"
)

const (
	FSM_KEY = "FSM:%s"
)

//------------------------------------------------ 状态机定义
const (
	UNKNOWN = byte(iota)
	OFF_FREE
	OFF_RAID
	OFF_PROT
	ON_FREE
	ON_PROT
)

type server struct {
	redis_client *redis.Client
}

type Stats struct {
	Uid    int32
	Host   string
	Status byte
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
}

//------------------------------------------------------- user login
func (s *server) Login(ctx context.Context, in *pb.User_Login) (*pb.User_Auth, error) {
	uuid := strings.ToUpper(in.Uuid)
	host := in.Host
	login_type := in.LoginType

	//TODO get lock from etcd or use redis watch.
	bin, err := s.redis_client.Cmd("get", fmt.Sprintf(FSM_KEY, uuid)).Bytes()
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
		bin, err := s.redis_client.Cmd("get", fmt.Sprintf("auth:%s", uuid)).Bytes()
		if err != nil {
			log.Critical(err)
			return nil, nil
		}
		if bin == nil {
			//TODO registe user
		}
		err = msgpack.Unmarshal(bin, auth)
		if err != nil {
			log.Critical(err)
			return nil, nil
		}
		user_stats.Uid = auth.Uid
		user_stats.Status = UNKNOWN
		user_stats.Host = host
	}

	switch user_stats.Status {
	case ON_FREE:
	case ON_PROT:
	case OFF_PROT:

	}

	return &pb.User_Auth{}, nil
}

//------------------------------------------------------- user logout
func (s *server) Logout(ctx context.Context, in *pb.User_Uid) (*pb.User_Auth, error) {
	return &pb.User_Auth{}, nil

}

//------------------------------------------------------- user login
func (s *server) Registe(ctx context.Context, in *pb.User_Login) (*pb.User_Auth, error) {
	return &pb.User_Auth{}, nil

}
