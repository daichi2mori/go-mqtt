package main

import (
	"context"
	"encoding/json"
	"go-mqtt/config"
	"go-mqtt/handler"
	mqttutil "go-mqtt/mqtt"
	"os"
	"testing"
	"time"
)

// テスト用のメイン処理をシミュレート
func TestMainFlow(t *testing.T) {
	// テスト用の設定を作成
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Broker:   "tcp://localhost:1883",
			Interval: 1,
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
				Topic3: "/test/topic3",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// キャンセル可能なコンテキストを作成
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Publisher、Subscriber1、Subscriber2を別々のゴルーチンで開始
	errCh := make(chan error, 3)

	go func() {
		publisher := handler.NewPublisher(mockClient, cfg)
		errCh <- publisher.Start(ctx)
	}()

	go func() {
		subscriber1 := handler.NewSubscriber1(mockClient, cfg)
		errCh <- subscriber1.Start(ctx)
	}()

	go func() {
		subscriber2 := handler.NewSubscriber2(mockClient, cfg)
		errCh <- subscriber2.Start(ctx)
	}()

	// 少し待って、すべてのコンポーネントが開始されるのを確認
	time.Sleep(1 * time.Second)

	// クライアントが接続され、サブスクライブが行われたか確認
	if !mockClient.IsConnected() {
		t.Error("Expected client to be connected")
	}

	// サブスクライブが2回呼ばれたか確認（Subscriber1とSubscriber2）
	if mockClient.GetSubscribeCount() != 2 {
		t.Errorf("Expected 2 subscribe calls, got %d", mockClient.GetSubscribeCount())
	}

	// パブリッシュが少なくとも1回呼ばれたか確認
	if mockClient.GetPublishCount() < 1 {
		t.Errorf("Expected at least 1 publish call, got %d", mockClient.GetPublishCount())
	}

	// シミュレートされたメッセージを送信
	testMessage := map[string]interface{}{
		"id":   "test-id",
		"name": "test-name",
		"type": 2,
	}
	messageJSON, _ := json.Marshal(testMessage)
	mockClient.SimulateMessage(cfg.Mqtt.Topics.Topic1, messageJSON)

	// 少し待って、メッセージ処理が完了するのを確認
	time.Sleep(500 * time.Millisecond)

	// すべてのコンポーネントをキャンセル
	cancel()

	// 終了を確認
	timeout := time.After(2 * time.Second)
	completed := 0

	for completed < 3 {
		select {
		case <-errCh:
			completed++
		case <-timeout:
			t.Fatalf("Timeout waiting for components to complete, only %d of 3 completed", completed)
			return
		}
	}
}

// テスト用の設定ファイルを作成するヘルパー関数
func createTestConfig(t *testing.T) (string, func()) {
	// 一時設定ファイルを作成
	content := `mqtt:
  broker: "tcp://127.0.0.1:1883"
  interval: 1
  topics:
    topic1: "/test/topic1"
    topic2: "/test/topic2"
    topic3: "/test/topic3"
`
	tmpfile, err := os.CreateTemp("", "config.*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// クリーンアップ関数を返す
	cleanup := func() {
		os.Remove(tmpfile.Name())
	}

	return tmpfile.Name(), cleanup
}

// 統合テスト用のユーティリティ関数
// 注意: この関数は実際のMQTTブローカーへの接続が必要です
func _TestIntegration(t *testing.T) {
	// CI環境など、統合テストを実行しない場合はスキップ
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	// テスト用の設定ファイルを作成
	configFile, cleanup := createTestConfig(t)
	defer cleanup()

	// テスト用の設定ファイルを使用するように環境を設定
	// バックアップと復元処理
	configBackup := "./config.yml.bak"
	configPath := "./config.yml"

	// 既存ファイルをバックアップ (存在する場合)
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, configBackup); err != nil {
			t.Fatalf("Failed to backup config file: %v", err)
		}
		defer os.Rename(configBackup, configPath) // テスト後に復元
	}

	// テスト用の設定ファイルをシンボリックリンク
	if err := os.Symlink(configFile, configPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	defer os.Remove(configPath)

	// コンテキストを作成
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 実際のMQTTクライアントを作成
	client := mqttutil.NewClient(cfg)
	client.Connect()
	defer client.Disconnect(250)

	// 実際のタスクを実行
	go handler.Pub(ctx, client, cfg)
	go handler.Sub1(ctx, client, cfg)
	go handler.Sub2(ctx, client, cfg)

	// 一定時間実行を続ける
	time.Sleep(5 * time.Second)

	// テストが成功したことを示す
	// （ここではエラーが発生しなければ成功とみなす）
}
