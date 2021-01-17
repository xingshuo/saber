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

type HeartBeat struct {
	Session int
}

func main() {
	server, err := sbapi.NewServer("config.json")
	if err != nil {
		log.Fatalf("new server err:%v", err)
	}
	gateSvc, err := server.NewService("gate", 1)
	if err != nil {
		log.Fatalf("new gate service err:%v", err)
	}
	gateSvc.RegisterSvcHandler("HeartBeat", func(ctx context.Context, req interface{}) (interface{}, error) {
		// svc := sbapi.GetSvcFromCtx(ctx)
		hb := &HeartBeat{}
		err := mapstructure.Decode(req, hb)
		if err != nil {
			return nil, fmt.Errorf("proto error %w", err)
		}
		log.Printf("recv heartbeat session %d\n", hb.Session)
		return nil, nil
	})
	rsp, err := gateSvc.CallCluster(context.Background(), "cluster_server", "lobby", 1, "ReqLogin", &ReqLogin{
		Gid:  101,
		Name: "lilei",
	})
	if err == nil {
		reply := &RspLogin{}
		mapstructure.Decode(rsp, reply)
		log.Printf("rpc result: %d\n", reply.Status)
	} else {
		log.Printf("rpc err: %v\n", err)
	}
	server.WaitExit(syscall.SIGINT)
}
