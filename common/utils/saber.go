package utils

import (
	"fmt"
	"hash/fnv"
)

//handleID生成规则:(ip << 32)|Hash(service_name|serviceID)
func MakeServiceHandle(serviceName string, serviceID uint32) uint64 {
	low32 := fmt.Sprintf("%s|%d", serviceName, serviceID)
	h := fnv.New32a()
	h.Write([]byte(low32))
	hashID := h.Sum32()
	handleID := uint64(INetAddr(GetIP()))
	handleID = (handleID << 32) | uint64(hashID)
	return handleID
}
