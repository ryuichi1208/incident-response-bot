package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
)

// TimekeeperManager はタイムキーパーのゴルーチンを管理
type TimekeeperManager struct {
	timekeepers map[int64]chan bool // incidentID -> stop channel
	mu          sync.RWMutex
}

var timekeeperManager = &TimekeeperManager{
	timekeepers: make(map[int64]chan bool),
}

// startTimekeeper はインシデントのタイムキーパーを開始
func (tm *TimekeeperManager) startTimekeeper(api *slack.Client, incidentID int64, channelID string, startTime time.Time) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 既に動いている場合は何もしない
	if _, exists := tm.timekeepers[incidentID]; exists {
		log.Printf("インシデント %d のタイムキーパーは既に動作中です", incidentID)
		return
	}

	// 停止用チャネルを作成
	stopChan := make(chan bool)
	tm.timekeepers[incidentID] = stopChan

	log.Printf("インシデント %d のタイムキーパーを開始します", incidentID)

	// ゴルーチンでタイムキーパーを開始
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				log.Printf("インシデント %d のタイムキーパーを停止しました", incidentID)
				return
			case <-ticker.C:
				// 経過時間を計算
				elapsed := time.Since(startTime)
				minutes := int(elapsed.Minutes())
				hours := minutes / 60
				mins := minutes % 60

				var elapsedStr string
				if hours > 0 {
					elapsedStr = fmt.Sprintf("%d時間%d分", hours, mins)
				} else {
					elapsedStr = fmt.Sprintf("%d分", mins)
				}

				// 経過時間メッセージを投稿
				message := fmt.Sprintf("⏱️ *インシデント経過時間:* %s\n*インシデントID:* #%d", elapsedStr, incidentID)

				_, _, err := api.PostMessage(
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
					log.Printf("タイムキーパーメッセージ投稿エラー: %v", err)

					// チャンネルがアーカイブされている場合は自動停止
					if strings.Contains(err.Error(), "is_archived") || strings.Contains(err.Error(), "channel_not_found") {
						log.Printf("チャンネル %s がアーカイブまたは削除されています。インシデント %d のタイムキーパーを自動停止します", channelID, incidentID)

						// タイムキーパーを停止
						tm.mu.Lock()
						if stopCh, exists := tm.timekeepers[incidentID]; exists {
							close(stopCh)
							delete(tm.timekeepers, incidentID)
							log.Printf("インシデント %d のタイムキーパーを自動停止しました", incidentID)
						}
						tm.mu.Unlock()

						// インシデントを自動的に復旧済みにする
						if db != nil {
							err := resolveIncident(incidentID, "system", "システム（チャンネルアーカイブ）")
							if err != nil {
								log.Printf("インシデント %d の自動復旧エラー: %v", incidentID, err)
							} else {
								log.Printf("インシデント %d を自動的に復旧済みにしました", incidentID)
							}
						}
						return
					}
				} else {
					log.Printf("インシデント %d の経過時間を投稿しました: %s", incidentID, elapsedStr)
				}
			}
		}
	}()
}

// stopTimekeeper はインシデントのタイムキーパーを停止
func (tm *TimekeeperManager) stopTimekeeper(incidentID int64) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	stopChan, exists := tm.timekeepers[incidentID]
	if !exists {
		log.Printf("インシデント %d のタイムキーパーは動作していません", incidentID)
		return false
	}

	// 停止シグナルを送信
	close(stopChan)

	// マップから削除
	delete(tm.timekeepers, incidentID)

	log.Printf("インシデント %d のタイムキーパーに停止シグナルを送信しました", incidentID)
	return true
}

// isTimekeeperRunning はタイムキーパーが動作中かチェック
func (tm *TimekeeperManager) isTimekeeperRunning(incidentID int64) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.timekeepers[incidentID]
	return exists
}
