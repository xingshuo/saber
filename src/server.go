package saber

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/xingshuo/saber/common/log"
	"github.com/xingshuo/saber/common/utils"
	// "github.com/xingshuo/ms-go/deadlock"
)

type ServerConfig struct {
	ClusterName    string            // 当前节点名
	LocalAddr      string            // 本进程/容器 mesh地址 ip:port
	RemoteAddrs    map[string]string // 远端节点地址表
	TickIntervalMs int64             // 定时器检测间隔:毫秒
}

type Server struct {
	config     ServerConfig
	rwMu       sync.RWMutex
	services   map[SVC_HANDLE]*Service
	svcGroup   map[string]map[uint32]*Service
	sidecar    *Sidecar
	timerStore *TimeStore
	log        *log.LoggerWrapper
	codec      Codec
}

func (s *Server) Init(config string) error {
	s.services = make(map[SVC_HANDLE]*Service)
	s.svcGroup = make(map[string]map[uint32]*Service)
	err := s.loadConfig(config)
	if err != nil {
		return err
	}
	s.sidecar = &Sidecar{server: s}
	err = s.sidecar.Init()
	if err != nil {
		return err
	}
	// log库有空再整理下, 这块需要重构
	s.log = log.NewLoggerWrapper()
	s.codec = &JsonCodec{}
	s.timerStore = &TimeStore{
		server: s,
	}
	s.timerStore.Init()
	return nil
}

func (s *Server) ClusterName() string {
	return s.config.ClusterName
}

// NewServer后即刻设置, 不加锁了
func (s *Server) SetLogger(l log.Logger) {
	if l != nil {
		s.log.SetLogger(l)
	}
}

func (s *Server) GetLogger() log.Logger {
	return s.log.GetLogger()
}

// NewServer后即刻设置, 不加锁了
func (s *Server) SetCodec(c Codec) {
	if c != nil {
		s.codec = c
	}
}

func (s *Server) loadConfig(config string) error {
	data, err := ioutil.ReadFile(config)
	if err != nil {
		s.log.Errorf("load config %s failed:%v\n", config, err)
		return err
	}
	err = json.Unmarshal(data, &s.config)
	if err != nil {
		s.log.Errorf("load config %s failed:%v.\n", config, err)
		return err
	}
	return nil
}

func (s *Server) NewService(svcName string, svcID uint32) (*Service, error) {
	s.rwMu.Lock()
	defer s.rwMu.Unlock()
	handle := SVC_HANDLE(utils.MakeServiceHandle(s.ClusterName(), svcName, svcID))
	if s.services[handle] != nil {
		return nil, fmt.Errorf("register same service: %s-%d", svcName, svcID)
	}
	svc := &Service{
		server: s,
		name:   svcName,
		instID: svcID,
		handle: handle,
	}
	svc.Init()
	s.services[handle] = svc
	if s.svcGroup[svcName] == nil {
		s.svcGroup[svcName] = make(map[uint32]*Service)
	}
	s.svcGroup[svcName][svcID] = svc
	go svc.Serve()
	return svc, nil
}

func (s *Server) DelService(svcName string, svcID uint32) {
	s.rwMu.Lock()
	defer s.rwMu.Unlock()
	handle := SVC_HANDLE(utils.MakeServiceHandle(s.ClusterName(), svcName, svcID))
	svc := s.services[handle]
	if svc != nil {
		svc.Exit()
	}
	delete(s.services, handle)
	delete(s.svcGroup[svcName], svcID)
}

func (s *Server) GetService(handle SVC_HANDLE) *Service {
	s.rwMu.RLock()
	defer s.rwMu.RUnlock()
	return s.services[handle]
}

//接收指定信号，优雅退出接口
func (s *Server) WaitExit(sigs ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sigs...)
	sig := <-c
	s.log.Infof("Server(%v) exitNotify with signal(%d)\n", syscall.Getpid(), sig)
	s.Exit()
}

func (s *Server) Exit() {
	s.sidecar.Exit()
	svcs := make([]*Service, 0)
	s.rwMu.RLock()
	for _, svc := range s.services {
		svcs = append(svcs, svc)
	}
	s.rwMu.RUnlock()
	for _, svc := range svcs {
		svc.Exit()
	}
}
