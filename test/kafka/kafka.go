package main

import (
	"fmt"
	"log"

	"github.com/Shopify/sarama"
)

func main() {
	// Kafka broker 地址
	brokers := []string{"47.115.200.76:9092"}

	// 创建一个新的 Kafka 生产者配置
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // 等待所有副本确认
	config.Producer.Retry.Max = 5                   // 重试次数
	config.Producer.Return.Successes = true         // 成功发送的消息将被返回

	// 创建一个新的 Kafka 生产者
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalln("Failed to start Kafka producer:", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalln("Failed to close Kafka producer:", err)
		}
	}()

	// 定义要发送的消息
	topic := "test-topic"
	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder("Hello, Kafka!"),
	}

	// 发送消息
	partition, offset, err := producer.SendMessage(message)
	if err != nil {
		log.Fatalln("Failed to send message:", err)
	}

	// 打印消息发送成功的信息
	fmt.Printf("Message sent successfully! Partition: %d, Offset: %d\n", partition, offset)
}
