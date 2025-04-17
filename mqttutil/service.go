package mqttutil

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

// Service は高レベルのMQTTサービスを表す
type Service struct {
	client    Client
	handlers  map[string][]MessageHandler
	mu        sync.RWMutex
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// NewService は新しいMQTTサービスを作成
func NewService(client Client) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		client:    client,
		handlers:  make(map[string][]MessageHandler),
		ctx:       ctx,
		cancelCtx: cancel,
	}
}

// Start はMQTTブローカーに接続しすべてのトピックをサブスクライブ
func (s *Service) Start() error {
	// ブローカーに接続
	if err := s.client.Connect(); err != nil {
		return err
	}

	// 登録されたトピックをサブスクライブ
	s.mu.RLock()
	defer s.mu.RUnlock()

	for topic := range s.handlers {
		if err := s.subscribeTopic(topic); err != nil {
			return err
		}
	}

	return nil
}

// Stop はMQTTブローカーとの接続を切断
func (s *Service) Stop() {
	s.cancelCtx()
	s.client.Disconnect()
}

// PublishJSON はJSONエンコードされたメッセージをトピックに公開
func (s *Service) PublishJSON(topic string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Publish(topic, payload)
}

// Subscribe はトピックにメッセージハンドラーを追加
func (s *Service) Subscribe(topic string, handler MessageHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ハンドラーを内部マップに追加
	if _, exists := s.handlers[topic]; !exists {
		s.handlers[topic] = []MessageHandler{}

		// クライアントが既に接続されている場合、トピックをサブスクライブ
		if s.client.IsConnected() {
			if err := s.subscribeTopic(topic); err != nil {
				return err
			}
		}
	}

	s.handlers[topic] = append(s.handlers[topic], handler)
	return nil
}

// subscribeTopic はトピックをサブスクライブし、登録されたハンドラーにメッセージをルーティング
// ロックが既に取得された状態で呼び出される内部メソッド
func (s *Service) subscribeTopic(topic string) error {
	return s.client.Subscribe(topic, func(t string, payload []byte) {
		s.handleMessage(t, payload)
	})
}

// handleMessage はメッセージをトピックに登録されたすべてのハンドラーにルーティング
func (s *Service) handleMessage(topic string, payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// このトピックのすべてのハンドラーを見つける
	handlers, exists := s.handlers[topic]
	if !exists {
		return
	}

	// 各ハンドラーを別のゴルーチンで呼び出す
	for _, handler := range handlers {
		h := handler // ゴルーチン用にコピーを作成
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("MQTTメッセージハンドラーでパニックから回復: %v", r)
				}
			}()
			h(topic, payload)
		}()
	}
}
