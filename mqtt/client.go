package mqttutil

import (
	"fmt"
	"go-mqtt/config"
	"sync"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// MQTTClient はMQTTクライアントの操作を定義するインターフェースです
type MQTTClient interface {
	Connect() error
	Disconnect(quiesce uint)
	IsConnected() bool
	Publish(topic string, qos byte, retained bool, payload interface{}) error
	Subscribe(topic string, qos byte, callback MQTT.MessageHandler) error
}

// PahoMQTTClient は実際のMQTTクライアントの実装です
type PahoMQTTClient struct {
	client MQTT.Client
	mutex  sync.Mutex
}

var (
	clientOnce sync.Once
)

func onConnectHandler(c MQTT.Client) {
	fmt.Println("接続された")
}

// NewClient は新しいMQTTクライアントを作成します
func NewClient(cfg *config.MqttConfig) MQTTClient {
	opts := MQTT.NewClientOptions().AddBroker(cfg.Mqtt.Broker).SetOnConnectHandler(onConnectHandler)
	client := MQTT.NewClient(opts)

	mqttClient := &PahoMQTTClient{
		client: client,
	}

	return mqttClient
}

// Connect はMQTTブローカーに接続します
func (c *PahoMQTTClient) Connect() error {
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Disconnect はMQTTブローカーから切断します
func (c *PahoMQTTClient) Disconnect(quiesce uint) {
	c.client.Disconnect(quiesce)
}

// IsConnected は接続状態を返します
func (c *PahoMQTTClient) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish はメッセージをパブリッシュします
func (c *PahoMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	token := c.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Subscribe はトピックをサブスクライブします
func (c *PahoMQTTClient) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	token := c.client.Subscribe(topic, qos, callback)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
