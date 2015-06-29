package main

import (
	"net"
	"os"

	log "github.com/GameGophers/libs/nsq-logger"
	_ "github.com/GameGophers/libs/statsd-pprof"
	"google.golang.org/grpc"
)

import (
	pb "proto"
)

const (
	port = ":50006"
)

func main() {
	log.SetPrefix(SERVICE)
	// 监听
	listen, err := net.Listen("tcp", port)
	if err != nil {
		log.Critical(err)
		os.Exit(-1)
	}
	log.Info("listening on ", listen.Addr())

	// 注册服务
	s := grpc.NewServer()
	ins := &server{}
	ins.init()
	pb.RegisterLoginServiceServer(s, ins)
	// 开始服务
	s.Serve(listen)
}
