package saber

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type waitPool struct {
	pool sync.Pool
}

func (p *waitPool) get() chan *SvcResponse {
	return p.pool.Get().(chan *SvcResponse)
}

func (p *waitPool) put(c chan *SvcResponse) {
	p.pool.Put(c)
}

func newWaitPool() *waitPool {
	return &waitPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make(chan *SvcResponse, 1) // 防止写端先写入,读端还未进入读取状态, 设置缓存大小为1
			},
		},
	}
}

type SessionStore struct {
	waitSessions map[uint32]chan *SvcResponse
	waitPool     *waitPool
	seq          uint32
}

func (ss *SessionStore) Init() {
	ss.waitSessions = make(map[uint32]chan *SvcResponse)
}

func (ss *SessionStore) NewSessionID() uint32 {
	id := atomic.AddUint32(&ss.seq, 1)
	if id != 0 {
		return id
	}
	return atomic.AddUint32(&ss.seq, 1)
}

func (ss *SessionStore) WakeUp(session uint32, rsp *SvcResponse) error {
	done := ss.waitSessions[session]
	if done == nil {
		return RPC_SESSION_NOEXIST_ERR
	}
	delete(ss.waitSessions, session)
	select {
	case done <- rsp:
		return nil
	default:
		return RPC_WAKEUP_ERR
	}
}

func (ss *SessionStore) Wait(ctx context.Context, session uint32, srcSvc *Service, onWait func() error) (interface{}, error) {
	done := ss.waitSessions[session]
	// 理论上不可能出现
	if done != nil {
		return nil, RPC_SESSION_REPEAT_ERR
	}
	done = ss.waitPool.get() // 防止写端先写入,读端还未进入读取状态, 设置缓存大小为1
	ss.waitSessions[session] = done
	err := onWait()
	if err != nil {
		delete(ss.waitSessions, session)
		ss.waitPool.put(done)
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
			delete(ss.waitSessions, session)
			ss.waitPool.put(done)
			return nil, fmt.Errorf("%w session %d", RPC_TIMEOUT_ERR, session)
		case rsp := <-done:
			ss.waitPool.put(done)
			timer.Stop()
			return rsp.Body, rsp.Err
		}
	}
	select {
	case rsp := <-done:
		ss.waitPool.put(done)
		return rsp.Body, rsp.Err
	}
}
