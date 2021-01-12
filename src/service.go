package saber

import (
	"context"
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
	log          *log.LoggerWrapper
}

func (s *Service) String() string {
	return fmt.Sprintf("[%s-%d]", s.name, s.instID)
}

func (s *Service) Init() {
	s.msgNotify = make(chan struct{}, 1)
	s.mqueue = NewMQueue(DEFAULT_MQ_SIZE)
	s.svcHandlers = make(map[string]SvcHandlerFunc)
	s.sessionStore = &SessionStore{}
	s.sessionStore.Init()
	s.svcTimers = make(map[uint32]*SvcTimer)
	s.exitNotify = lib.NewEvent()
	s.exitDone = lib.NewEvent()
	s.suspend = make(chan struct{}, 1)
	s.log = log.NewLoggerWrapper()
	s.log.SetLogger(s.server.GetLogger())
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
func (s *Service) SetLogger(logger log.Logger) {
	s.log.SetLogger(logger)
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
		sErr := s.rawSend(ctx, source, MSG_TYPE_SVC_RSP, session, &SvcResponse{
			Body: rsp,
			Err:  err,
		})
		if sErr != nil {
			s.log.Errorf("reply svc msg err:%v", sErr)
		}
	}
}

func (s *Service) onRecvSvcRsp(source SVC_HANDLE, session uint32, msg interface{}) {
	// 根据session唤醒, 这里不需要唤醒suspend chan, 等发起rpc的goroutine处理完自己唤醒
	rsp := msg.(*SvcResponse)
	err := s.sessionStore.WakeUp(session, rsp)
	if err != nil {
		s.log.Errorf("wakeup session %d err: %v", session, err)
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

func (s *Service) Send(ctx context.Context, svcName string, svcID uint32, method string, arg interface{}) error {
	dh := SVC_HANDLE(utils.MakeServiceHandle(svcName, svcID))
	req := &SvcRequest{
		Method: method,
		Body:   arg,
	}
	return s.rawSend(ctx, dh, MSG_TYPE_SVC_REQ, 0, req)
}

func (s *Service) Call(ctx context.Context, svcName string, svcID uint32, method string, arg interface{}) (rsp interface{}, err error) {
	dh := SVC_HANDLE(utils.MakeServiceHandle(svcName, svcID))
	ds := s.server.GetService(dh)
	if ds == nil {
		return nil, fmt.Errorf("unknown dst svc %d", dh)
	}
	req := &SvcRequest{
		Method: method,
		Body:   arg,
	}
	return s.sessionStore.Wait(ctx, s, ds, req)
}

func (s *Service) pushMsg(ctx context.Context, source SVC_HANDLE, msgType MsgType, session uint32, data interface{}) {
	s.mqueue.Push(&Message{
		source:  source,
		msgType: msgType,
		session: session,
		data:    data,
	})
	// 这里性能会不会有问题??
	select {
	case s.msgNotify <- struct{}{}:
	default:
	}
}

func (s *Service) dispatchMsg(source SVC_HANDLE, msgType MsgType, session uint32, msg interface{}) {
	s.log.Debugf("%s dispatch %s start from %s", s, msgType, s.server.GetService(source))
	if msgType == MSG_TYPE_TIMER {
		go s.onSvcTimer(session)
	} else if msgType == MSG_TYPE_SVC_REQ {
		go s.onRecvSvcReq(source, session, msg)
	} else if msgType == MSG_TYPE_SVC_RSP {
		go s.onRecvSvcRsp(source, session, msg)
	} else {
		return
	}
	if !s.config.Parallel { // 伪并发
		<-s.suspend
		s.log.Debugf("%s dispatch %s done from %s", s, msgType, s.server.GetService(source))
	}
}

func (s *Service) Serve() {
	for {
		select {
		case <-s.msgNotify:
			for {
				msg := s.mqueue.Pop()
				if msg == nil {
					break
				}
				s.dispatchMsg(msg.source, msg.msgType, msg.session, msg.data)
			}
		case <-s.exitNotify.Done():
			for {
				msg := s.mqueue.Pop()
				if msg == nil {
					break
				}
				s.dispatchMsg(msg.source, msg.msgType, msg.session, msg.data)
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
