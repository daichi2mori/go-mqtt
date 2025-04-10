package main

import (
	"fmt"
	mqttutil "go-mqtt/mqtt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// MQTT接続設定
const (
	brokerURL = "tcp://localhost:1883"
	clientID  = "mqtt-application"
	username  = "user"
	password  = "pass"
)

func main() {
	// 共有MQTTクライアントの取得
	client, err := mqttutil.GetClient(brokerURL, clientID, username, password)
	if err != nil {
		log.Fatalf("Failed to get MQTT client: %v", err)
	}

	var wg sync.WaitGroup

	// サブスクライバーを開始
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := startSubscriber1(client); err != nil {
			fmt.Printf("Subscriber1 error: %v\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := startSubscriber2(client); err != nil {
			fmt.Printf("Subscriber2 error: %v\n", err)
		}
	}()

	// 100ms毎にパブリッシュする処理を開始
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startPeriodicPublisher(client); err != nil {
			fmt.Printf("Periodic publisher error: %v\n", err)
		}
	}()

	// シグナル処理（Ctrl+Cなど）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down...")

	// MQTTクライアントを切断
	if client.IsConnected() {
		client.Disconnect(250)
		fmt.Println("MQTT client disconnected")
	}

	// すべてのゴルーチンが終了するのを待つ
	wg.Wait()
	fmt.Println("All tasks completed, exiting")
}
