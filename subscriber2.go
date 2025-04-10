package main

import (
	"encoding/json"
	"fmt"
	mqttutil "go-mqtt/mqtt"
	"log"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Subscriber2のトピック
const (
	subscriber2Topic = "device/2/data"
	publisher2Topic  = "device/2/response"
)

// デバイス2からのデータ構造
type Device2Data struct {
	SensorID     string             `json:"sensor_id"`
	Temperature  float64            `json:"temperature"`
	Humidity     float64            `json:"humidity"`
	Measurements map[string]float64 `json:"measurements"`
	RecordedAt   string             `json:"recorded_at"`
}

// デバイス2への応答データ構造
type Device2Response struct {
	AckID      string `json:"ack_id"`
	DeviceType string `json:"device_type"`
	Message    string `json:"message"`
	Timestamp  int64  `json:"timestamp"`
}

func startSubscriber2(client MQTT.Client) error {
	// ロガーの設定
	logFile, err := os.OpenFile("subscriber2.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "SUBSCRIBER2: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	// メッセージハンドラーを定義
	messageHandler := func(client MQTT.Client, msg MQTT.Message) {
		// JSONデータを受信
		var data Device2Data
		if err := json.Unmarshal(msg.Payload(), &data); err != nil {
			logger.Printf("Failed to unmarshal message: %v", err)
			return
		}

		// ログ出力
		logger.Printf("Received: %s", string(msg.Payload()))
		logger.Printf("Sensor ID: %s, Temperature: %.2f, Humidity: %.2f",
			data.SensorID, data.Temperature, data.Humidity)

		// 測定値の詳細をログ出力
		for key, value := range data.Measurements {
			logger.Printf("Measurement - %s: %.2f", key, value)
		}

		// 応答データの作成
		response := Device2Response{
			AckID:      "ack-" + data.SensorID,
			DeviceType: "environmental-sensor",
			Message:    "data received successfully",
			Timestamp:  time.Now().Unix(),
		}

		// 応答データをJSON形式に変換
		respData, err := json.Marshal(response)
		if err != nil {
			logger.Printf("Failed to marshal response: %v", err)
			return
		}

		// 応答データをパブリッシュ
		if err := mqttutil.Publish(client, publisher2Topic, 0, false, respData); err != nil {
			logger.Printf("Failed to publish response: %v", err)
		} else {
			logger.Printf("Published response: %s", string(respData))
		}
	}

	// トピックをサブスクライブ
	if err := mqttutil.Subscribe(client, subscriber2Topic, 0, messageHandler); err != nil {
		return fmt.Errorf("failed to subscribe to topic: %v", err)
	}

	logger.Printf("Subscribed to topic: %s", subscriber2Topic)

	// このゴルーチンを継続して実行
	select {}
}
