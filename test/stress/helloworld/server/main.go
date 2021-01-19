package main

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/goinggo/mapstructure"
	sbapi "github.com/xingshuo/saber/pkg"
)

type ReqLogin struct {
	Gid  uint64
	Name string
}

type RspLogin struct {
	Status int
}

func main() {
	server, err := sbapi.NewServer("config.json")
	if err != nil {
		log.Fatalf("new server err:%v", err)
	}
	lobbySvc, err := server.NewService("lobby", 101)
	if err != nil {
		log.Fatalf("new lobby service err:%v", err)
	}
	lobbySvc.RegisterSvcHandler("ReqLogin", func(ctx context.Context, req interface{}) (interface{}, error) {
		msg := &ReqLogin{}
		err := mapstructure.Decode(req, msg)
		if err != nil {
			return nil, fmt.Errorf("proto error %w", err)
		}
		if msg.Gid/100000 != 101 {
			log.Fatalf("gid err %d", msg.Gid)
		}
		log.Printf("%s on req login %d", msg.Name, msg.Gid)
		return &RspLogin{Status: 200}, nil
	})
	server.WaitExit(syscall.SIGINT)
}
