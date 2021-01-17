package utils

import (
	"fmt"
	"hash/fnv"
)

//handleID生成规则:Hash(clusterName)|Hash(service_name/serviceID)
func MakeServiceHandle(clusterName, serviceName string, serviceID uint32) uint64 {
	h := fnv.New32a()
	low32 := fmt.Sprintf("%s/%d", serviceName, serviceID)
	h.Write([]byte(low32))
	lowHash := h.Sum32()

	h = fnv.New32a()
	h.Write([]byte(clusterName))
	highHash := h.Sum32()

	handleID := uint64(highHash)
	handleID = (handleID << 32) | uint64(lowHash)
	return handleID
}

func ClusterNameToHash(clusterName string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(clusterName))
	return h.Sum32()
}
