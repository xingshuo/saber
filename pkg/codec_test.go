package saber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterReqHead(t *testing.T) {
	var buffer [PACK_BUFFER_SIZE]byte
	h := &ClusterReqHead{}
	err := h.Init(100001, 7999, 200002, "getFriends")
	assert.Nil(t, err)
	l, err := h.Pack(buffer[:])
	assert.Nil(t, err)
	h2 := &ClusterReqHead{}
	l2, err := h2.Unpack(buffer[:])
	assert.Nil(t, err)
	assert.Equal(t, uint64(100001), h2.source)
	assert.Equal(t, uint64(200002), h2.destination)
	assert.Equal(t, uint32(7999), h2.session)
	assert.Equal(t, h2.Method(), "getFriends")
	assert.Equal(t, l, l2)
	assert.Equal(t, uintptr(21+len(h2.Method())), l)
}

func TestClusterRspHead(t *testing.T) {
	var buffer [PACK_BUFFER_SIZE]byte
	h := &ClusterRspHead{}
	err := h.Init(100001, 7999, 200002, "getFriends", ErrCode_Usr, "no such player")
	assert.Nil(t, err)
	l, err := h.Pack(buffer[:])
	assert.Nil(t, err)
	h2 := &ClusterRspHead{}
	l2, err := h2.Unpack(buffer[:])
	assert.Nil(t, err)
	assert.Equal(t, uint64(100001), h2.source)
	assert.Equal(t, uint64(200002), h2.destination)
	assert.Equal(t, uint32(7999), h2.session)
	assert.Equal(t, h2.Method(), "getFriends")
	assert.Equal(t, h2.ErrMsg(), "no such player")
	assert.Equal(t, h2.errCode, ErrCode_Usr)
	assert.Equal(t, l, l2)
	assert.Equal(t, uintptr(21+len(h2.Method())+5+len(h2.ErrMsg())), l)
}
