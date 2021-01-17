package api

import (
	"context"

	"github.com/xingshuo/saber/common/log"
	"github.com/xingshuo/saber/common/utils"
	saber "github.com/xingshuo/saber/src"
)

const (
	LevelDebug   = log.LevelDebug
	LevelInfo    = log.LevelInfo
	LevelWarning = log.LevelWarning
	LevelError   = log.LevelError
)

func NewServer(config string) (*saber.Server, error) {
	s := &saber.Server{}
	err := s.Init(config)
	if err != nil {
		return nil, err
	}
	s.GetLogger().Logf(LevelInfo, "cluster %s start run on %s", s.ClusterName(), utils.GetIP())
	return s, nil
}

func GetSvcFromCtx(ctx context.Context) *saber.Service {
	svc, ok := ctx.Value(saber.CtxKeyService).(*saber.Service)
	if !ok {
		return nil
	}
	return svc
}
