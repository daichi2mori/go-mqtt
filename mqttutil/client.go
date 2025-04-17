package mqttutil

import (
	"crypto/tls"
	"errors"
	"fmt"
	"go-mqtt/config"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Config はMQTTクライアント設定を保持する
type Config struct {
	BrokerURL  string
	ClientID   string
	Username   string
	Password   string
	UseSSL     bool
	CACertPath string
	QoS        byte
	Retained   bool
}

// MessageHandler はメッセージ処理関数のシグネチャを定義
type MessageHandler func(topic string, payload []byte)

// Client はMQTT操作のインターフェース
type Client interface {
	Connect() error
	Disconnect()
	IsConnected() bool
	Publish(topic string, payload []byte) error
	Subscribe(topic string, handler MessageHandler) error
	Unsubscribe(topic string) error
	SetQoS(qos byte)
	SetRetained(retained bool)
}

// デフォルトの接続タイムアウト
const defaultConnectionTimeout = 10 * time.Second

// pahoClient はpaho MQTTを使用してClientインターフェースを実装
type pahoClient struct {
	config Config
	client paho.Client
}

// NewClient は新しいMQTTクライアントを作成
func NewClient(config Config) Client {
	return &pahoClient{
		config: config,
	}
}

// NewClientFromConfig はアプリケーション設定からMQTTクライアントを作成
func NewClientFromConfig(mqttConfig config.MQTTConfig) Client {
	return NewClient(Config{
		BrokerURL:  mqttConfig.BrokerURL,
		ClientID:   mqttConfig.ClientID,
		Username:   mqttConfig.Username,
		Password:   mqttConfig.Password,
		UseSSL:     mqttConfig.UseSSL,
		CACertPath: mqttConfig.CACertPath,
		QoS:        byte(mqttConfig.QoS),
		Retained:   mqttConfig.Retained,
	})
}

// Connect はMQTTブローカーへの接続を確立
func (c *pahoClient) Connect() error {
	if c.client != nil && c.client.IsConnected() {
		return nil
	}

	// MQTT接続オプション
	opts := paho.NewClientOptions().
		AddBroker(c.config.BrokerURL).
		SetClientID(c.config.ClientID).
		SetUsername(c.config.Username).
		SetPassword(c.config.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(defaultConnectionTimeout).
		SetCleanSession(true)

	// SSL設定
	if c.config.UseSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		if c.config.CACertPath != "" {
			// 実際のアプリケーションでは、ここで証明書をロードする
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// 接続切断ハンドラー
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		log.Printf("MQTT接続が切断されました: %v", err)
	})

	// MQTTクライアント作成
	c.client = paho.NewClient(opts)

	// ブローカーに接続
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTTブローカーへの接続に失敗: %w", token.Error())
	}

	return nil
}

// Disconnect はMQTTブローカーとの接続を終了
func (c *pahoClient) Disconnect() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250) // 250msタイムアウト
	}
}

// IsConnected はクライアントがブローカーに接続されているかどうかを返す
func (c *pahoClient) IsConnected() bool {
	return c.client != nil && c.client.IsConnected()
}

// Publish はトピックにメッセージを送信
func (c *pahoClient) Publish(topic string, payload []byte) error {
	if !c.IsConnected() {
		return errors.New("MQTTブローカーに接続されていません")
	}

	token := c.client.Publish(topic, c.config.QoS, c.config.Retained, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("メッセージの公開に失敗: %w", token.Error())
	}

	return nil
}

// Subscribe はトピックのメッセージを受信するサブスクリプションを追加
func (c *pahoClient) Subscribe(topic string, handler MessageHandler) error {
	if !c.IsConnected() {
		return errors.New("MQTTブローカーに接続されていません")
	}

	token := c.client.Subscribe(topic, c.config.QoS, func(_ paho.Client, msg paho.Message) {
		handler(msg.Topic(), msg.Payload())
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("トピック %s のサブスクライブに失敗: %w", topic, token.Error())
	}

	return nil
}

// Unsubscribe はトピックからサブスクリプションを削除
func (c *pahoClient) Unsubscribe(topic string) error {
	if !c.IsConnected() {
		return errors.New("MQTTブローカーに接続されていません")
	}

	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("トピック %s のサブスクリプション解除に失敗: %w", topic, token.Error())
	}

	return nil
}

// SetQoS はQoS値を設定
func (c *pahoClient) SetQoS(qos byte) {
	c.config.QoS = qos
}

// SetRetained はリテイン設定を変更
func (c *pahoClient) SetRetained(retained bool) {
	c.config.Retained = retained
}
