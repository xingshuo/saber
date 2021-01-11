package saber

const (
	DEFAULT_MQ_SIZE = 1024
	DEFAULT_TIMER_CAP = 1024
	MIN_TICK_INTERVAL_MS = 10
)

const (
	CtxKeyService = "SaberService"
	CtxKeyRpcTimeoutMS = "SaberRpcTimeout"
)

type SVC_HANDLE uint64