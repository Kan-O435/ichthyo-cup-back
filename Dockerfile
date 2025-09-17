# ビルドステージ
FROM golang:1.19 as builder

WORKDIR /app

# ソースコードをすべてコピー
COPY . .

# WebAssemblyとしてビルド (Goのソースファイルがclient/内にある場合)
# ここを修正して、正しいパスを指定
RUN GOARCH=wasm GOOS=js go build -o app.wasm ./client

# 本番ステージ
FROM nginx:alpine

# wasm_exec.jsをNginxのHTMLディレクトリにコピー
COPY --from=builder /usr/local/go/misc/wasm/wasm_exec.js /usr/share/nginx/html/

# ビルドしたWebAssemblyバイナリをコピー
COPY --from=builder /app/app.wasm /usr/share/nginx/html/

# index.htmlをコピー
# index.htmlもclient/内にあるため、パスを修正
COPY client/index.html /usr/share/nginx/html/

# Wplace風アプリをコピー
COPY client/wplace_leaflet.html /usr/share/nginx/html/

# カスタムNginx設定をコピー
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]