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

func TestMQTTSubscriber2_Start(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Broker:   "tcp://localhost:1883",
			Interval: 1,
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := NewSubscriber2(mockClient, cfg)

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

func TestMQTTSubscriber2_SubscribeError(t *testing.T) {
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
	subscriber := &MQTTSubscriber2{
		client: mockClient,
		cfg:    cfg,
	}

	// エラーが正しく返されるか確認
	err := subscriber.subscribe()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMQTTSubscriber2_MessageHandler_Type1(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := &MQTTSubscriber2{
		client: mockClient,
		cfg:    cfg,
	}

	// メッセージハンドラーを作成
	handler := subscriber.createMessageHandler()

	// Type 1のテストメッセージを作成
	testProps := Sub2Props{
		ID:   "test-id",
		Name: "test-name",
		Type: 1,
	}
	payload, _ := json.Marshal(testProps)

	// メッセージ受信をシミュレート - 修正したMockMessageの使い方
	message := &mqttutil.MockMessage{
		Topic1:   cfg.Mqtt.Topics.Topic1,
		Payload1: payload,
	}
	handler(nil, message)

	// Type 1のメッセージはTopic1にのみパブリッシュされるはず
	if mockClient.GetPublishCount() != 1 {
		t.Errorf("Expected 1 publish call for Type 1, got %d", mockClient.GetPublishCount())
	}

	if mockClient.GetLastPublishTopic() != cfg.Mqtt.Topics.Topic1 {
		t.Errorf("Expected publish to topic %s, got %s", cfg.Mqtt.Topics.Topic1, mockClient.GetLastPublishTopic())
	}
}

func TestMQTTSubscriber2_MessageHandler_Type2(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := &MQTTSubscriber2{
		client: mockClient,
		cfg:    cfg,
	}

	// メッセージハンドラーを作成
	handler := subscriber.createMessageHandler()

	// Type 2のテストメッセージを作成
	testProps := Sub2Props{
		ID:   "test-id",
		Name: "test-name",
		Type: 2,
	}
	payload, _ := json.Marshal(testProps)

	// メッセージ受信をシミュレート - 修正したMockMessageの使い方
	message := &mqttutil.MockMessage{
		Topic1:   cfg.Mqtt.Topics.Topic1,
		Payload1: payload,
	}
	handler(nil, message)

	// Type 2のメッセージはTopic1とTopic2の両方にパブリッシュされるはず
	if mockClient.GetPublishCount() != 2 {
		t.Errorf("Expected 2 publish calls for Type 2, got %d", mockClient.GetPublishCount())
	}
}

func TestMQTTSubscriber2_MessageHandler_InvalidJSON(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := &MQTTSubscriber2{
		client: mockClient,
		cfg:    cfg,
	}

	// メッセージハンドラーを作成
	handler := subscriber.createMessageHandler()

	// 無効なJSONデータでメッセージ受信をシミュレート
	invalidJSON := []byte("{invalid json}")
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

func TestMQTTSubscriber2_MessageHandler_UnknownType(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// サブスクライバーを作成
	subscriber := &MQTTSubscriber2{
		client: mockClient,
		cfg:    cfg,
	}

	// メッセージハンドラーを作成
	handler := subscriber.createMessageHandler()

	// 未知のTypeを持つテストメッセージを作成
	testProps := Sub2Props{
		ID:   "test-id",
		Name: "test-name",
		Type: 999, // 未知のタイプ
	}
	payload, _ := json.Marshal(testProps)

	// メッセージ受信をシミュレート
	message := &mqttutil.MockMessage{
		Topic1:   cfg.Mqtt.Topics.Topic1,
		Payload1: payload,
	}
	handler(nil, message)

	// 未知のType値のため、メッセージは処理されず、パブリッシュは呼ばれないはず
	if mockClient.GetPublishCount() != 0 {
		t.Errorf("Expected 0 publish calls for unknown Type, got %d", mockClient.GetPublishCount())
	}
}
