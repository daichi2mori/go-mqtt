package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// テスト設定ファイルを一時的に作成
	configContent := `
mqtt:
  broker_url: "tcp://test-broker:1883"
  client_id: "test-client"
  username: "testuser"
  password: "testpass"
  use_ssl: true
  ca_cert_path: "/path/to/ca.crt"
  qos: 2
  retained: true

topics:
  test:
    name: "test/topic"
    description: "テスト用トピック"
    qos: 1

  another:
    name: "another/topic"
    description: "別のトピック"
`
	tempFile, err := os.CreateTemp("", "config_test*.yaml")
	if err != nil {
		t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("設定内容の書き込みに失敗: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("一時ファイルのクローズに失敗: %v", err)
	}

	// 設定を読み込み
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("設定の読み込みに失敗: %v", err)
	}

	// MQTT設定をテスト
	if cfg.MQTT.BrokerURL != "tcp://test-broker:1883" {
		t.Errorf("BrokerURL = %s、期待値は tcp://test-broker:1883", cfg.MQTT.BrokerURL)
	}
	if cfg.MQTT.ClientID != "test-client" {
		t.Errorf("ClientID = %s、期待値は test-client", cfg.MQTT.ClientID)
	}
	if cfg.MQTT.Username != "testuser" {
		t.Errorf("Username = %s、期待値は testuser", cfg.MQTT.Username)
	}
	if cfg.MQTT.Password != "testpass" {
		t.Errorf("Password = %s、期待値は testpass", cfg.MQTT.Password)
	}
	if !cfg.MQTT.UseSSL {
		t.Error("UseSSL = false、期待値は true")
	}
	if cfg.MQTT.CACertPath != "/path/to/ca.crt" {
		t.Errorf("CACertPath = %s、期待値は /path/to/ca.crt", cfg.MQTT.CACertPath)
	}
	if cfg.MQTT.QoS != 2 {
		t.Errorf("QoS = %d、期待値は 2", cfg.MQTT.QoS)
	}
	if !cfg.MQTT.Retained {
		t.Error("Retained = false、期待値は true")
	}

	// トピック設定をテスト
	if len(cfg.Topics) != 2 {
		t.Errorf("Topics数 = %d、期待値は 2", len(cfg.Topics))
	}

	// トピック「test」の設定をチェック
	testTopic, exists := cfg.Topics["test"]
	if !exists {
		t.Fatal("トピック 'test' が見つかりません")
	}
	if testTopic.Name != "test/topic" {
		t.Errorf("test.Name = %s、期待値は test/topic", testTopic.Name)
	}
	if testTopic.Description != "テスト用トピック" {
		t.Errorf("test.Description = %s、期待値は テスト用トピック", testTopic.Description)
	}
	if *testTopic.QoS != 1 {
		t.Errorf("test.QoS = %d、期待値は 1", *testTopic.QoS)
	}

	// トピック「another」の設定をチェック
	anotherTopic, exists := cfg.Topics["another"]
	if !exists {
		t.Fatal("トピック 'another' が見つかりません")
	}
	if anotherTopic.Name != "another/topic" {
		t.Errorf("another.Name = %s、期待値は another/topic", anotherTopic.Name)
	}
	// QoSが指定されていない場合、グローバル設定のQoSが使用されるはず
	if *anotherTopic.QoS != 2 {
		t.Errorf("another.QoS = %d、期待値は 2 (グローバル設定からの継承)", *anotherTopic.QoS)
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// 最小限の設定ファイルを作成
	configContent := `
mqtt:
  # 最小限の設定
topics:
  minimal:
    name: "minimal/topic"
    description: "最小限の設定テスト"
`
	tempFile, err := os.CreateTemp("", "minimal_config_test*.yaml")
	if err != nil {
		t.Fatalf("テスト設定ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("設定内容の書き込みに失敗: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("一時ファイルのクローズに失敗: %v", err)
	}

	// 設定を読み込み
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("設定の読み込みに失敗: %v", err)
	}

	// デフォルト値をテスト
	if cfg.MQTT.BrokerURL != "tcp://localhost:1883" {
		t.Errorf("デフォルトBrokerURL = %s、期待値は tcp://localhost:1883", cfg.MQTT.BrokerURL)
	}
	if cfg.MQTT.QoS != 1 {
		t.Errorf("デフォルトQoS = %d、期待値は 1", cfg.MQTT.QoS)
	}

	// トピックのデフォルト値をチェック
	minimalTopic, exists := cfg.Topics["minimal"]
	if !exists {
		t.Fatal("トピック 'minimal' が見つかりません")
	}
	if *minimalTopic.QoS != 1 {
		t.Errorf("minimal.QoS = %d、期待値は 1 (グローバルデフォルト)", *minimalTopic.QoS)
	}
}
