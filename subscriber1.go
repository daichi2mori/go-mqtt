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

// Subscriber1のトピック
const (
	subscriber1Topic = "device/1/data"
	publisher1Topic  = "device/1/response"
)

// デバイス1からのデータ構造
type Device1Data struct {
	DeviceID  string  `json:"device_id"`
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
	Status    string  `json:"status"`
}

// デバイス1への応答データ構造
type Device1Response struct {
	ResponseID string `json:"response_id"`
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
}

func startSubscriber1(client MQTT.Client) error {
	// ロガーの設定
	logFile, err := os.OpenFile("subscriber1.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "SUBSCRIBER1: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	// メッセージハンドラーを定義
	messageHandler := func(client MQTT.Client, msg MQTT.Message) {
		// JSONデータを受信
		var data Device1Data
		if err := json.Unmarshal(msg.Payload(), &data); err != nil {
			logger.Printf("Failed to unmarshal message: %v", err)
			return
		}

		// ログ出力
		logger.Printf("Received: %s", string(msg.Payload()))
		logger.Printf("Device ID: %s, Value: %.2f, Status: %s", data.DeviceID, data.Value, data.Status)

		// 応答データの作成
		response := Device1Response{
			ResponseID: "resp-" + data.DeviceID,
			Status:     "processed",
			Timestamp:  time.Now().Unix(),
		}

		// 応答データをJSON形式に変換
		respData, err := json.Marshal(response)
		if err != nil {
			logger.Printf("Failed to marshal response: %v", err)
			return
		}

		// 応答データをパブリッシュ
		if err := mqttutil.Publish(client, publisher1Topic, 0, false, respData); err != nil {
			logger.Printf("Failed to publish response: %v", err)
		} else {
			logger.Printf("Published response: %s", string(respData))
		}
	}

	// トピックをサブスクライブ
	if err := mqttutil.Subscribe(client, subscriber1Topic, 0, messageHandler); err != nil {
		return fmt.Errorf("failed to subscribe to topic: %v", err)
	}

	logger.Printf("Subscribed to topic: %s", subscriber1Topic)

	// このゴルーチンを継続して実行
	select {}
}
