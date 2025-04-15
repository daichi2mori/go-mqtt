package mqttutil

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// MockClient はMQTTClientのモック実装です
type MockClient struct {
	ConnectFunc     func() error
	DisconnectFunc  func(quiesce uint)
	PublishFunc     func(topic string, qos byte, retained bool, payload string) error
	SubscribeFunc   func(topic string, qos byte, callback MQTT.MessageHandler) error
	UnsubscribeFunc func(topics []string) error
}

// MockError はテスト用のエラー型です
type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}

// NewMockClient は新しいモックMQTTクライアントを作成します
func NewMockClient() *MockClient {
	return &MockClient{
		ConnectFunc: func() error {
			return nil
		},
		DisconnectFunc: func(quiesce uint) {
			// 何もしない
		},
		PublishFunc: func(topic string, qos byte, retained bool, payload string) error {
			return nil
		},
		SubscribeFunc: func(topic string, qos byte, callback MQTT.MessageHandler) error {
			return nil
		},
		UnsubscribeFunc: func(topics []string) error {
			return nil
		},
	}
}

// Connect はモックの接続処理です
func (m *MockClient) Connect() error {
	return m.ConnectFunc()
}

// Disconnect はモックの切断処理です
func (m *MockClient) Disconnect(quiesce uint) {
	m.DisconnectFunc(quiesce)
}

// Publish はモックのパブリッシュ処理です
func (m *MockClient) Publish(topic string, qos byte, retained bool, payload string) error {
	return m.PublishFunc(topic, qos, retained, payload)
}

// Subscribe はモックのサブスクライブ処理です
func (m *MockClient) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) error {
	return m.SubscribeFunc(topic, qos, callback)
}

// Unsubscribe はモックのアンサブスクライブ処理です
func (m *MockClient) Unsubscribe(topics []string) error {
	return m.UnsubscribeFunc(topics)
}
