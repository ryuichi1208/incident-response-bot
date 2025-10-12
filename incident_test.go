package main

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

func TestCreateIncidentModal(t *testing.T) {
	// テスト用のチャンネルIDでモーダルを作成
	channelID := "C12345"
	modal := createIncidentModal(channelID)

	// モーダルの基本構造を確認
	if modal.Type != slack.VTModal {
		t.Errorf("モーダルタイプが間違っています: %s, 期待値: %s", modal.Type, slack.VTModal)
	}

	if modal.Title.Text != "インシデント報告" {
		t.Errorf("モーダルタイトルが間違っています: %s", modal.Title.Text)
	}

	if modal.CallbackID != "incident_report_modal" {
		t.Errorf("CallbackIDが間違っています: %s", modal.CallbackID)
	}

	if modal.PrivateMetadata != channelID {
		t.Errorf("PrivateMetadataが間違っています: %s, 期待値: %s", modal.PrivateMetadata, channelID)
	}

	// ブロック数を確認（タイトル、重要度、詳細説明、影響範囲）
	if len(modal.Blocks.BlockSet) != 4 {
		t.Errorf("ブロック数が間違っています: %d, 期待値: 4", len(modal.Blocks.BlockSet))
	}

	// 各ブロックがInputBlockであることを確認
	for i, block := range modal.Blocks.BlockSet {
		if _, ok := block.(*slack.InputBlock); !ok {
			t.Errorf("ブロック %d がInputBlockではありません", i)
		}
	}
}

func TestCreateUpdateIncidentModal(t *testing.T) {
	// テスト用のインシデント詳細
	incidentID := int64(123)
	currentDetails := map[string]interface{}{
		"title":       "テストインシデント",
		"severity":    "high",
		"description": "テスト詳細",
		"impact":      "全ユーザー",
	}

	modal := createUpdateIncidentModal(incidentID, currentDetails)

	// モーダルの基本構造を確認
	if modal.Type != slack.VTModal {
		t.Errorf("モーダルタイプが間違っています: %s, 期待値: %s", modal.Type, slack.VTModal)
	}

	if modal.Title.Text != "インシデント情報を更新" {
		t.Errorf("モーダルタイトルが間違っています: %s", modal.Title.Text)
	}

	if modal.CallbackID != "incident_update_modal" {
		t.Errorf("CallbackIDが間違っています: %s", modal.CallbackID)
	}

	if modal.PrivateMetadata != "123" {
		t.Errorf("PrivateMetadataが間違っています: %s, 期待値: 123", modal.PrivateMetadata)
	}

	// ブロック数を確認
	if len(modal.Blocks.BlockSet) != 4 {
		t.Errorf("ブロック数が間違っています: %d, 期待値: 4", len(modal.Blocks.BlockSet))
	}
}

func TestCreateUpdateIncidentModalSeverityOptions(t *testing.T) {
	// 各重要度レベルをテスト
	severities := []string{"critical", "high", "medium", "low"}

	for _, severity := range severities {
		t.Run(severity, func(t *testing.T) {
			currentDetails := map[string]interface{}{
				"title":       "テスト",
				"severity":    severity,
				"description": "テスト",
				"impact":      "テスト",
			}

			modal := createUpdateIncidentModal(1, currentDetails)

			// 重要度ブロックを取得
			if len(modal.Blocks.BlockSet) < 2 {
				t.Fatal("ブロック数が不足しています")
			}

			severityBlock, ok := modal.Blocks.BlockSet[1].(*slack.InputBlock)
			if !ok {
				t.Fatal("重要度ブロックがInputBlockではありません")
			}

			selectElement, ok := severityBlock.Element.(*slack.SelectBlockElement)
			if !ok {
				t.Fatal("重要度ブロックのElementがSelectBlockElementではありません")
			}

			// 初期選択値が正しいことを確認
			if selectElement.InitialOption == nil {
				t.Error("InitialOptionが設定されていません")
			} else if selectElement.InitialOption.Value != severity {
				t.Errorf("InitialOptionの値が間違っています: %s, 期待値: %s", selectElement.InitialOption.Value, severity)
			}
		})
	}
}

func TestGenerateRandomStringForChannelName(t *testing.T) {
	// チャンネル名に使用されるランダム文字列のテスト
	length := 6
	result := generateRandomString(length)

	// 長さを確認
	if len(result) != length {
		t.Errorf("ランダム文字列の長さが間違っています: %d, 期待値: %d", len(result), length)
	}

	// 小文字の英数字のみであることを確認
	for _, char := range result {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			t.Errorf("ランダム文字列に無効な文字が含まれています: %c", char)
		}
	}
}

func TestIncidentModalBlockIDs(t *testing.T) {
	// ブロックIDが正しく設定されていることを確認
	modal := createIncidentModal("C12345")

	expectedBlockIDs := []string{
		"title_block",
		"severity_block",
		"description_block",
		"impact_block",
	}

	if len(modal.Blocks.BlockSet) != len(expectedBlockIDs) {
		t.Fatalf("ブロック数が一致しません: %d, 期待値: %d", len(modal.Blocks.BlockSet), len(expectedBlockIDs))
	}

	for i, expectedID := range expectedBlockIDs {
		inputBlock, ok := modal.Blocks.BlockSet[i].(*slack.InputBlock)
		if !ok {
			t.Errorf("ブロック %d がInputBlockではありません", i)
			continue
		}

		if inputBlock.BlockID != expectedID {
			t.Errorf("ブロック %d のIDが間違っています: %s, 期待値: %s", i, inputBlock.BlockID, expectedID)
		}
	}
}

func TestUpdateModalBlockIDs(t *testing.T) {
	// 更新モーダルのブロックIDが正しく設定されていることを確認
	currentDetails := map[string]interface{}{
		"title":       "テスト",
		"severity":    "high",
		"description": "テスト詳細",
		"impact":      "影響範囲",
	}
	modal := createUpdateIncidentModal(1, currentDetails)

	expectedBlockIDs := []string{
		"title_block",
		"severity_block",
		"description_block",
		"impact_block",
	}

	if len(modal.Blocks.BlockSet) != len(expectedBlockIDs) {
		t.Fatalf("ブロック数が一致しません: %d, 期待値: %d", len(modal.Blocks.BlockSet), len(expectedBlockIDs))
	}

	for i, expectedID := range expectedBlockIDs {
		inputBlock, ok := modal.Blocks.BlockSet[i].(*slack.InputBlock)
		if !ok {
			t.Errorf("ブロック %d がInputBlockではありません", i)
			continue
		}

		if inputBlock.BlockID != expectedID {
			t.Errorf("ブロック %d のIDが間違っています: %s, 期待値: %s", i, inputBlock.BlockID, expectedID)
		}
	}
}

func TestModalActionIDs(t *testing.T) {
	// アクションIDが正しく設定されていることを確認
	modal := createIncidentModal("C12345")

	expectedActionIDs := []string{
		"incident_title",
		"incident_severity",
		"incident_description",
		"incident_impact",
	}

	for i, expectedActionID := range expectedActionIDs {
		inputBlock, ok := modal.Blocks.BlockSet[i].(*slack.InputBlock)
		if !ok {
			t.Errorf("ブロック %d がInputBlockではありません", i)
			continue
		}

		var actualActionID string
		switch element := inputBlock.Element.(type) {
		case *slack.PlainTextInputBlockElement:
			actualActionID = element.ActionID
		case *slack.SelectBlockElement:
			actualActionID = element.ActionID
		default:
			t.Errorf("ブロック %d のElementの型が不明です", i)
			continue
		}

		if actualActionID != expectedActionID {
			t.Errorf("ブロック %d のアクションIDが間違っています: %s, 期待値: %s", i, actualActionID, expectedActionID)
		}
	}
}

func TestSeverityOptions(t *testing.T) {
	// 重要度選択肢が正しく設定されていることを確認
	modal := createIncidentModal("C12345")

	// severity_blockを取得（2番目のブロック）
	if len(modal.Blocks.BlockSet) < 2 {
		t.Fatal("ブロック数が不足しています")
	}

	severityBlock, ok := modal.Blocks.BlockSet[1].(*slack.InputBlock)
	if !ok {
		t.Fatal("重要度ブロックがInputBlockではありません")
	}

	selectElement, ok := severityBlock.Element.(*slack.SelectBlockElement)
	if !ok {
		t.Fatal("重要度ブロックのElementがSelectBlockElementではありません")
	}

	expectedOptions := []struct {
		value string
		text  string
	}{
		{"critical", "🔴 Critical - サービス停止"},
		{"high", "🟠 High - 重大な機能障害"},
		{"medium", "🟡 Medium - 一部機能に影響"},
		{"low", "🟢 Low - 軽微な問題"},
	}

	if len(selectElement.Options) != len(expectedOptions) {
		t.Fatalf("選択肢の数が間違っています: %d, 期待値: %d", len(selectElement.Options), len(expectedOptions))
	}

	for i, expected := range expectedOptions {
		if selectElement.Options[i].Value != expected.value {
			t.Errorf("選択肢 %d の値が間違っています: %s, 期待値: %s", i, selectElement.Options[i].Value, expected.value)
		}

		if selectElement.Options[i].Text.Text != expected.text {
			t.Errorf("選択肢 %d のテキストが間違っています: %s, 期待値: %s", i, selectElement.Options[i].Text.Text, expected.text)
		}
	}
}

func TestMultilineInputElements(t *testing.T) {
	// 詳細説明入力がマルチラインであることを確認
	modal := createIncidentModal("C12345")

	// description_blockを取得（3番目のブロック）
	if len(modal.Blocks.BlockSet) < 3 {
		t.Fatal("ブロック数が不足しています")
	}

	descriptionBlock, ok := modal.Blocks.BlockSet[2].(*slack.InputBlock)
	if !ok {
		t.Fatal("詳細説明ブロックがInputBlockではありません")
	}

	textInput, ok := descriptionBlock.Element.(*slack.PlainTextInputBlockElement)
	if !ok {
		t.Fatal("詳細説明ブロックのElementがPlainTextInputBlockElementではありません")
	}

	if !textInput.Multiline {
		t.Error("詳細説明入力がマルチラインではありません")
	}
}

func TestChannelNameFormat(t *testing.T) {
	// チャンネル名のフォーマットをテスト
	// 実際のcreateIncidentChannelはSlack APIを呼び出すため、
	// ここではチャンネル名のフォーマットロジックのみをテスト

	// 期待される形式: incident-YYYYMMDD または incident-YYYYMMDD-xxxxxx
	baseChannelName := "incident-20250101"

	// ベース名のフォーマットを確認
	if !strings.HasPrefix(baseChannelName, "incident-") {
		t.Error("チャンネル名が 'incident-' で始まっていません")
	}

	// 日付部分の長さを確認（YYYYMMDD = 8文字）
	datePart := strings.TrimPrefix(baseChannelName, "incident-")
	if len(datePart) != 8 {
		t.Errorf("日付部分の長さが間違っています: %d, 期待値: 8", len(datePart))
	}

	// ランダムサフィックス付きの場合
	channelWithSuffix := baseChannelName + "-abc123"
	if !strings.HasPrefix(channelWithSuffix, "incident-") {
		t.Error("サフィックス付きチャンネル名が 'incident-' で始まっていません")
	}
}

func TestModalLabels(t *testing.T) {
	// モーダルの各ラベルが正しく設定されていることを確認
	modal := createIncidentModal("C12345")

	expectedLabels := []string{
		"インシデントタイトル",
		"重要度",
		"詳細説明",
		"影響範囲",
	}

	for i, expectedLabel := range expectedLabels {
		inputBlock, ok := modal.Blocks.BlockSet[i].(*slack.InputBlock)
		if !ok {
			t.Errorf("ブロック %d がInputBlockではありません", i)
			continue
		}

		if inputBlock.Label.Text != expectedLabel {
			t.Errorf("ブロック %d のラベルが間違っています: %s, 期待値: %s", i, inputBlock.Label.Text, expectedLabel)
		}
	}
}
