package saber

import (
	"context"

	"github.com/xingshuo/saber/common/utils"
)

func NewServer(config string) (*Server, error) {
	s := &Server{}
	err := s.Init(config)
	if err != nil {
		return nil, err
	}
	s.GetLogSystem().Infof("cluster %s start run on %s", s.ClusterName(), utils.GetIP())
	return s, nil
}

func GetSvcFromCtx(ctx context.Context) *Service {
	svc, ok := ctx.Value(CtxKeyService).(*Service)
	if !ok {
		return nil
	}
	return svc
}
