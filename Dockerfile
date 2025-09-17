# ビルドステージ
FROM golang:1.19 as builder

WORKDIR /app

# ソースコードをすべてコピー
COPY . .

# WebAssemblyとしてビルド
RUN GOARCH=wasm GOOS=js go build -o app.wasm ./client

# 本番ステージ - 非rootユーザー対応
FROM nginx:alpine

# 非rootユーザーで実行するための準備
RUN mkdir -p /tmp/nginx/cache /tmp/nginx/logs && \
    chown -R nginx:nginx /tmp/nginx && \
    chmod -R 755 /tmp/nginx

# nginxのディレクトリ権限設定
RUN touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid && \
    chown -R nginx:nginx /etc/nginx && \
    chown -R nginx:nginx /usr/share/nginx/html

# wasm_exec.jsをコピー
COPY --from=builder /usr/local/go/misc/wasm/wasm_exec.js /usr/share/nginx/html/

# ビルドしたWebAssemblyバイナリをコピー
COPY --from=builder /app/app.wasm /usr/share/nginx/html/

# HTMLファイルをコピー
COPY client/index.html /usr/share/nginx/html/
COPY client/wplace_leaflet.html /usr/share/nginx/html/

# Hugging Face Spaces用nginx設定をコピー
COPY nginx.conf /etc/nginx/nginx.conf

# 非rootユーザーで実行
USER nginx

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]