package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"go-mqtt/cache"
	"go-mqtt/config"
	mqttutil "go-mqtt/mqtt"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Sub2Props struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

// Subscriber2 はトピック2を購読するインターフェースです
type Subscriber2 interface {
	Start(ctx context.Context) error
}

// MQTTSubscriber2 はMQTTを使用したSubscriber2の実装です
type MQTTSubscriber2 struct {
	client mqttutil.MQTTClient
	cfg    *config.MqttConfig
}

// NewSubscriber2 は新しいSubscriber2を作成します
func NewSubscriber2(client mqttutil.MQTTClient, cfg *config.MqttConfig) Subscriber2 {
	return &MQTTSubscriber2{
		client: client,
		cfg:    cfg,
	}
}

// Start はサブスクライバーを開始します
func (s *MQTTSubscriber2) Start(ctx context.Context) error {
	// サブスクリプションを設定
	if err := s.subscribe(); err != nil {
		return err
	}

	// コンテキストのキャンセルを待つ
	<-ctx.Done()
	fmt.Println("Sub2 quit")
	return nil
}

// subscribe はトピックへのサブスクリプションを設定します
func (s *MQTTSubscriber2) subscribe() error {
	msgHandler := s.createMessageHandler()
	return s.client.Subscribe(s.cfg.Mqtt.Topics.Topic1, 0, msgHandler)
}

// createMessageHandler はメッセージハンドラーを作成します
func (s *MQTTSubscriber2) createMessageHandler() MQTT.MessageHandler {
	return func(client MQTT.Client, msg MQTT.Message) {
		payload := msg.Payload()

		var subProps Sub2Props
		if err := json.Unmarshal(payload, &subProps); err != nil {
			fmt.Println(err)
			return
		}

		// キャッシュにメッセージを保存
		cacheKey := fmt.Sprintf("sub2:%s:%d", subProps.ID, subProps.Type)
		cache.GetInstance().Set(cacheKey, subProps)
		fmt.Printf("Sub2: Stored message in cache with key: %s\n", cacheKey)

		// Sub1からのデータをキャッシュから取得
		sub1Key := fmt.Sprintf("sub1:%s", subProps.ID)
		sub1Data, exists := cache.GetInstance().Get(sub1Key)
		if exists {
			fmt.Printf("Sub2: Found data from Sub1 in cache with key: %s\n", sub1Key)
		}

		switch subProps.Type {
		case 1:
			if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic1, 0, false, string(payload)); err != nil {
				fmt.Println(err)
			}
		case 2:
			// キャッシュから取得したSub1のデータがあれば利用
			if exists {
				fmt.Printf("Sub2: Using Sub1 data from cache: %v\n", sub1Data)
			}

			if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic1, 0, false, string(payload)); err != nil {
				fmt.Println(err)
			}
			if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic2, 0, false, string(payload)); err != nil {
				fmt.Println(err)
			}
		}
	}
}

// 後方互換性のためのラッパー関数
func Sub2(ctx context.Context, client mqttutil.MQTTClient, cfg *config.MqttConfig) error {
	subscriber := NewSubscriber2(client, cfg)
	return subscriber.Start(ctx)
}
