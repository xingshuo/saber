package utils

import (
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
)

var defaultIP string = "0.0.0.0"

func GetInternalIP() string { //本机内网ip,取第一个
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return defaultIP
	}
	for _, address := range addrs {
		ipnet, ok := address.(*net.IPNet)
		if !ok {
			continue
		}
		if ipnet.IP.IsLoopback() || ipnet.IP.IsLinkLocalMulticast() || ipnet.IP.IsLinkLocalUnicast() {
			continue
		}
		if ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return defaultIP
}

func GetExternalIP() string { //本机外网ip
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return defaultIP
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return defaultIP
	}
	return string(content)
}

func INetAddr(ipaddr string) uint32 {
	var (
		ip                 = strings.Split(ipaddr, ".")
		ip1, ip2, ip3, ip4 uint64
	)
	ip1, _ = strconv.ParseUint(ip[0], 10, 8)
	ip2, _ = strconv.ParseUint(ip[1], 10, 8)
	ip3, _ = strconv.ParseUint(ip[2], 10, 8)
	ip4, _ = strconv.ParseUint(ip[3], 10, 8)
	return uint32(ip4)<<24 + uint32(ip3)<<16 + uint32(ip2)<<8 + uint32(ip1)
}

var ipCache = ""

func GetIP() string { //优先取外网ip, 取不到再取内网ip
	if ipCache != "" {
		return ipCache
	}
	ip := GetExternalIP() //http请求查询, 会阻塞. 只查一次, 本地缓存
	if ip != defaultIP {
		ipCache = ip
		return ip
	}
	ipCache = GetInternalIP()
	return ipCache
}
