package saber

import (
	"fmt"
)

type MsgType int

func (mt MsgType) IsClusterMsg() bool {
	return mt == MSG_TYPE_CLUSTER_REQ || mt == MSG_TYPE_CLUSTER_RSP
}

func (mt MsgType) String() string {
	switch mt {
	case MSG_TYPE_TIMER:
		return "TIMER"
	case MSG_TYPE_SVC_REQ:
		return "SVC_REQ"
	case MSG_TYPE_SVC_RSP:
		return "SVC_RSP"
	case MSG_TYPE_CLUSTER_REQ:
		return "CLUSTER_REQ"
	case MSG_TYPE_CLUSTER_RSP:
		return "CLUSTER_RSP"
	default:
		return "unknown"
	}
}

const (
	MSG_TYPE_TIMER MsgType = iota + 1
	MSG_TYPE_SVC_REQ
	MSG_TYPE_SVC_RSP
	MSG_TYPE_CLUSTER_REQ
	MSG_TYPE_CLUSTER_RSP
)

type SvcRequest struct {
	Method string
	Body   interface{}
}

type SvcResponse struct {
	Body interface{}
	Err  error
}

// 由sidecar挂载到SvcResponse.Body上,分发给目标service处理
type ClusterRspBody struct {
	Method string
	Body   []byte
}

type Message struct {
	Source  SVC_HANDLE
	MsgType MsgType
	Session uint32
	Data    interface{}
}

func (m *Message) String() string {
	return fmt.Sprintf("msg:[%d,%s,%d]", m.Source, m.MsgType, m.Session)
}
