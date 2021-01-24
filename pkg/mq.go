package saber

import (
	"fmt"
	"sync"
)

// 循环数组消息队列
func NewMQueue(cap int, exit <-chan struct{}) *MsgQueue {
	mq := &MsgQueue{
		head:        0,
		tail:        0,
		cap:         cap,
		data:        make([]Message, cap),
		waitConsume: false,
		waiting:     make(chan struct{}, 1),
		exit:        exit,
	}
	return mq
}

type MsgQueue struct {
	head        int // 队头
	tail        int // 队尾(指向下一个可放置位置)
	cap         int
	data        []Message
	rwMu        sync.RWMutex
	waitConsume bool
	waiting     chan struct{}
	exit        <-chan struct{}
}

func (mq *MsgQueue) expand() {
	newq := make([]Message, mq.cap*2)
	for i := 0; i < mq.cap; i++ {
		newq[i] = mq.data[(mq.head+i)%mq.cap]
	}
	mq.data = newq
	mq.tail = mq.cap
	mq.head = 0
	mq.cap *= 2
}

func (mq *MsgQueue) Push(source SVC_HANDLE, msgType MsgType, session uint32, data interface{}) {
	wakeUp := false
	mq.rwMu.Lock()
	if mq.waitConsume {
		wakeUp = true
		mq.waitConsume = false
	}
	back := &mq.data[mq.tail]
	back.Source = source
	back.MsgType = msgType
	back.Session = session
	back.Data = data
	mq.tail++
	if mq.tail >= mq.cap {
		mq.tail = 0
	}
	if mq.head == mq.tail {
		mq.expand()
	}
	mq.rwMu.Unlock()
	if wakeUp {
		select {
		case mq.waiting <- struct{}{}:
		default:
		}
	}
}

// 队列为空时会阻塞, 直到有新消息或退出
func (mq *MsgQueue) Pop() (empty bool, source SVC_HANDLE, msgType MsgType, session uint32, data interface{}) {
begin:
	mq.rwMu.Lock()
	if mq.head == mq.tail { // 由于Push时相等会扩容,所以相等只可能是空
		mq.waitConsume = true
		mq.rwMu.Unlock()
		select {
		case <-mq.waiting:
			goto begin
		case <-mq.exit:
			return true, 0, 0, 0, nil
		}
	}
	top := &mq.data[mq.head]
	mq.head++
	if mq.head >= mq.cap {
		mq.head = 0
	}
	mq.rwMu.Unlock()
	return false, top.Source, top.MsgType, top.Session, top.Data
}

func (mq *MsgQueue) Peek() (empty bool, source SVC_HANDLE, msgType MsgType, session uint32, data interface{}) {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	if mq.head == mq.tail { // 由于Push时相等会扩容,所以相等只可能是空
		return true, 0, 0, 0, nil
	}
	top := &mq.data[mq.head]
	return false, top.Source, top.MsgType, top.Session, top.Data
}

func (mq *MsgQueue) Len() int {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	if mq.tail >= mq.head {
		return mq.tail - mq.head
	}
	return mq.tail - mq.head + mq.cap
}

func (mq *MsgQueue) debug() {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	fmt.Printf("head:%d tail:%d cap:%d len:%d\n", mq.head, mq.tail, mq.cap, mq.Len())
	for i := mq.head; i != mq.tail; i = (i + 1) % mq.cap {
		fmt.Println(mq.data[i])
	}
}
