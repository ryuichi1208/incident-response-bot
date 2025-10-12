package main

import (
	"github.com/BurntSushi/toml"
)

// Config は設定ファイルの構造
type Config struct {
	Slack    SlackConfig    `toml:"slack"`
	Channels ChannelsConfig `toml:"channels"`
	Database DatabaseConfig `toml:"database"`
}

// SlackConfig はSlack関連の設定
type SlackConfig struct {
	BotToken string `toml:"bot_token"`
	AppToken string `toml:"app_token"`
}

// ChannelsConfig はチャンネル関連の設定
type ChannelsConfig struct {
	AnnouncementChannels []string `toml:"announcement_channels"`
	EnableAnnouncement   bool     `toml:"enable_announcement"`
}

// DatabaseConfig はデータベース接続の設定
type DatabaseConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
	SSLMode  string `toml:"sslmode"`
}

var config Config

// loadConfig は設定ファイルを読み込む
func loadConfig(filename string) error {
	_, err := toml.DecodeFile(filename, &config)
	return err
}
