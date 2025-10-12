# Kubernetes マニフェスト

このディレクトリには、Incident Response BotをKubernetesクラスタにデプロイするためのマニフェストファイルが含まれています。

## 構成ファイル

- `deployment.yaml` - Incident Response Botのデプロイメント設定
- `configmap.yaml` - アプリケーション設定（データベース接続情報など）
- `secret.yaml` - 機密情報（Slackトークン、データベース認証情報）
- `postgres-statefulset.yaml` - PostgreSQLのStatefulSet設定
- `postgres-service.yaml` - PostgreSQLのService設定
- `postgres-init-configmap.yaml` - PostgreSQL初期化スクリプト

## デプロイ手順

### 1. Secretの設定

`secret.yaml` を編集して、実際のトークンとパスワードを設定してください。

```yaml
# secret.yaml の編集が必要な箇所
stringData:
  # Slackのボットトークンとアプリトークン
  bot-token: "xoxb-your-actual-bot-token"
  app-token: "xapp-your-actual-app-token"

  # PostgreSQLの認証情報
  username: "postgres"
  password: "your-secure-password"
```

### 2. ConfigMapの確認

必要に応じて `configmap.yaml` の設定値を確認・変更してください。

```yaml
data:
  db-host: "postgres-service"
  db-port: "5432"
  db-name: "incidents"
  db-sslmode: "disable"
  timezone: "Asia/Tokyo"
```

### 3. Kubernetesクラスタへのデプロイ

以下のコマンドで全てのリソースをデプロイします。

```bash
# manifests ディレクトリに移動
cd manifests

# 全てのマニフェストを適用
kubectl apply -f .

# または、個別に適用する場合
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f postgres-init-configmap.yaml
kubectl apply -f postgres-service.yaml
kubectl apply -f postgres-statefulset.yaml
kubectl apply -f deployment.yaml
```

### 4. デプロイの確認

```bash
# Pod の状態を確認
kubectl get pods

# Deployment の状態を確認
kubectl get deployment incident-response-bot

# StatefulSet の状態を確認
kubectl get statefulset postgres

# ログを確認
kubectl logs -f deployment/incident-response-bot
kubectl logs -f statefulset/postgres
```

## リソース構成

### Incident Response Bot

- **イメージ**: `ryuichi1208/incident-response-bot:latest`
- **レプリカ数**: 1
- **リソース要求**:
  - CPU: 100m
  - メモリ: 128Mi
- **リソース制限**:
  - CPU: 500m
  - メモリ: 256Mi

### PostgreSQL

- **イメージ**: `postgres:15-alpine`
- **レプリカ数**: 1
- **ストレージ**: 10Gi (PersistentVolumeClaim)
- **リソース要求**:
  - CPU: 250m
  - メモリ: 256Mi
- **リソース制限**:
  - CPU: 500m
  - メモリ: 512Mi

## トラブルシューティング

### Podが起動しない場合

```bash
# Pod の詳細を確認
kubectl describe pod <pod-name>

# イベントを確認
kubectl get events --sort-by='.lastTimestamp'
```

### データベース接続エラーの場合

```bash
# PostgreSQL のログを確認
kubectl logs statefulset/postgres

# PostgreSQL に直接接続して確認
kubectl exec -it postgres-0 -- psql -U postgres -d incidents
```

### Secret が正しく設定されているか確認

```bash
# Secret の内容を確認（base64エンコードされた値が表示されます）
kubectl get secret slack-secrets -o yaml
kubectl get secret postgres-secrets -o yaml

# デコードして確認
kubectl get secret slack-secrets -o jsonpath='{.data.bot-token}' | base64 -d
```

## アンデプロイ

```bash
# 全てのリソースを削除
kubectl delete -f .

# または、個別に削除
kubectl delete deployment incident-response-bot
kubectl delete statefulset postgres
kubectl delete service postgres-service
kubectl delete configmap incident-bot-config postgres-init-scripts
kubectl delete secret slack-secrets postgres-secrets
kubectl delete pvc postgres-storage-postgres-0
```

## 注意事項

- PostgreSQL の PersistentVolumeClaim は自動的に削除されません。データを完全に削除する場合は、手動で削除してください。
- 本番環境では、Secretをコードリポジトリにコミットせず、別の方法（Sealed Secrets、外部のシークレット管理システムなど）で管理してください。
- リソース要求と制限は、環境に応じて調整してください。
- データベースのバックアップを定期的に取得することを推奨します。

## セキュリティ考慮事項

- `secret.yaml` は `.gitignore` に追加し、リポジトリにコミットしないでください
- 本番環境では以下の対策を検討してください：
  - Sealed Secrets や External Secrets Operator の使用
  - RBAC による適切なアクセス制御
  - Network Policy による通信制限
  - Pod Security Policy/Pod Security Standards の適用
  - イメージの脆弱性スキャン
  - 定期的なセキュリティパッチの適用
