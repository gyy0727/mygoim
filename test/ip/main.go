package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func getPublicIP() (string, error) {
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


func main() {
	ip, err := getPublicIP()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(ip)
}
