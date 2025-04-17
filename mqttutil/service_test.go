package mqttutil

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type TestMessage struct {
	Data string `json:"data"`
}

func TestServicePublishJSON(t *testing.T) {
	client := NewMockClient()
	service := NewService(client)

	// クライアントを接続
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() 失敗: %v", err)
	}

	// JSONメッセージの公開テスト
	topic := "test/json"
	testMsg := TestMessage{Data: "テストデータ"}

	err := service.PublishJSON(topic, testMsg)
	if err != nil {
		t.Errorf("PublishJSON() 失敗: %v", err)
	}

	// メッセージが正しく公開されたことを確認
	payload := client.GetLastPublishedMessage(topic)
	var receivedMsg TestMessage
	if err := json.Unmarshal(payload, &receivedMsg); err != nil {
		t.Errorf("公開されたメッセージのアンマーシャルに失敗: %v", err)
	}
	if receivedMsg.Data != testMsg.Data {
		t.Errorf("公開されたメッセージ = %+v、期待値は %+v", receivedMsg, testMsg)
	}

	// 公開エラーテスト
	testErr := errors.New("公開エラー")
	client.SetPublishError(testErr)

	err = service.PublishJSON(topic, testMsg)
	if err != testErr {
		t.Errorf("PublishJSON() エラー = %v、期待値は %v", err, testErr)
	}
}

func TestServiceSubscribe(t *testing.T) {
	client := NewMockClient()
	service := NewService(client)

	// クライアントを接続
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() 失敗: %v", err)
	}

	// トピックをサブスクライブ
	topic := "test/subscribe"
	var receivedPayload []byte
	var wg sync.WaitGroup
	wg.Add(1)

	err := service.Subscribe(topic, func(_ string, payload []byte) {
		receivedPayload = payload
		wg.Done()
	})
	if err != nil {
		t.Errorf("Subscribe() 失敗: %v", err)
	}

	// メッセージをシミュレート
	testPayload := []byte(`{"data":"テストデータ"}`)
	client.SimulateMessage(topic, testPayload)

	// メッセージが処理されるのを待つ
	waitTimeout := time.After(100 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 成功
	case <-waitTimeout:
		t.Fatal("メッセージハンドラーが呼び出されるのを待機してタイムアウト")
	}

	if string(receivedPayload) != string(testPayload) {
		t.Errorf("受信したペイロード = %s、期待値は %s", string(receivedPayload), string(testPayload))
	}
}

func TestServiceStartStop(t *testing.T) {
	client := NewMockClient()
	service := NewService(client)

	// 開始前にいくつかのトピックを登録
	topics := []string{"topic/1", "topic/2", "topic/3"}
	for _, topic := range topics {
		err := service.Subscribe(topic, func(_ string, _ []byte) {})
		if err != nil {
			t.Fatalf("Subscribe() 失敗: %v", err)
		}
	}

	// サービスを開始
	err := service.Start()
	if err != nil {
		t.Errorf("Start() 失敗: %v", err)
	}

	// クライアントが接続されていることを確認
	if !client.IsConnected() {
		t.Error("Start()後にクライアントが接続されていない")
	}

	// サービスを停止
	service.Stop()

	// クライアントが切断されていることを確認
	if client.IsConnected() {
		t.Error("Stop()後もクライアントが接続されたまま")
	}
}
