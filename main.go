package main

import (
	"log"
	"os"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	// 設定ファイルの読み込み
	if err := loadConfig("config.toml"); err != nil {
		log.Printf("設定ファイル読み込みエラー: %v", err)
		log.Println("環境変数からの読み込みを試みます")
	}

	// トークンの取得（設定ファイル優先、環境変数をフォールバック）
	botToken := config.Slack.BotToken
	appToken := config.Slack.AppToken

	if botToken == "" {
		botToken = os.Getenv("SLACK_BOT_TOKEN")
	}
	if appToken == "" {
		appToken = os.Getenv("SLACK_APP_TOKEN")
	}

	if botToken == "" || appToken == "" {
		log.Fatal("SLACK_BOT_TOKEN と SLACK_APP_TOKEN の設定が必要です（config.tomlまたは環境変数）")
	}

	// データベース接続を初期化
	if err := initDB(); err != nil {
		log.Printf("データベース接続エラー: %v", err)
		log.Println("データベース機能は無効化されます")
	} else {
		defer db.Close()
		log.Println("データベースに接続しました")
	}

	// Slack APIクライアントの作成
	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)

	// オープンなインシデントのタイムキーパーを復元
	if db != nil {
		openIncidents, err := getOpenIncidents()
		if err != nil {
			log.Printf("オープンなインシデント取得エラー: %v", err)
		} else if len(openIncidents) > 0 {
			log.Printf("オープンなインシデント %d 件のタイムキーパーを復元します", len(openIncidents))
			for _, incident := range openIncidents {
				incidentID := incident["id"].(int64)
				channelID := incident["channel_id"].(string)
				createdAt := incident["created_at"].(time.Time)

				timekeeperManager.startTimekeeper(api, incidentID, channelID, createdAt)
				log.Printf("インシデント %d のタイムキーパーを復元しました (開始時刻: %v)", incidentID, createdAt)
			}
		} else {
			log.Println("復元するオープンなインシデントはありません")
		}
	}

	// Socket Modeクライアントの作成
	client := socketmode.New(
		api,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	// Bot自身のユーザーIDを取得
	authTest, err := api.AuthTest()
	if err != nil {
		log.Fatalf("認証エラー: %v", err)
	}
	botUserID := authTest.UserID

	log.Printf("Botが起動しました。Bot ID: %s", botUserID)

	// 設定情報をログ出力
	log.Printf("全体周知機能: %v", config.Channels.EnableAnnouncement)
	log.Printf("全体周知チャンネル数: %d", len(config.Channels.AnnouncementChannels))
	if len(config.Channels.AnnouncementChannels) > 0 {
		log.Printf("全体周知チャンネル: %v", config.Channels.AnnouncementChannels)
	}

	// イベントハンドラの設定
	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				log.Println("イベントを受信しました")
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					log.Printf("イベントの型変換に失敗しました")
					continue
				}

				// イベントを確認応答
				client.Ack(*evt.Request)

				// イベントタイプに応じた処理
				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						// メンション受信時の処理
						handleAppMention(api, ev)
					case *slackevents.ChannelArchiveEvent:
						// チャンネルアーカイブ時の処理
						handleChannelArchive(api, ev)
					}
				}

			case socketmode.EventTypeInteractive:
				// モーダル送信などのインタラクティブイベント
				callback, ok := evt.Data.(slack.InteractionCallback)
				if !ok {
					log.Printf("インタラクティブイベントの型変換に失敗しました")
					continue
				}

				// イベントを確認応答
				client.Ack(*evt.Request)

				// インタラクションタイプに応じた処理
				switch callback.Type {
				case slack.InteractionTypeBlockActions:
					// ボタンクリック時の処理
					if len(callback.ActionCallback.BlockActions) > 0 {
						action := callback.ActionCallback.BlockActions[0]
						switch action.ActionID {
						case "open_incident_modal":
							handleOpenModal(api, callback)
						case "assign_handler":
							handleAssignHandler(api, callback)
						case "update_incident":
							handleUpdateIncident(api, callback)
						case "resolve_incident":
							handleResolveIncident(api, callback)
						case "stop_timekeeper":
							handleStopTimekeeper(api, callback)
						}
					}
				case slack.InteractionTypeViewSubmission:
					// モーダル送信時の処理
					if callback.View.CallbackID == "incident_report_modal" {
						handleModalSubmission(api, callback)
					} else if callback.View.CallbackID == "incident_update_modal" {
						handleUpdateModalSubmission(api, callback)
					}
				}

			case socketmode.EventTypeConnecting:
				log.Println("Slackに接続中...")

			case socketmode.EventTypeConnectionError:
				log.Println("接続エラーが発生しました")

			case socketmode.EventTypeConnected:
				log.Println("Slackに接続しました")
			}
		}
	}()

	// Socket Modeを開始
	if err := client.Run(); err != nil {
		log.Fatalf("Socket Modeの実行エラー: %v", err)
	}
}
