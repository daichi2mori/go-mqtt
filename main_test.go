package main

import (
	"context"
	"go-mqtt/cache"
	"go-mqtt/config"
	"go-mqtt/handler"
	"go-mqtt/mqtt"
	"testing"
	"time"
)

// モック構造体
type MockSubscriber1 struct {
	started bool
	err     error
}

func (m *MockSubscriber1) Start(ctx context.Context) error {
	m.started = true
	<-ctx.Done()
	return m.err
}

type MockSubscriber2 struct {
	started bool
	err     error
}

func (m *MockSubscriber2) Start(ctx context.Context) error {
	m.started = true
	<-ctx.Done()
	return m.err
}

type MockPublisher struct {
	started bool
	err     error
}

func (m *MockPublisher) Start(ctx context.Context) error {
	m.started = true
	<-ctx.Done()
	return m.err
}

// モックMQTTClient作成ユーティリティ
func createMockMQTTClient() *mqtt.MockClient {
	client := mqtt.NewMockClient()

	// Disconnect実装
	client.DisconnectFunc = func(quiesce uint) {
		// 何もしない
	}

	return client
}

func TestMain(t *testing.T) {
	// メインの初期化と実行フローのテスト
	t.Run("アプリケーション正常起動と終了", func(t *testing.T) {
		// キャッシュの初期化
		_ = cache.GetInstance()

		// テスト用の設定
		cfg := &config.MqttConfig{
			Mqtt: config.MqttSettings{
				Interval: 1,
				Topics: config.TopicConfig{
					Topic1: "test/topic1",
					Topic2: "test/topic2",
					Topic3: "test/topic3",
				},
			},
		}

		// モックMQTTクライアント
		mockClient := createMockMQTTClient()

		// モックスクライバーとパブリッシャー
		mockSub1 := &MockSubscriber1{}
		mockSub2 := &MockSubscriber2{}
		mockPub := &MockPublisher{}

		// コンテキスト
		ctx, cancel := context.WithCancel(context.Background())

		// エラーチャネル
		errChan := make(chan error, 3)

		// 各コンポーネントの起動
		go func() {
			errChan <- mockSub1.Start(ctx)
		}()

		go func() {
			errChan <- mockSub2.Start(ctx)
		}()

		go func() {
			errChan <- mockPub.Start(ctx)
		}()

		// 一定時間待機してキャンセル
		time.Sleep(200 * time.Millisecond)
		cancel()

		// シャットダウンタイムアウト
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer shutdownCancel()

		// クライアント切断
		mockClient.Disconnect(250)

		// シャットダウン待機
		<-shutdownCtx.Done()

		// 各コンポーネントが起動されたことを確認
		if !mockSub1.started {
			t.Error("Sub1が起動されなかった")
		}

		if !mockSub2.started {
			t.Error("Sub2が起動されなかった")
		}

		if !mockPub.started {
			t.Error("Pubが起動されなかった")
		}
	})
}

// インテグレーションテスト：コンポーネント間の連携をテスト
func TestComponentsIntegration(t *testing.T) {
	// キャッシュの初期化
	cache := cache.GetInstance()
	cache.Clear()

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Interval: 1,
			Topics: config.TopicConfig{
				Topic1: "test/topic1",
				Topic2: "test/topic2",
				Topic3: "test/topic3",
			},
		},
	}

	// モックMQTTクライアント
	mockClient := createMockMQTTClient()

	// パブリッシュとサブスクライブの監視
	publishedMessages := make(map[string][]string)
	mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
		publishedMessages[topic] = append(publishedMessages[topic], payload)
		return nil
	}

	var sub1Handler, sub2Handler mqtt.MessageHandler
	mockClient.SubscribeFunc = func(topic string, qos byte, callback mqtt.MessageHandler) error {
		if topic == cfg.Mqtt.Topics.Topic1 {
			if sub1Handler == nil {
				sub1Handler = callback
			} else {
				sub2Handler = callback
			}
		}
		return nil
	}

	// メッセージ受信通知チャネル
	messageReceived := make(chan struct{}, 1)

	// コンテキスト
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// コンポーネントの作成
	sub1 := handler.NewSubscriber1(mockClient, cfg, messageReceived)
	sub2 := handler.NewSubscriber2(mockClient, cfg)
	pub := handler.NewPublisher(mockClient, cfg, messageReceived)

	// コンポーネントの起動
	go sub1.Start(ctx)
	go sub2.Start(ctx)
	go pub.Start(ctx)

	// ハンドラーが設定されるまで待機
	time.Sleep(200 * time.Millisecond)

	// メッセージフロー：Sub1にメッセージ -> Sub1が通知 -> Pubが開始 -> Pubがメッセージを送信
	t.Run("メッセージフローテスト", func(t *testing.T) {
		if sub1Handler == nil {
			t.Fatal("Sub1ハンドラーが設定されていない")
		}

		// Sub1にメッセージを送信
		testData := `{"id":"1","name":"test-message"}`
		sub1Handler(nil, &mockMessage{
			topic:   cfg.Mqtt.Topics.Topic1,
			payload: []byte(testData),
		})

		// Pubがメッセージを送信するまで待機
		time.Sleep(500 * time.Millisecond)

		// キャッシュを確認
		sub1Data, exists := cache.Get("sub1:1")
		if !exists {
			t.Error("Sub1データがキャッシュに保存されていない")
		}

		pubData, exists := cache.Get("pub:1")
		if !exists {
			t.Error("Pubデータがキャッシュに保存されていない")
		}

		// Pubからのメッセージを確認
		topic3Messages, exists := publishedMessages[cfg.Mqtt.Topics.Topic3]
		if !exists || len(topic3Messages) == 0 {
			t.Error("Pubがトピック3にメッセージを送信していない")
		}

		// Sub2にメッセージを送信（Type 2）
		testData2 := `{"id":"1","name":"test-message-type2","type":2}`
		if sub2Handler == nil {
			t.Fatal("Sub2ハンドラーが設定されていない")
		}

		sub2Handler(nil, &mockMessage{
			topic:   cfg.Mqtt.Topics.Topic1,
			payload: []byte(testData2),
		})

		// 処理待機
		time.Sleep(200 * time.Millisecond)

		// Sub2データをキャッシュで確認
		sub2Data, exists := cache.Get("sub2:1:2")
		if !exists {
			t.Error("Sub2データがキャッシュに保存されていない")
		}

		// Sub2からのメッセージを確認（トピック1とトピック2の両方）
		topic1Messages, exists := publishedMessages[cfg.Mqtt.Topics.Topic1]
		if !exists || len(topic1Messages) < 2 {
			t.Error("Sub2がトピック1にメッセージを送信していない")
		}

		topic2Messages, exists := publishedMessages[cfg.Mqtt.Topics.Topic2]
		if !exists || len(topic2Messages) == 0 {
			t.Error("Sub2がトピック2にメッセージを送信していない")
		}
	})
}
