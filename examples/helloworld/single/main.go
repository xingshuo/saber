package main

import (
	"context"
	"fmt"
	"log"
	"syscall"

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
	lobbySvc.RegisterSvcHandler("ReqLogin", func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		msg, ok := req.(*ReqLogin)
		if !ok {
			return nil, fmt.Errorf("proto error")
		}
		svc := saber.GetSvcFromCtx(ctx)
		session := 500
		svc.Send(ctx, "gate", 1, "HeartBeat", &HeartBeat{Session: session})
		svc.RegisterTimer(func() {
			session++
			svc.Send(ctx, "gate", 1, "HeartBeat", &HeartBeat{Session: session})
		}, 1000, 3)
		log.Printf("%s on req login %d", msg.Name, msg.Gid)
		return &RspLogin{Status: 200}, nil
	})
	gateSvc, err := server.NewService("gate", 1)
	if err != nil {
		log.Fatalf("new gate service err:%v", err)
	}
	gateSvc.RegisterSvcHandler("HeartBeat", func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		// svc := sbapi.GetSvcFromCtx(ctx)
		hb, ok := req.(*HeartBeat)
		if !ok {
			return nil, fmt.Errorf("proto error")
		}
		log.Printf("recv heartbeat session %d\n", hb.Session)
		return nil, nil
	})
	rsp, err := gateSvc.Call(context.Background(), "lobby", 1, "ReqLogin", &ReqLogin{
		Gid:  101,
		Name: "lilei",
	})
	if err == nil {
		log.Printf("rpc result: %d\n", rsp.(*RspLogin).Status)
	} else {
		log.Printf("rpc err: %v\n", err)
	}
	server.WaitExit(syscall.SIGINT)
}
