package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"go-mqtt/cache"
	"go-mqtt/config"
	mqttutil "go-mqtt/mqtt"
	"time"
)

type PubReq struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Publisher はメッセージのパブリッシュを行うインターフェースです
type Publisher interface {
	Start(ctx context.Context) error
}

// MQTTPublisher はMQTTを使用したPublisherの実装です
type MQTTPublisher struct {
	client          mqttutil.MQTTClient
	cfg             *config.MqttConfig
	interval        time.Duration
	messageReceived <-chan struct{} // Sub1からのメッセージ受信通知を受け取るチャネル
}

// NewPublisher は新しいPublisherを作成します
func NewPublisher(client mqttutil.MQTTClient, cfg *config.MqttConfig, messageReceived <-chan struct{}) Publisher {
	return &MQTTPublisher{
		client:          client,
		cfg:             cfg,
		interval:        time.Duration(cfg.Mqtt.Interval) * time.Second,
		messageReceived: messageReceived,
	}
}

// Start はパブリッシャーを開始します
func (p *MQTTPublisher) Start(ctx context.Context) error {
	fmt.Println("Publisher waiting for Sub1 to receive a message...")

	// Sub1からのメッセージ受信通知を待つ
	select {
	case <-p.messageReceived:
		fmt.Println("Publisher received notification from Sub1, starting publishing cycle")
	case <-ctx.Done():
		fmt.Println("Publisher shutting down before receiving any messages")
		return ctx.Err()
	}

	// Sub1がメッセージを受信したら、定期的にメッセージのパブリッシュを開始
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// 最初に1回パブリッシュ
	if err := p.publishMessage(); err != nil {
		return fmt.Errorf("initial publish failed: %w", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.publishMessage(); err != nil {
				fmt.Printf("Publish error: %v\n", err)
				// エラーがあっても処理を続行
			}
		case <-ctx.Done():
			fmt.Println("Pub quit")
			return ctx.Err()
		}
	}
}

// publishMessage は単一のメッセージをパブリッシュします
func (p *MQTTPublisher) publishMessage() error {
	// 新しいメッセージを作成
	req := PubReq{
		ID:   "1",
		Name: "test",
	}

	// キャッシュからデータを取得
	sub1Key := "sub1:1" // IDが1のSub1データ
	sub1Data, exists := cache.GetInstance().Get(sub1Key)
	if exists {
		fmt.Printf("Found Sub1 data in cache: %v\n", sub1Data)
		// ここでキャッシュから取得したデータを利用できます
	}

	// キャッシュに保存
	cacheKey := fmt.Sprintf("pub:%s", req.ID)
	cache.GetInstance().Set(cacheKey, req)

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := p.client.Publish(p.cfg.Mqtt.Topics.Topic3, 0, false, string(payload)); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
