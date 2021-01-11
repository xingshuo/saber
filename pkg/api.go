package api

import "github.com/xingshuo/saber/src"

func NewServer(config string) (*saber.Server, error) {
	s := &saber.Server{}
	err := s.Init(config)
	if err != nil {
		return nil, err
	}
	return s, nil
}
