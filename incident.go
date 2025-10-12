package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// createIncidentModal はインシデント報告用のモーダルを作成
func createIncidentModal(channelID string) slack.ModalViewRequest {
	// タイトル入力
	titleInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "例: 本番環境でAPIエラーが発生", false, false),
		"incident_title",
	)
	titleBlock := slack.NewInputBlock(
		"title_block",
		slack.NewTextBlockObject("plain_text", "インシデントタイトル", false, false),
		nil,
		titleInput,
	)

	// 重要度選択
	severityOptions := []*slack.OptionBlockObject{
		slack.NewOptionBlockObject("critical", slack.NewTextBlockObject("plain_text", "🔴 Critical - サービス停止", false, false), nil),
		slack.NewOptionBlockObject("high", slack.NewTextBlockObject("plain_text", "🟠 High - 重大な機能障害", false, false), nil),
		slack.NewOptionBlockObject("medium", slack.NewTextBlockObject("plain_text", "🟡 Medium - 一部機能に影響", false, false), nil),
		slack.NewOptionBlockObject("low", slack.NewTextBlockObject("plain_text", "🟢 Low - 軽微な問題", false, false), nil),
	}
	severitySelect := slack.NewOptionsSelectBlockElement(
		"static_select",
		slack.NewTextBlockObject("plain_text", "重要度を選択", false, false),
		"incident_severity",
		severityOptions...,
	)
	severityBlock := slack.NewInputBlock(
		"severity_block",
		slack.NewTextBlockObject("plain_text", "重要度", false, false),
		nil,
		severitySelect,
	)

	// 詳細説明入力
	descriptionInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "インシデントの詳細を記載してください", false, false),
		"incident_description",
	)
	descriptionInput.Multiline = true
	descriptionBlock := slack.NewInputBlock(
		"description_block",
		slack.NewTextBlockObject("plain_text", "詳細説明", false, false),
		nil,
		descriptionInput,
	)

	// 影響範囲入力
	impactInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "例: 全ユーザー、特定の機能のみ", false, false),
		"incident_impact",
	)
	impactBlock := slack.NewInputBlock(
		"impact_block",
		slack.NewTextBlockObject("plain_text", "影響範囲", false, false),
		nil,
		impactInput,
	)

	// モーダルビューの構築
	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			titleBlock,
			severityBlock,
			descriptionBlock,
			impactBlock,
		},
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		Title:           slack.NewTextBlockObject("plain_text", "インシデント報告", false, false),
		Close:           slack.NewTextBlockObject("plain_text", "キャンセル", false, false),
		Submit:          slack.NewTextBlockObject("plain_text", "報告する", false, false),
		Blocks:          blocks,
		CallbackID:      "incident_report_modal",
		PrivateMetadata: channelID, // チャンネルIDを保存
	}
}

// handleModalSubmission はモーダル送信時の処理
func handleModalSubmission(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("モーダル送信を受信しました")

	// モーダルから入力値を取得
	values := callback.View.State.Values

	title := values["title_block"]["incident_title"].Value
	severity := values["severity_block"]["incident_severity"].SelectedOption.Value
	description := values["description_block"]["incident_description"].Value
	impact := values["impact_block"]["incident_impact"].Value

	log.Printf("インシデント報告: タイトル=%s, 重要度=%s", title, severity)

	// インシデント情報を構造化
	incident := map[string]interface{}{
		"title":       title,
		"severity":    severity,
		"description": description,
		"impact":      impact,
		"reported_by": callback.User.Name,
		"reported_at": time.Now().Format("2006-01-02 15:04:05"),
	}

	// JSON形式でログ出力（将来的にDBやAPIに送信可能）
	incidentJSON, _ := json.MarshalIndent(incident, "", "  ")
	log.Printf("インシデント情報:\n%s", string(incidentJSON))

	// 重要度に応じた絵文字を選択
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}
	emoji := severityEmoji[severity]

	// チャンネルに報告メッセージを投稿
	reportMessage := fmt.Sprintf(
		"%s *インシデントが報告されました*\n\n"+
			"*タイトル:* %s\n"+
			"*重要度:* %s %s\n"+
			"*影響範囲:* %s\n"+
			"*詳細:*\n%s\n\n"+
			"*報告者:* <@%s>\n"+
			"*報告日時:* %s",
		emoji,
		title,
		emoji,
		severity,
		impact,
		description,
		callback.User.ID,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// チャンネルIDを取得（モーダルを開いたチャンネル）
	channelID := callback.View.PrivateMetadata
	if channelID == "" {
		// PrivateMetadataが空の場合はユーザーのDMに送信
		channelID = callback.User.ID
	}

	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(reportMessage, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", reportMessage, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("メッセージ投稿エラー: %v", err)
		return
	}

	log.Println("インシデント報告をチャンネルに投稿しました")

	// インシデント対応用チャンネルを作成
	incidentChannel, err := createIncidentChannel(api, title, callback.User.ID)
	var incidentChannelID string
	var incidentID int64
	if err != nil {
		log.Printf("インシデントチャンネル作成エラー: %v", err)
	} else {
		incidentChannelID = incidentChannel.ID

		// ユーザー情報を取得
		user, err := api.GetUserInfo(callback.User.ID)
		reporterName := callback.User.Name
		if err == nil && user.RealName != "" {
			reporterName = user.RealName
		}

		// インシデントをデータベースに保存
		incidentID, err = saveIncident(
			title,
			severity,
			description,
			impact,
			incidentChannel.ID,
			incidentChannel.Name,
			callback.User.ID,
			reporterName,
		)
		if err != nil {
			log.Printf("データベース保存エラー: %v", err)
		}

		// 作成したチャンネルに報告を投稿
		log.Printf("インシデントチャンネル %s に報告を投稿します", incidentChannel.ID)
		postIncidentToChannel(api, incidentChannel.ID, reportMessage, channelID, incidentID)

		// タイムキーパーを開始
		timekeeperManager.startTimekeeper(api, incidentID, incidentChannel.ID, time.Now())
		log.Printf("インシデント %d のタイムキーパーを開始しました", incidentID)
	}

	// 全体周知チャンネルへの投稿
	log.Printf("全体周知チャンネルへの投稿チェック: enable=%v, channels=%d",
		config.Channels.EnableAnnouncement, len(config.Channels.AnnouncementChannels))

	if config.Channels.EnableAnnouncement && len(config.Channels.AnnouncementChannels) > 0 {
		log.Println("全体周知チャンネルへの投稿を開始します")
		postToAnnouncementChannels(api, reportMessage, incidentChannelID, severity)
	} else {
		if !config.Channels.EnableAnnouncement {
			log.Println("全体周知機能が無効になっています")
		}
		if len(config.Channels.AnnouncementChannels) == 0 {
			log.Println("全体周知チャンネルが設定されていません")
		}
	}
}

// createIncidentChannel はインシデント対応用のチャンネルを作成
func createIncidentChannel(api *slack.Client, title string, reporterID string) (*slack.Channel, error) {
	// チャンネル名を生成: incident-yyyymmdd
	now := time.Now()
	baseChannelName := fmt.Sprintf("incident-%s", now.Format("20060102"))
	channelName := baseChannelName

	// 最大10回リトライ
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		log.Printf("インシデントチャンネル %s を作成します (試行 %d/%d)", channelName, i+1, maxRetries)

		// チャンネルを作成（パブリックチャンネル）
		channel, err := api.CreateConversation(slack.CreateConversationParams{
			ChannelName: channelName,
			IsPrivate:   false,
		})

		if err != nil {
			// チャンネルが既に存在する場合は、英数字のランダム文字列を付けて再試行
			if err.Error() == "name_taken" {
				log.Printf("チャンネル %s は既に存在します。ランダム文字列を付けて再試行します", channelName)
				// 6文字の英数字ランダム文字列を生成
				randomSuffix := generateRandomString(6)
				channelName = fmt.Sprintf("%s-%s", baseChannelName, randomSuffix)
				continue
			}
			return nil, fmt.Errorf("チャンネル作成エラー: %v", err)
		}

		log.Printf("インシデントチャンネル %s (ID: %s) を作成しました", channelName, channel.ID)

		// 報告者をチャンネルに招待
		_, err = api.InviteUsersToConversation(channel.ID, reporterID)
		if err != nil {
			log.Printf("ユーザー招待エラー: %v", err)
		} else {
			log.Printf("報告者 %s をチャンネルに招待しました", reporterID)
		}

		// チャンネルのトピックを設定
		topic := fmt.Sprintf("インシデント対応: %s", title)
		_, err = api.SetTopicOfConversation(channel.ID, topic)
		if err != nil {
			log.Printf("トピック設定エラー: %v", err)
		}

		return channel, nil
	}

	return nil, fmt.Errorf("チャンネル作成に失敗しました: %d回試行しましたが、すべて名前が重複しています", maxRetries)
}

// postIncidentToChannel はインシデント対応チャンネルに報告とリンクを投稿
func postIncidentToChannel(api *slack.Client, incidentChannelID string, reportMessage string, originalChannelID string, incidentID int64) {
	// ウェルカムメッセージを投稿
	welcomeMessage := `🙏 *インシデント報告ありがとうございます！*

このチャンネルでインシデント対応を進めていきましょう。`

	_, _, err := api.PostMessage(
		incidentChannelID,
		slack.MsgOptionText(welcomeMessage, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", welcomeMessage, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("ウェルカムメッセージ投稿エラー: %v", err)
	}

	// インシデント報告を投稿
	_, _, err = api.PostMessage(
		incidentChannelID,
		slack.MsgOptionText(reportMessage, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", reportMessage, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("インシデントチャンネルへの投稿エラー: %v", err)
		return
	}

	log.Printf("インシデントチャンネル %s に報告を投稿しました", incidentChannelID)

	// インシデントハンドラーボタンを投稿
	if incidentID > 0 {
		postHandlerButton(api, incidentChannelID, incidentID)
		// インシデント操作ボタンを投稿
		postIncidentActionsButton(api, incidentChannelID, incidentID)
	}

	// 障害対応に役立つ情報を投稿
	postIncidentGuidelines(api, incidentChannelID)

	// 元のチャンネルにインシデントチャンネルへのリンクを投稿
	linkMessage := fmt.Sprintf("📋 インシデント対応チャンネルが作成されました: <#%s>", incidentChannelID)
	_, _, err = api.PostMessage(
		originalChannelID,
		slack.MsgOptionText(linkMessage, false),
	)

	if err != nil {
		log.Printf("元のチャンネルへのリンク投稿エラー: %v", err)
	} else {
		log.Printf("元のチャンネルにインシデントチャンネルへのリンクを投稿しました")
	}
}

// createUpdateIncidentModal はインシデント更新用のモーダルを作成
func createUpdateIncidentModal(incidentID int64, currentDetails map[string]interface{}) slack.ModalViewRequest {
	// タイトル入力（現在の値をプレースホルダーに）
	titleInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", fmt.Sprintf("現在: %s", currentDetails["title"]), false, false),
		"update_title",
	)
	titleInput.InitialValue = currentDetails["title"].(string)
	titleBlock := slack.NewInputBlock(
		"title_block",
		slack.NewTextBlockObject("plain_text", "インシデントタイトル", false, false),
		nil,
		titleInput,
	)

	// 重要度選択（現在の値を初期選択に）
	currentSeverity := currentDetails["severity"].(string)
	severityOptions := []*slack.OptionBlockObject{
		slack.NewOptionBlockObject("critical", slack.NewTextBlockObject("plain_text", "🔴 Critical - サービス停止", false, false), nil),
		slack.NewOptionBlockObject("high", slack.NewTextBlockObject("plain_text", "🟠 High - 重大な機能障害", false, false), nil),
		slack.NewOptionBlockObject("medium", slack.NewTextBlockObject("plain_text", "🟡 Medium - 一部機能に影響", false, false), nil),
		slack.NewOptionBlockObject("low", slack.NewTextBlockObject("plain_text", "🟢 Low - 軽微な問題", false, false), nil),
	}

	var initialOption *slack.OptionBlockObject
	for _, opt := range severityOptions {
		if opt.Value == currentSeverity {
			initialOption = opt
			break
		}
	}

	severitySelect := slack.NewOptionsSelectBlockElement(
		"static_select",
		slack.NewTextBlockObject("plain_text", "重要度を選択", false, false),
		"update_severity",
		severityOptions...,
	)
	if initialOption != nil {
		severitySelect.InitialOption = initialOption
	}
	severityBlock := slack.NewInputBlock(
		"severity_block",
		slack.NewTextBlockObject("plain_text", "重要度", false, false),
		nil,
		severitySelect,
	)

	// 詳細説明入力
	descriptionInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "インシデントの詳細を記載してください", false, false),
		"update_description",
	)
	descriptionInput.Multiline = true
	descriptionInput.InitialValue = currentDetails["description"].(string)
	descriptionBlock := slack.NewInputBlock(
		"description_block",
		slack.NewTextBlockObject("plain_text", "詳細説明", false, false),
		nil,
		descriptionInput,
	)

	// 影響範囲入力
	impactInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "例: 全ユーザー、特定の機能のみ", false, false),
		"update_impact",
	)
	impactInput.InitialValue = currentDetails["impact"].(string)
	impactBlock := slack.NewInputBlock(
		"impact_block",
		slack.NewTextBlockObject("plain_text", "影響範囲", false, false),
		nil,
		impactInput,
	)

	// モーダルビューの構築
	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			titleBlock,
			severityBlock,
			descriptionBlock,
			impactBlock,
		},
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		Title:           slack.NewTextBlockObject("plain_text", "インシデント情報を更新", false, false),
		Close:           slack.NewTextBlockObject("plain_text", "キャンセル", false, false),
		Submit:          slack.NewTextBlockObject("plain_text", "更新する", false, false),
		Blocks:          blocks,
		CallbackID:      "incident_update_modal",
		PrivateMetadata: fmt.Sprintf("%d", incidentID),
	}
}

// handleUpdateModalSubmission はインシデント更新モーダル送信時の処理
func handleUpdateModalSubmission(api *slack.Client, callback slack.InteractionCallback) {
	log.Println("インシデント更新モーダル送信を受信しました")

	// インシデントIDを取得
	var incidentID int64
	fmt.Sscanf(callback.View.PrivateMetadata, "%d", &incidentID)

	// 現在の詳細を取得
	currentDetails, err := getIncidentDetails(incidentID)
	if err != nil {
		log.Printf("インシデント詳細取得エラー: %v", err)
		return
	}

	// モーダルから入力値を取得
	values := callback.View.State.Values

	newTitle := values["title_block"]["update_title"].Value
	newSeverity := values["severity_block"]["update_severity"].SelectedOption.Value
	newDescription := values["description_block"]["update_description"].Value
	newImpact := values["impact_block"]["update_impact"].Value

	// ユーザー情報を取得
	user, err := api.GetUserInfo(callback.User.ID)
	updatedByName := callback.User.Name
	if err == nil && user.RealName != "" {
		updatedByName = user.RealName
	}

	channelID := currentDetails["channel_id"].(string)
	var updatedFields []string

	// 各フィールドをチェックして変更があれば更新
	if newTitle != currentDetails["title"].(string) {
		err := updateIncident(incidentID, "title", currentDetails["title"].(string), newTitle, callback.User.ID, updatedByName)
		if err != nil {
			log.Printf("タイトル更新エラー: %v", err)
		} else {
			updatedFields = append(updatedFields, "タイトル")
		}
	}

	if newSeverity != currentDetails["severity"].(string) {
		err := updateIncident(incidentID, "severity", currentDetails["severity"].(string), newSeverity, callback.User.ID, updatedByName)
		if err != nil {
			log.Printf("重要度更新エラー: %v", err)
		} else {
			updatedFields = append(updatedFields, "重要度")
		}
	}

	if newDescription != currentDetails["description"].(string) {
		err := updateIncident(incidentID, "description", currentDetails["description"].(string), newDescription, callback.User.ID, updatedByName)
		if err != nil {
			log.Printf("詳細説明更新エラー: %v", err)
		} else {
			updatedFields = append(updatedFields, "詳細説明")
		}
	}

	if newImpact != currentDetails["impact"].(string) {
		err := updateIncident(incidentID, "impact", currentDetails["impact"].(string), newImpact, callback.User.ID, updatedByName)
		if err != nil {
			log.Printf("影響範囲更新エラー: %v", err)
		} else {
			updatedFields = append(updatedFields, "影響範囲")
		}
	}

	if len(updatedFields) > 0 {
		// 更新通知メッセージを投稿
		updateMessage := fmt.Sprintf("📝 *インシデント情報が更新されました*\n\n"+
			"*更新者:* <@%s>\n"+
			"*更新項目:* %s\n"+
			"*インシデントID:* #%d",
			callback.User.ID,
			strings.Join(updatedFields, "、"),
			incidentID,
		)

		_, _, err := api.PostMessage(
			channelID,
			slack.MsgOptionText(updateMessage, false),
			slack.MsgOptionBlocks(
				slack.NewSectionBlock(
					slack.NewTextBlockObject("mrkdwn", updateMessage, false, false),
					nil, nil,
				),
			),
		)

		if err != nil {
			log.Printf("更新通知投稿エラー: %v", err)
		} else {
			log.Printf("インシデント %d の更新を通知しました", incidentID)
		}
	} else {
		log.Printf("インシデント %d に変更はありませんでした", incidentID)
	}
}

// postIncidentGuidelines はインシデント対応のガイドラインを投稿
func postIncidentGuidelines(api *slack.Client, channelID string) {
	guidelinesMessage := `📋 *インシデント対応のガイドライン*

*1️⃣ 初動対応 (最初の5分)*
• 影響範囲の確認
• 関係者への通知
• 暫定対応の検討

*2️⃣ 原因調査*
• ログの確認
• エラーメッセージの収集
• 最近の変更の確認
• モニタリングダッシュボードの確認

*3️⃣ 対応実施*
• 対応方針の決定と共有
• 実施前のバックアップ
• 段階的な実施
• 影響の確認

*4️⃣ 復旧確認*
• サービスの正常性確認
• モニタリング指標の確認
• ユーザー影響の確認

*5️⃣ 事後対応*
• インシデントレポートの作成
• 再発防止策の検討
• ポストモーテムの実施

---

*🔗 役立つリンク*
• モニタリングダッシュボード
• ログ検索ツール
• 障害対応手順書
• エスカレーションフロー

*💡 Tips*
• このチャンネルで進捗を随時共有しましょう
• 判断に迷ったら早めに相談しましょう
• 作業は複数人でレビューしながら進めましょう`

	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(guidelinesMessage, false),
		slack.MsgOptionBlocks(
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", guidelinesMessage, false, false),
				nil, nil,
			),
		),
	)

	if err != nil {
		log.Printf("ガイドライン投稿エラー: %v", err)
	} else {
		log.Printf("インシデント対応ガイドラインを投稿しました")
	}
}
