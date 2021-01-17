package netframe

import (
	"fmt"
	"sync"

	"github.com/xingshuo/saber/common/lib"
)

func NewDialer(address string, newReceiver func() Receiver, opts ...DialOption) (*Dialer, error) {
	// Dialer只处理发包时, 设置默认收包处理器
	if newReceiver == nil {
		newReceiver = func() Receiver {
			return &DefaultReceiver{}
		}
	}
	d := &Dialer{
		opts:        defaultDialOptions(),
		address:     address,
		quit:        lib.NewSyncEvent(),
		newReceiver: newReceiver,
		state:       Idle,
		notifys:     make(map[chan error]bool),
	}
	//处理参数
	for _, opt := range opts {
		opt.apply(&d.opts)
	}
	return d, nil
}

func NewListener(address string, newReceiver func() Receiver) (*Listener, error) {
	// Listener一定会处理收包事件, 必须设置收包处理器
	if newReceiver == nil {
		return nil, fmt.Errorf("newReceiver func nil")
	}
	l := &Listener{
		conns:       make(map[*Conn]bool),
		address:     address,
		quit:        lib.NewSyncEvent(),
		done:        lib.NewSyncEvent(),
		newReceiver: newReceiver,
	}
	l.cv = sync.NewCond(&l.mu)
	return l, nil
}
