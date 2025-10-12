.PHONY: help build up down restart logs clean test db-shell bot-shell ps

# デフォルトターゲット
.DEFAULT_GOAL := help

# ヘルプ
help: ## このヘルプメッセージを表示
	@echo "使用可能なコマンド:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ビルド
build: ## Dockerイメージをビルド
	@echo "Dockerイメージをビルド中..."
	docker-compose build

# 起動
up: ## コンテナを起動
	@echo "コンテナを起動中..."
	docker-compose up -d
	@echo "コンテナが起動しました"
	@echo "ログを確認: make logs"

# 起動（フォアグラウンド）
up-fg: ## コンテナをフォアグラウンドで起動
	docker-compose up

# 停止
down: ## コンテナを停止して削除
	@echo "コンテナを停止中..."
	docker-compose down

# 停止（ボリューム削除）
down-v: ## コンテナとボリュームを停止して削除
	@echo "コンテナとボリュームを削除中..."
	docker-compose down -v

# 再起動
restart: ## コンテナを再起動
	@echo "コンテナを再起動中..."
	docker-compose restart

# ログ表示
logs: ## ログを表示（全サービス）
	docker-compose logs -f

# ボットのログ表示
logs-bot: ## ボットのログを表示
	docker-compose logs -f bot

# PostgreSQLのログ表示
logs-db: ## PostgreSQLのログを表示
	docker-compose logs -f postgres

# コンテナ一覧
ps: ## 起動中のコンテナを表示
	docker-compose ps

# PostgreSQLシェル
db-shell: ## PostgreSQLシェルに接続
	docker-compose exec postgres psql -U postgres -d incident_bot

# ボットコンテナシェル
bot-shell: ## ボットコンテナのシェルに接続
	docker-compose exec bot /bin/sh

# データベースリセット
db-reset: ## データベースをリセット
	@echo "データベースをリセット中..."
	docker-compose exec postgres psql -U postgres -d incident_bot -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	docker-compose exec postgres psql -U postgres -d incident_bot -f /docker-entrypoint-initdb.d/schema.sql
	@echo "データベースをリセットしました"

# クリーンアップ
clean: down ## コンテナとイメージを削除
	@echo "イメージを削除中..."
	docker-compose down --rmi all

# ローカル実行
run: ## ローカルで直接実行（PostgreSQLはDockerを使用）
	@echo "PostgreSQLコンテナを起動中..."
	docker-compose up -d postgres
	@echo "PostgreSQLの起動を待機中..."
	@sleep 5
	@echo "ボットをローカルで起動中..."
	go run .

# テスト実行
test: ## テストを実行
	go test -v ./...

# 依存関係の更新
deps: ## Go依存関係を更新
	go mod tidy
	go mod download

# ビルドとテスト
build-test: deps test build ## 依存関係更新、テスト、ビルドを実行

# 開発環境セットアップ
setup: ## 開発環境をセットアップ
	@echo "開発環境をセットアップ中..."
	@if [ ! -f config.toml ]; then \
		cp config.toml.example config.toml; \
		echo "config.toml を作成しました。編集してトークンを設定してください。"; \
	else \
		echo "config.toml は既に存在します。"; \
	fi
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env を作成しました。編集してトークンを設定してください。"; \
	else \
		echo ".env は既に存在します。"; \
	fi
	@echo "依存関係をインストール中..."
	go mod download
	@echo "PostgreSQLコンテナを起動中..."
	docker-compose up -d postgres
	@echo ""
	@echo "セットアップ完了！"
	@echo "次のステップ:"
	@echo "  1. config.toml を編集してSlackトークンを設定"
	@echo "  2. make run でボットを起動"

# フルリビルド
rebuild: clean build ## クリーンアップして再ビルド
	@echo "リビルド完了"

# ステータス確認
status: ## システムステータスを確認
	@echo "=== Docker コンテナ ==="
	@docker-compose ps
	@echo ""
	@echo "=== PostgreSQL 接続テスト ==="
	@docker-compose exec -T postgres pg_isready -U postgres || echo "PostgreSQLが起動していません"
	@echo ""
	@echo "=== ディスク使用量 ==="
	@docker-compose exec -T postgres du -sh /var/lib/postgresql/data 2>/dev/null || echo "データベースボリュームが存在しません"
