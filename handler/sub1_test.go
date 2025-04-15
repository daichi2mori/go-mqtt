package handler

import (
	"context"
	"encoding/json"
	"go-mqtt/cache"
	"go-mqtt/config"
	"go-mqtt/mqtt"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// テスト用のMQTTメッセージの実装
type mockMessage struct {
	topic   string
	payload []byte
}

func (m *mockMessage) Duplicate() bool {
	return false
}

func (m *mockMessage) Qos() byte {
	return 0
}

func (m *mockMessage) Retained() bool {
	return false
}

func (m *mockMessage) Topic() string {
	return m.topic
}

func (m *mockMessage) MessageID() uint16 {
	return 0
}

func (m *mockMessage) Payload() []byte {
	return m.payload
}

func (m *mockMessage) Ack() {
}

func TestSubscriber1_Start(t *testing.T) {
	// キャッシュの初期化
	_ = cache.GetInstance()

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Topics: config.TopicConfig{
				Topic1: "test/topic1",
			},
		},
	}

	t.Run("通常動作：サブスクリプション成功後にコンテキストキャンセルで終了", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// サブスクリプションが呼ばれたことを確認するフラグ
		subscribed := false
		mockClient.SubscribeFunc = func(topic string, qos byte, callback MQTT.MessageHandler) error {
			if topic != cfg.Mqtt.Topics.Topic1 {
				t.Errorf("期待されるトピック %s に対して %s がサブスクライブされた", cfg.Mqtt.Topics.Topic1, topic)
			}
			subscribed = true
			return nil
		}

		// 通知用チャネル
		messageReceived := make(chan struct{}, 1)

		// サブスクライバー作成
		subscriber := NewSubscriber1(mockClient, cfg, messageReceived)

		// コンテキスト作成（すぐにキャンセルされるタイマー）
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// ゴルーチンでStart実行
		errCh := make(chan error)
		go func() {
			errCh <- subscriber.Start(ctx)
		}()

		// エラーを確認
		err := <-errCh
		if err != context.DeadlineExceeded {
			t.Errorf("context.DeadlineExceededエラーが期待されるが、%v だった", err)
		}

		// サブスクリプションが呼ばれたことを確認
		if !subscribed {
			t.Error("サブスクリプションが行われなかった")
		}
	})

	t.Run("サブスクリプションエラー", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// サブスクリプションでエラーを返すように設定
		expectedErr := &mqtt.MockError{Message: "サブスクリプションエラー"}
		mockClient.SubscribeFunc = func(topic string, qos byte, callback MQTT.MessageHandler) error {
			return expectedErr
		}

		// 通知用チャネル
		messageReceived := make(chan struct{}, 1)

		// サブスクライバー作成
		subscriber := NewSubscriber1(mockClient, cfg, messageReceived)

		// コンテキスト
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 直接Start呼び出し
		err := subscriber.Start(ctx)

		// エラーを確認
		if err == nil || err.Error() != "failed to subscribe: サブスクリプションエラー" {
			t.Errorf("期待されるエラーと異なる: %v", err)
		}
	})
}

func TestSubscriber1_MessageHandler(t *testing.T) {
	// キャッシュの初期化
	cache := cache.GetInstance()
	cache.Clear() // テスト前にクリア

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Topics: config.TopicConfig{
				Topic1: "test/topic1",
			},
		},
	}

	t.Run("メッセージ受信時の処理（初回）", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// Publish呼び出しをモニター
		publishCalled := false
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			if topic != cfg.Mqtt.Topics.Topic1 {
				t.Errorf("期待されるトピック %s に対して %s にパブリッシュされた", cfg.Mqtt.Topics.Topic1, topic)
			}
			publishCalled = true
			return nil
		}

		// Subscribe呼び出しをキャプチャしてハンドラーを取得
		var messageHandler MQTT.MessageHandler
		mockClient.SubscribeFunc = func(topic string, qos byte, callback MQTT.MessageHandler) error {
			messageHandler = callback
			return nil
		}

		// 通知用チャネル
		messageReceived := make(chan struct{}, 1)

		// サブスクライバー作成
		subscriber := NewSubscriber1(mockClient, cfg, messageReceived)

		// コンテキスト
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start呼び出し（サブスクリプション設定）
		go subscriber.Start(ctx)

		// ハンドラーが設定されるまで少し待機
		time.Sleep(100 * time.Millisecond)

		// テスト用のメッセージデータ
		testProps := Sub1Props{
			ID:   "1",
			Name: "test-message",
		}
		payload, _ := json.Marshal(testProps)

		// モックメッセージ作成
		msg := &mockMessage{
			topic:   cfg.Mqtt.Topics.Topic1,
			payload: payload,
		}

		// メッセージハンドラー呼び出し
		messageHandler(nil, msg)

		// 通知チャネルを確認
		select {
		case <-messageReceived:
			// 通知が送信されたことを確認
		case <-time.After(100 * time.Millisecond):
			t.Error("通知が送信されなかった")
		}

		// 再パブリッシュの確認
		if !publishCalled {
			t.Error("メッセージが再パブリッシュされなかった")
		}

		// キャッシュのデータ確認
		cacheKey := "sub1:1"
		cachedData, exists := cache.Get(cacheKey)
		if !exists {
			t.Error("キャッシュにデータが保存されていない")
		} else {
			sub1Props, ok := cachedData.(Sub1Props)
			if !ok {
				t.Error("キャッシュに保存されたデータが正しい型でない")
			} else if sub1Props.ID != "1" || sub1Props.Name != "test-message" {
				t.Errorf("キャッシュに保存されたデータが期待値と異なる: %+v", sub1Props)
			}
		}
	})

	t.Run("メッセージ受信時の処理（2回目以降）", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// Subscribe呼び出しをキャプチャしてハンドラーを取得
		var messageHandler MQTT.MessageHandler
		mockClient.SubscribeFunc = func(topic string, qos byte, callback MQTT.MessageHandler) error {
			messageHandler = callback
			return nil
		}

		// 通知用チャネル（バッファ1）
		messageReceived := make(chan struct{}, 1)

		// サブスクライバー作成
		sub1 := NewSubscriber1(mockClient, cfg, messageReceived).(*MQTTSubscriber1)

		// 初回メッセージ受信済みに設定
		sub1.firstMessageReceived.Store(true)

		// コンテキスト
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start呼び出し（サブスクリプション設定）
		go sub1.Start(ctx)

		// ハンドラーが設定されるまで少し待機
		time.Sleep(100 * time.Millisecond)

		// テスト用のメッセージデータ
		testProps := Sub1Props{
			ID:   "2",
			Name: "second-message",
		}
		payload, _ := json.Marshal(testProps)

		// モックメッセージ作成
		msg := &mockMessage{
			topic:   cfg.Mqtt.Topics.Topic1,
			payload: payload,
		}

		// メッセージハンドラー呼び出し
		messageHandler(nil, msg)

		// 通知チャネルを確認（通知は送信されないはず）
		select {
		case <-messageReceived:
			t.Error("2回目のメッセージで通知が送信された")
		case <-time.After(100 * time.Millisecond):
			// 通知が送信されないことを確認
		}

		// キャッシュのデータ確認
		cacheKey := "sub1:2"
		cachedData, exists := cache.Get(cacheKey)
		if !exists {
			t.Error("2回目のキャッシュにデータが保存されていない")
		} else {
			sub1Props, ok := cachedData.(Sub1Props)
			if !ok {
				t.Error("2回目のキャッシュに保存されたデータが正しい型でない")
			} else if sub1Props.ID != "2" || sub1Props.Name != "second-message" {
				t.Errorf("2回目のキャッシュに保存されたデータが期待値と異なる: %+v", sub1Props)
			}
		}
	})
}
