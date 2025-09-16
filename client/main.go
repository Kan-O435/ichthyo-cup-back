package main

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/prop"
)

// MapView コンポーネント
type MapView struct {
	vecty.Core
}

// Render 地図表示 - RenderBodyにはbody要素を返す必要がある
func (m *MapView) Render() vecty.ComponentOrHTML {
	// より安定したタイルURL
	tileURL := "https://tile.openstreetmap.org/3/7/3.png"

	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("padding", "0"),
			vecty.Style("font-family", "Arial, sans-serif"),
		),
		
		// メインコンテナ
		elem.Div(
			vecty.Markup(
				vecty.Style("display", "flex"),
				vecty.Style("flex-direction", "column"),
				vecty.Style("justify-content", "center"),
				vecty.Style("align-items", "center"),
				vecty.Style("min-height", "100vh"),
				vecty.Style("background-color", "#f0f0f0"),
				vecty.Style("padding", "20px"),
				vecty.Style("box-sizing", "border-box"),
			),
			
			// タイトル
			elem.Heading1(
				vecty.Text("Go + Vecty Map Demo"),
				vecty.Markup(
					vecty.Style("margin-bottom", "20px"),
					vecty.Style("color", "#333"),
					vecty.Style("text-align", "center"),
				),
			),

			// 地図コンテナ
			elem.Div(
				vecty.Markup(
					vecty.Style("border", "2px solid #333"),
					vecty.Style("border-radius", "8px"),
					vecty.Style("overflow", "hidden"),
					vecty.Style("box-shadow", "0 4px 6px rgba(0,0,0,0.1)"),
					vecty.Style("background", "white"),
				),
				elem.Image(
					vecty.Markup(
						prop.Src(tileURL),
						prop.Alt("OpenStreetMap Tile"),
						vecty.Style("display", "block"),
						vecty.Style("width", "256px"),
						vecty.Style("height", "256px"),
					),
				),
			),

			// 説明テキスト
			elem.Paragraph(
				vecty.Text("OpenStreetMapタイルの表示テスト"),
				vecty.Markup(
					vecty.Style("margin-top", "20px"),
					vecty.Style("color", "#666"),
					vecty.Style("font-size", "14px"),
					vecty.Style("text-align", "center"),
				),
			),

			// デバッグ情報
			elem.Paragraph(
				vecty.Text("地図が表示されない場合は、ブラウザの開発者ツールでエラーをチェックしてください。"),
				vecty.Markup(
					vecty.Style("margin-top", "10px"),
					vecty.Style("color", "#999"),
					vecty.Style("font-size", "12px"),
					vecty.Style("text-align", "center"),
					vecty.Style("max-width", "400px"),
				),
			),
		),
	)
}

func main() {
	// アプリケーションをHTMLの<body>要素にレンダリングします
	vecty.RenderBody(&HelloWorld{})
	/* components/login */
	vecty.SetTitle("Login Form")
	vecty.RenderBody(&components.Login{})
	// ブラウザのイベントループをブロックしないように、この関数をブロックします
	select {}
}