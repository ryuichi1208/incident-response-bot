package main

import (
	"fmt"
	"log"
	"strings"

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

	// インシデントチャンネル（incident-で始まる）の場合は操作ボタンを表示
	if channel != nil && strings.HasPrefix(channel.Name, "incident-") {
		// インシデントIDを取得
		incidentID, _, err := getIncidentByChannelID(event.Channel)
		if err != nil {
			log.Printf("インシデント取得エラー: %v", err)
			showHelp(api, event.Channel)
			return
		}

		// ハンドラーボタンを表示
		postHandlerButton(api, event.Channel, incidentID)

		// インシデント操作ボタンを表示
		postIncidentActionsButton(api, event.Channel, incidentID)

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

	// 復旧ボタン
	resolveButton := slack.NewButtonBlockElement(
		"resolve_incident",
		fmt.Sprintf("incident_%d", incidentID),
		slack.NewTextBlockObject("plain_text", "✅ 復旧完了", true, false),
	)
	resolveButton.Style = "primary"
	resolveButton.Confirm = &slack.ConfirmationBlockObject{
		Title:   slack.NewTextBlockObject("plain_text", "復旧完了の確認", false, false),
		Text:    slack.NewTextBlockObject("mrkdwn", "このインシデントを復旧済みにしますか？\n復旧通知が全体周知チャンネルに送信されます。", false, false),
		Confirm: slack.NewTextBlockObject("plain_text", "復旧完了", false, false),
		Deny:    slack.NewTextBlockObject("plain_text", "キャンセル", false, false),
	}

	// タイムキーパー停止ボタン
	stopTimekeeperButton := slack.NewButtonBlockElement(
		"stop_timekeeper",
		fmt.Sprintf("incident_%d", incidentID),
		slack.NewTextBlockObject("plain_text", "⏹️ タイムキーパーを止める", true, false),
	)
	stopTimekeeperButton.Style = "danger"

	actionBlock := slack.NewActionBlock(
		fmt.Sprintf("incident_actions_%d", incidentID),
		updateButton,
		resolveButton,
		stopTimekeeperButton,
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

// handleResolveIncident はインシデント復旧ボタンがクリックされた時の処理
func handleResolveIncident(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("インシデント復旧ボタンがクリックされました")

	// ボタンのValueからインシデントIDを取得
	action := callback.ActionCallback.BlockActions[0]
	var incidentID int64
	_, err := fmt.Sscanf(action.Value, "incident_%d", &incidentID)
	if err != nil {
		log.Printf("インシデントID解析エラー: %v", err)
		return
	}

	// インシデント詳細を取得
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

	// ユーザー情報を取得
	user, err := api.GetUserInfo(callback.User.ID)
	resolvedByName := callback.User.Name
	if err == nil && user.RealName != "" {
		resolvedByName = user.RealName
	}

	// インシデントを復旧済みにする
	err = resolveIncident(incidentID, callback.User.ID, resolvedByName)
	if err != nil {
		log.Printf("インシデント復旧エラー: %v", err)
		api.PostEphemeral(
			callback.Channel.ID,
			callback.User.ID,
			slack.MsgOptionText(fmt.Sprintf("❌ インシデントの復旧に失敗しました: %v", err), false),
		)
		return
	}

	// 重要度に応じた絵文字
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}
	emoji := severityEmoji[details["severity"].(string)]

	// チャンネルメンバーを取得（対応メンバー一覧）
	contributors, err := getChannelContributors(api, callback.Channel.ID)
	if err != nil {
		log.Printf("対応メンバー取得エラー: %v", err)
	}

	// 復旧メッセージを構築
	resolveMessage := fmt.Sprintf(
		"✅ *インシデントが復旧しました*\n\n"+
			"%s *タイトル:* %s\n"+
			"*重要度:* %s %s\n"+
			"*復旧者:* <@%s>\n"+
			"*インシデントID:* #%d\n"+
			"*チャンネル:* <#%s>",
		emoji,
		details["title"].(string),
		emoji,
		details["severity"].(string),
		callback.User.ID,
		incidentID,
		callback.Channel.ID,
	)

	// 対応メンバー一覧を追加
	if len(contributors) > 0 {
		resolveMessage += fmt.Sprintf("\n\n👥 *対応メンバー:* %s", contributors)
	}

	// インシデントチャンネルに復旧メッセージを投稿（緑の縦棒）
	attachment := slack.Attachment{
		Color: "good", // 緑色の縦棒
		Text:  resolveMessage,
	}

	_, _, err = api.PostMessage(
		callback.Channel.ID,
		slack.MsgOptionText("インシデントが復旧しました", false),
		slack.MsgOptionAttachments(attachment),
	)

	if err != nil {
		log.Printf("復旧メッセージ投稿エラー: %v", err)
	} else {
		log.Printf("インシデント %d の復旧をチャンネルに通知しました", incidentID)
	}

	// 全体周知チャンネルに復旧通知を送信（緑の縦棒付き）
	if config.Channels.EnableAnnouncement && len(config.Channels.AnnouncementChannels) > 0 {
		log.Println("全体周知チャンネルに復旧通知を送信します")
		postResolveToAnnouncementChannels(api, resolveMessage, callback.Channel.ID)
	}

	// タイムキーパーを自動停止
	if timekeeperManager.stopTimekeeper(incidentID) {
		log.Printf("インシデント %d のタイムキーパーを自動停止しました", incidentID)
	}
}

// handleStopTimekeeper はタイムキーパー停止ボタンがクリックされた時の処理
func handleStopTimekeeper(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("タイムキーパー停止ボタンがクリックされました")

	// ボタンのValueからインシデントIDを取得
	action := callback.ActionCallback.BlockActions[0]
	var incidentID int64
	_, err := fmt.Sscanf(action.Value, "incident_%d", &incidentID)
	if err != nil {
		log.Printf("インシデントID解析エラー: %v", err)
		return
	}

	// タイムキーパーを停止
	if timekeeperManager.stopTimekeeper(incidentID) {
		successMessage := fmt.Sprintf("⏹️ インシデント #%d のタイムキーパーを停止しました", incidentID)
		_, _, err := api.PostMessage(
			callback.Channel.ID,
			slack.MsgOptionText(successMessage, false),
		)

		if err != nil {
			log.Printf("停止メッセージ投稿エラー: %v", err)
		} else {
			log.Printf("インシデント %d のタイムキーパーを手動停止しました", incidentID)
		}
	} else {
		// 既に停止している場合
		api.PostEphemeral(
			callback.Channel.ID,
			callback.User.ID,
			slack.MsgOptionText("ℹ️ タイムキーパーは既に停止しています。", false),
		)
	}
}

// postToAnnouncementChannels は全体周知チャンネルにメッセージを投稿（赤/黄色の縦棒）
func postToAnnouncementChannels(api *slack.Client, message string, incidentChannelID string, severity string) {
	// 重要度に応じた色を決定
	var color string
	switch severity {
	case "critical", "high":
		color = "danger" // 赤色
	case "medium":
		color = "warning" // 黄色
	default:
		color = "#439FE0" // 青色（低重要度）
	}

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

		// アタッチメントを使用して色付き縦棒で投稿
		attachment := slack.Attachment{
			Color: color,
			Text:  announcementMessage,
		}

		_, _, err := api.PostMessage(
			channelID,
			slack.MsgOptionText("インシデント通知", false),
			slack.MsgOptionAttachments(attachment),
		)

		if err != nil {
			log.Printf("全体周知チャンネル %s への投稿エラー: %v", channelID, err)
		} else {
			log.Printf("全体周知チャンネル %s に投稿しました", channelID)
		}
	}
}

// postResolveToAnnouncementChannels は全体周知チャンネルに復旧通知を投稿（緑の縦棒）
func postResolveToAnnouncementChannels(api *slack.Client, message string, incidentChannelID string) {
	for _, channelID := range config.Channels.AnnouncementChannels {
		if channelID == "" {
			continue
		}

		log.Printf("全体周知チャンネル %s に復旧通知を投稿中...", channelID)

		// インシデントチャンネルのリンクを追加
		announcementMessage := message
		if incidentChannelID != "" {
			announcementMessage = fmt.Sprintf("%s\n\n📋 *対応チャンネル:* <#%s>", message, incidentChannelID)
		}

		// 緑色の縦棒で投稿
		attachment := slack.Attachment{
			Color: "good", // 緑色
			Text:  announcementMessage,
		}

		_, _, err := api.PostMessage(
			channelID,
			slack.MsgOptionText("インシデント復旧通知", false),
			slack.MsgOptionAttachments(attachment),
		)

		if err != nil {
			log.Printf("全体周知チャンネル %s への復旧通知投稿エラー: %v", channelID, err)
		} else {
			log.Printf("全体周知チャンネル %s に復旧通知を投稿しました", channelID)
		}
	}
}

// getChannelContributors はチャンネルでメッセージを投稿したユーザー一覧を取得
func getChannelContributors(api *slack.Client, channelID string) (string, error) {
	log.Printf("チャンネル %s の対応メンバーを取得中...", channelID)

	// チャンネルの会話履歴を取得（最大1000件）
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     1000,
	}

	history, err := api.GetConversationHistory(params)
	if err != nil {
		log.Printf("会話履歴取得エラー: %v", err)
		return "", fmt.Errorf("会話履歴取得エラー: %v", err)
	}

	log.Printf("取得したメッセージ数: %d", len(history.Messages))

	// ユニークなユーザーIDを収集（Botは除外）
	userSet := make(map[string]bool)
	botCount := 0
	userCount := 0

	for _, msg := range history.Messages {
		log.Printf("メッセージ - User: %s, BotID: %s, SubType: %s", msg.User, msg.BotID, msg.SubType)

		// Botのメッセージはスキップ
		if msg.BotID != "" || msg.SubType == "bot_message" {
			botCount++
			continue
		}

		// ユーザーIDがある場合のみ追加
		if msg.User != "" {
			userSet[msg.User] = true
			userCount++
		}
	}

	log.Printf("Bot メッセージ: %d, ユーザーメッセージ: %d", botCount, userCount)

	// ユーザーIDをスライスに変換
	var userIDs []string
	for userID := range userSet {
		userIDs = append(userIDs, userID)
		log.Printf("対応メンバー: %s", userID)
	}

	log.Printf("対応メンバー %d 人を検出しました", len(userIDs))

	// メンション形式に変換
	if len(userIDs) == 0 {
		log.Println("対応メンバーが0人のため、空文字列を返します")
		return "", nil
	}

	var mentions []string
	for _, userID := range userIDs {
		mentions = append(mentions, fmt.Sprintf("<@%s>", userID))
	}

	result := strings.Join(mentions, ", ")
	log.Printf("対応メンバー文字列: %s", result)

	return result, nil
}

// handleChannelArchive はチャンネルアーカイブ時の処理
func handleChannelArchive(api *slack.Client, event *slackevents.ChannelArchiveEvent) {
	log.Printf("チャンネルアーカイブイベントを受信しました: %s", event.Channel)

	// チャンネルがインシデントチャンネルかどうかを確認
	incidentID, title, err := getIncidentByChannelID(event.Channel)
	if err != nil {
		log.Printf("アーカイブされたチャンネル %s にはオープンなインシデントがありません: %v", event.Channel, err)
		return
	}

	log.Printf("インシデント %d (%s) のチャンネルがアーカイブされました", incidentID, title)

	// タイムキーパーを停止
	if timekeeperManager.stopTimekeeper(incidentID) {
		log.Printf("インシデント %d のタイムキーパーを自動停止しました（チャンネルアーカイブ）", incidentID)
	}

	// インシデントを自動的に復旧済みにする
	if db != nil {
		err := resolveIncident(incidentID, "system", "システム（チャンネルアーカイブ）")
		if err != nil {
			log.Printf("インシデント %d の自動復旧エラー: %v", incidentID, err)
		} else {
			log.Printf("インシデント %d を自動的に復旧済みにしました（チャンネルアーカイブ）", incidentID)
		}
	}
}
