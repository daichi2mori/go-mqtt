package main

import (
	"encoding/json"
	"flag"
	"go-mqtt/config"
	"go-mqtt/mqttutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type SensorData struct {
	DeviceID  string    `json:"device_id"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// コマンドラインフラグ
	configPath := flag.String("config", "./config.yaml", "設定ファイルのパス")
	flag.Parse()

	// 設定を読み込み
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("設定の読み込みに失敗: %v", err)
	}

	// MQTTクライアントを作成
	client := mqttutil.NewClientFromConfig(cfg.MQTT)

	// MQTTサービスを作成
	service := mqttutil.NewService(client)

	// 設定ファイルからトピックを処理
	for _, topicInfo := range cfg.Topics {
		log.Printf("トピックをサブスクライブ: %s (%s)", topicInfo.Name, topicInfo.Description)

		// トピック固有のQoS設定があれば適用
		if topicInfo.QoS != nil {
			client.SetQoS(byte(*topicInfo.QoS))
		}

		// トピックをサブスクライブ
		err := service.Subscribe(topicInfo.Name, func(topic string, payload []byte) {
			var data SensorData
			if err := json.Unmarshal(payload, &data); err != nil {
				log.Printf("メッセージのアンマーシャルエラー: %v", err)
				return
			}
			log.Printf("%s でメッセージを受信: DeviceID=%s, Value=%.2f, Time=%s",
				topic, data.DeviceID, data.Value, data.Timestamp.Format(time.RFC3339))
		})

		if err != nil {
			log.Printf("トピック %s のサブスクライブに失敗: %v", topicInfo.Name, err)
		}

		// グローバルQoS設定に戻す
		client.SetQoS(byte(cfg.MQTT.QoS))
	}

	// サービスを開始
	log.Printf("MQTTブローカーに接続: %s", cfg.MQTT.BrokerURL)
	if err := service.Start(); err != nil {
		log.Fatalf("MQTTサービスの開始に失敗: %v", err)
	}
	defer service.Stop()

	// テストメッセージを公開
	sensorsTopic := cfg.Topics["sensors"].Name
	if sensorsTopic != "" {
		data := SensorData{
			DeviceID:  "device-001",
			Value:     23.5,
			Timestamp: time.Now(),
		}
		log.Printf("%s にテストメッセージを公開", sensorsTopic)
		if err := service.PublishJSON(sensorsTopic, data); err != nil {
			log.Printf("メッセージの公開に失敗: %v", err)
		}
	}

	// 割り込み信号を待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("シャットダウン中...")
}
