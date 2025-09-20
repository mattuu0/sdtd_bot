# 7 Days to Die Discord Bot

7 Days to Die サーバーを自動管理する Discord Bot です。サーバーの監視、自動起動・停止、プレイヤー情報の表示などを行い、効率的なサーバー運用を支援します。

## 🚀 主な機能

### サーバー管理
- **自動起動**: `/start` コマンドでサーバー起動
- **自動停止**: プレイヤー0人が70秒継続で自動停止
- **監視機能**: 10秒間隔でサーバー状態を監視

### Discord 連携
- **リアルタイム更新**: サーバー情報とプレイヤー数をリアルタイム表示
- **通知機能**: 起動・停止・警告をDiscordで通知
- **接続情報表示**: IP、ポート、パスワードを自動表示

### 自動停止機能
- プレイヤー0人の状態が70秒継続で自動停止
- 50秒経過時点で警告メッセージ
- 起動後5分間は自動停止を無効化


## 🛠 セットアップ

### 必要なソフトウェア

1. **Go 1.21以上**
   ```bash
   # https://golang.org/dl/ からダウンロード
   ```

2. **Node.js と gamedig**
   ```bash
   npm install -g gamedig
   ```

3. **7DTD サーバー管理スクリプト**
   ```bash
   # ./sdtdserver スクリプトを実行可能にする
   chmod +x ./sdtdserver
   ```

### Discord Bot の作成

1. [Discord Developer Portal](https://discord.com/developers/applications) にアクセス
2. "New Application" をクリックして新しいアプリケーションを作成
3. "Bot" タブに移動して "Add Bot" をクリック
4. "Token" をコピーして保存

### Bot の招待

1. Developer Portal の "OAuth2" → "URL Generator" に移動
2. **SCOPES**: `bot`, `applications.commands` を選択
3. **BOT PERMISSIONS**: 
   - ✅ Send Messages
   - ✅ Use Slash Commands  
   - ✅ Manage Messages
4. 生成されたURLでBotをDiscordサーバーに招待

### 設定ファイル

```bash
# .env ファイルを作成
cp .env.example .env
```

`.env` の設定例：
```env
# 必須項目
DISCORD_BOT_TOKEN=your_discord_bot_token_here
CHANNEL_ID=your_channel_id_here

# オプション項目
SERVER_IP=127.0.0.1
SERVER_PORT=26900
SERVER_PASSWORD=your_server_password
```

### 実行

```bash
# 依存関係のダウンロード
go mod tidy

# 実行
go run main.go

# または、ビルドしてから実行
go build -o 7dtd-bot main.go
./7dtd-bot
```

## 📖 使用方法

### コマンド

| コマンド | 説明 |
|---------|------|
| `/start` | サーバーを起動します |

### メッセージの種類

#### 1. ステータスメッセージ（10秒間隔で更新）
```
オンライン: 2人
ping: 17ms
バージョン: 01.02.03
```

#### 2. 起動完了メッセージ（動的更新）
```
🟢 サーバーの起動が完了しました
```
IP: 127.0.0.1
ポート: 26900
パスワード: your_server_password
```
⏰ 245秒以内に参加してください
👥 現在のプレイヤー数: 0人
```

#### 3. 警告メッセージ
```
⚠️ あと20秒以内に誰も参加しない場合にはサーバーを停止します
```

## ⚙️ 技術仕様

### 監視間隔
- **通常監視**: 10秒間隔
- **プレイヤー参加待機**: 3秒間隔

### 自動停止設定
```go
MAX_EMPTY_CHECKS = 7        // 自動停止までのチェック回数（70秒）
WARNING_CHECK = 5           // 警告メッセージ送信タイミング（50秒）
STARTUP_GRACE_PERIOD = 5分  // 起動後の猶予期間
```

### ファイル構成
```
7dtd-discord-bot/
├── main.go              # メインプログラム
├── go.mod               # Go モジュール設定
├── .env                 # 環境変数設定
├── .env.example         # 環境変数テンプレート
├── message_ids.json     # メッセージID保存（自動生成）
└── README.md           # このファイル
```

## 🔧 トラブルシューティング

### よくある問題

#### Bot が応答しない
- ✅ Discord Bot トークンが正しく設定されているか確認
- ✅ Bot に必要な権限（Send Messages, Use Slash Commands, Manage Messages）が付与されているか確認
- ✅ チャンネルIDが正しく設定されているか確認

#### gamedig コマンドが見つからない
```bash
# Node.js と gamedig を再インストール
npm install -g gamedig

# インストール確認
gamedig --help
```

#### サーバー起動・停止に失敗
- ✅ `./sdtdserver` スクリプトが存在し、実行可能か確認
- ✅ サーバーの状態を手動で確認
```bash
# スクリプトの実行権限を付与
chmod +x ./sdtdserver

# 手動でサーバー起動テスト
./sdtdserver start
./sdtdserver stop
```

#### メッセージが更新されない
- ✅ Bot にメッセージ送信・編集権限があるか確認
- ✅ `message_ids.json` ファイルの権限を確認
- ✅ Discord API の制限に引っかかっていないか確認

### ログの確認

実行時のログで状況を確認できます：

```bash
# 正常ログ例
Bot が起動しました。Ctrl+C で終了します。
サーバーの起動が完了しました
新しいステータスメッセージを作成しました: 1234567890
最初のプレイヤーが参加しました。自動停止監視を開始します。

# エラーログ例
gamedig実行エラー: exit status 1
メッセージ更新エラー: HTTP 404 Not Found
サーバー停止エラー: permission denied
```

## 📊 設定項目詳細

### 必須設定
| 項目 | 説明 |
|------|------|
| `DISCORD_BOT_TOKEN` | Discord Bot のトークン |
| `CHANNEL_ID` | 通知を送信するDiscordチャンネルのID |

### オプション設定
| 項目 | デフォルト値 | 説明 |
|------|-------------|------|
| `SERVER_IP` | `127.0.0.1` | サーバーのIPアドレス |
| `SERVER_PORT` | `26900` | サーバーのポート |
| `SERVER_PASSWORD` | (なし) | サーバーのパスワード |

## 🛡 セキュリティ

### 権限管理
- Bot には必要最小限のDiscord権限のみ付与
- メッセージ送信・編集・スラッシュコマンド権限のみ

### 情報保護
- サーバーパスワードは環境変数で管理
- 機密情報はログに出力されません
- スラッシュコマンドは適切なチャンネルでのみ動作

## 🚀 停止方法

Bot の停止は `Ctrl+C` で安全に終了できます。

```bash
# 実行中の Bot を停止
Ctrl+C
```

## 📝 ライセンス

このプロジェクトはMITライセンスの下で公開されています。

## 🤝 貢献

Issues や Pull Requests を歓迎します！

---

**更新日**: 2025年9月20日  
**バージョン**: 1.0.0