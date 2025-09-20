package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// 定数定義
const (
	MAX_EMPTY_CHECKS      = 7
	WARNING_CHECK         = 5
	CHECK_INTERVAL        = 10 * time.Second
	STARTUP_GRACE_PERIOD  = 5 * time.Minute
	WAIT_PLAYER_INTERVAL  = 3 * time.Second
	SERVER_START_TIMEOUT  = 5 * time.Minute
)

// サーバー情報構造体
type ServerStatus struct {
	Name       string `json:"name"`
	Map        string `json:"map"`
	NumPlayers int    `json:"numplayers"`
	Ping       int    `json:"ping"`
	Version    string `json:"version"`
	Error      string `json:"error,omitempty"`
}

// メッセージID管理
type MessageIDs struct {
	StatusMessageID   string `json:"status_message_id"`
	StartupMessageID  string `json:"startup_message_id"`
}

// Bot構造体
type Bot struct {
	session         *discordgo.Session
	channelID       string
	serverIP        string
	serverPort      string
	serverPassword  string
	messageIDs      MessageIDs
	emptyCheckCount int
	gracePeriodEnd  *time.Time
	isMonitoring    bool
	isWaitingPlayer bool
}

func main() {
	// 環境変数読み込み
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	channelID := os.Getenv("CHANNEL_ID")
	
	if token == "" || channelID == "" {
		log.Fatal("DISCORD_BOT_TOKEN と CHANNEL_ID は必須です")
	}

	// Bot初期化
	bot := &Bot{
		channelID:      channelID,
		serverIP:       getEnvOrDefault("SERVER_IP", "127.0.0.1"),
		serverPort:     getEnvOrDefault("SERVER_PORT", "26900"),
		serverPassword: getEnvOrDefault("SERVER_PASSWORD", ""),
	}

	// メッセージID読み込み
	bot.loadMessageIDs()

	// Discord セッション作成
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Discord セッション作成エラー:", err)
	}
	bot.session = dg

	// インテント設定
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// スラッシュコマンド設定
	dg.AddHandler(bot.handleSlashCommand)
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot が起動しました。Ctrl+C で終了します。")
	})

	// Discord接続
	err = dg.Open()
	if err != nil {
		log.Fatal("Discord接続エラー:", err)
	}
	defer dg.Close()

	// スラッシュコマンド登録
	_, err = dg.ApplicationCommandCreate(dg.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "start",
		Description: "7DTDサーバーを起動します",
	})
	if err != nil {
		log.Fatal("スラッシュコマンド登録エラー:", err)
	}

	// 定期監視開始
	bot.startMonitoring()

	// シグナル待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

// 環境変数取得（デフォルト値付き）
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// メッセージID読み込み
func (b *Bot) loadMessageIDs() {
	data, err := os.ReadFile("message_ids.json")
	if err != nil {
		log.Println("message_ids.json が見つかりません。新規作成します。")
		return
	}
	
	err = json.Unmarshal(data, &b.messageIDs)
	if err != nil {
		log.Println("message_ids.json の読み込みに失敗しました:", err)
	}
}

// メッセージID保存
func (b *Bot) saveMessageIDs() {
	data, err := json.Marshal(b.messageIDs)
	if err != nil {
		log.Println("メッセージID保存エラー:", err)
		return
	}
	
	err = os.WriteFile("message_ids.json", data, 0644)
	if err != nil {
		log.Println("メッセージID保存エラー:", err)
	}
}

// スラッシュコマンドハンドラ
func (b *Bot) handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name == "start" {
		b.handleStartCommand(s, i)
	}
}

// サーバー起動コマンド処理
func (b *Bot) handleStartCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 応答
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "サーバーを起動しています...",
		},
	})
	if err != nil {
		log.Println("コマンド応答エラー:", err)
		return
	}

	// サーバー起動
	go b.startServer()
}

// サーバー起動処理
func (b *Bot) startServer() {
	log.Println("サーバー起動コマンドを実行します")
	
	// サーバー起動コマンド実行
	cmd := exec.Command("./sdtdserver", "start")
	err := cmd.Run()
	if err != nil {
		msg := fmt.Sprintf("❌ サーバー起動に失敗しました: %v", err)
		b.sendMessage(msg)
		return
	}

	// サーバー起動確認（最大5分間）
	timeout := time.Now().Add(SERVER_START_TIMEOUT)
	for time.Now().Before(timeout) {
		status := b.getServerStatus()
		if status.Error == "" {
			log.Println("サーバーの起動が完了しました")
			b.onServerStarted()
			return
		}
		time.Sleep(5 * time.Second)
	}

	b.sendMessage("❌ サーバー起動がタイムアウトしました")
}

// サーバー起動完了時の処理
func (b *Bot) onServerStarted() {
	// 起動完了メッセージ送信
	msg := b.createStartupMessage(0, 300)
	messageRef, err := b.sendMessage(msg)
	if err != nil {
		log.Println("起動完了メッセージ送信エラー:", err)
		return
	}

	// メッセージID保存
	b.messageIDs.StartupMessageID = messageRef.ID
	b.saveMessageIDs()

	// プレイヤー参加待機開始
	b.isWaitingPlayer = true
	go b.waitForFirstPlayer()
}

// 最初のプレイヤー参加待機
func (b *Bot) waitForFirstPlayer() {
	countdown := 300 // 5分 = 300秒
	ticker := time.NewTicker(WAIT_PLAYER_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !b.isWaitingPlayer {
				return
			}

			status := b.getServerStatus()
			if status.Error != "" {
				// サーバーが停止した場合
				b.isWaitingPlayer = false
				return
			}

			if status.NumPlayers > 0 {
				// プレイヤーが参加した
				b.onFirstPlayerJoined()
				return
			}

			// カウントダウン更新
			countdown -= int(WAIT_PLAYER_INTERVAL.Seconds())
			if countdown <= 0 {
				// タイムアウト（通常は自動停止ループで処理されるが、念のため）
				b.isWaitingPlayer = false
				return
			}

			// メッセージ更新
			msg := b.createStartupMessage(status.NumPlayers, countdown)
			b.updateStartupMessage(msg)
		}
	}
}

// 最初のプレイヤー参加時の処理
func (b *Bot) onFirstPlayerJoined() {
	b.isWaitingPlayer = false
	
	status := b.getServerStatus()
	
	// プレイヤー参加メッセージ
	msg := b.createStartupMessage(status.NumPlayers, 0)
	msg += "\n\n✅ プレイヤーが参加しました！"
	b.updateStartupMessage(msg)

	// 自動停止警告メッセージ
	warningMsg := "ℹ️ プレイヤーが0人の状態が70秒続くとサーバーが自動停止します"
	b.sendMessage(warningMsg)

	log.Println("最初のプレイヤーが参加しました。自動停止監視を開始します。")
}

// 起動メッセージ作成
func (b *Bot) createStartupMessage(playerCount, countdown int) string {
	msg := "🟢 サーバーの起動が完了しました\n```\n"
	msg += fmt.Sprintf("IP: %s\n", b.serverIP)
	msg += fmt.Sprintf("ポート: %s\n", b.serverPort)
	if b.serverPassword != "" {
		msg += fmt.Sprintf("パスワード: %s\n", b.serverPassword)
	}
	msg += "```\n"

	if countdown > 0 && !b.isWaitingPlayer {
		// 猶予期間中
		msg += fmt.Sprintf("⏰ %d秒以内に参加してください\n", countdown)
	} else if countdown > 0 && b.isWaitingPlayer {
		// プレイヤー待機中
		msg += fmt.Sprintf("⏰ %d秒以内に参加してください\n", countdown)
	}
	
	msg += fmt.Sprintf("👥 現在のプレイヤー数: %d人", playerCount)
	
	return msg
}

// 定期監視開始
func (b *Bot) startMonitoring() {
	b.isMonitoring = true
	go func() {
		ticker := time.NewTicker(CHECK_INTERVAL)
		defer ticker.Stop()

		for b.isMonitoring {
			select {
			case <-ticker.C:
				b.checkServerStatus()
			}
		}
	}()
}

// サーバー状態確認
func (b *Bot) checkServerStatus() {
	status := b.getServerStatus()
	
	// ステータスメッセージ更新
	b.updateStatusMessage(status)

	if status.Error != "" {
		// サーバー停止中
		b.emptyCheckCount = 0
		b.gracePeriodEnd = nil
		b.messageIDs.StartupMessageID = ""
		b.saveMessageIDs()
		return
	}

	// 猶予期間チェック
	if b.gracePeriodEnd != nil && time.Now().Before(*b.gracePeriodEnd) {
		log.Printf("起動後猶予期間中 (残り: %.0f秒)", time.Until(*b.gracePeriodEnd).Seconds())
		return
	}

	// プレイヤー待機中は自動停止チェックしない
	if b.isWaitingPlayer {
		return
	}

	// 自動停止チェック
	if status.NumPlayers == 0 {
		b.emptyCheckCount++
		log.Printf("プレイヤー0人の状態: %d/%d", b.emptyCheckCount, MAX_EMPTY_CHECKS)

		if b.emptyCheckCount == WARNING_CHECK {
			// 警告送信
			warningMsg := fmt.Sprintf("⚠️ あと%d秒以内に誰も参加しない場合にはサーバーを停止します", 
				(MAX_EMPTY_CHECKS-WARNING_CHECK)*int(CHECK_INTERVAL.Seconds()))
			b.sendMessage(warningMsg)
			
			// 起動メッセージ更新
			if b.messageIDs.StartupMessageID != "" {
				msg := b.createStartupMessage(0, 0)
				msg += fmt.Sprintf("\n\n⚠️ あと%d秒以内にサーバーに誰も参加しない場合はサーバーを自動停止します",
					(MAX_EMPTY_CHECKS-WARNING_CHECK)*int(CHECK_INTERVAL.Seconds()))
				b.updateStartupMessage(msg)
			}
		} else if b.emptyCheckCount >= MAX_EMPTY_CHECKS {
			// 自動停止実行
			b.autoStopServer()
		}
	} else {
		// プレイヤーが参加している場合
		if b.emptyCheckCount > 0 {
			b.emptyCheckCount = 0
			log.Println("プレイヤーが参加したため、自動停止カウンタをリセットしました")
			
			// 起動メッセージ更新（プレイヤー参加表示）
			if b.messageIDs.StartupMessageID != "" {
				msg := b.createStartupMessage(status.NumPlayers, 0)
				msg += "\n\n✅ プレイヤーが参加しました！"
				b.updateStartupMessage(msg)
			}
		}
	}
}

// サーバー状態取得
func (b *Bot) getServerStatus() ServerStatus {
	cmd := exec.Command("gamedig", "--type", "protocol-valve", "--host", b.serverIP, "--port", b.serverPort)
	output, err := cmd.Output()
	
	var status ServerStatus
	if err != nil {
		status.Error = "Failed all attempts"
		return status
	}

	err = json.Unmarshal(output, &status)
	if err != nil {
		status.Error = "Parse error"
		return status
	}

	return status
}

// ステータスメッセージ更新
func (b *Bot) updateStatusMessage(status ServerStatus) {
	var content string
	
	if status.Error != "" {
		if strings.Contains(status.Error, "Failed") {
			content = "サーバー: オフライン\nステータス: 停止中"
		} else {
			content = "サーバー: エラー\nステータス: 状態確認失敗"
		}
	} else {
		content = fmt.Sprintf("オンライン: %d人\nping: %dms\nバージョン: %s", 
			status.NumPlayers, status.Ping, status.Version)
	}

	if b.messageIDs.StatusMessageID == "" {
		// 新規メッセージ作成
		messageRef, err := b.sendMessage(content)
		if err == nil {
			b.messageIDs.StatusMessageID = messageRef.ID
			b.saveMessageIDs()
			log.Println("新しいステータスメッセージを作成しました:", messageRef.ID)
		}
	} else {
		// 既存メッセージ更新
		b.updateMessage(b.messageIDs.StatusMessageID, content)
	}
}

// 起動メッセージ更新
func (b *Bot) updateStartupMessage(content string) {
	if b.messageIDs.StartupMessageID != "" {
		b.updateMessage(b.messageIDs.StartupMessageID, content)
	}
}

// 自動停止実行
func (b *Bot) autoStopServer() {
	log.Println("プレイヤーが存在しないため、サーバーを自動停止します")
	
	b.sendMessage("🔴 プレイヤーが存在しないためサーバーを自動停止します")
	
	cmd := exec.Command("./sdtdserver", "stop")
	err := cmd.Run()
	
	if err != nil {
		msg := fmt.Sprintf("❌ サーバー停止に失敗しました: %v", err)
		b.sendMessage(msg)
		log.Println("サーバー停止エラー:", err)
	} else {
		b.sendMessage("✅ サーバーが正常に停止しました")
		log.Println("サーバーが正常に停止しました")
	}
	
	b.emptyCheckCount = 0
	b.gracePeriodEnd = nil
	b.messageIDs.StartupMessageID = ""
	b.saveMessageIDs()
}

// メッセージ送信
func (b *Bot) sendMessage(content string) (*discordgo.Message, error) {
	return b.session.ChannelMessageSend(b.channelID, content)
}

// メッセージ更新
func (b *Bot) updateMessage(messageID, content string) error {
	_, err := b.session.ChannelMessageEdit(b.channelID, messageID, content)
	if err != nil {
		log.Printf("メッセージ更新エラー: %v", err)
		// メッセージが存在しない場合は新規作成
		if strings.Contains(err.Error(), "404") {
			messageRef, newErr := b.sendMessage(content)
			if newErr == nil {
				// メッセージIDを更新
				if messageID == b.messageIDs.StatusMessageID {
					b.messageIDs.StatusMessageID = messageRef.ID
				} else if messageID == b.messageIDs.StartupMessageID {
					b.messageIDs.StartupMessageID = messageRef.ID
				}
				b.saveMessageIDs()
			}
		}
	}
	return err
}