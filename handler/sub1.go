package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"go-mqtt/cache"
	"go-mqtt/config"
	mqttutil "go-mqtt/mqtt"
	"sync"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Sub1Props struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Subscriber1 はトピック1を購読するインターフェースです
type Subscriber1 interface {
	Start(ctx context.Context) error
}

// MQTTSubscriber1 はMQTTを使用したSubscriber1の実装です
type MQTTSubscriber1 struct {
	client         mqttutil.MQTTClient
	cfg            *config.MqttConfig
	messageReceived chan<- struct{} // メッセージ受信を通知するチャネル
	firstMessageReceived bool       // 最初のメッセージを受信したかどうかのフラグ
	mutex          sync.Mutex      // フラグアクセス用のミューテックス
}

// NewSubscriber1 は新しいSubscriber1を作成します
func NewSubscriber1(client mqttutil.MQTTClient, cfg *config.MqttConfig, messageReceived chan<- struct{}) Subscriber1 {
	return &MQTTSubscriber1{
		client:         client,
		cfg:            cfg,
		messageReceived: messageReceived,
		firstMessageReceived: false,
	}
}

// Start はサブスクライバーを開始します
func (s *MQTTSubscriber1) Start(ctx context.Context) error {
	// サブスクリプションを設定
	if err := s.subscribe(); err != nil {
		return err
	}

	// コンテキストのキャンセルを待つ
	<-ctx.Done()
	fmt.Println("Sub1 quit")
	return nil
}

// subscribe はトピックへのサブスクリプションを設定します
func (s *MQTTSubscriber1) subscribe() error {
	msgHandler := s.createMessageHandler()
	return s.client.Subscribe(s.cfg.Mqtt.Topics.Topic1, 0, msgHandler)
}

// createMessageHandler はメッセージハンドラーを作成します
func (s *MQTTSubscriber1) createMessageHandler() MQTT.MessageHandler {
	return func(client MQTT.Client, msg MQTT.Message) {
		payload := msg.Payload()

		var subProps Sub1Props
		if err := json.Unmarshal(payload, &subProps); err != nil {
			fmt.Println(err)
			return
		}

		// キャッシュにメッセージを保存
		cacheKey := fmt.Sprintf("sub1:%s", subProps.ID)
		cache.GetInstance().Set(cacheKey, subProps)
		fmt.Printf("Sub1: Stored message in cache with key: %s\n", cacheKey)

		// 最初のメッセージを受信した場合、パブリッシャーに通知
		s.mutex.Lock()
		if !s.firstMessageReceived {
			s.firstMessageReceived = true
			s.mutex.Unlock()
			// チャネルに通知を送信（ノンブロッキング）
			fmt.Println("Sub1: First message received, notifying publisher")
			select {
			case s.messageReceived <- struct{}{}:
				fmt.Println("Sub1: Publisher notified successfully")
			default:
				// チャネルがすでに通知を受けている場合は無視
				fmt.Println("Sub1: Publisher already notified")
			}
		} else {
			s.mutex.Unlock()
		}

		if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic1, 0, false, string(payload)); err != nil {
			fmt.Println(err)
		}
	}
}

// 後方互換性のためのラッパー関数
func Sub1(ctx context.Context, client mqttutil.MQTTClient, cfg *config.MqttConfig, messageReceived chan<- struct{}) error {
	subscriber := NewSubscriber1(client, cfg, messageReceived)
	return subscriber.Start(ctx)
}
