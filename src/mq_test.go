package saber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgQueue(t *testing.T) {
	mq := NewMQueue(3)
	data := []*Message{
		{0, MSG_TYPE_LOGIC,1, nil},
		{0, MSG_TYPE_LOGIC,2, nil},
		{0, MSG_TYPE_LOGIC,3, nil},
		{0, MSG_TYPE_LOGIC,4, nil},
		{0, MSG_TYPE_LOGIC,5, nil},
		{0, MSG_TYPE_LOGIC,6, nil},
		{0, MSG_TYPE_LOGIC,7, nil},
		{0, MSG_TYPE_LOGIC,8, nil},
	}
	for i := range data {
		mq.Push(data[i])
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
	data = []*Message{
		{0, MSG_TYPE_LOGIC,9, nil},
		{0, MSG_TYPE_LOGIC,10, nil},
		{0, MSG_TYPE_LOGIC,11, nil},
		{0, MSG_TYPE_LOGIC,12, nil},
		{0, MSG_TYPE_LOGIC,13, nil},
		{0, MSG_TYPE_LOGIC,14, nil},
		{0, MSG_TYPE_LOGIC,15, nil},
	}
	for i := range data {
		mq.Push(data[i])
	}
	assert.Equal(t, 4, mq.head)
	assert.Equal(t, 3, mq.tail)
	assert.Equal(t, 11, mq.Len())
	assert.Equal(t, 12, mq.cap)
	assert.Equal(t, 5, mq.Peek().session)
	// 打印一下
	mq.debug()
	// 临界触发expand
	mq.Push(&Message{0, MSG_TYPE_LOGIC,16, nil})
	assert.Equal(t, 0, mq.head)
	assert.Equal(t, 12, mq.tail)
	assert.Equal(t, 12, mq.Len())
	assert.Equal(t, 24, mq.cap)
	assert.Equal(t, 5, mq.Peek().session)
}
