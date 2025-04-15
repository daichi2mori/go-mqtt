package handler

import (
	"context"
	"encoding/json"
	"go-mqtt/cache"
	"go-mqtt/config"
	"go-mqtt/mqtt"
	"testing"
	"time"
)

func TestPublisher_Start(t *testing.T) {
	// キャッシュの初期化
	_ = cache.GetInstance()

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Interval: 1, // 1秒間隔でパブリッシュ
			Topics: config.TopicConfig{
				Topic3: "test/topic3",
			},
		},
	}

	t.Run("通常動作：Sub1からの通知を受け取ってパブリッシュ開始", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// Sub1からの通知用チャネル
		messageReceived := make(chan struct{}, 1)

		// コンテキスト作成（短いタイムアウト）
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// パブリッシャー作成
		publisher := NewPublisher(mockClient, cfg, messageReceived)

		// パブリッシュされたメッセージをカウントするための変数
		var publishCount int

		// モックのPublish関数をオーバーライド
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			// トピックをチェック
			if topic != cfg.Mqtt.Topics.Topic3 {
				t.Errorf("期待されるトピック %s に対して %s が使用された", cfg.Mqtt.Topics.Topic3, topic)
			}

			// ペイロードをパース
			var req PubReq
			if err := json.Unmarshal([]byte(payload), &req); err != nil {
				t.Errorf("ペイロードのパースに失敗: %v", err)
			}

			// 期待値と比較
			if req.ID != "1" || req.Name != "test" {
				t.Errorf("期待されるペイロード {ID:1, Name:test} に対して %+v が送信された", req)
			}

			// カウント増加
			publishCount++
			return nil
		}

		// ゴルーチンでStart実行
		errCh := make(chan error)
		go func() {
			errCh <- publisher.Start(ctx)
		}()

		// Sub1からの通知シミュレーション
		time.Sleep(500 * time.Millisecond) // 少し待機
		messageReceived <- struct{}{}

		// コンテキストタイムアウトまで待機
		<-ctx.Done()

		// エラーを確認
		err := <-errCh
		if err != context.DeadlineExceeded {
			t.Errorf("予期しないエラー: %v", err)
		}

		// 少なくとも初期パブリッシュと1回以上の定期パブリッシュが行われたことを確認
		if publishCount < 2 {
			t.Errorf("最低2回のパブリッシュが期待されるが、%d回だった", publishCount)
		}

		// キャッシュにデータが保存されたことを確認
		cacheKey := "pub:1"
		cachedData, exists := cache.GetInstance().Get(cacheKey)
		if !exists {
			t.Error("キャッシュにデータが保存されていない")
		} else {
			pubReq, ok := cachedData.(PubReq)
			if !ok {
				t.Error("キャッシュに保存されたデータが正しい型でない")
			} else if pubReq.ID != "1" || pubReq.Name != "test" {
				t.Errorf("キャッシュに保存されたデータが期待値と異なる: %+v", pubReq)
			}
		}
	})

	t.Run("コンテキストキャンセル：通知を受け取る前に終了", func(t *testing.T) {
		// モックMQTTクライアントの作成
		mockClient := mqtt.NewMockClient()

		// Sub1からの通知用チャネル
		messageReceived := make(chan struct{}, 1)

		// コンテキスト作成して即キャンセル
		ctx, cancel := context.WithCancel(context.Background())

		// パブリッシャー作成
		publisher := NewPublisher(mockClient, cfg, messageReceived)

		// パブリッシュがされないことを確認
		mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
			t.Error("パブリッシュが行われるべきではない")
			return nil
		}

		// ゴルーチンでStart実行
		errCh := make(chan error)
		go func() {
			errCh <- publisher.Start(ctx)
		}()

		// すぐにキャンセル
		cancel()

		// エラーを確認
		err := <-errCh
		if err != context.Canceled {
			t.Errorf("context.Canceledエラーが期待されるが、%v だった", err)
		}
	})
}

func TestPublisher_publishMessage(t *testing.T) {
	// キャッシュの初期化
	cache := cache.GetInstance()
	cache.Clear() // テスト前にクリア

	// テスト用のSub1データをキャッシュに追加
	cache.Set("sub1:1", Sub1Props{ID: "1", Name: "sub1-test"})

	// テスト用の設定
	cfg := &config.MqttConfig{
		Mqtt: config.MqttSettings{
			Topics: config.TopicConfig{
				Topic3: "test/topic3",
			},
		},
	}

	// モックMQTTクライアントの作成
	mockClient := mqtt.NewMockClient()

	// パブリッシャー作成
	publisher := &MQTTPublisher{
		client:   mockClient,
		cfg:      cfg,
		interval: 1 * time.Second,
	}

	// モックのPublish関数をオーバーライド
	var publishedPayload string
	mockClient.PublishFunc = func(topic string, qos byte, retained bool, payload string) error {
		publishedPayload = payload
		return nil
	}

	// publishMessageの呼び出し
	err := publisher.publishMessage()

	// エラーがないことを確認
	if err != nil {
		t.Errorf("publishMessageがエラーを返した: %v", err)
	}

	// パブリッシュされたペイロードを確認
	var pubReq PubReq
	if err := json.Unmarshal([]byte(publishedPayload), &pubReq); err != nil {
		t.Fatalf("パブリッシュされたペイロードのパースに失敗: %v", err)
	}

	// 期待値と比較
	if pubReq.ID != "1" || pubReq.Name != "test" {
		t.Errorf("期待されるペイロード {ID:1, Name:test} に対して %+v が送信された", pubReq)
	}

	// キャッシュにデータが保存されたことを確認
	cacheKey := "pub:1"
	cachedData, exists := cache.Get(cacheKey)
	if !exists {
		t.Error("キャッシュにデータが保存されていない")
	} else {
		cachedPubReq, ok := cachedData.(PubReq)
		if !ok {
			t.Error("キャッシュに保存されたデータが正しい型でない")
		} else if cachedPubReq.ID != "1" || cachedPubReq.Name != "test" {
			t.Errorf("キャッシュに保存されたデータが期待値と異なる: %+v", cachedPubReq)
		}
	}
}
