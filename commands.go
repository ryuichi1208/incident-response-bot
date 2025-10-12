package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// showHelp はヘルプメッセージを表示
func showHelp(api *slack.Client, channelID string) {
	helpMessage := "📚 *インシデントレスポンスボット - ヘルプ*\n\n" +
		"*基本的な使い方:*\n" +
		"• ボットをメンションするとインシデント報告ボタンが表示されます\n" +
		"• ボタンをクリックしてインシデント情報を入力してください\n\n" +
		"*利用可能なコマンド:*\n" +
		"• `@bot help` または `@bot ヘルプ`\n" +
		"  このヘルプメッセージを表示\n\n" +
		"• `@bot handler` または `@bot ハンドラー` または `@bot 担当`\n" +
		"  このチャンネルのインシデントハンドラーを確認\n\n" +
		"• `@bot list` または `@bot 一覧` または `@bot リスト`\n" +
		"  オープン中のインシデント一覧を表示\n\n" +
		"*インシデント報告の流れ:*\n" +
		"1️⃣ ボットをメンション\n" +
		"2️⃣ 「🚨 インシデントを報告」ボタンをクリック\n" +
		"3️⃣ モーダルで詳細情報を入力\n" +
		"4️⃣ 自動的にインシデントチャンネルが作成されます\n" +
		"5️⃣ 「🙋 担当者になる」ボタンで担当者を割り当て\n\n" +
		"*機能:*\n" +
		"• インシデントチャンネルの自動作成\n" +
		"• 担当者の割り当てと管理\n" +
		"• インシデント対応ガイドラインの自動表示\n" +
		"• 全体周知チャンネルへの通知\n" +
		"• データベースでのインシデント管理"

	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(helpMessage, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", helpMessage, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("ヘルプメッセージ投稿エラー: %v", err)
	} else {
		log.Println("ヘルプメッセージを表示しました")
	}
}

// showHandler はチャンネルのハンドラー情報を表示
func showHandler(api *slack.Client, channelID string) {
	// データベースが無効な場合
	if db == nil {
		msg := "⚠️ データベース機能が無効のため、ハンドラー情報を取得できません。"
		api.PostMessage(channelID, slack.MsgOptionText(msg, false))
		return
	}

	// チャンネルのインシデント情報を取得
	query := `
		SELECT id, title, severity, handler_id, handler_name, reporter_name, created_at
		FROM incidents
		WHERE channel_id = $1 AND status = 'open'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var incidentID int64
	var title, severity, handlerID, handlerName, reporterName string
	var createdAt time.Time
	var handlerIDNull, handlerNameNull sql.NullString

	err := db.QueryRow(query, channelID).Scan(&incidentID, &title, &severity, &handlerIDNull, &handlerNameNull, &reporterName, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "ℹ️ このチャンネルにはオープンなインシデントがありません。"
			api.PostMessage(channelID, slack.MsgOptionText(msg, false))
		} else {
			log.Printf("ハンドラー情報取得エラー: %v", err)
			msg := fmt.Sprintf("❌ ハンドラー情報の取得に失敗しました: %v", err)
			api.PostMessage(channelID, slack.MsgOptionText(msg, false))
		}
		return
	}

	if handlerIDNull.Valid {
		handlerID = handlerIDNull.String
	}
	if handlerNameNull.Valid {
		handlerName = handlerNameNull.String
	}

	// 重要度に応じた絵文字
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}
	emoji := severityEmoji[severity]

	// メッセージを構築
	var message string
	if handlerID != "" {
		message = fmt.Sprintf(
			"%s *インシデント情報*\n\n"+
				"*タイトル:* %s\n"+
				"*重要度:* %s %s\n"+
				"*報告者:* %s\n"+
				"*担当者:* <@%s> (%s)\n"+
				"*作成日時:* %s",
			emoji,
			title,
			emoji,
			severity,
			reporterName,
			handlerID,
			handlerName,
			createdAt.Format("2006-01-02 15:04:05"),
		)
	} else {
		message = fmt.Sprintf(
			"%s *インシデント情報*\n\n"+
				"*タイトル:* %s\n"+
				"*重要度:* %s %s\n"+
				"*報告者:* %s\n"+
				"*担当者:* 未割り当て\n"+
				"*作成日時:* %s\n\n"+
				"💡 「🙋 担当者になる」ボタンで担当者を割り当ててください。",
			emoji,
			title,
			emoji,
			severity,
			reporterName,
			createdAt.Format("2006-01-02 15:04:05"),
		)
	}

	_, _, err = api.PostMessage(
		channelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", message, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("ハンドラー情報投稿エラー: %v", err)
	} else {
		log.Println("ハンドラー情報を表示しました")
	}
}

// showIncidentList はオープンなインシデント一覧を表示
func showIncidentList(api *slack.Client, channelID string) {
	// データベースが無効な場合
	if db == nil {
		msg := "⚠️ データベース機能が無効のため、インシデント一覧を取得できません。"
		api.PostMessage(channelID, slack.MsgOptionText(msg, false))
		return
	}

	// オープンなインシデント一覧を取得
	query := `
		SELECT id, title, severity, channel_id, channel_name, handler_name, reporter_name, created_at
		FROM incidents
		WHERE status = 'open'
		ORDER BY created_at DESC
		LIMIT 10
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("インシデント一覧取得エラー: %v", err)
		msg := fmt.Sprintf("❌ インシデント一覧の取得に失敗しました: %v", err)
		api.PostMessage(channelID, slack.MsgOptionText(msg, false))
		return
	}
	defer rows.Close()

	// 重要度に応じた絵文字
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}

	var incidents []string
	for rows.Next() {
		var id int64
		var title, severity, incidentChannelID, incidentChannelName, reporterName string
		var handlerName sql.NullString
		var createdAt time.Time

		err := rows.Scan(&id, &title, &severity, &incidentChannelID, &incidentChannelName, &handlerName, &reporterName, &createdAt)
		if err != nil {
			log.Printf("インシデント情報スキャンエラー: %v", err)
			continue
		}

		emoji := severityEmoji[severity]
		handler := "未割り当て"
		if handlerName.Valid {
			handler = handlerName.String
		}

		incident := fmt.Sprintf(
			"%s *#%d* - %s\n  チャンネル: <#%s> | 担当: %s | 報告: %s",
			emoji,
			id,
			title,
			incidentChannelID,
			handler,
			reporterName,
		)
		incidents = append(incidents, incident)
	}

	if len(incidents) == 0 {
		msg := "✅ 現在オープンなインシデントはありません。"
		api.PostMessage(channelID, slack.MsgOptionText(msg, false))
		return
	}

	message := fmt.Sprintf("📋 *オープン中のインシデント一覧* (%d件)\n\n%s", len(incidents), strings.Join(incidents, "\n\n"))

	_, _, err = api.PostMessage(
		channelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", message, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("インシデント一覧投稿エラー: %v", err)
	} else {
		log.Printf("インシデント一覧を表示しました (%d件)", len(incidents))
	}
}
