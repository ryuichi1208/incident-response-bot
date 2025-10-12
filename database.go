package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// initDB はデータベース接続を初期化
func initDB() error {
	// 環境変数を優先、次にconfig.toml、最後にデフォルト値
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = config.Database.Host
		if host == "" {
			host = "127.0.0.1"
		}
	}

	port := config.Database.Port
	if portEnv := os.Getenv("DB_PORT"); portEnv != "" {
		fmt.Sscanf(portEnv, "%d", &port)
	}
	if port == 0 {
		port = 5432
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = config.Database.User
		if user == "" {
			user = "postgres"
		}
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = config.Database.Password
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = config.Database.DBName
		if dbname == "" {
			dbname = "incident_bot"
		}
	}

	sslmode := config.Database.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	// 接続文字列を構築
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	// データベースに接続
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("データベース接続オープンエラー: %v", err)
	}

	// 接続テスト
	if err = db.Ping(); err != nil {
		return fmt.Errorf("データベース接続テストエラー: %v", err)
	}

	// 接続プールの設定
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Printf("データベースに接続しました: %s@%s:%d/%s", user, host, port, dbname)

	return nil
}

// saveIncident はインシデントをデータベースに保存
func saveIncident(title, severity, description, impact, channelID, channelName, reporterID, reporterName string) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("データベース接続が初期化されていません")
	}

	query := `
		INSERT INTO incidents (title, severity, description, impact, channel_id, channel_name, reporter_id, reporter_name, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'open')
		RETURNING id
	`

	var incidentID int64
	err := db.QueryRow(query, title, severity, description, impact, channelID, channelName, reporterID, reporterName).Scan(&incidentID)
	if err != nil {
		return 0, fmt.Errorf("インシデント保存エラー: %v", err)
	}

	log.Printf("インシデントをデータベースに保存しました (ID: %d)", incidentID)
	return incidentID, nil
}

// assignHandler はインシデントハンドラーを割り当て
func assignHandler(incidentID int64, handlerID, handlerName, assignedBy string) error {
	if db == nil {
		return fmt.Errorf("データベース接続が初期化されていません")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始エラー: %v", err)
	}
	defer tx.Rollback()

	// インシデントのハンドラーを更新
	updateQuery := `
		UPDATE incidents
		SET handler_id = $1, handler_name = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, handlerID, handlerName, incidentID)
	if err != nil {
		return fmt.Errorf("ハンドラー更新エラー: %v", err)
	}

	// ハンドラー履歴を記録
	historyQuery := `
		INSERT INTO incident_handler_history (incident_id, old_handler_id, new_handler_id, assigned_by)
		SELECT $1, handler_id, $2, $3
		FROM incidents
		WHERE id = $1
	`
	_, err = tx.Exec(historyQuery, incidentID, handlerID, assignedBy)
	if err != nil {
		return fmt.Errorf("ハンドラー履歴記録エラー: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションコミットエラー: %v", err)
	}

	log.Printf("インシデント %d のハンドラーを %s に割り当てました", incidentID, handlerName)
	return nil
}

// getIncidentByChannelID はチャンネルIDからインシデントを取得
func getIncidentByChannelID(channelID string) (int64, string, error) {
	if db == nil {
		return 0, "", fmt.Errorf("データベース接続が初期化されていません")
	}

	query := `
		SELECT id, title
		FROM incidents
		WHERE channel_id = $1 AND status = 'open'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var incidentID int64
	var title string
	err := db.QueryRow(query, channelID).Scan(&incidentID, &title)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("チャンネル %s のオープンなインシデントが見つかりません", channelID)
		}
		return 0, "", fmt.Errorf("インシデント取得エラー: %v", err)
	}

	return incidentID, title, nil
}

// getIncidentDetails はインシデントIDから詳細情報を取得
func getIncidentDetails(incidentID int64) (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("データベース接続が初期化されていません")
	}

	query := `
		SELECT title, severity, description, impact, status, channel_id, channel_name,
		       reporter_id, reporter_name, handler_id, handler_name, created_at, updated_at
		FROM incidents
		WHERE id = $1
	`

	var title, severity, description, impact, status, channelID, channelName, reporterID, reporterName string
	var handlerID, handlerName sql.NullString
	var createdAt, updatedAt time.Time

	err := db.QueryRow(query, incidentID).Scan(
		&title, &severity, &description, &impact, &status, &channelID, &channelName,
		&reporterID, &reporterName, &handlerID, &handlerName, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("インシデントID %d が見つかりません", incidentID)
		}
		return nil, fmt.Errorf("インシデント詳細取得エラー: %v", err)
	}

	details := map[string]interface{}{
		"id":            incidentID,
		"title":         title,
		"severity":      severity,
		"description":   description,
		"impact":        impact,
		"status":        status,
		"channel_id":    channelID,
		"channel_name":  channelName,
		"reporter_id":   reporterID,
		"reporter_name": reporterName,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	}

	if handlerID.Valid {
		details["handler_id"] = handlerID.String
	}
	if handlerName.Valid {
		details["handler_name"] = handlerName.String
	}

	return details, nil
}

// updateIncident はインシデントの詳細情報を更新
func updateIncident(incidentID int64, field, oldValue, newValue, updatedBy, updatedByName string) error {
	if db == nil {
		return fmt.Errorf("データベース接続が初期化されていません")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始エラー: %v", err)
	}
	defer tx.Rollback()

	// インシデントを更新
	var updateQuery string
	switch field {
	case "title":
		updateQuery = "UPDATE incidents SET title = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2"
	case "severity":
		updateQuery = "UPDATE incidents SET severity = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2"
	case "description":
		updateQuery = "UPDATE incidents SET description = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2"
	case "impact":
		updateQuery = "UPDATE incidents SET impact = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2"
	default:
		return fmt.Errorf("更新できないフィールド: %s", field)
	}

	_, err = tx.Exec(updateQuery, newValue, incidentID)
	if err != nil {
		return fmt.Errorf("インシデント更新エラー: %v", err)
	}

	// 更新履歴を記録
	historyQuery := `
		INSERT INTO incident_update_history (incident_id, field_name, old_value, new_value, updated_by, updated_by_name)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(historyQuery, incidentID, field, oldValue, newValue, updatedBy, updatedByName)
	if err != nil {
		return fmt.Errorf("更新履歴記録エラー: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションコミットエラー: %v", err)
	}

	log.Printf("インシデント %d の %s を更新しました", incidentID, field)
	return nil
}

// changeHandler はインシデントハンドラーを変更（交代）
func changeHandler(incidentID int64, newHandlerID, newHandlerName, changedBy string) error {
	if db == nil {
		return fmt.Errorf("データベース接続が初期化されていません")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始エラー: %v", err)
	}
	defer tx.Rollback()

	// 現在のハンドラーを取得
	var oldHandlerID sql.NullString
	err = tx.QueryRow("SELECT handler_id FROM incidents WHERE id = $1", incidentID).Scan(&oldHandlerID)
	if err != nil {
		return fmt.Errorf("現在のハンドラー取得エラー: %v", err)
	}

	// ハンドラーを更新
	updateQuery := `
		UPDATE incidents
		SET handler_id = $1, handler_name = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, newHandlerID, newHandlerName, incidentID)
	if err != nil {
		return fmt.Errorf("ハンドラー更新エラー: %v", err)
	}

	// ハンドラー履歴を記録
	var oldHandlerIDStr string
	if oldHandlerID.Valid {
		oldHandlerIDStr = oldHandlerID.String
	}

	historyQuery := `
		INSERT INTO incident_handler_history (incident_id, old_handler_id, new_handler_id, assigned_by)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(historyQuery, incidentID, oldHandlerIDStr, newHandlerID, changedBy)
	if err != nil {
		return fmt.Errorf("ハンドラー履歴記録エラー: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションコミットエラー: %v", err)
	}

	log.Printf("インシデント %d のハンドラーを %s に変更しました", incidentID, newHandlerName)
	return nil
}

// getUpdateHistory はインシデントの更新履歴を取得
func getUpdateHistory(incidentID int64, limit int) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("データベース接続が初期化されていません")
	}

	query := `
		SELECT field_name, old_value, new_value, updated_by, updated_by_name, updated_at, note
		FROM incident_update_history
		WHERE incident_id = $1
		ORDER BY updated_at DESC
		LIMIT $2
	`

	rows, err := db.Query(query, incidentID, limit)
	if err != nil {
		return nil, fmt.Errorf("更新履歴取得エラー: %v", err)
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var fieldName, oldValue, newValue, updatedBy, updatedByName string
		var note sql.NullString
		var updatedAt time.Time

		err := rows.Scan(&fieldName, &oldValue, &newValue, &updatedBy, &updatedByName, &updatedAt, &note)
		if err != nil {
			log.Printf("履歴スキャンエラー: %v", err)
			continue
		}

		record := map[string]interface{}{
			"field_name":      fieldName,
			"old_value":       oldValue,
			"new_value":       newValue,
			"updated_by":      updatedBy,
			"updated_by_name": updatedByName,
			"updated_at":      updatedAt,
		}

		if note.Valid {
			record["note"] = note.String
		}

		history = append(history, record)
	}

	return history, nil
}

// getOpenIncidents はオープンなインシデント一覧を取得（タイムキーパー復元用）
func getOpenIncidents() ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("データベース接続が初期化されていません")
	}

	query := `
		SELECT id, channel_id, created_at
		FROM incidents
		WHERE status = 'open'
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("オープンなインシデント取得エラー: %v", err)
	}
	defer rows.Close()

	var incidents []map[string]interface{}
	for rows.Next() {
		var id int64
		var channelID string
		var createdAt time.Time

		err := rows.Scan(&id, &channelID, &createdAt)
		if err != nil {
			log.Printf("インシデント情報スキャンエラー: %v", err)
			continue
		}

		incident := map[string]interface{}{
			"id":         id,
			"channel_id": channelID,
			"created_at": createdAt,
		}
		incidents = append(incidents, incident)
	}

	return incidents, nil
}

// resolveIncident はインシデントを復旧済みにする
func resolveIncident(incidentID int64, resolvedBy, resolvedByName string) error {
	if db == nil {
		return fmt.Errorf("データベース接続が初期化されていません")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始エラー: %v", err)
	}
	defer tx.Rollback()

	// インシデントのステータスを更新
	updateQuery := `
		UPDATE incidents
		SET status = 'resolved', resolved_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'open'
	`
	result, err := tx.Exec(updateQuery, incidentID)
	if err != nil {
		return fmt.Errorf("インシデント復旧更新エラー: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("更新行数取得エラー: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("インシデント %d は既に復旧済みか存在しません", incidentID)
	}

	// ステータス変更履歴を記録
	historyQuery := `
		INSERT INTO incident_status_history (incident_id, old_status, new_status, changed_by, note)
		VALUES ($1, 'open', 'resolved', $2, $3)
	`
	note := fmt.Sprintf("%s により復旧完了", resolvedByName)
	_, err = tx.Exec(historyQuery, incidentID, resolvedBy, note)
	if err != nil {
		return fmt.Errorf("ステータス履歴記録エラー: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションコミットエラー: %v", err)
	}

	log.Printf("インシデント %d を復旧済みに更新しました (復旧者: %s)", incidentID, resolvedByName)
	return nil
}
