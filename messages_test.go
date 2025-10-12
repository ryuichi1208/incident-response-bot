package main

import (
	"strings"
	"testing"
)

func TestGetIncidentGuidelines(t *testing.T) {
	guidelines := GetIncidentGuidelines()

	// 空でないことを確認
	if guidelines == "" {
		t.Error("GetIncidentGuidelines() が空文字列を返しました")
	}

	// 必須のキーワードが含まれていることを確認
	requiredKeywords := []string{
		"インシデント対応のガイドライン",
		"初動対応",
		"原因調査",
		"対応実施",
		"復旧確認",
		"事後対応",
		"役立つリンク",
		"Tips",
	}

	for _, keyword := range requiredKeywords {
		if !strings.Contains(guidelines, keyword) {
			t.Errorf("GetIncidentGuidelines() に必須キーワード '%s' が含まれていません", keyword)
		}
	}

	// 絵文字が含まれていることを確認
	requiredEmojis := []string{
		"📋",
		"1️⃣",
		"2️⃣",
		"3️⃣",
		"4️⃣",
		"5️⃣",
		"🔗",
		"💡",
	}

	for _, emoji := range requiredEmojis {
		if !strings.Contains(guidelines, emoji) {
			t.Errorf("GetIncidentGuidelines() に絵文字 '%s' が含まれていません", emoji)
		}
	}

	// 最小文字数のチェック（適切な長さのガイドラインであることを確認）
	minLength := 200
	if len(guidelines) < minLength {
		t.Errorf("GetIncidentGuidelines() の長さが短すぎます: %d 文字 (最小: %d)", len(guidelines), minLength)
	}
}
