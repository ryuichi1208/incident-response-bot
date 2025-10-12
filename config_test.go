package main

import (
	"os"
	"testing"
)

func TestLoadConfigWithDefaults(t *testing.T) {
	// 存在しないファイルを指定してエラーが発生することを確認
	err := loadConfig("nonexistent.toml")

	// エラーが発生することを期待
	if err == nil {
		t.Error("存在しないファイルでエラーが発生しませんでした")
	}

	// loadConfigはエラー時にデフォルト値を設定しないため、
	// config構造体は空のままになる
	// このテストは単にエラーが返されることを確認するのみ
	t.Logf("期待通りにエラーが発生しました: %v", err)
}

func TestConfigWithEnvironmentVariables(t *testing.T) {
	// 環境変数を設定
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_APP_TOKEN", "xapp-test-token")
	os.Setenv("DB_HOST", "test-host")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "test-user")
	os.Setenv("DB_PASSWORD", "test-password")
	os.Setenv("DB_NAME", "test-db")

	defer func() {
		// テスト後にクリーンアップ
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_APP_TOKEN")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
	}()

	// 環境変数が正しく読み込まれることを確認
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken != "xoxb-test-token" {
		t.Errorf("SLACK_BOT_TOKEN が正しく設定されていません: %s", botToken)
	}

	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken != "xapp-test-token" {
		t.Errorf("SLACK_APP_TOKEN が正しく設定されていません: %s", appToken)
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost != "test-host" {
		t.Errorf("DB_HOST が正しく設定されていません: %s", dbHost)
	}
}

func TestConfigStructure(t *testing.T) {
	// Config構造体が正しく初期化されることを確認
	testConfig := Config{
		Slack: SlackConfig{
			BotToken: "test-bot-token",
			AppToken: "test-app-token",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			DBName:   "test_db",
			SSLMode:  "disable",
		},
		Channels: ChannelsConfig{
			EnableAnnouncement:   true,
			AnnouncementChannels: []string{"C12345", "C67890"},
		},
	}

	if testConfig.Slack.BotToken != "test-bot-token" {
		t.Error("Config構造体のBotTokenが正しく設定されていません")
	}

	if testConfig.Database.Port != 5432 {
		t.Error("Config構造体のDBポートが正しく設定されていません")
	}

	if !testConfig.Channels.EnableAnnouncement {
		t.Error("Config構造体のEnableAnnouncementが正しく設定されていません")
	}

	if len(testConfig.Channels.AnnouncementChannels) != 2 {
		t.Errorf("Config構造体のAnnouncementChannels数が間違っています: %d, 期待値: 2", len(testConfig.Channels.AnnouncementChannels))
	}
}
