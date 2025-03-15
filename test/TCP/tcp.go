package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	// 连接到本地的 8080 端口
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("无法连接到服务器: %v", err)
	}
	defer conn.Close()

	log.Println("已连接到服务器")

	// 创建一个带缓冲的读取器，用于读取用户输入
	reader := bufio.NewReader(os.Stdin)

	for {
		// 读取用户输入
		fmt.Print("请输入要发送的消息: ")
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("读取输入失败: %v", err)
			continue
		}

		// 发送消息到服务器
		_, err = conn.Write([]byte(message))
		if err != nil {
			log.Printf("发送消息失败: %v", err)
			break
		}

		// 读取服务器的响应
		response, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Printf("读取服务器响应失败: %v", err)
			break
		}

		// 打印服务器的响应
		fmt.Printf("服务器响应: %s", response)
	}
}
