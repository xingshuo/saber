package main

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/goinggo/mapstructure"
	saber "github.com/xingshuo/saber/pkg"
)

type ReqLogin struct {
	Gid  uint64
	Name string
}

type RspLogin struct {
	Status int
}

type HeartBeat struct {
	Session int
}

func main() {
	server, err := saber.NewServer("config.json")
	if err != nil {
		log.Fatalf("new server err:%v", err)
	}
	// server.GetLogSystem().SetLevel(sblog.LevelDebug)
	lobbySvc, err := server.NewService("lobby", 1)
	if err != nil {
		log.Fatalf("new lobby service err:%v", err)
	}
	lobbySvc.RegisterSvcHandler("ReqLogin", func(ctx context.Context, req interface{}) (interface{}, error) {
		msg := &ReqLogin{}
		err := mapstructure.Decode(req, msg)
		if err != nil {
			return nil, fmt.Errorf("proto error %w", err)
		}
		svc := saber.GetSvcFromCtx(ctx)
		session := 500
		svc.SendCluster(ctx, "cluster_client", "gate", 1, "HeartBeat", &HeartBeat{Session: session})
		svc.RegisterTimer(func() {
			session++
			svc.SendCluster(ctx, "cluster_client", "gate", 1, "HeartBeat", &HeartBeat{Session: session})
		}, 1000, 3)
		log.Printf("%s on req login %d", msg.Name, msg.Gid)
		return &RspLogin{Status: 200}, nil
	})
	server.WaitExit(syscall.SIGINT)
}
