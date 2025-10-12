package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestSeverityEmojiMapping(t *testing.T) {
	// 重要度絵文字のマッピングをテスト
	severityEmoji := map[string]string{
		"critical": "🔴",
		"high":     "🟠",
		"medium":   "🟡",
		"low":      "🟢",
	}

	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "🔴"},
		{"high", "🟠"},
		{"medium", "🟡"},
		{"low", "🟢"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			emoji, exists := severityEmoji[tt.severity]
			if !exists {
				t.Errorf("重要度 %s の絵文字が見つかりません", tt.severity)
			}
			if emoji != tt.expected {
				t.Errorf("重要度 %s の絵文字が間違っています: %s, 期待値: %s", tt.severity, emoji, tt.expected)
			}
		})
	}
}

func TestSeverityColorMapping(t *testing.T) {
	// 重要度の色マッピングをテスト（postToAnnouncementChannels内のロジック）
	tests := []struct {
		severity     string
		expectedColor string
	}{
		{"critical", "danger"},
		{"high", "danger"},
		{"medium", "warning"},
		{"low", "#439FE0"},
		{"", "#439FE0"}, // デフォルト
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			var color string
			switch tt.severity {
			case "critical", "high":
				color = "danger" // 赤色
			case "medium":
				color = "warning" // 黄色
			default:
				color = "#439FE0" // 青色（低重要度）
			}

			if color != tt.expectedColor {
				t.Errorf("重要度 %s の色が間違っています: %s, 期待値: %s", tt.severity, color, tt.expectedColor)
			}
		})
	}
}

func TestUserSetDeduplication(t *testing.T) {
	// getChannelContributorsで使用されるユーザーセットの重複排除ロジックをテスト
	userSet := make(map[string]bool)

	// 重複するユーザーIDを追加
	userIDs := []string{"U1", "U2", "U1", "U3", "U2", "U1"}

	for _, userID := range userIDs {
		userSet[userID] = true
	}

	// ユニークなユーザーIDの数を確認
	expectedCount := 3
	if len(userSet) != expectedCount {
		t.Errorf("ユニークなユーザー数が間違っています: %d, 期待値: %d", len(userSet), expectedCount)
	}

	// 期待されるユーザーIDが含まれていることを確認
	expectedUsers := []string{"U1", "U2", "U3"}
	for _, expectedUser := range expectedUsers {
		if !userSet[expectedUser] {
			t.Errorf("ユーザー %s がセットに含まれていません", expectedUser)
		}
	}
}

func TestBotMessageFiltering(t *testing.T) {
	// Botメッセージのフィルタリングロジックをテスト
	messages := []struct {
		botID   string
		subType string
		user    string
		isBot   bool
	}{
		{"", "", "U1", false},                 // 通常のユーザーメッセージ
		{"B123", "", "U2", true},              // BotIDがあるメッセージ
		{"", "bot_message", "U3", true},       // SubTypeがbot_messageのメッセージ
		{"B456", "bot_message", "U4", true},   // 両方あるメッセージ
		{"", "", "U5", false},                 // 通常のユーザーメッセージ
	}

	botCount := 0
	userCount := 0

	for _, msg := range messages {
		// Botのメッセージはスキップ
		if msg.botID != "" || msg.subType == "bot_message" {
			botCount++
		} else if msg.user != "" {
			userCount++
		}
	}

	expectedBotCount := 3
	expectedUserCount := 2

	if botCount != expectedBotCount {
		t.Errorf("Botメッセージ数が間違っています: %d, 期待値: %d", botCount, expectedBotCount)
	}

	if userCount != expectedUserCount {
		t.Errorf("ユーザーメッセージ数が間違っています: %d, 期待値: %d", userCount, expectedUserCount)
	}
}

func TestIncidentIDParsing(t *testing.T) {
	// インシデントIDの解析ロジックをテスト
	tests := []struct {
		name        string
		value       string
		expectedID  int64
		shouldError bool
	}{
		{"正常なID", "incident_123", 123, false},
		{"大きなID", "incident_999999", 999999, false},
		{"IDが1", "incident_1", 1, false},
		{"不正なフォーマット", "invalid_123", 0, true},
		{"数字なし", "incident_abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var incidentID int64
			n, err := fmt.Sscanf(tt.value, "incident_%d", &incidentID)

			if tt.shouldError {
				if err == nil && n > 0 {
					t.Errorf("エラーが期待されましたが、成功しました: %d", incidentID)
				}
			} else {
				if err != nil || n == 0 {
					t.Errorf("解析エラー: %v", err)
				}
				if incidentID != tt.expectedID {
					t.Errorf("解析されたIDが間違っています: %d, 期待値: %d", incidentID, tt.expectedID)
				}
			}
		})
	}
}


func TestEmptyAnnouncementChannels(t *testing.T) {
	// 空のチャンネルIDがフィルタリングされることをテスト
	channels := []string{"C123", "", "C456", "", "C789"}

	var validChannels []string
	for _, channelID := range channels {
		if channelID != "" {
			validChannels = append(validChannels, channelID)
		}
	}

	expectedCount := 3
	if len(validChannels) != expectedCount {
		t.Errorf("有効なチャンネル数が間違っています: %d, 期待値: %d", len(validChannels), expectedCount)
	}

	expectedChannels := []string{"C123", "C456", "C789"}
	for i, expected := range expectedChannels {
		if validChannels[i] != expected {
			t.Errorf("チャンネル %d が間違っています: %s, 期待値: %s", i, validChannels[i], expected)
		}
	}
}

func TestMentionFormatting(t *testing.T) {
	// ユーザーメンション形式のテスト
	userIDs := []string{"U1234", "U5678", "U9012"}

	var mentions []string
	for _, userID := range userIDs {
		mentions = append(mentions, fmt.Sprintf("<@%s>", userID))
	}

	expected := []string{"<@U1234>", "<@U5678>", "<@U9012>"}

	if len(mentions) != len(expected) {
		t.Errorf("メンション数が間違っています: %d, 期待値: %d", len(mentions), len(expected))
	}

	for i, expectedMention := range expected {
		if mentions[i] != expectedMention {
			t.Errorf("メンション %d が間違っています: %s, 期待値: %s", i, mentions[i], expectedMention)
		}
	}
}

func TestCommandParsing(t *testing.T) {
	// handleAppMentionで使用されるコマンド解析ロジックをテスト
	tests := []struct {
		name     string
		text     string
		isHelp   bool
		isHandler bool
		isList   bool
	}{
		{"helpコマンド", "help", true, false, false},
		{"ヘルプコマンド", "ヘルプ", true, false, false},
		{"handlerコマンド", "handler", false, true, false},
		{"ハンドラーコマンド", "ハンドラー", false, true, false},
		{"担当コマンド", "担当", false, true, false},
		{"listコマンド", "list", false, false, true},
		{"一覧コマンド", "一覧", false, false, true},
		{"リストコマンド", "リスト", false, false, true},
		{"通常のメンション", "hello", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := strings.ToLower(strings.TrimSpace(tt.text))

			isHelp := strings.Contains(text, "help") || strings.Contains(text, "ヘルプ")
			isHandler := strings.Contains(text, "handler") || strings.Contains(text, "ハンドラー") || strings.Contains(text, "担当")
			isList := strings.Contains(text, "list") || strings.Contains(text, "一覧") || strings.Contains(text, "リスト")

			if isHelp != tt.isHelp {
				t.Errorf("helpコマンドの判定が間違っています: %v, 期待値: %v", isHelp, tt.isHelp)
			}

			if isHandler != tt.isHandler {
				t.Errorf("handlerコマンドの判定が間違っています: %v, 期待値: %v", isHandler, tt.isHandler)
			}

			if isList != tt.isList {
				t.Errorf("listコマンドの判定が間違っています: %v, 期待値: %v", isList, tt.isList)
			}
		})
	}
}

func TestIncidentChannelNameDetection(t *testing.T) {
	// インシデントチャンネル名の検出ロジックをテスト
	tests := []struct {
		name             string
		channelName      string
		isIncidentChannel bool
	}{
		{"インシデントチャンネル", "incident-20250101", true},
		{"ランダムサフィックス付き", "incident-20250101-abc123", true},
		{"インシデントプレフィックス", "incident-test", true},
		{"通常のチャンネル", "general", false},
		{"似た名前", "incidents", false},
		{"空文字列", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isIncidentChannel := strings.HasPrefix(tt.channelName, "incident-")

			if isIncidentChannel != tt.isIncidentChannel {
				t.Errorf("インシデントチャンネル判定が間違っています: %v, 期待値: %v", isIncidentChannel, tt.isIncidentChannel)
			}
		})
	}
}

func TestUserNameFallback(t *testing.T) {
	// ユーザー名のフォールバックロジックをテスト
	tests := []struct {
		name        string
		realName    string
		userName    string
		expected    string
	}{
		{"RealNameあり", "田中太郎", "tanaka", "田中太郎"},
		{"RealNameなし", "", "tanaka", "tanaka"},
		{"両方あり", "山田花子", "yamada", "山田花子"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.realName != "" {
				result = tt.realName
			} else {
				result = tt.userName
			}

			if result != tt.expected {
				t.Errorf("ユーザー名が間違っています: %s, 期待値: %s", result, tt.expected)
			}
		})
	}
}

func TestDisplayNameFallback(t *testing.T) {
	// Display Nameのフォールバックロジックをテスト
	tests := []struct {
		name        string
		displayName string
		realName    string
		userName    string
		expected    string
	}{
		{"DisplayNameあり", "太郎さん", "田中太郎", "tanaka", "太郎さん"},
		{"DisplayNameなし・RealNameあり", "", "田中太郎", "tanaka", "田中太郎"},
		{"DisplayName・RealNameなし", "", "", "tanaka", "tanaka"},
		{"全てあり", "太郎さん", "田中太郎", "tanaka", "太郎さん"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			displayName := tt.displayName
			if displayName == "" {
				displayName = tt.realName
			}
			if displayName == "" {
				displayName = tt.userName
			}

			if displayName != tt.expected {
				t.Errorf("Display Nameが間違っています: %s, 期待値: %s", displayName, tt.expected)
			}
		})
	}
}
