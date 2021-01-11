package saber

import "fmt"

type MsgType int

func (mt MsgType) String() string {
	switch mt {
	case MSG_TYPE_TIMER:
		return "TIMER"
	case MSG_TYPE_SVC_REQ:
		return "SVC_REQ"
	case MSG_TYPE_SVC_RSP:
		return "SVC_RSP"
	default:
		return "unknown"
	}
}

const (
	MSG_TYPE_TIMER MsgType = iota + 1
	MSG_TYPE_SVC_REQ
	MSG_TYPE_SVC_RSP
)

type SvcRequest struct {
	Method string
	Body   interface{}
}

type SvcResponse struct {
	Body interface{}
	Err  error
}

type Message struct {
	source  SVC_HANDLE
	msgType MsgType
	session uint32
	data    interface{}
}

func (m *Message) String() string {
	return fmt.Sprintf("msg:[%d,%s,%d]", m.source, m.msgType, m.session)
}