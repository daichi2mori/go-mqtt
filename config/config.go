package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// MQTTConfig はMQTT接続の設定を保持する
type MQTTConfig struct {
	BrokerURL  string `mapstructure:"broker_url"`
	ClientID   string `mapstructure:"client_id"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	UseSSL     bool   `mapstructure:"use_ssl"`
	CACertPath string `mapstructure:"ca_cert_path"`
	QoS        uint8  `mapstructure:"qos"`
	Retained   bool   `mapstructure:"retained"`
}

// AppConfig はアプリケーションの全体的な設定を保持する
type AppConfig struct {
	MQTT   MQTTConfig           `mapstructure:"mqtt"`
	Topics map[string]TopicInfo `mapstructure:"topics"`
}

// TopicInfo はトピックに関する設定情報を保持する
type TopicInfo struct {
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	QoS         *uint8 `mapstructure:"qos"`
}

// LoadConfig は設定ファイルを読み込み、AppConfig構造体を返す
func LoadConfig(configPath string) (*AppConfig, error) {
	v := viper.New()

	// 設定ファイルパスとファイル名を分離
	dir, file := filepath.Split(configPath)
	ext := filepath.Ext(file)
	name := strings.TrimSuffix(file, ext)

	// Viperの設定
	v.AddConfigPath(dir)
	v.SetConfigName(name)
	v.SetConfigType(strings.TrimPrefix(ext, "."))

	// 環境変数によるオーバーライドを有効化
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 設定ファイルを読み込み
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	// 設定を構造体にアンマーシャル
	var config AppConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("設定の解析に失敗: %w", err)
	}

	// デフォルト値の設定
	setDefaults(&config)

	return &config, nil
}

// setDefaults は設定にデフォルト値を設定する
func setDefaults(config *AppConfig) {
	// MQTTブローカーURLのデフォルト
	if config.MQTT.BrokerURL == "" {
		config.MQTT.BrokerURL = "tcp://localhost:1883"
	}

	// ClientIDが指定されていない場合、mqtt.NewClientでランダムなものが生成される
	// QoSのデフォルト
	if config.MQTT.QoS == 0 {
		config.MQTT.QoS = 1 // デフォルトはQoS 1
	}

	// トピックごとのQoSを確認し、指定されていない場合はグローバル設定を使用
	for topic, info := range config.Topics {
		if info.QoS == nil {
			qos := config.MQTT.QoS
			config.Topics[topic] = TopicInfo{
				Name:        info.Name,
				Description: info.Description,
				QoS:         &qos,
			}
		}
	}
}
