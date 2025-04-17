package mqttutil

import (
	"errors"
	"testing"
)

func TestMockClientConnect(t *testing.T) {
	client := NewMockClient()

	// 接続成功テスト
	err := client.Connect()
	if err != nil {
		t.Errorf("Connect() 失敗: %v", err)
	}
	if !client.IsConnected() {
		t.Error("Connect()後にIsConnected() = false、期待値はtrue")
	}

	// 接続エラーテスト
	testErr := errors.New("接続エラー")
	client.SetConnectError(testErr)

	err = client.Connect()
	if err != testErr {
		t.Errorf("Connect() エラー = %v、期待値は %v", err, testErr)
	}
}

func TestMockClientDisconnect(t *testing.T) {
	client := NewMockClient()

	// 最初に接続
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() 失敗: %v", err)
	}

	// 切断テスト
	client.Disconnect()
	if client.IsConnected() {
		t.Error("Disconnect()後にIsConnected() = true、期待値はfalse")
	}
}

func TestMockClientPublish(t *testing.T) {
	client := NewMockClient()

	// 接続されていない状態での公開テスト
	topic := "test/topic"
	payload := []byte("テストメッセージ")

	err := client.Publish(topic, payload)
	if err == nil {
		t.Error("未接続クライアントでのPublish()がエラーを返さなかった")
	}

	// 接続して公開テスト
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() 失敗: %v", err)
	}

	err = client.Publish(topic, payload)
	if err != nil {
		t.Errorf("Publish() 失敗: %v", err)
	}

	// メッセージが保存されたことを確認
	msg := client.GetLastPublishedMessage(topic)
	if string(msg) != string(payload) {
		t.Errorf("GetLastPublishedMessage() = %s、期待値は %s", string(msg), string(payload))
	}

	// 公開エラーテスト
	testErr := errors.New("公開エラー")
	client.SetPublishError(testErr)

	err = client.Publish(topic, payload)
	if err != testErr {
		t.Errorf("Publish() エラー = %v、期待値は %v", err, testErr)
	}
}

func TestMockClientSubscribe(t *testing.T) {
	client := NewMockClient()

	// 接続されていない状態でのサブスクライブテスト
	topic := "test/topic"
	var receivedTopic string
	var receivedPayload []byte

	handler := func(t string, p []byte) {
		receivedTopic = t
		receivedPayload = p
	}

	err := client.Subscribe(topic, handler)
	if err == nil {
		t.Error("未接続クライアントでのSubscribe()がエラーを返さなかった")
	}

	// 接続してサブスクライブテスト
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() 失敗: %v", err)
	}

	err = client.Subscribe(topic, handler)
	if err != nil {
		t.Errorf("Subscribe() 失敗: %v", err)
	}

	// メッセージをシミュレートしてハンドラーが呼び出されたか確認
	testPayload := []byte("テストメッセージ")
	client.SimulateMessage(topic, testPayload)

	if receivedTopic != topic {
		t.Errorf("ハンドラーが受信したトピック = %s、期待値は %s", receivedTopic, topic)
	}
	if string(receivedPayload) != string(testPayload) {
		t.Errorf("ハンドラーが受信したペイロード = %s、期待値は %s", string(receivedPayload), string(testPayload))
	}

	// サブスクライブエラーテスト
	testErr := errors.New("サブスクライブエラー")
	client.SetSubscribeError(testErr)

	err = client.Subscribe("another/topic", handler)
	if err != testErr {
		t.Errorf("Subscribe() エラー = %v、期待値は %v", err, testErr)
	}
}

func TestMockClientUnsubscribe(t *testing.T) {
	client := NewMockClient()

	// 接続されていない状態でのサブスクリプション解除テスト
	topic := "test/topic"

	err := client.Unsubscribe(topic)
	if err == nil {
		t.Error("未接続クライアントでのUnsubscribe()がエラーを返さなかった")
	}

	// 接続して最初にサブスクライブ
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() 失敗: %v", err)
	}

	var messageReceived bool
	err = client.Subscribe(topic, func(_ string, _ []byte) { messageReceived = true })
	if err != nil {
		t.Fatalf("Subscribe() 失敗: %v", err)
	}

	// サブスクリプション解除テスト
	err = client.Unsubscribe(topic)
	if err != nil {
		t.Errorf("Unsubscribe() 失敗: %v", err)
	}

	// サブスクリプション解除後にハンドラーが呼び出されないことを確認
	client.SimulateMessage(topic, []byte("テスト"))
	if messageReceived {
		t.Error("Unsubscribe()後にハンドラーが呼び出された")
	}

	// サブスクリプション解除エラーテスト
	testErr := errors.New("サブスクリプション解除エラー")
	client.SetUnsubscribeError(testErr)

	err = client.Unsubscribe("another/topic")
	if err != testErr {
		t.Errorf("Unsubscribe() エラー = %v、期待値は %v", err, testErr)
	}
}

func TestMockClientQoSAndRetained(t *testing.T) {
	client := NewMockClient()

	// デフォルト値をチェック
	if client.GetQoS() != 1 {
		t.Errorf("デフォルトQoS = %d、期待値は 1", client.GetQoS())
	}
	if client.GetRetained() {
		t.Error("デフォルトRetained = true、期待値は false")
	}

	// 値を変更してチェック
	client.SetQoS(2)
	if client.GetQoS() != 2 {
		t.Errorf("QoS = %d、期待値は 2", client.GetQoS())
	}

	client.SetRetained(true)
	if !client.GetRetained() {
		t.Error("Retained = false、期待値は true")
	}

	// Publish時に設定が使用されることを確認するには、Publish実装を更新して
	// これらの値を使用するようにすることも検討してください
}
