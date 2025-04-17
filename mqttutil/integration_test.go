//go:build integration
// +build integration

package mqttutil

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"
)

// TestIntegration_MQTT は実行中のMQTTブローカーを必要とする統合テスト
// このテストを実行するには: go test -tags=integration -v ./mqtt
func TestIntegration_MQTT(t *testing.T) {
	// 環境からブローカーURLを取得するか、デフォルトを使用
	brokerURL := os.Getenv("MQTT_BROKER_URL")
	if brokerURL == "" {
		brokerURL = "tcp://localhost:1883"
	}

	// MQTT設定を作成
	config := Config{
		BrokerURL: brokerURL,
		ClientID:  "mqtt-integration-test",
		QoS:       1,
		Retained:  false,
	}

	// 公開用と購読用の2つのクライアントを作成
	publisher := NewClient(config)
	subscriber := NewClient(Config{
		BrokerURL: brokerURL,
		ClientID:  "mqtt-integration-test-sub",
		QoS:       1,
		Retained:  false,
	})

	// 両方のクライアントを接続
	if err := publisher.Connect(); err != nil {
		t.Fatalf("パブリッシャーの接続に失敗: %v", err)
	}
	defer publisher.Disconnect()

	if err := subscriber.Connect(); err != nil {
		t.Fatalf("サブスクライバーの接続に失敗: %v", err)
	}
	defer subscriber.Disconnect()

	// テストトピックとメッセージを設定
	topic := "mqtt-integration-test/topic"
	type TestMsg struct {
		Text      string    `json:"text"`
		Value     int       `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}
	testMsg := TestMsg{
		Text:      "Hello MQTT",
		Value:     42,
		Timestamp: time.Now(),
	}
	jsonData, err := json.Marshal(testMsg)
	if err != nil {
		t.Fatalf("JSONのマーシャルに失敗: %v", err)
	}

	// サブスクリプションコールバック用の待機グループを設定
	var wg sync.WaitGroup
	wg.Add(1)

	// 受信したメッセージを格納する変数
	var receivedMsg []byte
	var receivedTopic string

	// テストトピックをサブスクライブ
	if err := subscriber.Subscribe(topic, func(t string, payload []byte) {
		receivedTopic = t
		receivedMsg = make([]byte, len(payload))
		copy(receivedMsg, payload)
		wg.Done()
	}); err != nil {
		t.Fatalf("サブスクライブに失敗: %v", err)
	}

	// サブスクリプションが有効になるのを少し待つ
	time.Sleep(500 * time.Millisecond)

	// テストメッセージを公開
	if err := publisher.Publish(topic, jsonData); err != nil {
		t.Fatalf("公開に失敗: %v", err)
	}

	// タイムアウト付きでメッセージを受信するのを待つ
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 成功、メッセージ受信
	case <-time.After(5 * time.Second):
		t.Fatal("メッセージ待機がタイムアウト")
	}

	// メッセージを検証
	if receivedTopic != topic {
		t.Errorf("受信したトピック = %s、期待値は %s", receivedTopic, topic)
	}

	var receivedData TestMsg
	if err := json.Unmarshal(receivedMsg, &receivedData); err != nil {
		t.Fatalf("受信したメッセージのアンマーシャルに失敗: %v", err)
	}

	if receivedData.Text != testMsg.Text {
		t.Errorf("受信したメッセージテキスト = %s、期待値は %s", receivedData.Text, testMsg.Text)
	}
	if receivedData.Value != testMsg.Value {
		t.Errorf("受信したメッセージ値 = %d、期待値は %d", receivedData.Value, testMsg.Value)
	}
}
