package ip

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func GetPublicIP() (string, error) {
	// 尝试多个 IP 查询服务
	services := []string{
		"https://api.ipify.org?format=text",
		"https://ifconfig.me/ip",
		"https://ipinfo.io/ip",
	}

	for _, service := range services {
		resp, err := http.Get(service)
		if err == nil {
			defer resp.Body.Close()
			ip, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				return string(ip), nil
			}
		}
	}

	return "", fmt.Errorf("failed to get public IP from all services")
}

// *获取当前机器的内部 IPv4 地址
func InternalIP() string {
	inters, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, inter := range inters {
		if inter.Flags&net.FlagUp != 0 && !strings.HasPrefix(inter.Name, "lo") {
			addrs, err := inter.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	return ""
}
