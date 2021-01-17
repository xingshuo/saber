package netframe

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/xingshuo/saber/common/lib"
)

// Tcp连接之上的封装层,主要处理网络流读写缓存相关

type Conn struct {
	rawConn         net.Conn
	rbuff           *bytes.Buffer
	wbuff           *bytes.Buffer
	receiver        Receiver
	mu              sync.Mutex
	waiting         chan struct{}
	close           *lib.SyncEvent
	consumerWaiting bool
}

func (c *Conn) Init(conn net.Conn, r Receiver) error {
	c.rawConn = conn
	c.rbuff = bytes.NewBuffer(make([]byte, MIN_CONN_READ_BUFFER))
	c.rbuff.Reset()
	c.wbuff = bytes.NewBuffer(make([]byte, MIN_CONN_WRITE_BUFFER))
	c.wbuff.Reset()
	c.receiver = r
	c.waiting = make(chan struct{}, 1)
	c.close = lib.NewSyncEvent()
	c.consumerWaiting = false
	err := r.OnConnected(c)
	return err
}

func (c *Conn) loopRead() error {
	rsize := MIN_CONN_READ_BUFFER
	rbuf := make([]byte, rsize)
	for {
		n, err := c.rawConn.Read(rbuf) //将网络流写进rbuf
		if err != nil {
			return err
		}
		if n <= 0 {
			return fmt.Errorf("EOF")
		}
		c.rbuff.Write(rbuf[:n]) //将rbuf数据写入tc.rbuff
		if n == rsize {
			rsize = 2 * rsize
			rbuf = make([]byte, rsize)
		}
		if c.rbuff.Len() <= 0 {
			continue
		}
		for {
			rn, err := c.receiver.OnMessage(c, c.rbuff.Bytes())
			if err != nil { //业务层消息异常不应该引起网络连接断开
				log.Printf("tcp conn %p on message err:%v\n", c, err)
			}
			if rn > 0 {
				c.rbuff.Next(rn)
			} else {
				break
			}
		}

	}
}

func (c *Conn) loopWrite() error {
	var wn int
	var err error
	wsize := MIN_CONN_WRITE_BUFFER
	wbuf := make([]byte, wsize)
	for {
		c.mu.Lock()
		len := c.wbuff.Len()
		if len <= 0 {
			c.mu.Unlock()
			goto waitdata
		}
		if len > wsize {
			wsize = len
			wbuf = make([]byte, wsize)
		}
		wn, err = c.wbuff.Read(wbuf) //将数据从tc.wbuff 写入wbuf
		if err != nil {
			c.mu.Unlock()
			log.Printf("tcp conn %p wbuff.Read err: %v", c, err)
			return err
		}
		c.wbuff.Next(wn)
		c.mu.Unlock()
		if wn > 0 {
			_, err := c.rawConn.Write(wbuf[:wn]) //tcp conn once time drain off
			if err != nil {
				log.Printf("tcp conn %p Write err: %v", c, err)
				return err
			}
			continue
		}
	waitdata:
		c.mu.Lock()
		c.consumerWaiting = true
		c.mu.Unlock()
		select {
		case <-c.waiting:
		case <-c.close.Done(): //连接断开通知
			log.Print("chan closed.\n")
			return nil
		}
	}
}

func (c *Conn) Send(b []byte) {
	var wakeUp bool
	c.mu.Lock()
	if c.consumerWaiting {
		c.consumerWaiting = false
		wakeUp = true
	}
	c.wbuff.Write(b)
	c.mu.Unlock()
	if wakeUp {
		select {
		case c.waiting <- struct{}{}:
		default:
		}
	}
}

func (c *Conn) Close() error { //主动关闭调用
	if c.close.Fire() {
		c.receiver.OnClosed(c)
		c.rawConn.Close()
		return nil
	}
	return fmt.Errorf("repeat close")
}

func (c *Conn) Done() <-chan struct{} {
	return c.close.Done()
}

func (c *Conn) PeerAddr() string {
	return c.rawConn.RemoteAddr().String()
}
