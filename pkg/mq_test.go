package saber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgQueue(t *testing.T) {
	mq := NewMQueue(3)
	data := []Message{
		{0, MSG_TYPE_TIMER, 1, nil},
		{0, MSG_TYPE_TIMER, 2, nil},
		{0, MSG_TYPE_TIMER, 3, nil},
		{0, MSG_TYPE_TIMER, 4, nil},
		{0, MSG_TYPE_TIMER, 5, nil},
		{0, MSG_TYPE_TIMER, 6, nil},
		{0, MSG_TYPE_TIMER, 7, nil},
		{0, MSG_TYPE_TIMER, 8, nil},
	}
	for i := range data {
		mq.Push(data[i].Source, data[i].MsgType, data[i].Session, data[i].Data)
	}
	assert.Equal(t, 0, mq.head)
	assert.Equal(t, 8, mq.tail)
	assert.Equal(t, 8, mq.Len())
	assert.Equal(t, 12, mq.cap)
	// 绕圈
	for i := 0; i < 4; i++ {
		mq.Pop()
	}
	assert.Equal(t, 4, mq.head)
	assert.Equal(t, 8, mq.tail)
	assert.Equal(t, 4, mq.Len())
	data = []Message{
		{0, MSG_TYPE_TIMER, 9, nil},
		{0, MSG_TYPE_TIMER, 10, nil},
		{0, MSG_TYPE_TIMER, 11, nil},
		{0, MSG_TYPE_TIMER, 12, nil},
		{0, MSG_TYPE_TIMER, 13, nil},
		{0, MSG_TYPE_TIMER, 14, nil},
		{0, MSG_TYPE_TIMER, 15, nil},
	}
	for i := range data {
		mq.Push(data[i].Source, data[i].MsgType, data[i].Session, data[i].Data)
	}
	assert.Equal(t, 4, mq.head)
	assert.Equal(t, 3, mq.tail)
	assert.Equal(t, 11, mq.Len())
	assert.Equal(t, 12, mq.cap)
	_, _, _, session, _ := mq.Peek()
	assert.Equal(t, uint32(5), session)
	// 打印一下
	mq.debug()
	// 临界触发expand
	mq.Push(0, MSG_TYPE_TIMER, 16, nil)
	assert.Equal(t, 0, mq.head)
	assert.Equal(t, 12, mq.tail)
	assert.Equal(t, 12, mq.Len())
	assert.Equal(t, 24, mq.cap)
	_, _, _, session, _ = mq.Peek()
	assert.Equal(t, uint32(5), session)
}
