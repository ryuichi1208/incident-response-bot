package main

import (
	"testing"
)

// モックデータベース接続のテスト
func TestDatabaseConnectionNil(t *testing.T) {
	// データベース接続がnilの場合のテスト
	originalDB := db
	db = nil
	defer func() { db = originalDB }()

	// saveIncident
	_, err := saveIncident("test", "high", "desc", "impact", "ch1", "channel", "u1", "user")
	if err == nil {
		t.Error("データベースがnilの場合、saveIncidentはエラーを返すべきです")
	}

	// assignHandler
	err = assignHandler(1, "u1", "user", "u2")
	if err == nil {
		t.Error("データベースがnilの場合、assignHandlerはエラーを返すべきです")
	}

	// getIncidentByChannelID
	_, _, err = getIncidentByChannelID("ch1")
	if err == nil {
		t.Error("データベースがnilの場合、getIncidentByChannelIDはエラーを返すべきです")
	}

	// getIncidentDetails
	_, err = getIncidentDetails(1)
	if err == nil {
		t.Error("データベースがnilの場合、getIncidentDetailsはエラーを返すべきです")
	}

	// updateIncident
	err = updateIncident(1, "title", "old", "new", "u1", "user")
	if err == nil {
		t.Error("データベースがnilの場合、updateIncidentはエラーを返すべきです")
	}

	// changeHandler
	err = changeHandler(1, "u2", "user2", "u1")
	if err == nil {
		t.Error("データベースがnilの場合、changeHandlerはエラーを返すべきです")
	}

	// getUpdateHistory
	_, err = getUpdateHistory(1, 10)
	if err == nil {
		t.Error("データベースがnilの場合、getUpdateHistoryはエラーを返すべきです")
	}

	// getOpenIncidents
	_, err = getOpenIncidents()
	if err == nil {
		t.Error("データベースがnilの場合、getOpenIncidentsはエラーを返すべきです")
	}

	// resolveIncident
	err = resolveIncident(1, "u1", "user")
	if err == nil {
		t.Error("データベースがnilの場合、resolveIncidentはエラーを返すべきです")
	}
}

func TestDatabaseErrorMessages(t *testing.T) {
	// データベース接続がnilの場合のエラーメッセージをテスト
	originalDB := db
	db = nil
	defer func() { db = originalDB }()

	_, err := saveIncident("test", "high", "desc", "impact", "ch1", "channel", "u1", "user")
	if err != nil && err.Error() != "データベース接続が初期化されていません" {
		t.Errorf("予期しないエラーメッセージ: %v", err)
	}
}
