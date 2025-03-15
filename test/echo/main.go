package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func handleConnection(conn net.Conn) {
	defer conn.Close() // 确保连接关闭

	// 创建一个带缓冲的读取器
	reader := bufio.NewReader(conn)

	for {
		// 读取客户端发送的数据
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Client disconnected: %v", err)
			return
		}

		// 打印接收到的消息
		fmt.Printf("Received: %s", message)

		// 将消息原样返回给客户端
		_, err = conn.Write([]byte(message))
		if err != nil {
			log.Printf("Failed to send data: %v", err)
			return
		}
	}
}

func main() {
	// 监听 TCP 端口
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Println("TCP Echo Server is running on port 8080...")

	for {
		// 等待客户端连接
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// 启动一个 goroutine 处理连接
		go handleConnection(conn)
	}
}
