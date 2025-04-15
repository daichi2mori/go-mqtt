package handler

import (
	"context"
	"encoding/json"
	"errors"
	"go-mqtt/config"
	mqttutil "go-mqtt/mqtt"
	"testing"
	"time"
)

func TestMQTTSubscriber1_Start(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Broker:   "tcp://localhost:1883",
			Interval: 1,
			Topics: config.Topics{
				Topic1: "/test/topic1",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := NewSubscriber1(mockClient, cfg)

	// コンテキストを作成して、すぐにキャンセルできるようにする
	ctx, cancel := context.WithCancel(context.Background())

	// エラーチャネルを作成して、非同期でサブスクライバーを開始
	errCh := make(chan error)
	go func() {
		errCh <- subscriber.Start(ctx)
	}()

	// サブスクライバーがトピックをサブスクライブするのを待つ
	time.Sleep(500 * time.Millisecond)

	// サブスクライバーが正しいトピックをサブスクライブしたか確認
	if mockClient.GetSubscribeCount() != 1 {
		t.Errorf("Expected 1 subscribe call, got %d", mockClient.GetSubscribeCount())
	}

	if mockClient.GetLastSubscribeTopic() != cfg.Mqtt.Topics.Topic1 {
		t.Errorf("Expected subscribe to topic %s, got %s", cfg.Mqtt.Topics.Topic1, mockClient.GetLastSubscribeTopic())
	}

	// テストメッセージを作成
	testProps := Sub1Props{
		ID:   "test-id",
		Name: "test-name",
	}
	payload, _ := json.Marshal(testProps)

	// メッセージ受信をシミュレート
	mockClient.SimulateMessage(cfg.Mqtt.Topics.Topic1, payload)

	// 少し待ってから結果を確認
	time.Sleep(100 * time.Millisecond)

	// ハンドラーがメッセージを処理して再パブリッシュしたか確認
	if mockClient.GetPublishCount() != 1 {
		t.Errorf("Expected 1 publish call, got %d", mockClient.GetPublishCount())
	}

	if mockClient.GetLastPublishTopic() != cfg.Mqtt.Topics.Topic1 {
		t.Errorf("Expected publish to topic %s, got %s", cfg.Mqtt.Topics.Topic1, mockClient.GetLastPublishTopic())
	}

	// サブスクライバーを停止
	cancel()

	// 少し待ってからエラーをチェック
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for subscriber to stop")
	}
}

func TestMQTTSubscriber1_SubscribeError(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Topics: config.Topics{
				Topic1: "/test/topic1",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライブエラーを設定
	expectedErr := errors.New("subscribe failed")
	mockClient.SetSubscribeError(expectedErr)

	// サブスクライバーを作成
	subscriber := &MQTTSubscriber1{
		client: mockClient,
		cfg:    cfg,
	}

	// エラーが正しく返されるか確認
	err := subscriber.subscribe()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMQTTSubscriber1_MessageHandler_InvalidJSON(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Topics: config.Topics{
				Topic1: "/test/topic1",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := &MQTTSubscriber1{
		client: mockClient,
		cfg:    cfg,
	}

	// メッセージハンドラーを作成
	handler := subscriber.createMessageHandler()

	// 無効なJSONデータでメッセージ受信をシミュレート
	invalidJSON := []byte("{invalid json}")

	// MockMessageを正しく初期化
	message := &mqttutil.MockMessage{
		Topic1:   cfg.Mqtt.Topics.Topic1,
		Payload1: invalidJSON,
	}

	// ハンドラーを呼び出す
	handler(nil, message)

	// JSON解析エラーのため、メッセージは処理されず、パブリッシュは呼ばれないはず
	if mockClient.GetPublishCount() != 0 {
		t.Errorf("Expected 0 publish calls for invalid JSON, got %d", mockClient.GetPublishCount())
	}
}
