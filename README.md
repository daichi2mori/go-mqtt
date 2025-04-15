# MQTT アプリケーションのテストコード実装ガイド

このドキュメントでは、MQTT アプリケーションのテスト実装について説明します。テストはモックを使用したユニットテストと、実際の MQTT ブローカーに接続する統合テストの 2 種類があります。

## 1. ユニットテスト

ユニットテストは実際の MQTT ブローカーに接続せずに、モックを使用して行います。

### テスト実行方法

```bash
# すべてのテストを実行
go test ./...

# 単一のパッケージのテストを実行
go test ./handler

# 特定のテストを実行
go test -run TestMQTTPublisher_Start ./handler
```

### テストカバレッジの確認

```bash
# カバレッジレポートを生成
go test -coverprofile=coverage.out ./...

# カバレッジを確認
go tool cover -html=coverage.out -o coverage.html
```

## 2. 統合テスト

統合テストは実際の MQTT ブローカーに接続して行います。これらは環境変数によって制御されます。

### 準備

1. MQTT ブローカー（例: Mosquitto）を起動します。
2. 環境変数を設定します：

```bash
export RUN_INTEGRATION_TESTS=true
```

### テスト実行方法

```bash
# 統合テストを含むすべてのテストを実行
go test -v ./...

# 特定の統合テストを実行（_を取り除く）
go test -v -run TestIntegration ./...
```

## 3. コード改善点

テストを行いやすくするために、以下の改善を行いました：

1. **インターフェースの導入**：

   - `MQTTClient` インターフェースを作成し、実際の MQTT クライアントとモッククライアントを切り替え可能に
   - ハンドラーごとにインターフェースを定義し、テスト可能な構造に

2. **依存性の注入**：

   - MQTT クライアントをハンドラーに注入し、テスト時にモックに置き換え可能に
   - 設定（config）も注入可能に

3. **モックの実装**：

   - `MockMQTTClient` で MQTT クライアントの動作をシミュレート
   - テスト用のヘルパーメソッドを追加（例: SimulateMessage）

4. **テスト分類**：
   - ユニットテスト：モックを使用して各コンポーネントを個別にテスト
   - 統合テスト：実際の MQTT ブローカーに接続してエンドツーエンドでテスト

## 4. テストケース一覧

### Publisher テスト

- `TestMQTTPublisher_Start`: パブリッシャーの起動と実行
- `TestMQTTPublisher_PublishError`: パブリッシュエラーの処理

### Subscriber1 テスト

- `TestMQTTSubscriber1_Start`: サブスクライバー 1 の起動と実行
- `TestMQTTSubscriber1_SubscribeError`: サブスクライブエラーの処理
- `TestMQTTSubscriber1_MessageHandler_InvalidJSON`: 無効な JSON の処理

### Subscriber2 テスト

- `TestMQTTSubscriber2_Start`: サブスクライバー 2 の起動と実行
- `TestMQTTSubscriber2_MessageHandler_Type1`: タイプ 1 のメッセージ処理
- `TestMQTTSubscriber2_MessageHandler_Type2`: タイプ 2 のメッセージ処理
- `TestMQTTSubscriber2_MessageHandler_InvalidJSON`: 無効な JSON の処理
- `TestMQTTSubscriber2_MessageHandler_UnknownType`: 未知のタイプの処理
- `TestMQTTSubscriber2_SubscribeError`: サブスクライブエラーの処理

### メインフロー テスト

- `TestMainFlow`: アプリケーション全体のフロー
- `_TestIntegration`: 実際の MQTT ブローカーとの統合テスト

## 5. 今後の改善点

1. **テストカバレッジの向上**：

   - エラーケースのテストを充実させる
   - 境界値のテストを追加する

2. **パフォーマンステスト**：

   - 大量のメッセージを処理する場合のパフォーマンステスト
   - 長時間運用時の安定性テスト

3. **モックの強化**：

   - より詳細な動作検証のためのモック機能の追加
   - メソッド呼び出し順序の検証機能

4. **CI/CD 統合**：
   - GitHub アクションなどでのテスト自動化
   - Docker コンテナでの MQTT ブローカー提供による統合テスト
