package saber

type Sidecar struct {
	server       *Server
	clusterAddrs map[string]string // clustername: address
}

func (sc *Sidecar) Init() error {
	sc.clusterAddrs = make(map[string]string)
	// 从配置中读取clustername表, 绑定本地端口
	return nil
}

func (sc *Sidecar) Exit() {

}
