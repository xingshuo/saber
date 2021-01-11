package saber

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Ticker struct {
	// 所属服务
	handle SVC_HANDLE
	// 定时器标识
	session uint32
	// Tick间隔:毫秒
	interval int64
	// 执行次数: > 0有限次, == 0无限次
	count int
	// 下一次触发时间: 纳秒
	nextTime int64
	// 是否失效
	expired bool
}

type Heapq struct {
	size int
	cap  int
	// 注意: 堆头在索引1处
	data []*Ticker
}

func (hq *Heapq) push(t *Ticker) error {
	if hq.size >= hq.cap {
		newq := make([]*Ticker, 2*hq.cap + 1)
		for i := 1; i <= hq.size; i++ {
			newq[i] = hq.data[i]
		}
		hq.data = newq
		hq.cap *= 2
	}
	hq.size++
	hq.data[hq.size] = t
	hq.shiftup(hq.size)
	return nil
}

func (hq *Heapq) top() *Ticker {
	if hq.size < 1 {
		return nil
	}
	return hq.data[1]
}

func (hq *Heapq) pop() *Ticker {
	if hq.size < 1 {
		return nil
	}
	top := hq.data[1]
	hq.data[1] = hq.data[hq.size]
	hq.size--
	hq.shiftdown(1)
	return top
}

func (hq *Heapq) PopUntil(now time.Time) []TimerSeq {
	var tickSeqs []TimerSeq
	nowTime := now.UnixNano()
	for hq.size >= 1 {
		top := hq.data[1]
		if top.expired {
			hq.pop()
			continue
		}
		if nowTime < top.nextTime {
			break
		}
		tickSeqs = append(tickSeqs, TimerSeq{
			handle:  top.handle,
			session: top.session,
		})
		if top.count > 0 {
			top.count--
			if top.count == 0 { // 移除
				hq.pop()
				continue
			}
		}
		top.nextTime = top.nextTime + top.interval* int64(time.Millisecond)
		hq.shiftdown(1)
	}
	return tickSeqs
}

func (hq *Heapq) shiftup(k int) {
	v := hq.data[k]
	for {
		c := k / 2
		if c <= 0 || hq.data[c].nextTime <= v.nextTime {
			break
		}
		hq.data[k] = hq.data[c]
		k = c
	}
	hq.data[k] = v
}

func (hq *Heapq) shiftdown(k int) {
	v := hq.data[k]
	for {
		c := k * 2
		if c > hq.size {
			break
		}
		if c < hq.size && hq.data[c].nextTime > hq.data[c+1].nextTime {
			c++
		}
		if v.nextTime <= hq.data[c].nextTime {
			break
		}
		hq.data[k] = hq.data[c]
		k = c
	}
	hq.data[k] = v
}

func (hq *Heapq) debug() {
	fmt.Printf("========heapq debug========\n")
	for i := 1; i <= hq.size; i++ {
		fmt.Printf("%dst %p %d\n", i, hq.data[i], hq.data[i].nextTime)
	}
}

type TimerSeq struct {
	handle  SVC_HANDLE
	session uint32
}

type TimeStore struct {
	server *Server
	rwMu sync.RWMutex
	queue  *Heapq
	timers map[TimerSeq]*Ticker
}

func (ts *TimeStore) Init() {
	ts.queue = &Heapq{
		size: 0,
		cap:  DEFAULT_TIMER_CAP,
		data: make([]*Ticker, DEFAULT_TIMER_CAP + 1),
	}
	ts.timers = make(map[TimerSeq]*Ticker)
	go ts.Start()
}

// 阻塞的,需要单独启动一个goroutine调用
func (ts *TimeStore) Start() {
	interval := ts.server.config.TickIntervalMs
	if interval < MIN_TICK_INTERVAL_MS {
		interval = MIN_TICK_INTERVAL_MS
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	for range ticker.C {
		ts.OnTick(time.Now())
	}
}

func (ts *TimeStore) OnTick(now time.Time) {
	ts.rwMu.Lock()
	tickSeqs := ts.queue.PopUntil(now)
	ts.rwMu.Unlock()
	for _,seq := range tickSeqs {
		svc := ts.server.GetService(seq.handle)
		if svc != nil {
			svc.pushMsg(context.Background(), SVC_HANDLE(0), MSG_TYPE_TIMER, seq.session, nil)
		}
	}
}

func (ts *TimeStore) Push(handle SVC_HANDLE, session uint32, interval int64, count int) {
	ts.rwMu.Lock()
	defer ts.rwMu.Unlock()
	t := &Ticker{
		handle:   handle,
		session:  session,
		interval: interval,
		count:    count,
		nextTime: time.Now().UnixNano() + interval * int64(time.Millisecond),
		expired:  false,
	}
	seq := TimerSeq{
		handle:  handle,
		session: session,
	}
	old := ts.timers[seq]
	if old != nil { // 理论上不可能
		old.expired = true
	}
	ts.timers[seq] = t
	ts.queue.push(t)
}

func (ts *TimeStore) Remove(handle SVC_HANDLE, session uint32) {
	ts.rwMu.Lock()
	defer ts.rwMu.Unlock()
	seq := TimerSeq{
		handle:  handle,
		session: session,
	}
	old := ts.timers[seq]
	if old != nil { // 理论上不可能
		old.expired = true
	}
	delete(ts.timers, seq)
}
