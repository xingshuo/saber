package saber

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type SessionStore struct {
	pending map[uint32]chan *SvcResponse
	mu      sync.Mutex
	seq     uint32
}

func (ss *SessionStore) Init() {
	ss.mu.Lock()
	ss.pending = make(map[uint32]chan *SvcResponse)
	ss.mu.Unlock()
}

func (ss *SessionStore) NewSessionID() uint32 {
	id := atomic.AddUint32(&ss.seq, 1)
	if id != 0 {
		return id
	}
	return atomic.AddUint32(&ss.seq, 1)
}

func (ss *SessionStore) WakeUp(session uint32, rsp *SvcResponse) error {
	ss.mu.Lock()
	done, ok := ss.pending[session]
	if !ok {
		ss.mu.Unlock()
		return RPC_SESSION_NOEXIST_ERR
	}
	delete(ss.pending, session)
	ss.mu.Unlock()
	select {
	case done <- rsp:
		return nil
	default:
		return RPC_WAKEUP_ERR
	}
}

func (ss *SessionStore) Wait(ctx context.Context, session uint32, srcSvc *Service, onWait func() error) (interface{}, error) {
	ss.mu.Lock()
	done := ss.pending[session]
	// 理论上不可能出现
	if done != nil {
		ss.mu.Unlock()
		return nil, RPC_SESSION_REPEAT_ERR
	}
	// 这里可以复用chan降低内存分配开销??
	done = make(chan *SvcResponse, 1) // 防止写端先写入,读端还未进入读取状态, 设置缓存大小为1
	ss.pending[session] = done
	ss.mu.Unlock()
	err := onWait()
	if err != nil {
		return nil, err
	}
	if !srcSvc.isParallel() { // 通知Serve继续处理其他消息
		srcSvc.suspend <- struct{}{}
	}
	timeout, _ := ctx.Value(CtxKeyRpcTimeoutMS).(time.Duration)
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		select {
		case <-timer.C:
			ss.mu.Lock()
			delete(ss.pending, session)
			ss.mu.Unlock()
			return nil, RPC_TIMEOUT_ERR
		case rsp := <-done:
			timer.Stop()
			return rsp.Body, rsp.Err
		}
	}
	select {
	case rsp := <-done:
		return rsp.Body, rsp.Err
	}
}
