package main

import (
	"context"
	"fmt"
	"go-mqtt/cache"
	"go-mqtt/config"
	"go-mqtt/handler"
	mqttutil "go-mqtt/mqtt"
	"log"
	"os"
	"os/signal"
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
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// キャッシュの初期化
	_ = cache.GetInstance()
	fmt.Println("Cache initialized")

	client := mqttutil.NewClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Sub1からのメッセージ受信を通知するチャネルを作成
	messageReceived := make(chan struct{}, 1)

	// Sub1とSub2を起動
	go handler.Sub1(ctx, client, cfg, messageReceived)
	go handler.Sub2(ctx, client, cfg)
	
	// Pubは、Sub1がメッセージを受信した後に開始するよう変更
	go handler.Pub(ctx, client, cfg, messageReceived)

	// シグナル処理（Ctrl+Cなど）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down...")
	cancel()
	client.Disconnect(250)
	fmt.Println("All tasks completed, exiting")
}
