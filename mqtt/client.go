package mqttutil

import (
	"fmt"
	"log"
	"sync"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var (
	client     MQTT.Client
	clientOnce sync.Once
	mutex      sync.Mutex
)

// GetClient はシングルトンのMQTTクライアントを返します
func GetClient(brokerURL, clientID, username, password string) (MQTT.Client, error) {
	clientOnce.Do(func() {
		// MQTTクライアントオプションの設定
		opts := MQTT.NewClientOptions()
		opts.AddBroker(brokerURL)
		opts.SetClientID(clientID)
		opts.SetUsername(username)
		opts.SetPassword(password)
		opts.SetAutoReconnect(true)
		opts.SetMaxReconnectInterval(5000)

		// 接続時のコールバック
		opts.SetOnConnectHandler(func(client MQTT.Client) {
			log.Println("Connected to MQTT broker")
		})

		// 接続失敗時のコールバック
		opts.SetConnectionLostHandler(func(client MQTT.Client, err error) {
			log.Printf("Connection lost: %v", err)
		})

		// MQTTクライアントの作成
		client = MQTT.NewClient(opts)

		// ブローカーに接続
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("Failed to connect to MQTT broker: %v", token.Error())
		}
	})

	if !client.IsConnected() {
		return nil, fmt.Errorf("MQTT client is not connected")
	}

	return client, nil
}

// Subscribe はトピックをサブスクライブします
func Subscribe(client MQTT.Client, topic string, qos byte, callback MQTT.MessageHandler) error {
	mutex.Lock()
	defer mutex.Unlock()

	if token := client.Subscribe(topic, qos, callback); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %v", topic, token.Error())
	}

	log.Printf("Subscribed to topic: %s", topic)
	return nil
}

// Publish はメッセージをパブリッシュします
func Publish(client MQTT.Client, topic string, qos byte, retained bool, payload any) error {
	mutex.Lock()
	defer mutex.Unlock()

	if token := client.Publish(topic, qos, retained, payload); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %v", topic, token.Error())
	}

	return nil
}
