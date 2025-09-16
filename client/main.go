package main

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
)

// HelloWorld コンポーネント
type HelloWorld struct {
	vecty.Core
}

// Render はコンポーネントのHTML構造を定義します
func (h *HelloWorld) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("text-align", "center"),
		),
		elem.Heading1(
			vecty.Text("Hello, Go and Vecty!"),
		),
		elem.Paragraph(
			vecty.Text("This is a simple front-end application built with Go and Vecty."),
		),
	)
}

func main() {
	// アプリケーションをHTMLの<body>要素にレンダリングします
	vecty.RenderBody(&HelloWorld{})
	// ブラウザのイベントループをブロックしないように、この関数をブロックします
	select {}
}