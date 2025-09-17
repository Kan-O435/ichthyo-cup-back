# ビルドステージ
FROM golang:1.19 as builder

WORKDIR /app
COPY . .

# WebAssemblyとしてビルド
RUN GOARCH=wasm GOOS=js go build -o app.wasm ./client

# 本番ステージ - シンプルなHTTPサーバーを使用
FROM python:3.9-slim

WORKDIR /app

# 必要なファイルをコピー
COPY --from=builder /usr/local/go/misc/wasm/wasm_exec.js ./
COPY --from=builder /app/app.wasm ./
COPY client/index.html ./
COPY client/wplace_leaflet.html ./

# シンプルなHTTPサーバーでファイルを提供
EXPOSE 8080

CMD ["python", "-m", "http.server", "8080"]