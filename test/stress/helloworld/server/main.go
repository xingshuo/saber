package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"syscall"

	_ "net/http/pprof"

	"github.com/goinggo/mapstructure"
	saber "github.com/xingshuo/saber/pkg"
)

var (
	rpcSvcNum int
)

const (
	BaseSvcID uint32 = 101
)

type ReqLogin struct {
	Gid  uint64
	Name string
}

type RspLogin struct {
	Status int
}

func init() {
	flag.IntVar(&rpcSvcNum, "svc", 1, "rpc svc num")
}

func main() {
	flag.Parse()
	// pprof
	go func() {
		err := http.ListenAndServe(":9090", nil)
		if err != nil {
			log.Fatalf("failed to serve pprof: %v", err)
		}
	}()

	server, err := saber.NewServer("config.json")
	if err != nil {
		log.Fatalf("new server err:%v", err)
	}
	for i := 0; i < rpcSvcNum; i++ {
		svcID := BaseSvcID + uint32(i)
		lobbySvc, err := server.NewService("lobby", svcID)
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
			//fmt.Printf("%s on req login %d", msg.Name, msg.Gid)
			return &RspLogin{Status: 200}, nil
		})
	}
	server.WaitExit(syscall.SIGINT)
}
