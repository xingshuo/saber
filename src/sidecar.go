package saber

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/xingshuo/saber/common/netframe"
	"github.com/xingshuo/saber/common/utils"
)

type ClusterProxy struct {
	// 不加锁应该没问题
	rmtClusters map[string]string           // clustername: address
	hashToNames map[uint32]string           // hashID : clustername
	dialers     map[string]*netframe.Dialer // clustername: dialer
	rwMu        sync.RWMutex
}

func (p *ClusterProxy) Reload(clusterAddrs map[string]string) {
	for name, addr := range p.rmtClusters {
		// 移除失效的Dialer
		if clusterAddrs[name] != addr {
			p.rwMu.Lock()
			d := p.dialers[name]
			delete(p.dialers, name)
			p.rwMu.Unlock()
			if d != nil {
				d.Shutdown()
			}
		}
	}
	p.rmtClusters = clusterAddrs
	// 重新生成hash表
	hashs := make(map[uint32]string)
	for name := range clusterAddrs {
		hashs[utils.ClusterNameToHash(name)] = name
	}
	p.hashToNames = hashs
}

func (p *ClusterProxy) GetDialer(clusterName string) (*netframe.Dialer, error) {
	addr, ok := p.rmtClusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("no such cluster %s", clusterName)
	}
	p.rwMu.Lock()
	defer p.rwMu.Unlock()
	if p.dialers == nil {
		p.dialers = make(map[string]*netframe.Dialer)
	}
	if p.dialers[clusterName] == nil {
		d, err := netframe.NewDialer(addr, nil)
		if err != nil {
			return nil, err
		}
		err = d.Start()
		if err != nil {
			return nil, err
		}
		p.dialers[clusterName] = d
	}
	return p.dialers[clusterName], nil
}

func (p *ClusterProxy) Exit() {
	p.rwMu.Lock()
	defer p.rwMu.Unlock()
	for _, d := range p.dialers {
		go d.Shutdown()
	}
	p.dialers = nil
}

type GateReceiver struct {
	server     *Server
	packBuffer [PACK_BUFFER_SIZE]byte
}

func (r *GateReceiver) OnConnected(s netframe.Sender) error {
	return nil
}

func (r *GateReceiver) OnMessage(s netframe.Sender, b []byte) (int, error) {
	n, data := NetUnpack(b)
	if n == 0 { // 没解够长度
		return n, nil
	}
	if len(data) == 0 { // 几乎不可能发生
		return n, fmt.Errorf("data is nil")
	}
	msgType := MsgType(data[0])
	// 剔除msgType
	data = data[1:]
	if msgType == MSG_TYPE_CLUSTER_REQ {
		head := &ClusterReqHead{}
		pos, err := head.Unpack(data)
		if err != nil {
			return n, err
		}
		dstSvc := r.server.GetService(SVC_HANDLE(head.destination))
		if dstSvc != nil {
			dstSvc.pushClusterRequest(context.Background(), head, data[pos:])
		} else {
			return n, fmt.Errorf("%s not find dst svc %d", msgType, head.destination)
		}
	} else if msgType == MSG_TYPE_CLUSTER_RSP {
		head := &ClusterRspHead{}
		pos, err := head.Unpack(data)
		if err != nil {
			return n, err
		}
		dstSvc := r.server.GetService(SVC_HANDLE(head.destination))
		if dstSvc != nil {
			dstSvc.pushClusterResponse(context.Background(), head, data[pos:])
		} else {
			return n, fmt.Errorf("%s not find dst svc %d", msgType, head.destination)
		}
	}
	return n, nil
}

func (r *GateReceiver) OnClosed(s netframe.Sender) error {
	return nil
}

type Sidecar struct {
	server       *Server
	clusterName  string
	gateListener *netframe.Listener
	clusterProxy *ClusterProxy
}

func (sc *Sidecar) Init() error {
	sc.clusterName = sc.server.config.ClusterName
	// 从配置中读取clustername表
	sc.clusterProxy = &ClusterProxy{}
	sc.clusterProxy.Reload(sc.server.config.RemoteAddrs)
	// 绑定本地端口
	l, err := netframe.NewListener(sc.server.config.LocalAddr, func() netframe.Receiver {
		return &GateReceiver{server: sc.server}
	})
	if err != nil {
		return err
	}
	sc.gateListener = l
	go func() {
		err := l.Serve()
		if err != nil {
			log.Fatalf("gate listener serve err:%v", err)
		} else {
			log.Println("gate listener quit serve")
		}
	}()
	return nil
}

// 更新cluster节点信息
func (sc *Sidecar) Reload() {
	sc.clusterProxy.Reload(sc.server.config.RemoteAddrs)
}

func (sc *Sidecar) GetClusterName(handle SVC_HANDLE) (string, bool) {
	hashID := uint32(handle >> 32)
	cluster, ok := sc.clusterProxy.hashToNames[hashID]
	if !ok {
		return "unknown", false
	}
	return cluster, true
}

func (sc *Sidecar) Send(clusterName string, data []byte) error {
	d, err := sc.clusterProxy.GetDialer(clusterName)
	if err != nil {
		return err
	}
	return d.Send(data)
}

func (sc *Sidecar) Exit() {
	sc.clusterProxy.Exit()
	sc.gateListener.GracefulStop()
}
