package main

import (
	"testing"
	"time"
)

func TestTimekeeperManager(t *testing.T) {
	// テスト用のTimekeeperManagerを作成
	tm := &TimekeeperManager{
		timekeepers: make(map[int64]chan bool),
	}

	// 初期状態ではタイムキーパーが動作していないことを確認
	if tm.isTimekeeperRunning(1) {
		t.Error("初期状態でタイムキーパーが動作していると報告されました")
	}

	// タイムキーパーを手動で登録
	stopChan := make(chan bool)
	tm.mu.Lock()
	tm.timekeepers[1] = stopChan
	tm.mu.Unlock()

	// タイムキーパーが動作していることを確認
	if !tm.isTimekeeperRunning(1) {
		t.Error("タイムキーパーを登録したのに動作していないと報告されました")
	}

	// タイムキーパーを停止
	stopped := tm.stopTimekeeper(1)
	if !stopped {
		t.Error("タイムキーパーの停止に失敗しました")
	}

	// 停止後は動作していないことを確認
	if tm.isTimekeeperRunning(1) {
		t.Error("停止後もタイムキーパーが動作していると報告されました")
	}

	// 既に停止したタイムキーパーを再度停止しようとすると失敗することを確認
	stopped = tm.stopTimekeeper(1)
	if stopped {
		t.Error("存在しないタイムキーパーの停止が成功してしまいました")
	}
}

func TestTimekeeperMultipleInstances(t *testing.T) {
	// テスト用のTimekeeperManagerを作成
	tm := &TimekeeperManager{
		timekeepers: make(map[int64]chan bool),
	}

	// 複数のタイムキーパーを登録
	incidentIDs := []int64{1, 2, 3, 4, 5}
	for _, id := range incidentIDs {
		stopChan := make(chan bool)
		tm.mu.Lock()
		tm.timekeepers[id] = stopChan
		tm.mu.Unlock()
	}

	// すべてのタイムキーパーが動作していることを確認
	for _, id := range incidentIDs {
		if !tm.isTimekeeperRunning(id) {
			t.Errorf("インシデント %d のタイムキーパーが動作していません", id)
		}
	}

	// いくつかのタイムキーパーを停止
	tm.stopTimekeeper(2)
	tm.stopTimekeeper(4)

	// 停止したものは動作していないことを確認
	if tm.isTimekeeperRunning(2) {
		t.Error("停止したインシデント 2 のタイムキーパーが動作していると報告されました")
	}
	if tm.isTimekeeperRunning(4) {
		t.Error("停止したインシデント 4 のタイムキーパーが動作していると報告されました")
	}

	// 停止していないものは動作していることを確認
	if !tm.isTimekeeperRunning(1) {
		t.Error("インシデント 1 のタイムキーパーが停止していると報告されました")
	}
	if !tm.isTimekeeperRunning(3) {
		t.Error("インシデント 3 のタイムキーパーが停止していると報告されました")
	}
	if !tm.isTimekeeperRunning(5) {
		t.Error("インシデント 5 のタイムキーパーが停止していると報告されました")
	}
}

func TestTimekeeperConcurrency(t *testing.T) {
	// テスト用のTimekeeperManagerを作成
	tm := &TimekeeperManager{
		timekeepers: make(map[int64]chan bool),
	}

	// 複数のゴルーチンから同時にアクセス
	done := make(chan bool)
	iterations := 10

	for i := 0; i < iterations; i++ {
		go func(id int64) {
			stopChan := make(chan bool)
			tm.mu.Lock()
			tm.timekeepers[id] = stopChan
			tm.mu.Unlock()

			// 少し待機
			time.Sleep(10 * time.Millisecond)

			// 確認
			if !tm.isTimekeeperRunning(id) {
				t.Errorf("ゴルーチン %d: タイムキーパーが動作していません", id)
			}

			// 停止
			tm.stopTimekeeper(id)

			done <- true
		}(int64(i))
	}

	// すべてのゴルーチンが完了するまで待機
	for i := 0; i < iterations; i++ {
		<-done
	}

	// すべてのタイムキーパーが停止していることを確認
	tm.mu.RLock()
	if len(tm.timekeepers) != 0 {
		t.Errorf("タイムキーパーが残っています: %d 個", len(tm.timekeepers))
	}
	tm.mu.RUnlock()
}
