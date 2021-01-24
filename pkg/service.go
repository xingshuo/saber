package saber

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/xingshuo/saber/common/lib"
	"github.com/xingshuo/saber/common/log"
	"github.com/xingshuo/saber/common/utils"
)

type SvcHandlerFunc func(ctx context.Context, req interface{}) (rsp interface{}, err error)
type SvcTimerFunc func()
type SvcTimer struct {
	onTick SvcTimerFunc
	count  int
}

type ServiceConfig struct {
	Parallel bool // 是否真并发
}

type Service struct {
	config       ServiceConfig
	server       *Server
	name         string // 服务名 如: chat, agent
	instID       uint32 // 服务实例ID
	handle       SVC_HANDLE
	mqueue       *MsgQueue
	svcHandlers  map[string]SvcHandlerFunc
	svcTimers    map[uint32]*SvcTimer
	rwMu         sync.RWMutex // 其实伪并发模式不需要
	msgNotify    chan struct{}
	exitNotify   *lib.SyncEvent
	exitDone     *lib.SyncEvent
	sessionStore *SessionStore
	suspend      chan struct{}
	log          *log.LogSystem
	codec        Codec
	packBuffer   [PACK_BUFFER_SIZE]byte
}

func (s *Service) String() string {
	return fmt.Sprintf("[%s-%d]", s.name, s.instID)
}

//as C++ constructor
func (s *Service) Init() {
	s.msgNotify = make(chan struct{}, 1)
	s.mqueue = NewMQueue(DEFAULT_MQ_SIZE)
	s.svcHandlers = make(map[string]SvcHandlerFunc)
	s.sessionStore = &SessionStore{waitPool: s.server.waitPool}
	s.sessionStore.Init()
	s.svcTimers = make(map[uint32]*SvcTimer)
	s.exitNotify = lib.NewSyncEvent()
	s.exitDone = lib.NewSyncEvent()
	s.suspend = make(chan struct{}, 1)
	s.log = s.server.GetLogSystem()
	s.codec = s.server.codec
}

// 服务启动时注册
func (s *Service) RegisterSvcHandler(method string, handler SvcHandlerFunc) {
	s.rwMu.Lock()
	s.svcHandlers[method] = handler
	s.rwMu.Unlock()
}

// interval:执行间隔, 单位:毫秒
// count: 执行次数, > 0:有限次, == 0:无限次
func (s *Service) RegisterTimer(onTick SvcTimerFunc, interval int64, count int) uint32 {
	if interval <= MIN_TICK_INTERVAL_MS {
		interval = MIN_TICK_INTERVAL_MS
	}
	session := s.sessionStore.NewSessionID()
	t := &SvcTimer{
		onTick: onTick,
		count:  count,
	}
	s.rwMu.Lock()
	s.svcTimers[session] = t
	s.rwMu.Unlock()
	s.server.timerStore.Push(s.handle, session, interval, count)
	return session
}

func (s *Service) UnRegisterTimer(session uint32) {
	s.rwMu.Lock()
	delete(s.svcTimers, session)
	s.rwMu.Unlock()
	s.server.timerStore.Remove(s.handle, session)
}

// 服务自定义logger实现
func (s *Service) SetLogSystem(logger log.Logger, lv log.LogLevel) {
	s.log = log.NewLogSystem(logger, lv)
}

func (s *Service) GetLogSystem() *log.LogSystem {
	return s.log
}

// 支持服务自定义, 以便与三方服务对接
func (s *Service) SetCodec(c Codec) {
	if c != nil {
		s.codec = c
	}
}

func (s *Service) isParallel() bool {
	return s.config.Parallel
}

func (s *Service) onSvcTimer(session uint32) {
	if !s.config.Parallel {
		defer func() {
			s.suspend <- struct{}{}
		}()
	}
	s.rwMu.RLock()
	t := s.svcTimers[session]
	s.rwMu.RUnlock()
	if t != nil {
		t.onTick()
		// 有限次执行
		s.rwMu.Lock()
		if t.count > 0 {
			t.count--
			if t.count == 0 {
				delete(s.svcTimers, session)
			}
		}
		s.rwMu.Unlock()
	}
}

func (s *Service) onRecvSvcReq(source SVC_HANDLE, session uint32, msg interface{}) {
	if !s.config.Parallel {
		defer func() {
			s.suspend <- struct{}{}
			if e := recover(); e != nil {
				s.log.Errorf("panic occurred on recv svc req: %v", e)
			}
		}()
	}
	req := msg.(*SvcRequest)
	handler := s.svcHandlers[req.Method]
	if handler == nil {
		if session != 0 {
			err := s.rawSend(context.Background(), source, MSG_TYPE_SVC_RSP, session, &SvcResponse{
				Err: fmt.Errorf("call unknown func %s", req.Method),
			})
			if err != nil {
				s.log.Errorf("reply svc msg err:%v", err)
			}
		}
		return
	}
	ctx := context.WithValue(context.Background(), CtxKeyService, s)
	// svc, _ := ctx.Value(CtxKeyService).(*Service)
	rsp, err := handler(ctx, req.Body)
	if session != 0 {
		rspErr := s.rawSend(ctx, source, MSG_TYPE_SVC_RSP, session, &SvcResponse{
			Body: rsp,
			Err:  err,
		})
		if rspErr != nil {
			s.log.Errorf("reply svc msg err:%v", rspErr)
		}
	}
}

func (s *Service) onRecvSvcRsp(source SVC_HANDLE, session uint32, msg interface{}) {
	// 根据session唤醒:如果成功, 这里不需要唤醒suspend chan, 等发起rpc的goroutine处理完自己唤醒
	rsp := msg.(*SvcResponse)
	err := s.sessionStore.WakeUp(session, rsp)
	if err != nil {
		s.log.Errorf("wakeup Session %d from %d err: %v", session, source, err)
		// 失败要唤醒下
		if !s.config.Parallel {
			s.suspend <- struct{}{}
		}
	}
}

func (s *Service) onRecvClusterReq(source SVC_HANDLE, session uint32, msg interface{}) {
	if !s.config.Parallel {
		defer func() {
			s.suspend <- struct{}{}
			if e := recover(); e != nil {
				s.log.Errorf("panic occurred on recv cluster req: %v", e)
			}
		}()
	}
	cluster, exist := s.server.sidecar.GetClusterName(source)
	req := msg.(*SvcRequest)
	arg, err := s.codec.Unmarshal(MSG_TYPE_CLUSTER_REQ, req.Method, req.Body.([]byte))
	if err != nil {
		s.log.Errorf("codec.Unmarshal cluster req err:%v", err)
		return
	}

	handler := s.svcHandlers[req.Method]
	if handler == nil {
		if session != 0 {
			rpcErr := fmt.Errorf("unknown rpc func %s", req.Method)
			if !exist {
				s.log.Errorf("recv %v from unknown cluster", rpcErr)
				return
			}
			data, err := NetPackResponse(s.packBuffer[:], s.codec, s.handle, session, source, req.Method, nil, rpcErr)
			if err != nil {
				s.log.Errorf("netpack rsp err:%v", err)
				return
			}
			err = s.server.sidecar.Send(cluster, data)
			if err != nil {
				s.log.Errorf("reply cluster rpc err:%v", err)
			}
		}
		return
	}
	ctx := context.WithValue(context.Background(), CtxKeyService, s)
	rsp, rpcErr := handler(ctx, arg)
	if session != 0 {
		data, err := NetPackResponse(s.packBuffer[:], s.codec, s.handle, session, source, req.Method, rsp, rpcErr)
		if err != nil {
			s.log.Errorf("netpack rsp err:[%v]", err)
			return
		}
		err = s.server.sidecar.Send(cluster, data)
		if err != nil {
			s.log.Errorf("reply cluster rpc err:[%v]", err)
		}
	}
}

func (s *Service) onRecvClusterRsp(source SVC_HANDLE, session uint32, msg interface{}) {
	// 根据session唤醒:如果成功, 这里不需要唤醒suspend chan, 等发起rpc的goroutine处理完自己唤醒
	rsp := msg.(*SvcResponse)
	if rsp.Err == nil {
		body := rsp.Body.(*ClusterRspBody)
		arg, err := s.codec.Unmarshal(MSG_TYPE_CLUSTER_RSP, body.Method, body.Body)
		if err != nil {
			s.log.Errorf("codec.Unmarshal cluster rsp err:%v", err)
			return
		}
		rsp.Body = arg
	}
	err := s.sessionStore.WakeUp(session, rsp)
	if err != nil {
		s.log.Errorf("wakeup cluster Session %d from err: %v", session, source, err)
		// 失败要唤醒下
		if !s.config.Parallel {
			s.suspend <- struct{}{}
		}
	}
}

func (s *Service) rawSend(ctx context.Context, dh SVC_HANDLE, msgType MsgType, session uint32, msg interface{}) error {
	ds := s.server.GetService(dh)
	if ds == nil {
		return fmt.Errorf("unknown dst svc handle %d", dh)
	}
	ds.pushMsg(ctx, s.handle, msgType, session, msg)
	return nil
}

// 节点内Notify
func (s *Service) Send(ctx context.Context, svcName string, svcID uint32, method string, arg interface{}) error {
	dh := SVC_HANDLE(utils.MakeServiceHandle(s.server.ClusterName(), svcName, svcID))
	req := &SvcRequest{
		Method: method,
		Body:   arg,
	}
	return s.rawSend(ctx, dh, MSG_TYPE_SVC_REQ, 0, req)
}

// 节点内Rpc
func (s *Service) Call(ctx context.Context, svcName string, svcID uint32, method string, arg interface{}) (rsp interface{}, err error) {
	dh := SVC_HANDLE(utils.MakeServiceHandle(s.server.ClusterName(), svcName, svcID))
	ds := s.server.GetService(dh)
	if ds == nil {
		return nil, fmt.Errorf("unknown dst svc %d", dh)
	}
	req := &SvcRequest{
		Method: method,
		Body:   arg,
	}
	session := s.sessionStore.NewSessionID()
	onWait := func() error {
		ds.pushMsg(ctx, s.handle, MSG_TYPE_SVC_REQ, session, req)
		return nil
	}
	return s.sessionStore.Wait(ctx, session, s, onWait)
}

// 跨节点Notify
func (s *Service) SendCluster(ctx context.Context, clusterName, svcName string, svcID uint32, method string, arg interface{}) error {
	dh := SVC_HANDLE(utils.MakeServiceHandle(clusterName, svcName, svcID))
	data, err := NetPackRequest(s.packBuffer[:], s.codec, s.handle, 0, dh, method, arg)
	if err != nil {
		return err
	}
	return s.server.sidecar.Send(clusterName, data)
}

// 跨节点Rpc
func (s *Service) CallCluster(ctx context.Context, clusterName, svcName string, svcID uint32, method string, arg interface{}) (rsp interface{}, err error) {
	dh := SVC_HANDLE(utils.MakeServiceHandle(clusterName, svcName, svcID))
	session := s.sessionStore.NewSessionID()
	data, err := NetPackRequest(s.packBuffer[:], s.codec, s.handle, session, dh, method, arg)
	if err != nil {
		return nil, err
	}
	onWait := func() error {
		return s.server.sidecar.Send(clusterName, data)
	}
	return s.sessionStore.Wait(ctx, session, s, onWait)
}

func (s *Service) pushMsg(ctx context.Context, source SVC_HANDLE, msgType MsgType, session uint32, data interface{}) {
	if s.mqueue.Push(source, msgType, session, data) {
		s.msgNotify <- struct{}{}
	}
}

func (s *Service) pushClusterRequest(ctx context.Context, head *ClusterReqHead, body []byte) {
	req := &SvcRequest{
		Method: head.Method(),
		Body:   body,
	}
	s.pushMsg(ctx, SVC_HANDLE(head.source), MSG_TYPE_CLUSTER_REQ, head.session, req)
}

func (s *Service) pushClusterResponse(ctx context.Context, head *ClusterRspHead, body []byte) {
	if head.errCode == ErrCode_OK {
		rsp := &SvcResponse{
			Body: &ClusterRspBody{
				Method: head.Method(),
				Body:   body,
			},
		}
		s.pushMsg(ctx, SVC_HANDLE(head.source), MSG_TYPE_CLUSTER_RSP, head.session, rsp)
	} else {
		rsp := &SvcResponse{
			Err: errors.New(head.ErrMsg()),
		}
		s.pushMsg(ctx, SVC_HANDLE(head.source), MSG_TYPE_CLUSTER_RSP, head.session, rsp)
	}
}

func (s *Service) dispatchMsg(source SVC_HANDLE, msgType MsgType, session uint32, msg interface{}) {
	if msgType.IsClusterMsg() {
		cluster, _ := s.server.sidecar.GetClusterName(source)
		s.log.Debugf("%s dispatch %s start from %s", s, msgType, cluster)
	} else {
		s.log.Debugf("%s dispatch %s start from %s", s, msgType, s.server.GetService(source))
	}

	if msgType == MSG_TYPE_TIMER {
		go s.onSvcTimer(session)
	} else if msgType == MSG_TYPE_SVC_REQ {
		go s.onRecvSvcReq(source, session, msg)
	} else if msgType == MSG_TYPE_SVC_RSP {
		go s.onRecvSvcRsp(source, session, msg)
	} else if msgType == MSG_TYPE_CLUSTER_REQ {
		go s.onRecvClusterReq(source, session, msg)
	} else if msgType == MSG_TYPE_CLUSTER_RSP {
		go s.onRecvClusterRsp(source, session, msg)
	} else {
		return
	}

	if !s.config.Parallel { // 伪并发
		<-s.suspend
		s.log.Debugf("%s dispatch %s done from %s", s, msgType, s.server.GetService(source))
	}
}

func (s *Service) Serve() {
	s.log.Infof("cluster %s new service %s handle:%d", s.server.ClusterName(), s, s.handle)
	for {
		select {
		case <-s.msgNotify:
			for {
				empty, source, msgType, session, data := s.mqueue.Pop()
				if empty {
					break
				}
				s.dispatchMsg(source, msgType, session, data)
			}
		case <-s.exitNotify.Done():
			for {
				empty, source, msgType, session, data := s.mqueue.Pop()
				if empty {
					break
				}
				s.dispatchMsg(source, msgType, session, data)
			}
			s.exitDone.Fire()
			return
		}
	}
}

func (s *Service) Exit() {
	if s.exitNotify.Fire() {
		<-s.exitDone.Done()
	}
}
