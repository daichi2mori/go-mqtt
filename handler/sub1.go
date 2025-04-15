package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"go-mqtt/cache"
	"go-mqtt/config"
	mqttutil "go-mqtt/mqtt"
	"sync/atomic"

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
	client               mqttutil.MQTTClient
	cfg                  *config.MqttConfig
	messageReceived      chan<- struct{} // メッセージ受信を通知するチャネル
	firstMessageReceived atomic.Bool     // 最初のメッセージを受信したかどうかのフラグ（アトミック操作用）
}

// NewSubscriber1 は新しいSubscriber1を作成します
func NewSubscriber1(client mqttutil.MQTTClient, cfg *config.MqttConfig, messageReceived chan<- struct{}) Subscriber1 {
	return &MQTTSubscriber1{
		client:          client,
		cfg:             cfg,
		messageReceived: messageReceived,
	}
}

// Start はサブスクライバーを開始します
func (s *MQTTSubscriber1) Start(ctx context.Context) error {
	// サブスクリプションを設定
	if err := s.subscribe(ctx); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// コンテキストのキャンセルを待つ
	<-ctx.Done()
	fmt.Println("Sub1 quit")
	return ctx.Err()
}

// subscribe はトピックへのサブスクリプションを設定します
func (s *MQTTSubscriber1) subscribe(ctx context.Context) error {
	msgHandler := s.createMessageHandler(ctx)
	return s.client.Subscribe(s.cfg.Mqtt.Topics.Topic1, 0, msgHandler)
}

// createMessageHandler はメッセージハンドラーを作成します
func (s *MQTTSubscriber1) createMessageHandler(ctx context.Context) MQTT.MessageHandler {
	return func(client MQTT.Client, msg MQTT.Message) {
		select {
		case <-ctx.Done():
			return // コンテキストがキャンセルされていたら処理しない
		default:
			// 処理を続行
		}

		payload := msg.Payload()

		var subProps Sub1Props
		if err := json.Unmarshal(payload, &subProps); err != nil {
			fmt.Printf("Failed to unmarshal message: %v\n", err)
			return
		}

		// キャッシュにメッセージを保存
		cacheKey := fmt.Sprintf("sub1:%s", subProps.ID)
		cache.GetInstance().Set(cacheKey, subProps)
		fmt.Printf("Sub1: Stored message in cache with key: %s\n", cacheKey)

		// 最初のメッセージを受信した場合、パブリッシャーに通知（アトミック操作で競合を排除）
		if !s.firstMessageReceived.Load() && s.firstMessageReceived.CompareAndSwap(false, true) {
			fmt.Println("Sub1: First message received, notifying publisher")
			select {
			case s.messageReceived <- struct{}{}:
				fmt.Println("Sub1: Publisher notified successfully")
			default:
				// チャネルがすでに通知を受けている場合は無視
				fmt.Println("Sub1: Publisher already notified or not listening")
			}
		}

		// 同じトピックに再パブリッシュ
		if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic1, 0, false, string(payload)); err != nil {
			fmt.Printf("Failed to republish message: %v\n", err)
		}
	}
}
