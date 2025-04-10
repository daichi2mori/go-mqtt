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

// 定期的なパブリッシュ用のトピック
const periodicTopic = "system/heartbeat"

// シンプルなハートビートデータ構造（プロパティが1つだけ）
type HeartbeatData struct {
	Timestamp int64 `json:"timestamp"`
}

func startPeriodicPublisher(client MQTT.Client) error {
	// ロガーの設定
	logFile, err := os.OpenFile("publisher.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "PUBLISHER: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	logger.Println("Periodic publisher started")

	// 100ms毎にハートビートデータをパブリッシュするタイマーを設定
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// タイマーイベントの処理
	for range ticker.C {
		// 現在時刻のUNIXタイムスタンプを取得
		currentTime := time.Now().UnixNano() / int64(time.Millisecond)

		// ハートビートデータの作成
		heartbeat := HeartbeatData{
			Timestamp: currentTime,
		}

		// ハートビートデータをJSON形式に変換
		payload, err := json.Marshal(heartbeat)
		if err != nil {
			logger.Printf("Failed to marshal heartbeat data: %v", err)
			continue
		}

		// ハートビートデータをパブリッシュ
		if err := mqttutil.Publish(client, periodicTopic, 0, false, payload); err != nil {
			logger.Printf("Failed to publish heartbeat: %v", err)
		} else {
			logger.Printf("Published heartbeat: %s", string(payload))
		}
	}

	return nil
}
