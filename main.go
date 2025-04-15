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
	"time"
)

func main() {
	// コンテキストの作成（タイムアウトも設定可能）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 設定の読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// キャッシュの初期化
	_ = cache.GetInstance()
	fmt.Println("Cache initialized")

	// MQTTクライアントの作成と接続
	client := mqttutil.NewClient(cfg)

	// Sub1からのメッセージ受信を通知するチャネルを作成
	messageReceived := make(chan struct{}, 1)

	// サブスクライバーとパブリッシャーのインスタンス作成
	sub1 := handler.NewSubscriber1(client, cfg, messageReceived)
	sub2 := handler.NewSubscriber2(client, cfg)
	pub := handler.NewPublisher(client, cfg, messageReceived)

	// エラーチャネル（各ゴルーチンからのエラーを受け取る）
	errChan := make(chan error, 3)

	// 各ハンドラーを並行起動
	go func() {
		errChan <- sub1.Start(ctx)
	}()

	go func() {
		errChan <- sub2.Start(ctx)
	}()

	go func() {
		errChan <- pub.Start(ctx)
	}()

	// シグナル処理（Ctrl+Cなど）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// シグナルかエラーを待機
	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v\n", sig)
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			fmt.Printf("Component error: %v\n", err)
		}
	}

	// グレースフルシャットダウン
	fmt.Println("Shutting down...")
	cancel()

	// 短いタイムアウトを設定して終了を待機
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// MQTTクライアントの切断
	client.Disconnect(250)

	// シャットダウンタイムアウトまで待機（すべてのゴルーチンに終了の機会を与える）
	<-shutdownCtx.Done()
	fmt.Println("All tasks completed, exiting")
}
