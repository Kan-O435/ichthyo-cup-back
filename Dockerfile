# ビルドステージ
FROM golang:1.19 as builder

WORKDIR /app

# wasm_exec.jsをコピー
COPY --from=golang:1.19 /usr/local/go/misc/wasm/wasm_exec.js .

# Go modulesファイルをコピー
COPY go.mod ./
COPY go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY . .

# WebAssemblyとしてビルド (main.goがclient/内にある場合)
RUN GOARCH=wasm GOOS=js go build -o app.wasm ./client/main.go

# 本番ステージ
FROM nginx:alpine

# 静的ファイルをコピー
COPY --from=builder /app/app.wasm /usr/share/nginx/html/
COPY --from=builder /app/wasm_exec.js /usr/share/nginx/html/

# index.htmlを作成またはコピー
COPY client/index.html /usr/share/nginx/html/

# カスタムNginx設定をコピー
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]