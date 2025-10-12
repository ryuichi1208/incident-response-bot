# ビルドステージ
FROM golang:1.21-alpine AS builder

# 必要なパッケージをインストール
RUN apk add --no-cache git ca-certificates tzdata

# 作業ディレクトリを設定
WORKDIR /app

# 依存関係をコピーしてダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# バイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o incident-bot .

# 実行ステージ
FROM alpine:latest

# 必要なパッケージをインストール
RUN apk --no-cache add ca-certificates tzdata

# タイムゾーンを設定
ENV TZ=Asia/Tokyo

# 作業ディレクトリを設定
WORKDIR /root/

# ビルドステージからバイナリをコピー
COPY --from=builder /app/incident-bot .
COPY --from=builder /app/config.toml.example .

# ポートの公開は不要（Slack Socket Modeを使用）

# ヘルスチェック用のラベル
LABEL maintainer="ryuichi1208"
LABEL description="Slack Incident Response Bot"

# 実行
CMD ["./incident-bot"]
