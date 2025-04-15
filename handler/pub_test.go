package handler

import (
	"context"
	"errors"
	"go-mqtt/config"
	mqttutil "go-mqtt/mqtt"
	"testing"
	"time"
)

func TestMQTTPublisher_Start(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Broker:   "tcp://localhost:1883",
			Interval: 1, // テストを速く終わらせるために短くする
			Topics: config.Topics{
				Topic1: "/test/topic1",
				Topic2: "/test/topic2",
				Topic3: "/test/topic3",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// パブリッシャーを作成
	publisher := NewPublisher(mockClient, cfg)

	// コンテキストを作成して、すぐにキャンセルできるようにする
	ctx, cancel := context.WithCancel(context.Background())

	// テスト終了時にキャンセルする
	defer cancel()

	// エラーチャネルを作成して、非同期でパブリッシャーを開始
	errCh := make(chan error)
	go func() {
		errCh <- publisher.Start(ctx)
	}()

	// パブリッシャーがメッセージを送信するのを待つ
	time.Sleep(1500 * time.Millisecond)

	// パブリッシャーが正しいトピックにパブリッシュしたか確認
	if mockClient.GetPublishCount() == 0 {
		t.Errorf("Expected at least one publish call, got 0")
	}

	if mockClient.GetLastPublishTopic() != cfg.Mqtt.Topics.Topic3 {
		t.Errorf("Expected publish to topic %s, got %s", cfg.Mqtt.Topics.Topic3, mockClient.GetLastPublishTopic())
	}

	// パブリッシャーを停止
	cancel()

	// 少し待ってからエラーをチェック
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for publisher to stop")
	}
}

func TestMQTTPublisher_PublishError(t *testing.T) {
	// テスト用の設定を準備
	cfg := &config.MqttConfig{
		Mqtt: config.Config{
			Broker:   "tcp://localhost:1883",
			Interval: 1,
			Topics: config.Topics{
				Topic3: "/test/topic3",
			},
		},
	}

	// モックMQTTクライアントを作成
	mockClient := mqttutil.NewMockClient()

	// パブリッシュエラーを設定
	expectedErr := errors.New("publish failed")
	mockClient.SetPublishError(expectedErr)

	// publishMessageメソッドをテスト
	publisher := &MQTTPublisher{
		client:   mockClient,
		cfg:      cfg,
		interval: time.Duration(cfg.Mqtt.Interval) * time.Second,
	}

	err := publisher.publishMessage()

	// エラーが正しく返されるか確認
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}
