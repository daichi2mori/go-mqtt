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
	if err := s.subscribe(ctx); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// コンテキストのキャンセルを待つ
	<-ctx.Done()
	fmt.Println("Sub2 quit")
	return ctx.Err()
}

// subscribe はトピックへのサブスクリプションを設定します
func (s *MQTTSubscriber2) subscribe(ctx context.Context) error {
	msgHandler := s.createMessageHandler(ctx)
	return s.client.Subscribe(s.cfg.Mqtt.Topics.Topic1, 0, msgHandler)
}

// createMessageHandler はメッセージハンドラーを作成します
func (s *MQTTSubscriber2) createMessageHandler(ctx context.Context) MQTT.MessageHandler {
	return func(client MQTT.Client, msg MQTT.Message) {
		select {
		case <-ctx.Done():
			return // コンテキストがキャンセルされていたら処理しない
		default:
			// 処理を続行
		}

		payload := msg.Payload()

		var subProps Sub2Props
		if err := json.Unmarshal(payload, &subProps); err != nil {
			fmt.Printf("Failed to unmarshal message: %v\n", err)
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

		// Typeに基づいて異なる処理を実行
		if err := s.processMessageByType(subProps, exists, sub1Data); err != nil {
			fmt.Printf("Error processing message: %v\n", err)
		}
	}
}

// processMessageByType はメッセージタイプに基づいて処理を行います
func (s *MQTTSubscriber2) processMessageByType(props Sub2Props, hasSub1Data bool, sub1Data interface{}) error {
	payload, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	switch props.Type {
	case 1:
		// Type 1: トピック1に再パブリッシュ
		if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic1, 0, false, string(payload)); err != nil {
			return fmt.Errorf("failed to publish to topic1: %w", err)
		}
	case 2:
		// Type 2: トピック1とトピック2の両方に再パブリッシュ
		// キャッシュから取得したSub1のデータがあれば利用
		if hasSub1Data {
			fmt.Printf("Sub2: Using Sub1 data from cache: %v\n", sub1Data)
			// ここでsub1Dataを利用した処理を追加できます
		}

		// トピック1に再パブリッシュ
		if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic1, 0, false, string(payload)); err != nil {
			return fmt.Errorf("failed to publish to topic1: %w", err)
		}

		// トピック2に再パブリッシュ
		if err := s.client.Publish(s.cfg.Mqtt.Topics.Topic2, 0, false, string(payload)); err != nil {
			return fmt.Errorf("failed to publish to topic2: %w", err)
		}
	default:
		return fmt.Errorf("unknown message type: %d", props.Type)
	}

	return nil
}
