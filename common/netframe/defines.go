package netframe

const (
	MIN_CONN_READ_BUFFER  = 128
	MIN_CONN_WRITE_BUFFER = 128
)

//Conn 发包能力的的interface抽象
type Sender interface {
	Send(b []byte)
	PeerAddr() string //获取连接对端地址
}

//流事件接收器
type Receiver interface {
	//连接建立后
	OnConnected(s Sender) error

	//接收流消息,返回已经处理的n个字节流和异常信息,发生异常会关闭连接
	//当返回n > 0,Conn会主动Pop掉n字节的缓存,接口内部无需处理
	OnMessage(s Sender, b []byte) (n int, err error)

	//连接断开前
	OnClosed(s Sender) error
}

type DefaultReceiver struct {
}

func (r *DefaultReceiver) OnConnected(s Sender) error {
	return nil
}

func (r *DefaultReceiver) OnMessage(s Sender, b []byte) (n int, err error) {
	return len(b), nil
}

func (r *DefaultReceiver) OnClosed(s Sender) error {
	return nil
}
