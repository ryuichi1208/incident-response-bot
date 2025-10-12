-- インシデント管理テーブル
CREATE TABLE IF NOT EXISTS incidents (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    description TEXT,
    impact TEXT,
    status VARCHAR(50) DEFAULT 'open',
    channel_id VARCHAR(100) NOT NULL,
    channel_name VARCHAR(100) NOT NULL,
    reporter_id VARCHAR(100) NOT NULL,
    reporter_name VARCHAR(255),
    handler_id VARCHAR(100),
    handler_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_channel_id ON incidents(channel_id);
CREATE INDEX IF NOT EXISTS idx_incidents_handler_id ON incidents(handler_id);
CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents(created_at);

-- インシデントステータスの更新履歴テーブル
CREATE TABLE IF NOT EXISTS incident_status_history (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER REFERENCES incidents(id) ON DELETE CASCADE,
    old_status VARCHAR(50),
    new_status VARCHAR(50) NOT NULL,
    changed_by VARCHAR(100) NOT NULL,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    note TEXT
);

-- インシデントハンドラー履歴テーブル
CREATE TABLE IF NOT EXISTS incident_handler_history (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER REFERENCES incidents(id) ON DELETE CASCADE,
    old_handler_id VARCHAR(100),
    new_handler_id VARCHAR(100),
    assigned_by VARCHAR(100) NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_status_history_incident_id ON incident_status_history(incident_id);
CREATE INDEX IF NOT EXISTS idx_handler_history_incident_id ON incident_handler_history(incident_id);

-- インシデント詳細更新履歴テーブル
CREATE TABLE IF NOT EXISTS incident_update_history (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER REFERENCES incidents(id) ON DELETE CASCADE,
    field_name VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    updated_by VARCHAR(100) NOT NULL,
    updated_by_name VARCHAR(255),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    note TEXT
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_update_history_incident_id ON incident_update_history(incident_id);
CREATE INDEX IF NOT EXISTS idx_update_history_updated_at ON incident_update_history(updated_at);
