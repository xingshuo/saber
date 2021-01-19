package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	kite "github.com/xingshuo/kite/pkg"
	sbapi "github.com/xingshuo/saber/pkg"
	saber "github.com/xingshuo/saber/src"
)

var (
	concyNum       int
	reqNumPerConcy int
	rpcSvcNum      int
	seq            uint32 = 1
)

const (
	MSG_SABER kite.MsgType = 4
	BaseSvcID uint32       = 101
)

type ReqLogin struct {
	Gid  uint64
	Name string
}

type RspLogin struct {
	Status int
}

type ReqHandler struct {
	server    *saber.Server
	client    *saber.Service
	svcID     uint32
	results   chan<- *kite.Response
	recvBytes uint64
}

// impl of saber.Codec.Marshal
func (rh *ReqHandler) Marshal(msgType saber.MsgType, method string, v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// impl of saber.Codec.Unmarshal
func (rh *ReqHandler) Unmarshal(msgType saber.MsgType, method string, data []byte) (interface{}, error) {
	rh.recvBytes = uint64(len(data))
	var v interface{}
	if msgType == saber.MSG_TYPE_CLUSTER_RSP && method == "ReqLogin" {
		v = &RspLogin{}
	}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	return v, err
}

func (rh *ReqHandler) Init(req *kite.Request, results chan<- *kite.Response) error {
	s, err := rh.server.NewService("kiteproxy", rh.svcID)
	if err != nil {
		return err
	}
	s.SetCodec(rh)
	rh.results = results
	rh.client = s
	return nil
}

func (rh *ReqHandler) OnRequest() error {
	startTime := time.Now()
	svcID := BaseSvcID
	if rpcSvcNum > 0 {
		svcID += rh.svcID % uint32(rpcSvcNum)
	}
	reply, err := rh.client.CallCluster(context.Background(), "stress_server", "lobby", svcID, "ReqLogin", &ReqLogin{
		Gid:  uint64(10100000 + rh.svcID),
		Name: uuid.New().String()[:8],
	})
	result := &kite.Response{}
	result.UseTime = uint64(time.Since(startTime))
	result.Method = "ReqLogin"
	result.MsgType = MSG_SABER
	if err == nil && reply.(*RspLogin).Status == 200 {
		result.IsSucceed = true
		result.ErrCode = 0
	} else {
		result.IsSucceed = false
		result.ErrCode = -1001
	}
	result.ReceivedBytes = rh.recvBytes
	rh.results <- result
	return err
}

func (rh *ReqHandler) Close() {
	rh.server.DelService("kiteproxy", rh.svcID)
}

func init() {
	flag.IntVar(&concyNum, "c", 20, "concurrency num")
	flag.IntVar(&reqNumPerConcy, "n", 50, "per concurrency req num")
	flag.IntVar(&rpcSvcNum, "svc", 1, "rpc svc num")
	kite.RegisterMsgType(MSG_SABER, "SABER")
}

func main() {
	flag.Parse()
	ss, err := sbapi.NewServer("config.json")
	if err != nil {
		log.Fatalf("new server err:%v", err)
	}

	ks := kite.NewServer()
	_, err = ks.RunWithSimpleArgs("stress_server.lobby.101", concyNum, reqNumPerConcy, func() kite.ReqHandler {
		return &ReqHandler{
			server: ss,
			svcID:  atomic.AddUint32(&seq, 1),
		}
	})
	if err != nil {
		log.Fatalf("run failed:%v\n", err)
	}
	ss.Exit()
	log.Println("run done")
}
