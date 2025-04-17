package mqttutil

import (
	"errors"
	"log"
	"sync"
)

// MockClient はテスト用のClientインターフェースのモック実装
type MockClient struct {
	connected        bool
	publishedMsgs    map[string][]byte
	subscriptions    map[string]MessageHandler
	mu               sync.RWMutex
	connectError     error
	publishError     error
	subscribeError   error
	unsubscribeError error
	qos              byte
	retained         bool
}

// NewMockClient は新しいモックMQTTクライアントを作成
func NewMockClient() *MockClient {
	return &MockClient{
		publishedMsgs: make(map[string][]byte),
		subscriptions: make(map[string]MessageHandler),
		qos:           1, // デフォルトQoS
	}
}

// Connect モック実装
func (m *MockClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.connectError != nil {
		return m.connectError
	}
	m.connected = true
	return nil
}

// Disconnect モック実装
func (m *MockClient) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
}

// IsConnected モック実装
func (m *MockClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// Publish モック実装
func (m *MockClient) Publish(topic string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishError != nil {
		return m.publishError
	}
	if !m.connected {
		return errors.New("MQTTブローカーに接続されていません")
	}

	// この時点でQoSとRetainedの値をログに出力したり
	// 検証用に保存したりすることもできます
	log.Printf("トピック: %s にメッセージを公開 (QoS: %d, Retained: %t)",
		topic, m.qos, m.retained)

	m.publishedMsgs[topic] = payload
	return nil
}

// Subscribe モック実装
func (m *MockClient) Subscribe(topic string, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.subscribeError != nil {
		return m.subscribeError
	}
	if !m.connected {
		return errors.New("MQTTブローカーに接続されていません")
	}
	m.subscriptions[topic] = handler
	return nil
}

// Unsubscribe モック実装
func (m *MockClient) Unsubscribe(topic string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unsubscribeError != nil {
		return m.unsubscribeError
	}
	if !m.connected {
		return errors.New("MQTTブローカーに接続されていません")
	}
	delete(m.subscriptions, topic)
	return nil
}

// SetConnectError はConnectが返すエラーを設定
func (m *MockClient) SetConnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectError = err
}

// SetPublishError はPublishが返すエラーを設定
func (m *MockClient) SetPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishError = err
}

// SetSubscribeError はSubscribeが返すエラーを設定
func (m *MockClient) SetSubscribeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeError = err
}

// SetUnsubscribeError はUnsubscribeが返すエラーを設定
func (m *MockClient) SetUnsubscribeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unsubscribeError = err
}

// GetLastPublishedMessage はトピックに公開された最後のメッセージを返す
func (m *MockClient) GetLastPublishedMessage(topic string) []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.publishedMsgs[topic]
}

// SimulateMessage はブローカーからの受信メッセージをシミュレート
func (m *MockClient) SimulateMessage(topic string, payload []byte) {
	m.mu.RLock()
	handler, exists := m.subscriptions[topic]
	m.mu.RUnlock()

	if exists {
		handler(topic, payload)
	}
}

// SetQoS モック実装
func (m *MockClient) SetQoS(qos byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.qos = qos
}

// SetRetained モック実装
func (m *MockClient) SetRetained(retained bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retained = retained
}

// GetQoS はテスト用に現在のQoS設定を取得
func (m *MockClient) GetQoS() byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.qos
}

// GetRetained はテスト用に現在のRetained設定を取得
func (m *MockClient) GetRetained() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.retained
}
