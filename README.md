# incident-response-bot

Slackでインシデントを報告・管理するためのボットです。メンションするとモーダルが表示され、インシデント情報を入力できます。

## クイックスタート（Docker）

```bash
# リポジトリをクローン
git clone https://github.com/ryuichi1208/incident-response-bot.git
cd incident-response-bot

# 環境をセットアップ
make setup

# config.tomlを編集してSlackトークンを設定
vi config.toml

# ビルドして起動
make build
make up

# ログを確認
make logs-bot
```

## 機能

- 🚨 メンションでインシデント報告モーダルを表示
- 📝 インシデントタイトル、重要度、詳細説明、影響範囲を入力
- 🎨 重要度に応じた色分け（Critical/High/Medium/Low）
- ✍️ 入力開始時に「〇〇さんが入力中です」メッセージを表示
- 💬 チャンネルに整形されたインシデント報告を投稿
- 📢 複数の全体周知チャンネルへ同時投稿（設定ファイルで管理）
- 🗂️ インシデント対応用チャンネルの自動作成（incident-YYYYMMDD形式、重複時は英数字サフィックス）
- 📋 インシデント対応ガイドラインの自動投稿
- 🙋 インシデントハンドラー割り当て機能（担当者ボタン）
- 🗄️ PostgreSQLによるインシデント管理とハンドラー履歴の記録
- 💬 helpコマンド、handlerコマンド、listコマンド

## 必要なもの

- Go 1.21以上
- Slack Appの作成とトークンの取得
- Azure App ServiceとSQL Serverが今のところ必須

## Slack Appの設定

### 1. アプリの作成
[Slack API](https://api.slack.com/apps)で「Create New App」→「From scratch」を選択

### 2. OAuth & Permissions の設定
以下のBot Token Scopesを追加:
- `app_mentions:read` - メンションを読み取る
- `chat:write` - メッセージを送信する
- `users:read` - ユーザー情報（Display Name）を取得する
- `channels:manage` - チャンネルを作成する
- `channels:read` - チャンネル情報を取得する
- `groups:write` - プライベートチャンネルを作成する（必要に応じて）

その後、「Install to Workspace」でアプリをインストール

### 3. Socket Mode の有効化
1. 「Socket Mode」セクションで Socket Mode を有効化
2. 「Generate an app-level token」でトークンを生成（スコープ: `connections:write`）

### 4. Event Subscriptions の設定
1. 「Event Subscriptions」で「Enable Events」をON
2. 「Subscribe to bot events」で`app_mention`イベントを追加
3. 変更を保存

### 5. Interactivity の有効化
「Interactivity & Shortcuts」で「Interactivity」をON（モーダル送信に必要）

## セットアップ

### 方法1: Dockerを使用（推奨）

1. リポジトリをクローン:
```bash
git clone https://github.com/ryuichi1208/incident-response-bot.git
cd incident-response-bot
```

2. 環境をセットアップ:
```bash
make setup
```

3. config.tomlを編集してSlackトークンを設定:
```bash
vi config.toml
```

または、.envファイルを作成して環境変数を設定:
```bash
cp .env.example .env
vi .env
```

.envファイルの例:
```bash
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-token
```

4. Dockerコンテナをビルドして起動:
```bash
make build
make up
```

5. ログを確認:
```bash
make logs
```

成功すると以下のようなログが表示されます:
```
incident-bot | socketmode: Slackに接続しました
incident-bot | slack-bot: Botが起動しました。Bot ID: U12345678
incident-bot | データベースに接続しました: postgres@postgres:5432/incident_bot
```

### 方法2: ローカル実行（PostgreSQLのみDocker）

1. リポジトリをクローン:
```bash
git clone https://github.com/ryuichi1208/incident-response-bot.git
cd incident-response-bot
```

2. 依存関係をインストール:
```bash
go mod download
```

3. PostgreSQLコンテナのみ起動:
```bash
docker-compose up -d postgres
```

4. 設定ファイルを作成:
```bash
cp config.toml.example config.toml
```

設定ファイルを編集してトークンとチャンネル情報を設定:
```toml
[slack]
bot_token = "xoxb-your-bot-token"
app_token = "xapp-your-app-token"

[channels]
# 全体周知チャンネルのリスト（複数指定可能）
announcement_channels = ["C1234567890", "C0987654321"]
enable_announcement = true

[database]
host = "localhost"
port = 5432
user = "postgres"
password = "your-password"
dbname = "incident_bot"
sslmode = "disable"
```

または、環境変数で設定（config.tomlがない場合のフォールバック）:
```bash
export SLACK_BOT_TOKEN=xoxb-your-bot-token
export SLACK_APP_TOKEN=xapp-your-app-token
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=your-password
export DB_NAME=incident_bot
```

5. ボットを実行:
```bash
make run
# または
go run .
```

成功すると以下のようなログが表示されます:
```
socketmode: Slackに接続しました
slack-bot: Botが起動しました。Bot ID: U12345678
データベースに接続しました: postgres@localhost:5432/incident_bot
```

### 方法3: 完全にローカル実行

1. PostgreSQLをローカルにインストール

2. データベースを作成:
```bash
createdb incident_bot
```

3. スキーマを適用:
```bash
psql -d incident_bot -f schema.sql
```

4. 設定ファイルを作成して編集（方法2の手順4と同じ）

5. 実行:
```bash
go run .
```

## 使い方

### インシデントの報告

1. Slackのチャンネルでボットをメンション:
```
@incident-bot
```

2. 表示された「🚨 インシデントを報告」ボタンをクリック

3. 「〇〇さんが入力中です」メッセージが表示され、モーダルが開きます

4. 以下の情報を入力:
   - **インシデントタイトル**: 簡潔なタイトル
   - **重要度**: Critical/High/Medium/Low から選択
   - **詳細説明**: インシデントの詳細
   - **影響範囲**: どの範囲に影響があるか

5. 「報告する」をクリック

6. 以下が自動的に実行されます:
   - インシデント対応用チャンネルの作成（`incident-YYYYMMDD`形式）
   - インシデント報告の投稿
   - インシデント対応ガイドラインの投稿
   - インシデントハンドラー割り当てボタンの表示
   - 設定した全体周知チャンネルへの通知（設定している場合）
   - PostgreSQLへのインシデント情報の保存（データベースが有効な場合）

### インシデントハンドラーの割り当て

1. インシデントチャンネルで「🙋 担当者になる」ボタンをクリック

2. クリックしたユーザーがインシデントハンドラーに割り当てられます

3. 「✅ 〇〇さんがこのインシデントの担当者になりました！」というメッセージが投稿されます

4. データベースに割り当て履歴が記録されます（データベースが有効な場合）

### ボットコマンド

**通常のチャンネル:**
- `@bot` - インシデント報告ボタンを表示
- `@bot help` / `@bot ヘルプ` - ヘルプを表示
- `@bot handler` / `@bot ハンドラー` / `@bot 担当` - そのチャンネルのハンドラー情報を表示
- `@bot list` / `@bot 一覧` / `@bot リスト` - オープン中のインシデント一覧を表示

**インシデントチャンネル (incident-で始まる):**
- `@bot` - 自動的にヘルプを表示
- `@bot handler` - そのチャンネルのハンドラー情報を表示
- その他のコマンドも利用可能

## 設定ファイル詳細

`config.toml`で以下の設定が可能です：

```toml
[slack]
bot_token = "xoxb-..." # Slack Bot Token
app_token = "xapp-..." # Slack App Token

[channels]
# 全体周知チャンネルのリスト（複数指定可能）
announcement_channels = ["C1234567890", "C0987654321"]
# 全体周知チャンネルへの投稿を有効化
enable_announcement = true

[database]
# PostgreSQL接続情報（オプション）
host = "localhost"
port = 5432
user = "postgres"
password = "your-password"
dbname = "incident_bot"
sslmode = "disable"
```

**チャンネルIDの確認方法:**
1. Slackでチャンネルを右クリック
2. 「チャンネルの詳細を表示」を選択
3. 一番下にチャンネルIDが表示されます

## Makeコマンド一覧

Dockerを使用する場合、以下のMakeコマンドが利用できます：

```bash
make help          # ヘルプを表示
make setup         # 開発環境をセットアップ
make build         # Dockerイメージをビルド
make up            # コンテナを起動（バックグラウンド）
make up-fg         # コンテナを起動（フォアグラウンド）
make down          # コンテナを停止
make down-v        # コンテナとボリュームを削除
make restart       # コンテナを再起動
make logs          # 全サービスのログを表示
make logs-bot      # ボットのログを表示
make logs-db       # PostgreSQLのログを表示
make ps            # コンテナ一覧を表示
make db-shell      # PostgreSQLシェルに接続
make bot-shell     # ボットコンテナのシェルに接続
make db-reset      # データベースをリセット
make clean         # コンテナとイメージを削除
make run           # ローカルで実行（PostgreSQLのみDocker）
make test          # テストを実行
make deps          # Go依存関係を更新
make rebuild       # クリーンアップして再ビルド
make status        # システムステータスを確認
```

### よく使うコマンド例

初回セットアップ:
```bash
make setup
# config.tomlを編集
make build
make up
```

ログ確認:
```bash
make logs-bot
```

データベース操作:
```bash
make db-shell
# PostgreSQLシェル内で:
# \dt              # テーブル一覧
# SELECT * FROM incidents;  # インシデント一覧
# \q               # 終了
```

データベースリセット:
```bash
make db-reset
```

コンテナ再起動:
```bash
make restart
```

完全クリーンアップ:
```bash
make down-v
make clean
```

## データベーススキーマ

PostgreSQLを使用する場合、以下のテーブルが作成されます：

### incidents テーブル
インシデントの基本情報を管理:
- id: インシデントID（自動採番）
- title: インシデントタイトル
- severity: 重要度（critical/high/medium/low）
- description: 詳細説明
- impact: 影響範囲
- status: ステータス（open/resolved）
- channel_id: インシデントチャンネルID
- channel_name: インシデントチャンネル名
- reporter_id: 報告者のユーザーID
- reporter_name: 報告者名
- handler_id: 担当者のユーザーID
- handler_name: 担当者名
- created_at: 作成日時
- updated_at: 更新日時
- resolved_at: 解決日時

### incident_status_history テーブル
インシデントのステータス変更履歴:
- id: 履歴ID
- incident_id: インシデントID（外部キー）
- old_status: 変更前のステータス
- new_status: 変更後のステータス
- changed_by: 変更者のユーザーID
- changed_at: 変更日時
- note: 備考

### incident_handler_history テーブル
インシデントハンドラーの割り当て履歴:
- id: 履歴ID
- incident_id: インシデントID（外部キー）
- old_handler_id: 変更前の担当者ID
- new_handler_id: 変更後の担当者ID
- assigned_by: 割り当てを行ったユーザーID
- assigned_at: 割り当て日時

## 実装の詳細

### 主要な関数

- `handleAppMention` - メンション受信時にボタンを表示
- `handleOpenModal` - ボタンクリック時に入力中メッセージを投稿しモーダルを開く
- `createIncidentModal` - インシデント報告用モーダルの作成
- `handleModalSubmission` - モーダル送信時の処理とチャンネルへの投稿
- `createIncidentChannel` - インシデント対応チャンネルの作成（重複時は英数字ランダムサフィックス追加）
- `generateRandomString` - ランダムな英数字文字列を生成（チャンネル名の重複回避用）
- `postIncidentToChannel` - インシデントチャンネルへの投稿
- `postHandlerButton` - インシデントハンドラーボタンの投稿
- `handleAssignHandler` - インシデントハンドラー割り当て処理
- `postIncidentGuidelines` - インシデント対応ガイドラインの投稿
- `postToAnnouncementChannels` - 全体周知チャンネルへの投稿
- `initDB` - PostgreSQL接続の初期化
- `saveIncident` - インシデントのデータベース保存
- `assignHandler` - インシデントハンドラーの割り当てとデータベース更新
- `showHelp` - ヘルプメッセージの表示
- `showHandler` - チャンネルのハンドラー情報を表示
- `showIncidentList` - オープン中のインシデント一覧を表示
- `loadConfig` - TOML設定ファイルの読み込み

### 技術スタック

- Socket Modeを使用してSlackとリアルタイム通信
- インタラクティブコンポーネント（モーダル、ボタン）に対応
- 設定ファイル優先、環境変数をフォールバックとして対応
- PostgreSQLによる永続化（オプション）
- トランザクション処理によるデータ整合性の保証

## トラブルシューティング

### モーダルが表示されない
- 「Interactivity & Shortcuts」が有効になっているか確認
- ボットがチャンネルに追加されているか確認（`/invite @bot-name`）

### イベントが起動しない
- Event Subscriptionsで`app_mention`が設定されているか確認
- 設定変更後、アプリを再インストール

### 権限エラー
- 必要な権限（`app_mentions:read`, `chat:write`, `users:read`）が設定されているか確認

### 全体周知チャンネルに投稿されない
- `config.toml`の`enable_announcement`が`true`になっているか確認
- `announcement_channels`にチャンネルIDが正しく設定されているか確認
- ボットが全体周知チャンネルに追加されているか確認（`/invite @bot-name`）

### インシデントチャンネルが作成されない
- ボットに`channels:manage`権限が付与されているか確認
- Slackワークスペースのチャンネル作成制限を確認

### データベース接続エラー

**"connection refused" エラーの場合:**

PostgreSQLコンテナが起動しているか確認:
```bash
make ps
# または
docker-compose ps postgres
```

ローカル実行時に "dial tcp [::1]:5432: connect: connection refused" が出る場合は、config.tomlで `host = "127.0.0.1"` を使用してください（IPv6の問題を回避）:
```toml
[database]
host = "127.0.0.1"  # localhost ではなく 127.0.0.1 を使用
port = 5432
user = "postgres"
password = "postgres"
dbname = "incident_bot"
sslmode = "disable"
```

Docker実行時は環境変数で `DB_HOST=postgres` が設定されているため問題ありません。

**その他の確認項目:**
- PostgreSQLが起動しているか確認（`make status`）
- データベースとユーザーが存在するか確認
- `schema.sql`が適用されているか確認:
  ```bash
  make db-shell
  # psql内で:
  \dt  # テーブル一覧を表示
  ```

データベースなしでも動作しますが、インシデントハンドラー機能は利用できません。

### Dockerコンテナが起動しない
- Dockerが起動しているか確認（`docker ps`）
- ポート5432が既に使用されていないか確認（`lsof -i :5432`）
- ログを確認（`make logs`）

### ローカルからPostgreSQLに接続できない
- PostgreSQLコンテナが起動しているか確認（`make ps`）
- ポート5432がホストに公開されているか確認
- 接続情報を確認: `psql -h localhost -U postgres -d incident_bot`

## Docker構成

### サービス構成

- **postgres**: PostgreSQL 15データベース
  - ポート: 5432（ホストからアクセス可能）
  - ボリューム: postgres_data（データ永続化）
  - 自動初期化: schema.sqlを自動適用

- **bot**: インシデントレスポンスボット
  - PostgreSQLコンテナに自動接続
  - config.tomlまたは環境変数で設定
  - 自動再起動設定

### ボリューム

- `postgres_data`: PostgreSQLのデータを永続化

### ネットワーク

- `incident-bot-network`: コンテナ間通信用のブリッジネットワーク
