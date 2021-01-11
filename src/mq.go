package saber

import (
	"fmt"
	"sync"
)

// 循环数组消息队列
func NewMQueue(cap int) *MsgQueue {
	mq := &MsgQueue{
		head: 0,
		tail: 0,
		cap:  cap,
		data: make([]*Message, cap),
	}
	return mq
}

type MsgQueue struct {
	head int // 队头
	tail int // 队尾(指向下一个可放置位置)
	cap  int
	data []*Message
	rwMu sync.RWMutex
}

func (mq *MsgQueue) expand() {
	newq := make([]*Message, mq.cap*2)
	for i := 0; i < mq.cap; i++ {
		newq[i] = mq.data[(mq.head+i)%mq.cap]
	}
	mq.data = newq
	mq.tail = mq.cap
	mq.head = 0
	mq.cap *= 2
}

func (mq *MsgQueue) Push(m *Message) {
	mq.rwMu.Lock()
	defer mq.rwMu.Unlock()
	mq.data[mq.tail] = m
	mq.tail++
	if mq.tail >= mq.cap {
		mq.tail = 0
	}
	if mq.head == mq.tail {
		mq.expand()
	}
}

func (mq *MsgQueue) Pop() *Message {
	mq.rwMu.Lock()
	defer mq.rwMu.Unlock()
	if mq.head == mq.tail { // 由于Push时相等会扩容,所以相等只可能是空
		return nil
	}
	top := mq.data[mq.head]
	mq.head++
	if mq.head >= mq.cap {
		mq.head = 0
	}
	return top
}

func (mq *MsgQueue) Peek() *Message {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	if mq.head == mq.tail { // 由于Push时相等会扩容,所以相等只可能是空
		return nil
	}
	return mq.data[mq.head]
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
