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

func TestSubscriber2_Start(t *testing.T) {
	// キャッシュの初期化
	_ = cache.GetInstance()

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Topics: config.TopicConfig{
				Topic1: "test/topic1",
				Topic2: "test/topic2",
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

		// サブスクライバー作成
		subscriber := NewSubscriber2(mockClient, cfg)

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

		// サブスクライバー作成
		subscriber := NewSubscriber2(mockClient, cfg)

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

func TestSubscriber2_MessageHandler(t *testing.T) {
	// キャッシュの初期化
	cache := cache.GetInstance()
	cache.Clear() // テスト前にクリア

	// テスト用のSub1データをキャッシュに追加
	cache.Set("sub1:1", Sub1Props{ID: "1", Name: "sub1-test"})

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Topics: config.TopicConfig{
				Topic1: "test/topic1",
				Topic2: "test/topic2",
			},
		},
	}

	t.Run("Type 1のメッセージ処理", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// パブリッシュ呼び出しをモニター
		publishedTopics := make(map[string]bool)
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			publishedTopics[topic] = true
			return nil
		}

		// Subscribe呼び出しをキャプチャしてハンドラーを取得
		var messageHandler MQTT.MessageHandler
		mockClient.SubscribeFunc = func(topic string, qos byte, callback MQTT.MessageHandler) error {
			messageHandler = callback
			return nil
		}

		// サブスクライバー作成
		subscriber := NewSubscriber2(mockClient, cfg)

		// コンテキスト
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start呼び出し（サブスクリプション設定）
		go subscriber.Start(ctx)

		// ハンドラーが設定されるまで少し待機
		time.Sleep(100 * time.Millisecond)

		// テスト用のメッセージデータ（Type 1）
		testProps := Sub2Props{
			ID:   "1",
			Name: "test-message-type1",
			Type: 1,
		}
		payload, _ := json.Marshal(testProps)

		// モックメッセージ作成
		msg := &mockMessage{
			topic:   cfg.Mqtt.Topics.Topic1,
			payload: payload,
		}

		// メッセージハンドラー呼び出し
		messageHandler(nil, msg)

		// パブリッシュ先トピックの確認
		if !publishedTopics[cfg.Mqtt.Topics.Topic1] {
			t.Error("トピック1へのパブリッシュが行われなかった")
		}
		if publishedTopics[cfg.Mqtt.Topics.Topic2] {
			t.Error("Type 1でトピック2へのパブリッシュが行われた")
		}

		// キャッシュのデータ確認
		cacheKey := "sub2:1:1" // ID:1, Type:1
		cachedData, exists := cache.Get(cacheKey)
		if !exists {
			t.Error("キャッシュにデータが保存されていない")
		} else {
			sub2Props, ok := cachedData.(Sub2Props)
			if !ok {
				t.Error("キャッシュに保存されたデータが正しい型でない")
			} else if sub2Props.ID != "1" || sub2Props.Name != "test-message-type1" || sub2Props.Type != 1 {
				t.Errorf("キャッシュに保存されたデータが期待値と異なる: %+v", sub2Props)
			}
		}
	})

	t.Run("Type 2のメッセージ処理", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// パブリッシュ呼び出しをモニター
		publishedTopics := make(map[string]bool)
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			publishedTopics[topic] = true
			return nil
		}

		// Subscribe呼び出しをキャプチャしてハンドラーを取得
		var messageHandler MQTT.MessageHandler
		mockClient.SubscribeFunc = func(topic string, qos byte, callback MQTT.MessageHandler) error {
			messageHandler = callback
			return nil
		}

		// サブスクライバー作成
		subscriber := NewSubscriber2(mockClient, cfg)

		// コンテキスト
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start呼び出し（サブスクリプション設定）
		go subscriber.Start(ctx)

		// ハンドラーが設定されるまで少し待機
		time.Sleep(100 * time.Millisecond)

		// テスト用のメッセージデータ（Type 2）
		testProps := Sub2Props{
			ID:   "1",
			Name: "test-message-type2",
			Type: 2,
		}
		payload, _ := json.Marshal(testProps)

		// モックメッセージ作成
		msg := &mockMessage{
			topic:   cfg.Mqtt.Topics.Topic1,
			payload: payload,
		}

		// メッセージハンドラー呼び出し
		messageHandler(nil, msg)

		// パブリッシュ先トピックの確認
		if !publishedTopics[cfg.Mqtt.Topics.Topic1] {
			t.Error("トピック1へのパブリッシュが行われなかった")
		}
		if !publishedTopics[cfg.Mqtt.Topics.Topic2] {
			t.Error("トピック2へのパブリッシュが行われなかった")
		}

		// キャッシュのデータ確認
		cacheKey := "sub2:1:2" // ID:1, Type:2
		cachedData, exists := cache.Get(cacheKey)
		if !exists {
			t.Error("キャッシュにデータが保存されていない")
		} else {
			sub2Props, ok := cachedData.(Sub2Props)
			if !ok {
				t.Error("キャッシュに保存されたデータが正しい型でない")
			} else if sub2Props.ID != "1" || sub2Props.Name != "test-message-type2" || sub2Props.Type != 2 {
				t.Errorf("キャッシュに保存されたデータが期待値と異なる: %+v", sub2Props)
			}
		}
	})
}

func TestSubscriber2_ProcessMessageByType(t *testing.T) {
	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Topics: config.TopicConfig{
				Topic1: "test/topic1",
				Topic2: "test/topic2",
			},
		},
	}

	// モックMQTTクライアントの作成
	mockClient := mqtt.NewMockClient()

	// サブスクライバー作成
	sub2 := NewSubscriber2(mockClient, cfg).(*MQTTSubscriber2)

	t.Run("Type 1のメッセージ処理", func(t *testing.T) {
		// パブリッシュ呼び出しをモニター
		publishedTopics := make(map[string]bool)
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			publishedTopics[topic] = true
			return nil
		}

		// テスト用のメッセージデータ
		testProps := Sub2Props{
			ID:   "1",
			Name: "test-message-type1",
			Type: 1,
		}

		// processMessageByTypeの呼び出し
		err := sub2.processMessageByType(testProps, true, Sub1Props{ID: "1", Name: "sub1-test"})

		// エラーがないことを確認
		if err != nil {
			t.Errorf("processMessageByTypeがエラーを返した: %v", err)
		}

		// パブリッシュ先トピックの確認
		if !publishedTopics[cfg.Mqtt.Topics.Topic1] {
			t.Error("トピック1へのパブリッシュが行われなかった")
		}
		if publishedTopics[cfg.Mqtt.Topics.Topic2] {
			t.Error("Type 1でトピック2へのパブリッシュが行われた")
		}
	})

	t.Run("Type 2のメッセージ処理", func(t *testing.T) {
		// パブリッシュ呼び出しをリセット
		publishedTopics := make(map[string]bool)
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			publishedTopics[topic] = true
			return nil
		}

		// テスト用のメッセージデータ
		testProps := Sub2Props{
			ID:   "1",
			Name: "test-message-type2",
			Type: 2,
		}

		// processMessageByTypeの呼び出し
		err := sub2.processMessageByType(testProps, true, Sub1Props{ID: "1", Name: "sub1-test"})

		// エラーがないことを確認
		if err != nil {
			t.Errorf("processMessageByTypeがエラーを返した: %v", err)
		}

		// パブリッシュ先トピックの確認
		if !publishedTopics[cfg.Mqtt.Topics.Topic1] {
			t.Error("トピック1へのパブリッシュが行われなかった")
		}
		if !publishedTopics[cfg.Mqtt.Topics.Topic2] {
			t.Error("トピック2へのパブリッシュが行われなかった")
		}
	})

	t.Run("不明なタイプのエラー処理", func(t *testing.T) {
		// パブリッシュ呼び出しをリセット
		publishedTopics := make(map[string]bool)
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			publishedTopics[topic] = true
			return nil
		}

		// テスト用のメッセージデータ（不明なタイプ）
		testProps := Sub2Props{
			ID:   "1",
			Name: "test-message-unknown-type",
			Type: 999, // 未定義のタイプ
		}

		// processMessageByTypeの呼び出し
		err := sub2.processMessageByType(testProps, false, nil)

		// エラーを確認
		if err == nil {
			t.Error("不明なタイプでエラーが返されなかった")
		} else if err.Error() != "unknown message type: 999" {
			t.Errorf("期待されるエラーと異なる: %v", err)
		}

		// パブリッシュが行われないことを確認
		if len(publishedTopics) > 0 {
			t.Error("不明なタイプでパブリッシュが行われた")
		}
	})

	t.Run("パブリッシュエラーの処理", func(t *testing.T) {
		// パブリッシュでエラーを返すように設定
		expectedErr := &mqtt.MockError{Message: "パブリッシュエラー"}
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			return expectedErr
		}

		// テスト用のメッセージデータ
		testProps := Sub2Props{
			ID:   "1",
			Name: "test-message-type1",
			Type: 1,
		}

		// processMessageByTypeの呼び出し
		err := sub2.processMessageByType(testProps, false, nil)

		// エラーを確認
		if err == nil {
			t.Error("パブリッシュエラーが返されなかった")
		} else if err.Error() != "failed to publish to topic1: パブリッシュエラー" {
			t.Errorf("期待されるエラーと異なる: %v", err)
		}
	})
}
