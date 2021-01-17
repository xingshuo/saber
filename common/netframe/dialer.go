package netframe

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/xingshuo/saber/common/lib"
)

// Tcp Dial流程封装, 并提供断线重连的能力

type State int

func (s State) String() string {
	switch s {
	case Idle:
		return "IDLE"
	case Connecting:
		return "CONNECTING"
	case Ready:
		return "READY"
	case TransientFailure:
		return "TRANSIENT_FAILURE"
	case Shutdown:
		return "SHUTDOWN"
	default:
		return "Invalid-State"
	}
}

const (
	// Idle indicates the ClientConn is idle.
	Idle State = iota
	// Connecting indicates the ClientConn is connecting.
	Connecting
	// Ready indicates the ClientConn is ready for work.
	Ready
	// TransientFailure indicates the ClientConn has seen a failure but expects to recover.
	TransientFailure
	// Shutdown indicates the ClientConn has started shutting down.
	Shutdown
)

type Dialer struct {
	opts        dialOptions
	conn        *Conn
	address     string
	newReceiver func() Receiver
	wg          sync.WaitGroup
	quit        *lib.SyncEvent
	state       State
	notifys     map[chan error]bool
	rwMu        sync.RWMutex
}

func (d *Dialer) dial() error {
	s := d.getState()
	if s == Connecting {
		return fmt.Errorf("Already In Connecting")
	} else if s == Ready {
		return nil
	} else if s == Shutdown {
		return fmt.Errorf("Already Shutdown")
	}
	var (
		err     error
		rawConn net.Conn
	)
	d.setState(Connecting)
	//wait last connection close done.
	d.wg.Wait()
	if d.opts.dialTimeout > 0 {
		rawConn, err = net.DialTimeout("tcp", d.address, time.Duration(d.opts.dialTimeout)*time.Second)
	} else {
		rawConn, err = net.Dial("tcp", d.address)
	}
	if err != nil {
		d.setState(TransientFailure)
		d.doNotify(err)
		return err
	}
	//connect过程中状态发生变化, 例如Dialer关闭了
	if !d.isInState(Connecting) {
		err = fmt.Errorf("Connecting State Changed.")
		d.doNotify(err)
		return err
	}
	d.rwMu.Lock()
	d.conn = new(Conn)
	d.rwMu.Unlock()
	err = d.conn.Init(rawConn, d.newReceiver())
	if err != nil {
		d.setState(TransientFailure)
		d.doNotify(err)
		return err
	}
	d.setState(Ready)
	d.wg.Add(2)
	go func() {
		err := d.conn.loopRead()
		if err != nil {
			d.conn.Close()
			d.setState(TransientFailure)
			log.Printf("loop read err:%v", err)
		}
		d.wg.Done()
	}()
	go func() {
		err := d.conn.loopWrite()
		if err != nil {
			d.conn.Close()
			d.setState(TransientFailure)
			log.Printf("loop write err:%v", err)
		}
		d.wg.Done()
	}()
	d.doNotify(nil)
	return nil
}

func (d *Dialer) reConnect() error {
	for {
		select {
		case <-d.conn.Done():
			log.Printf("connection close")
		case <-d.quit.Done():
			log.Printf("quit dial")
			return nil
		}
		//---下面是重连逻辑---
		retryTimes := 0
	retry:
		for {
			select {
			case <-d.quit.Done():
				log.Printf("quit dial.")
				return nil
			default:
				retryTimes++
				if err := d.dial(); err != nil {
					log.Printf("reconnect %dth failed %v", retryTimes, err)
					time.Sleep(time.Duration(d.opts.retryInterval) * time.Second)
					continue
				}
				log.Printf("reconnect %dth succeed!", retryTimes)
				break retry
			}
		}
	}
}

func (d *Dialer) setState(s State) {
	d.rwMu.Lock()
	defer d.rwMu.Unlock()
	if d.state == Shutdown { //Dialer已彻底关闭
		return
	}
	d.state = s
}

func (d *Dialer) getState() State {
	d.rwMu.RLock()
	defer d.rwMu.RUnlock()
	return d.state
}

func (d *Dialer) isInState(s State) bool {
	d.rwMu.RLock()
	defer d.rwMu.RUnlock()
	return s == d.state
}

func (d *Dialer) getNotify() chan error {
	d.rwMu.Lock()
	defer d.rwMu.Unlock()
	c := make(chan error)
	d.notifys[c] = true
	return c
}

func (d *Dialer) delNotify(c chan error) {
	d.rwMu.Lock()
	defer d.rwMu.Unlock()
	delete(d.notifys, c)
}

func (d *Dialer) doNotify(err error) {
	d.rwMu.Lock()
	notifys := d.notifys
	d.notifys = make(map[chan error]bool)
	d.rwMu.Unlock()
	for c := range notifys {
		select {
		case c <- err:
		default:
		}
	}
}

//外部调用接口
func (d *Dialer) Start() error {
	if err := d.dial(); err != nil {
		return err
	}
	if d.opts.retryInterval > 0 {
		go d.reConnect()
	}
	return nil
}

func (d *Dialer) Shutdown() error {
	err := fmt.Errorf("repeat shutdown")
	if d.quit.Fire() {
		err = nil
		d.setState(Shutdown)
		d.rwMu.RLock()
		c := d.conn
		d.rwMu.RUnlock()
		if c != nil {
			c.Close()
		}
	}
	return err
}

func (d *Dialer) Send(b []byte) error {
	s := d.getState()
	if s == Ready {
		d.conn.Send(b)
		return nil
	} else if s == Connecting {
		c := d.getNotify()
		select {
		case <-time.After(3 * time.Second):
			d.delNotify(c)
			return fmt.Errorf("Connecting Timeout")
		case err := <-c:
			if err != nil {
				return err
			}
			d.conn.Send(b)
			return nil
		}
	} else if s == Shutdown {
		return fmt.Errorf("Already Shutdown.")
	} else {
		err := d.dial()
		if err != nil {
			return err
		}
		d.conn.Send(b)
		return nil
	}
}
