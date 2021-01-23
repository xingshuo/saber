package saber

const (
	DEFAULT_MQ_SIZE      = 1024
	DEFAULT_TIMER_CAP    = 1024
	MIN_TICK_INTERVAL_MS = 10
	CLUSTER_NAME_MAX_LEN = 64
	METHOD_MAX_LEN       = 64
	PACK_BUFFER_SIZE     = 8192
	ERR_MSG_MAX_LEN      = 256
)

const (
	CtxKeyService      = "SaberService"
	CtxKeyRpcTimeoutMS = "SaberRpcTimeout"
)

type SVC_HANDLE uint64
