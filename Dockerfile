# ビルドステージ
FROM golang:1.19-alpine as builder

WORKDIR /app

# wasm_exec.jsをコピー
COPY --from=golang:1.19 /usr/local/go/misc/wasm/wasm_exec.js .

# Go modulesファイルをコピー
COPY go.mod go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY . .

# 軽量地図用のWebAssemblyビルド
RUN GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o app.wasm ./client/main.go

# 本番ステージ
FROM nginx:alpine

# Nginxの最適化設定
COPY <<'EOF' /etc/nginx/conf.d/default.conf
server {
    listen 80;
    server_name localhost;
    
    # 圧縮設定（パフォーマンス向上）
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types
        text/css
        text/javascript
        text/xml
        text/plain
        application/javascript
        application/xml+rss
        application/json
        application/wasm;
    
    # キャッシュ設定
    location ~* \.(wasm|js)$ {
        expires 1d;
        add_header Cache-Control "public, immutable";
        add_header Cross-Origin-Embedder-Policy "require-corp";
        add_header Cross-Origin-Opener-Policy "same-origin";
    }
    
    # WASM用のMIMEタイプ設定
    location ~* \.wasm$ {
        add_header Content-Type "application/wasm";
    }
    
    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
}
EOF

# WebAssemblyとJSファイルをコピー
COPY --from=builder /app/app.wasm /usr/share/nginx/html/
COPY --from=builder /app/wasm_exec.js /usr/share/nginx/html/

# 最適化されたindex.htmlを作成
COPY <<'EOF' /usr/share/nginx/html/index.html
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>軽量地図表示</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            background: #1a1a1a;
            font-family: Arial, sans-serif;
        }
        
        #loading {
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            color: white;
            font-size: 18px;
            z-index: 9999;
        }
        
        .spinner {
            border: 3px solid rgba(255,255,255,0.3);
            border-radius: 50%;
            border-top: 3px solid white;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div id="loading">
        <div class="spinner"></div>
        <div>地図を読み込み中...</div>
    </div>

    <script src="wasm_exec.js"></script>
    <script>
        const go = new Go();
        
        // パフォーマンス最適化のためのプリロード
        const wasmUrl = 'app.wasm';
        
        // WebAssemblyの読み込みと実行
        async function loadWasm() {
            try {
                const result = await WebAssembly.instantiateStreaming(
                    fetch(wasmUrl), 
                    go.importObject
                );
                
                // ローディング画面を非表示
                document.getElementById('loading').style.display = 'none';
                
                // Goアプリケーション開始
                go.run(result.instance);
                
            } catch (err) {
                console.error('WebAssembly loading failed:', err);
                document.getElementById('loading').innerHTML = `
                    <div style="color: #ff6b6b;">
                        <h3>読み込みエラー</h3>
                        <p>地図の読み込みに失敗しました</p>
                        <button onclick="location.reload()" 
                                style="background: #4CAF50; color: white; border: none; 
                                       padding: 10px 20px; border-radius: 4px; cursor: pointer;">
                            再読み込み
                        </button>
                    </div>
                `;
            }
        }
        
        // ページ読み込み完了後にWebAssembly開始
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', loadWasm);
        } else {
            loadWasm();
        }
    </script>
</body>
</html>
EOF

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]