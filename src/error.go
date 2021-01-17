package saber

import "fmt"

var (
	RPC_SESSION_REPEAT_ERR    = fmt.Errorf("rpc Session repeat")
	RPC_SESSION_NOEXIST_ERR   = fmt.Errorf("rpc Session no exist")
	RPC_TIMEOUT_ERR           = fmt.Errorf("rpc timeout")
	RPC_WAKEUP_ERR            = fmt.Errorf("rpc wake up err")
	RPC_METHOD_LEN_OVER_ERR   = fmt.Errorf("rpc method len over")
	CLUSTER_NAME_LEN_OVER_ERR = fmt.Errorf("cluster name len over")
	ERR_MSG_LEN_OVER          = fmt.Errorf("err msg len over")
	RPC_RETVALUE_NUM_ERR      = fmt.Errorf("rpc return value num error")
	RPC_RETVALUE_TYPE_ERR     = fmt.Errorf("rpc return value type error")
	PACK_BUFFER_SHORT_ERR     = fmt.Errorf("pack buffer not enough")
	UNPACK_BUFFER_SHORT_ERR   = fmt.Errorf("unpack buffer not enough")
	MSG_TYPE_ERR              = fmt.Errorf("msg type error")
)

var (
	ErrCode_OK  uint32 = 0
	ErrCode_Usr uint32 = 10001 // 业务层逻辑错误
)
