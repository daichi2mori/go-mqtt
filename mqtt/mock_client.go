package mqttutil

import (
	"sync"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// MockMQTTClient はテスト用のモッククライアントです
type MockMQTTClient struct {
	connected      bool
	publishCalled  int
	publishTopic   string
	publishPayload interface{}
	publishError   error

	subscribeCalled  int
	subscribeTopic   string
	subscribeHandler MQTT.MessageHandler
	subscribeError   error

	mu sync.Mutex
}

// NewMockClient は新しいモッククライアントを作成します
func NewMockClient() *MockMQTTClient {
	return &MockMQTTClient{
		connected: false,
	}
}

// Connect はモックの接続操作をシミュレートします
func (m *MockMQTTClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Disconnect はモックの切断操作をシミュレートします
func (m *MockMQTTClient) Disconnect(quiesce uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
}

// IsConnected は接続状態を返します
func (m *MockMQTTClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// Publish はモックのパブリッシュ操作をシミュレートします
func (m *MockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishCalled++
	m.publishTopic = topic
	m.publishPayload = payload
	return m.publishError
}

// SetPublishError はパブリッシュ時に返すエラーを設定します
func (m *MockMQTTClient) SetPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishError = err
}

// GetPublishCount はパブリッシュが呼ばれた回数を返します
func (m *MockMQTTClient) GetPublishCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishCalled
}

// GetLastPublishTopic は最後にパブリッシュされたトピックを返します
func (m *MockMQTTClient) GetLastPublishTopic() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishTopic
}

// GetLastPublishPayload は最後にパブリッシュされたペイロードを返します
func (m *MockMQTTClient) GetLastPublishPayload() interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishPayload
}

// Subscribe はモックのサブスクライブ操作をシミュレートします
func (m *MockMQTTClient) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeCalled++
	m.subscribeTopic = topic
	m.subscribeHandler = callback
	return m.subscribeError
}

// SetSubscribeError はサブスクライブ時に返すエラーを設定します
func (m *MockMQTTClient) SetSubscribeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeError = err
}

// GetSubscribeCount はサブスクライブが呼ばれた回数を返します
func (m *MockMQTTClient) GetSubscribeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.subscribeCalled
}

// GetLastSubscribeTopic は最後にサブスクライブされたトピックを返します
func (m *MockMQTTClient) GetLastSubscribeTopic() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.subscribeTopic
}

// GetSubscribeHandler は設定されたハンドラーを返します
func (m *MockMQTTClient) GetSubscribeHandler() MQTT.MessageHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.subscribeHandler
}

// SimulateMessage はサブスクライブしたトピックにメッセージが来たことをシミュレートします
func (m *MockMQTTClient) SimulateMessage(topic string, payload []byte) {
	m.mu.Lock()
	handler := m.subscribeHandler
	m.mu.Unlock()

	if handler != nil && m.subscribeTopic == topic {
		handler(nil, &MockMessage{
			Topic1:   topic,
			Payload1: payload,
		})
	}
}

// MockMessage はMQTTメッセージのモック実装です
type MockMessage struct {
	Topic1   string
	Payload1 []byte
}

func (m *MockMessage) Duplicate() bool {
	return false
}

func (m *MockMessage) Qos() byte {
	return 0
}

func (m *MockMessage) Retained() bool {
	return false
}

func (m *MockMessage) Topic() string {
	return m.Topic1
}

func (m *MockMessage) MessageID() uint16 {
	return 1
}

func (m *MockMessage) Payload() []byte {
	return m.Payload1
}

func (m *MockMessage) Ack() {
	// 何もしない
}
