# Stage 1: ビルドステージ
FROM golang:1.19 AS builder

# 作業ディレクトリの設定
WORKDIR /app

# Goのツールチェーンからwasm_exec.jsをコピー
COPY --from=golang:1.19 /usr/local/go/misc/wasm/wasm_exec.js ./client/

# Goモジュールをキャッシュするために、先にgo.modとgo.sumをコピー
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# go.modの依存関係にあるVectyをビルド
RUN GOARCH=wasm GOOS=js go build -o ./client/app.wasm ./client/main.go

# 静的HTMLファイルを作成
RUN echo '<!DOCTYPE html><html><head><meta charset="utf-8"><title>Go + Vecty App</title></head><body><script src="wasm_exec.js"></script><script>const go = new Go();WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then((result) => {go.run(result.instance);});</script></body></html>' > ./client/index.html

# ---
# Stage 2: 実行ステージ
# ここでは軽量なnginxコンテナを使用
FROM nginx:alpine

# ビルドステージで作成したファイルをコピー
COPY --from=builder /app/client /usr/share/nginx/html

# nginxを起動
CMD ["nginx", "-g", "daemon off;"]