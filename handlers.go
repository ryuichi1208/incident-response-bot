package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// handleAppMention はメンション受信時の処理（ボタンを表示）
func handleAppMention(api *slack.Client, event *slackevents.AppMentionEvent) {
	log.Printf("メンションを受信しました: %s", event.Text)

	// コマンドを解析
	text := strings.ToLower(strings.TrimSpace(event.Text))

	// helpコマンドの処理
	if strings.Contains(text, "help") || strings.Contains(text, "ヘルプ") {
		showHelp(api, event.Channel)
		return
	}

	// handlerコマンドの処理（チャンネルのハンドラー情報を表示）
	if strings.Contains(text, "handler") || strings.Contains(text, "ハンドラー") || strings.Contains(text, "担当") {
		showHandler(api, event.Channel)
		return
	}

	// listコマンドの処理（オープンなインシデント一覧）
	if strings.Contains(text, "list") || strings.Contains(text, "一覧") || strings.Contains(text, "リスト") {
		showIncidentList(api, event.Channel)
		return
	}

	// チャンネル情報を取得してインシデントチャンネルかどうかを判定
	channel, err := api.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID: event.Channel,
	})
	if err != nil {
		log.Printf("チャンネル情報取得エラー: %v", err)
	}

	// インシデントチャンネル（incident-で始まる）の場合はhelpを表示
	if channel != nil && strings.HasPrefix(channel.Name, "incident-") {
		showHelp(api, event.Channel)
		return
	}

	// デフォルト: インシデント報告ボタンを表示
	button := slack.NewButtonBlockElement(
		"open_incident_modal",
		"open_modal",
		slack.NewTextBlockObject("plain_text", "🚨 インシデントを報告", true, false),
	)
	button.Style = slack.StyleDanger

	actionBlock := slack.NewActionBlock(
		"incident_report_action",
		button,
	)

	headerText := slack.NewTextBlockObject("mrkdwn", "インシデントを報告するには、下のボタンをクリックしてください。", false, false)
	headerBlock := slack.NewSectionBlock(headerText, nil, nil)

	// メッセージを送信
	_, _, err = api.PostMessage(
		event.Channel,
		slack.MsgOptionBlocks(headerBlock, actionBlock),
	)

	if err != nil {
		log.Printf("メッセージ送信エラー: %v", err)
		return
	}

	log.Println("インシデント報告ボタンを表示しました")
}

// handleOpenModal はボタンクリック時にモーダルを開く
func handleOpenModal(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("モーダル表示ボタンがクリックされました")

	// ユーザー情報を取得してDisplay Nameを取得
	user, err := api.GetUserInfo(callback.User.ID)
	if err != nil {
		log.Printf("ユーザー情報取得エラー: %v", err)
		return
	}

	// Display Nameを取得（Profile.DisplayNameが空の場合はRealNameを使用）
	displayName := user.Profile.DisplayName
	if displayName == "" {
		displayName = user.RealName
	}
	if displayName == "" {
		displayName = user.Name
	}

	// 「入力中です」メッセージを投稿
	typingMessage := fmt.Sprintf("✍️ %sさんがインシデント報告を入力中です...", displayName)
	_, _, err = api.PostMessage(
		callback.Channel.ID,
		slack.MsgOptionText(typingMessage, false),
	)
	if err != nil {
		log.Printf("入力中メッセージの投稿エラー: %v", err)
	}

	// インシデント報告用のモーダルを作成
	modalView := createIncidentModal(callback.Channel.ID)

	// モーダルを開く（trigger IDを使用）
	_, err = api.OpenView(callback.TriggerID, modalView)
	if err != nil {
		log.Printf("モーダル表示エラー: %v", err)
		return
	}

	log.Println("インシデント報告モーダルを表示しました")
}

// handleAssignHandler はインシデントハンドラー割り当て/更新ボタンがクリックされた時の処理（冪等）
func handleAssignHandler(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("インシデントハンドラー割り当て/更新ボタンがクリックされました")

	// ボタンのValueからインシデントIDを取得
	action := callback.ActionCallback.BlockActions[0]
	var incidentID int64
	_, err := fmt.Sscanf(action.Value, "incident_%d", &incidentID)
	if err != nil {
		log.Printf("インシデントID解析エラー: %v", err)
		return
	}

	// ユーザー情報を取得
	user, err := api.GetUserInfo(callback.User.ID)
	if err != nil {
		log.Printf("ユーザー情報取得エラー: %v", err)
		return
	}

	handlerName := user.RealName
	if handlerName == "" {
		handlerName = user.Name
	}

	// ハンドラーを割り当て/更新（冪等操作）
	err = changeHandler(incidentID, callback.User.ID, handlerName, callback.User.ID)
	if err != nil {
		log.Printf("ハンドラー割り当てエラー: %v", err)
		// エラーメッセージを投稿
		api.PostEphemeral(
			callback.Channel.ID,
			callback.User.ID,
			slack.MsgOptionText(fmt.Sprintf("❌ ハンドラー割り当てに失敗しました: %v", err), false),
		)
		return
	}

	// 成功メッセージを投稿
	successMessage := fmt.Sprintf("✅ <@%s> さんがこのインシデントの担当者になりました！", callback.User.ID)
	_, _, err = api.PostMessage(
		callback.Channel.ID,
		slack.MsgOptionText(successMessage, false),
	)

	if err != nil {
		log.Printf("成功メッセージ投稿エラー: %v", err)
	} else {
		log.Printf("インシデント %d のハンドラーを %s に設定しました", incidentID, handlerName)
	}
}

// postHandlerButton はインシデントハンドラー割り当てボタンを投稿
func postHandlerButton(api *slack.Client, channelID string, incidentID int64) {
	// ボタンを作成（冪等：何回でも押せる）
	assignButton := slack.NewButtonBlockElement(
		"assign_handler",
		fmt.Sprintf("incident_%d", incidentID),
		slack.NewTextBlockObject("plain_text", "🙋 担当者になる", true, false),
	)
	assignButton.Style = slack.StylePrimary

	actionBlock := slack.NewActionBlock(
		fmt.Sprintf("handler_action_%d", incidentID),
		assignButton,
	)

	headerText := slack.NewTextBlockObject("mrkdwn", "このインシデントの担当者を設定してください（何回でも変更可能）", false, false)
	headerBlock := slack.NewSectionBlock(headerText, nil, nil)

	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionBlocks(headerBlock, actionBlock),
	)

	if err != nil {
		log.Printf("ハンドラーボタン投稿エラー: %v", err)
	} else {
		log.Println("インシデントハンドラーボタンを投稿しました")
	}
}

// postIncidentActionsButton はインシデント操作ボタンを投稿
func postIncidentActionsButton(api *slack.Client, channelID string, incidentID int64) {
	// 更新ボタン
	updateButton := slack.NewButtonBlockElement(
		"update_incident",
		fmt.Sprintf("incident_%d", incidentID),
		slack.NewTextBlockObject("plain_text", "📝 詳細を更新", true, false),
	)
	updateButton.Style = slack.StylePrimary

	actionBlock := slack.NewActionBlock(
		fmt.Sprintf("incident_actions_%d", incidentID),
		updateButton,
	)

	headerText := slack.NewTextBlockObject("mrkdwn", "インシデント情報を管理:", false, false)
	headerBlock := slack.NewSectionBlock(headerText, nil, nil)

	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionBlocks(headerBlock, actionBlock),
	)

	if err != nil {
		log.Printf("インシデント操作ボタン投稿エラー: %v", err)
	} else {
		log.Println("インシデント操作ボタンを投稿しました")
	}
}

// handleUpdateIncident はインシデント更新ボタンがクリックされた時の処理
func handleUpdateIncident(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("インシデント更新ボタンがクリックされました")

	// ボタンのValueからインシデントIDを取得
	action := callback.ActionCallback.BlockActions[0]
	var incidentID int64
	_, err := fmt.Sscanf(action.Value, "incident_%d", &incidentID)
	if err != nil {
		log.Printf("インシデントID解析エラー: %v", err)
		return
	}

	// 現在のインシデント詳細を取得
	details, err := getIncidentDetails(incidentID)
	if err != nil {
		log.Printf("インシデント詳細取得エラー: %v", err)
		api.PostEphemeral(
			callback.Channel.ID,
			callback.User.ID,
			slack.MsgOptionText(fmt.Sprintf("❌ インシデント情報の取得に失敗しました: %v", err), false),
		)
		return
	}

	// 更新用モーダルを作成
	modalView := createUpdateIncidentModal(incidentID, details)

	// モーダルを開く
	_, err = api.OpenView(callback.TriggerID, modalView)
	if err != nil {
		log.Printf("更新モーダル表示エラー: %v", err)
		return
	}

	log.Println("インシデント更新モーダルを表示しました")
}

// postToAnnouncementChannels は全体周知チャンネルにメッセージを投稿
func postToAnnouncementChannels(api *slack.Client, message string, incidentChannelID string) {
	for _, channelID := range config.Channels.AnnouncementChannels {
		if channelID == "" {
			continue
		}

		log.Printf("全体周知チャンネル %s に投稿中...", channelID)

		// インシデントチャンネルのリンクを追加
		announcementMessage := message
		if incidentChannelID != "" {
			announcementMessage = fmt.Sprintf("%s\n\n📋 *対応チャンネル:* <#%s>", message, incidentChannelID)
		}

		_, _, err := api.PostMessage(
			channelID,
			slack.MsgOptionText(announcementMessage, false),
			slack.MsgOptionBlocks(
				slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", announcementMessage, false, false),
					nil, nil,
				),
			),
		)

		if err != nil {
			log.Printf("全体周知チャンネル %s への投稿エラー: %v", channelID, err)
		} else {
			log.Printf("全体周知チャンネル %s に投稿しました", channelID)
		}
	}
}
