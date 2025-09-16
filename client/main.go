package main

import (
	"fmt"
	"strconv"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
	router "marwan.io/vecty-router" // app routing

	"ichthyo-cup-front/components" // login.go
)

// 地図コンポーネント
type MapDisplay struct {
	vecty.Core
	
	// 地図設定
	CenterLat   float64
	CenterLng   float64
	ZoomLevel   int
	TileSize    int
}

// 新しい地図を作成
func NewMapDisplay() *MapDisplay {
	return &MapDisplay{
		CenterLat: 34.6937,  // 大阪の緯度
		CenterLng: 135.5023, // 大阪の経度
		ZoomLevel: 10,       // ズームレベル
		TileSize:  256,      // タイルサイズ（標準）
	}
}

// OpenStreetMapのタイルURLを生成
func (m *MapDisplay) getTileUrl(x, y, z int) string {
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", z, x, y)
}

// 緯度経度からタイル座標を計算
func (m *MapDisplay) getTileCoordinates() (int, int) {
	// 簡易的な計算（正確には数学的変換が必要）
	// ズームレベルに基づいてタイル座標を決定
	zoomPower := 1 << uint(m.ZoomLevel)
	x := int((m.CenterLng + 180.0) * float64(zoomPower) / 360.0)
	
	zoomPowerMinusOne := 1 << uint(m.ZoomLevel-1)
	y := int((1.0 - (m.CenterLat*3.14159/180.0)) * float64(zoomPowerMinusOne) / 3.14159)
	return x, y
}

// ズームイン
func (m *MapDisplay) zoomIn() {
	if m.ZoomLevel < 18 {
		m.ZoomLevel++
		vecty.Rerender(m)
	}
}

// ズームアウト
func (m *MapDisplay) zoomOut() {
	if m.ZoomLevel > 1 {
		m.ZoomLevel--
		vecty.Rerender(m)
	}
}

// 地図を北に移動
func (m *MapDisplay) moveNorth() {
	m.CenterLat += 0.01
	vecty.Rerender(m)
}

// 地図を南に移動
func (m *MapDisplay) moveSouth() {
	m.CenterLat -= 0.01
	vecty.Rerender(m)
}

// 地図を東に移動
func (m *MapDisplay) moveEast() {
	m.CenterLng += 0.01
	vecty.Rerender(m)
}

// 地図を西に移動
func (m *MapDisplay) moveWest() {
	m.CenterLng -= 0.01
	vecty.Rerender(m)
}

// レンダリング
func (m *MapDisplay) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("padding", "0"),
			vecty.Style("font-family", "Arial, sans-serif"),
			vecty.Style("background", "#f0f0f0"),
		),
		
		// ヘッダー
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "#2C3E50"),
				vecty.Style("color", "white"),
				vecty.Style("padding", "15px"),
				vecty.Style("text-align", "center"),
			),
			elem.Heading1(
				vecty.Text("🗺️ 地図表示アプリ"),
				vecty.Markup(
					vecty.Style("margin", "0"),
					vecty.Style("font-size", "24px"),
				),
			),
		),
		
		// メインコンテンツ
		elem.Div(
			vecty.Markup(
				vecty.Style("display", "flex"),
				vecty.Style("height", "calc(100vh - 80px)"),
			),
			
			// 地図表示エリア
			elem.Div(
				vecty.Markup(
					vecty.Style("flex", "1"),
					vecty.Style("position", "relative"),
					vecty.Style("background", "#34495E"),
					vecty.Style("display", "flex"),
					vecty.Style("justify-content", "center"),
					vecty.Style("align-items", "center"),
				),
				m.renderMapTiles(),
			),
			
			// 操作パネル
			elem.Div(
				vecty.Markup(
					vecty.Style("width", "300px"),
					vecty.Style("background", "#ECF0F1"),
					vecty.Style("padding", "20px"),
					vecty.Style("overflow-y", "auto"),
				),
				m.renderControlPanel(),
			),
		),
	)
}

// 地図タイルの表示
func (m *MapDisplay) renderMapTiles() vecty.ComponentOrHTML {
	// 中心タイルの座標を取得
	centerX, centerY := m.getTileCoordinates()
	
	// 3x3のタイル表示（中心を囲む9枚のタイル）
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "relative"),
			vecty.Style("width", strconv.Itoa(m.TileSize*3)+"px"),
			vecty.Style("height", strconv.Itoa(m.TileSize*3)+"px"),
			vecty.Style("border", "2px solid #2C3E50"),
			vecty.Style("box-shadow", "0 4px 8px rgba(0,0,0,0.3)"),
		),
		
		// 9枚のタイルを配置
		m.createTile(centerX-1, centerY-1, 0, 0), // 左上
		m.createTile(centerX, centerY-1, 1, 0),   // 中上
		m.createTile(centerX+1, centerY-1, 2, 0), // 右上
		m.createTile(centerX-1, centerY, 0, 1),   // 左中
		m.createTile(centerX, centerY, 1, 1),     // 中心
		m.createTile(centerX+1, centerY, 2, 1),   // 右中
		m.createTile(centerX-1, centerY+1, 0, 2), // 左下
		m.createTile(centerX, centerY+1, 1, 2),   // 中下
		m.createTile(centerX+1, centerY+1, 2, 2), // 右下
	)
}

// 個別タイルを作成
func (m *MapDisplay) createTile(tileX, tileY, gridX, gridY int) vecty.ComponentOrHTML {
	x := gridX * m.TileSize
	y := gridY * m.TileSize
	tileUrl := m.getTileUrl(tileX, tileY, m.ZoomLevel)
	
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", strconv.Itoa(x)+"px"),
			vecty.Style("top", strconv.Itoa(y)+"px"),
			vecty.Style("width", strconv.Itoa(m.TileSize)+"px"),
			vecty.Style("height", strconv.Itoa(m.TileSize)+"px"),
		),
		elem.Image(
			vecty.Markup(
				prop.Src(tileUrl),
				prop.Alt(fmt.Sprintf("Map tile %d,%d", tileX, tileY)),
				vecty.Style("width", "100%"),
				vecty.Style("height", "100%"),
				vecty.Style("display", "block"),
			),
		),
	)
}

// 操作パネルの表示
func (m *MapDisplay) renderControlPanel() vecty.ComponentOrHTML {
	return elem.Div(
		elem.Heading2(
			vecty.Text("🎮 地図操作"),
			vecty.Markup(
				vecty.Style("margin", "0 0 20px 0"),
				vecty.Style("color", "#2C3E50"),
			),
		),
		
		// 現在位置情報
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "white"),
				vecty.Style("padding", "15px"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("margin-bottom", "20px"),
				vecty.Style("box-shadow", "0 2px 4px rgba(0,0,0,0.1)"),
			),
			elem.Heading3(
				vecty.Text("📍 現在位置"),
				vecty.Markup(
					vecty.Style("margin", "0 0 10px 0"),
					vecty.Style("font-size", "16px"),
				),
			),
			elem.Paragraph(
				vecty.Text(fmt.Sprintf("緯度: %.4f", m.CenterLat)),
				vecty.Markup(
					vecty.Style("margin", "5px 0"),
					vecty.Style("font-size", "14px"),
				),
			),
			elem.Paragraph(
				vecty.Text(fmt.Sprintf("経度: %.4f", m.CenterLng)),
				vecty.Markup(
					vecty.Style("margin", "5px 0"),
					vecty.Style("font-size", "14px"),
				),
			),
			elem.Paragraph(
				vecty.Text(fmt.Sprintf("ズーム: %d", m.ZoomLevel)),
				vecty.Markup(
					vecty.Style("margin", "5px 0"),
					vecty.Style("font-size", "14px"),
				),
			),
		),
		
		// ズーム操作
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "white"),
				vecty.Style("padding", "15px"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("margin-bottom", "20px"),
				vecty.Style("box-shadow", "0 2px 4px rgba(0,0,0,0.1)"),
			),
			elem.Heading3(
				vecty.Text("🔍 ズーム"),
				vecty.Markup(
					vecty.Style("margin", "0 0 15px 0"),
					vecty.Style("font-size", "16px"),
				),
			),
			elem.Button(
				vecty.Text("➕ 拡大"),
				vecty.Markup(
					vecty.Style("width", "100%"),
					vecty.Style("padding", "10px"),
					vecty.Style("margin-bottom", "10px"),
					vecty.Style("background", "#3498DB"),
					vecty.Style("color", "white"),
					vecty.Style("border", "none"),
					vecty.Style("border-radius", "5px"),
					vecty.Style("cursor", "pointer"),
					vecty.Style("font-size", "16px"),
					event.Click(func(e *vecty.Event) {
						m.zoomIn()
					}),
				),
			),
			elem.Button(
				vecty.Text("➖ 縮小"),
				vecty.Markup(
					vecty.Style("width", "100%"),
					vecty.Style("padding", "10px"),
					vecty.Style("background", "#E74C3C"),
					vecty.Style("color", "white"),
					vecty.Style("border", "none"),
					vecty.Style("border-radius", "5px"),
					vecty.Style("cursor", "pointer"),
					vecty.Style("font-size", "16px"),
					event.Click(func(e *vecty.Event) {
						m.zoomOut()
					}),
				),
			),
		),
		
		// 移動操作
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "white"),
				vecty.Style("padding", "15px"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("margin-bottom", "20px"),
				vecty.Style("box-shadow", "0 2px 4px rgba(0,0,0,0.1)"),
			),
			elem.Heading3(
				vecty.Text("🧭 移動"),
				vecty.Markup(
					vecty.Style("margin", "0 0 15px 0"),
					vecty.Style("font-size", "16px"),
				),
			),
			
			// 上移動ボタン
			elem.Div(
				vecty.Markup(
					vecty.Style("text-align", "center"),
					vecty.Style("margin-bottom", "10px"),
				),
				elem.Button(
					vecty.Text("⬆️ 北"),
					vecty.Markup(
						vecty.Style("padding", "8px 16px"),
						vecty.Style("background", "#27AE60"),
						vecty.Style("color", "white"),
						vecty.Style("border", "none"),
						vecty.Style("border-radius", "5px"),
						vecty.Style("cursor", "pointer"),
						event.Click(func(e *vecty.Event) {
							m.moveNorth()
						}),
					),
				),
			),
			
			// 左右移動ボタン
			elem.Div(
				vecty.Markup(
					vecty.Style("display", "flex"),
					vecty.Style("justify-content", "space-between"),
					vecty.Style("margin-bottom", "10px"),
				),
				elem.Button(
					vecty.Text("⬅️ 西"),
					vecty.Markup(
						vecty.Style("padding", "8px 16px"),
						vecty.Style("background", "#27AE60"),
						vecty.Style("color", "white"),
						vecty.Style("border", "none"),
						vecty.Style("border-radius", "5px"),
						vecty.Style("cursor", "pointer"),
						event.Click(func(e *vecty.Event) {
							m.moveWest()
						}),
					),
				),
				elem.Button(
					vecty.Text("➡️ 東"),
					vecty.Markup(
						vecty.Style("padding", "8px 16px"),
						vecty.Style("background", "#27AE60"),
						vecty.Style("color", "white"),
						vecty.Style("border", "none"),
						vecty.Style("border-radius", "5px"),
						vecty.Style("cursor", "pointer"),
						event.Click(func(e *vecty.Event) {
							m.moveEast()
						}),
					),
				),
			),
			
			// 下移動ボタン
			elem.Div(
				vecty.Markup(
					vecty.Style("text-align", "center"),
				),
				elem.Button(
					vecty.Text("⬇️ 南"),
					vecty.Markup(
						vecty.Style("padding", "8px 16px"),
						vecty.Style("background", "#27AE60"),
						vecty.Style("color", "white"),
						vecty.Style("border", "none"),
						vecty.Style("border-radius", "5px"),
						vecty.Style("cursor", "pointer"),
						event.Click(func(e *vecty.Event) {
							m.moveSouth()
						}),
					),
				),
			),
		),
		
		// 情報表示
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "#D5DBDB"),
				vecty.Style("padding", "10px"),
				vecty.Style("border-radius", "5px"),
				vecty.Style("font-size", "12px"),
			),
			elem.Paragraph(
				vecty.Text("🌍 OpenStreetMap使用"),
				vecty.Markup(
					vecty.Style("margin", "0"),
				),
			),
		),
	)
}

func main() {
	vecty.SetTitle("地図表示アプリ")
	vecty.RenderBody(&App{})
	select {}
}

// App は アプリケーションのルートコンポーネント
type App struct {
	vecty.Core
}

// Render はアプリケーションのルーティングを定義
func (a *App) Render() vecty.ComponentOrHTML {
	return elem.Body(
		router.NewRoute("/", NewMapDisplay(), router.NewRouteOpts{ExactMatch: true}),
		router.NewRoute("/login", &components.Login{}, router.NewRouteOpts{ExactMatch: true}),
		router.NotFoundHandler(&NotFound{}),
	)
}

// NotFound は 404ページのコンポーネント
type NotFound struct {
	vecty.Core
}

// Render は 404ページをレンダリング
func (nf *NotFound) Render() vecty.ComponentOrHTML {
	return elem.Div(
		elem.Heading1(vecty.Text("404 - Page Not Found")),
		elem.Paragraph(vecty.Text("申し訳ありませんが、お探しのページが見つかりません。")),
		router.Link("/", "ホームに戻る", router.LinkOptions{}),
	)
}