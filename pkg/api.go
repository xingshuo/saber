package api

import (
	"context"

	"github.com/xingshuo/saber/common/log"
	saber "github.com/xingshuo/saber/src"
)

const (
	LevelDebug   = log.LevelDebug
	LevelInfo    = log.LevelInfo
	LevelWarning = log.LevelWarning
	LevelError   = log.LevelError
)

func NewServer(config string, logger log.Logger) (*saber.Server, error) {
	s := &saber.Server{}
	err := s.Init(config, logger)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetSvcFromCtx(ctx context.Context) *saber.Service {
	svc, ok := ctx.Value(saber.CtxKeyService).(*saber.Service)
	if !ok {
		return nil
	}
	return svc
}
