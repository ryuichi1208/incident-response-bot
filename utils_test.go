package main

import (
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"6文字", 6},
		{"10文字", 10},
		{"1文字", 1},
		{"20文字", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.length)

			// 長さが正しいことを確認
			if len(result) != tt.length {
				t.Errorf("generateRandomString(%d) の長さが %d, 期待値 %d", tt.length, len(result), tt.length)
			}

			// 英数字のみで構成されていることを確認
			for _, char := range result {
				if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
					t.Errorf("generateRandomString(%d) に無効な文字が含まれています: %c", tt.length, char)
				}
			}
		})
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	// 複数回実行して異なる値が生成されることを確認
	results := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		result := generateRandomString(6)
		if results[result] {
			t.Logf("重複が検出されました（これは稀に発生する可能性があります）: %s", result)
		}
		results[result] = true
	}

	// 少なくとも95%は異なる値であることを期待
	uniqueCount := len(results)
	expectedUnique := int(float64(iterations) * 0.95)
	if uniqueCount < expectedUnique {
		t.Errorf("ランダム文字列の一意性が低い: %d/%d (期待値: >=%d)", uniqueCount, iterations, expectedUnique)
	}
}
