package saber

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"unsafe"
)

//协议格式: 4字节包头长度 + 内容
const PkgHeadLen = 4

// 用户数据编解码器
type Codec interface {
	Marshal(msgType MsgType, method string, v interface{}) ([]byte, error)
	Unmarshal(msgType MsgType, method string, data []byte) (interface{}, error)
}

type JsonCodec struct {
}

func (c *JsonCodec) Marshal(msgType MsgType, method string, v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// reflect.TypeOf(v) : map[string]interface{}
func (c *JsonCodec) Unmarshal(msgType MsgType, method string, data []byte) (interface{}, error) {
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	return v, err
}

var CLUSTER_REQ_HEAD_LEN = int(new(ClusterReqHead).Size())
var CLUSTER_RSP_HEAD_LEN = int(new(ClusterRspHead).Size())

type ClusterBaseHead struct {
	source      uint64
	session     uint32
	destination uint64
	mdLen       uint8
	method      [METHOD_MAX_LEN]byte
}

func (h *ClusterBaseHead) Init(source SVC_HANDLE, session uint32, destination SVC_HANDLE, method string) error {
	if len(h.method) > METHOD_MAX_LEN {
		return RPC_METHOD_LEN_OVER_ERR
	}
	h.source = uint64(source)
	h.session = session
	h.destination = uint64(destination)
	h.mdLen = uint8(len(method))
	copy(h.method[:], method)
	return nil
}

func (h *ClusterBaseHead) Method() string {
	return string(h.method[:h.mdLen])
}

func (h *ClusterBaseHead) Pack(b []byte) (uintptr, error) {
	var pos uintptr
	// source
	binary.LittleEndian.PutUint64(b[pos:], h.source)
	pos = pos + unsafe.Sizeof(h.source)
	// session
	binary.LittleEndian.PutUint32(b[pos:], h.session)
	pos = pos + unsafe.Sizeof(h.session)
	// destination
	binary.LittleEndian.PutUint64(b[pos:], h.destination)
	pos = pos + unsafe.Sizeof(h.destination)
	// mdLen
	copy(b[pos:], []byte{h.mdLen})
	pos = pos + unsafe.Sizeof(h.mdLen)
	// method
	copy(b[pos:], h.method[:h.mdLen])
	pos = pos + uintptr(h.mdLen)

	return pos, nil
}

func (h *ClusterBaseHead) Unpack(b []byte) (uintptr, error) {
	var (
		pos     uintptr
		nextPos uintptr
	)
	// source
	h.source = binary.LittleEndian.Uint64(b[pos:])
	pos = pos + unsafe.Sizeof(h.source)
	// session
	h.session = binary.LittleEndian.Uint32(b[pos:])
	pos = pos + unsafe.Sizeof(h.session)
	// destination
	h.destination = binary.LittleEndian.Uint64(b[pos:])
	pos = pos + unsafe.Sizeof(h.destination)
	// mdLen
	h.mdLen = b[pos]
	pos = pos + unsafe.Sizeof(h.mdLen)
	// method
	nextPos = pos + uintptr(h.mdLen)
	copy(h.method[:], b[pos:nextPos])
	pos = nextPos

	return pos, nil
}

func (h *ClusterBaseHead) Size() uintptr {
	var size uintptr
	hType := reflect.TypeOf(h)
	for i := 0; i < hType.Elem().NumField(); i++ {
		field := hType.Elem().Field(i)
		size = size + field.Type.Size()
	}
	return size
}

type ClusterReqHead struct {
	ClusterBaseHead
}

func (h *ClusterReqHead) Pack(b []byte) (uintptr, error) {
	// 检查buffer够不够上限
	// FIXME: 这里需要做精确判断
	/*	if len(b) < CLUSTER_REQ_HEAD_LEN {
		return 0, PACK_BUFFER_SHORT_ERR
	}*/
	return h.ClusterBaseHead.Pack(b)
}

func (h *ClusterReqHead) Unpack(b []byte) (uintptr, error) {
	// 检查buffer够不够上限
	// FIXME: 这里需要做精确判断
	/*	if len(b) < CLUSTER_REQ_HEAD_LEN {
		return 0, PACK_BUFFER_SHORT_ERR
	}*/
	return h.ClusterBaseHead.Unpack(b)
}

type ClusterRspHead struct {
	ClusterBaseHead
	errCode uint32
	emLen   uint8
	errMsg  [ERR_MSG_MAX_LEN]byte
}

func (h *ClusterRspHead) Init(source SVC_HANDLE, session uint32, destination SVC_HANDLE, method string, errCode uint32, errMsg string) error {
	if len(errMsg) > ERR_MSG_MAX_LEN {
		return ERR_MSG_LEN_OVER
	}
	h.errCode = errCode
	h.emLen = uint8(len(errMsg))
	copy(h.errMsg[:], errMsg)
	return h.ClusterBaseHead.Init(source, session, destination, method)
}

func (h *ClusterRspHead) ErrMsg() string {
	return string(h.errMsg[:h.emLen])
}

func (h *ClusterRspHead) Pack(b []byte) (uintptr, error) {
	// 检查buffer够不够上限
	// FIXME: 这里需要做精确判断
	/*	if len(b) < CLUSTER_RSP_HEAD_LEN {
		return 0, PACK_BUFFER_SHORT_ERR
	}*/
	pos, err := h.ClusterBaseHead.Pack(b)
	if err != nil {
		return pos, err
	}
	binary.LittleEndian.PutUint32(b[pos:], h.errCode)
	pos = pos + unsafe.Sizeof(h.errCode)
	if h.errCode != ErrCode_OK {
		// emLen
		copy(b[pos:], []byte{h.emLen})
		pos = pos + unsafe.Sizeof(h.emLen)
		// errMsg
		copy(b[pos:], h.errMsg[:h.emLen])
		pos = pos + uintptr(h.emLen)
	}
	return pos, nil
}

func (h *ClusterRspHead) Unpack(b []byte) (uintptr, error) {
	// 检查buffer够不够上限
	// FIXME: 这里需要做精确判断
	/*	if len(b) < CLUSTER_RSP_HEAD_LEN {
		return 0, PACK_BUFFER_SHORT_ERR
	}*/
	pos, err := h.ClusterBaseHead.Unpack(b)
	if err != nil {
		return pos, err
	}
	h.errCode = binary.LittleEndian.Uint32(b[pos:])
	pos = pos + unsafe.Sizeof(h.errCode)
	if h.errCode != ErrCode_OK {
		// emLen
		h.emLen = b[pos]
		pos = pos + unsafe.Sizeof(h.emLen)
		// errMsg
		nextPos := pos + uintptr(h.emLen)
		copy(h.errMsg[:], b[pos:nextPos])
		pos = nextPos
	}
	return pos, nil
}

func (h *ClusterRspHead) Size() uintptr {
	size := unsafe.Sizeof(h.errCode) + unsafe.Sizeof(h.emLen) + unsafe.Sizeof(h.errMsg)
	return size + h.ClusterBaseHead.Size()
}

func NetPackRequest(buffer []byte, cc Codec, source SVC_HANDLE, session uint32, destination SVC_HANDLE, method string, req interface{}) ([]byte, error) {
	if PkgHeadLen+1 > len(buffer) {
		return nil, PACK_BUFFER_SHORT_ERR
	}
	buffer[PkgHeadLen] = uint8(MSG_TYPE_CLUSTER_REQ)
	head := &ClusterReqHead{}
	err := head.Init(source, session, destination, method)
	if err != nil {
		return nil, err
	}
	hsize, err := head.Pack(buffer[PkgHeadLen+1:])
	if err != nil {
		return nil, err
	}
	pos := PkgHeadLen + 1 + int(hsize)
	if pos > len(buffer) {
		return nil, PACK_BUFFER_SHORT_ERR
	}
	body, err := cc.Marshal(MSG_TYPE_CLUSTER_REQ, method, req)
	if err != nil {
		return nil, err
	}
	if pos+len(body) > len(buffer) {
		return nil, PACK_BUFFER_SHORT_ERR
	}
	bsize := copy(buffer[pos:], body)
	pos += bsize
	binary.LittleEndian.PutUint32(buffer, uint32(pos-PkgHeadLen))
	return buffer[:pos], nil
}

func NetPackResponse(buffer []byte, cc Codec, source SVC_HANDLE, session uint32, destination SVC_HANDLE, method string, rsp interface{}, rpcErr error) ([]byte, error) {
	if PkgHeadLen+1 > len(buffer) {
		return nil, PACK_BUFFER_SHORT_ERR
	}
	buffer[PkgHeadLen] = uint8(MSG_TYPE_CLUSTER_RSP)
	head := &ClusterRspHead{}
	errCode := ErrCode_OK
	errMsg := ""
	if rpcErr != nil {
		errCode = ErrCode_Usr
		errMsg = rpcErr.Error()
	}
	err := head.Init(source, session, destination, method, errCode, errMsg)
	if err != nil {
		return nil, err
	}
	hsize, err := head.Pack(buffer[PkgHeadLen+1:])
	if err != nil {
		return nil, err
	}
	pos := PkgHeadLen + 1 + int(hsize)
	if pos > len(buffer) {
		return nil, PACK_BUFFER_SHORT_ERR
	}
	// 未出现错误才Marshal Rsp
	if rpcErr == nil {
		body, err := cc.Marshal(MSG_TYPE_CLUSTER_RSP, method, rsp)
		if err != nil {
			return nil, err
		}
		if pos+len(body) > len(buffer) {
			return nil, PACK_BUFFER_SHORT_ERR
		}
		bsize := copy(buffer[pos:], body)
		pos += bsize
	}
	binary.LittleEndian.PutUint32(buffer, uint32(pos-PkgHeadLen))
	return buffer[:pos], nil
}

func NetUnpack(b []byte) (int, []byte) { //返回(消耗字节数,实际内容)
	if len(b) < PkgHeadLen { //不够包头长度
		return 0, nil
	}
	bodyLen := int(binary.LittleEndian.Uint32(b))
	if len(b) < PkgHeadLen+bodyLen { //不够body长度
		return 0, nil
	}
	msgLen := PkgHeadLen + bodyLen
	return msgLen, b[PkgHeadLen:msgLen]
}
