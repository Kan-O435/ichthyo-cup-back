package main

import (
	"fmt"
	"math"
	"strconv"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

// OpenFreeMapベースの地図コンポーネント
type OpenFreeMap struct {
	vecty.Core

	// 地図設定
	CenterLat float64
	CenterLng float64
	ZoomLevel int
	TileSize  int

	// UI状態
	IsControlsVisible bool
	// タイルプロバイダー
	TileProvider int // 0: OSM, 1: OpenFreeMap, 2: CartoDB

	// ドラッグ状態
	IsDragging bool
	DragStartX int
	DragStartY int
	LastDragX  int
	LastDragY  int
	mapOffsetX float64
	mapOffsetY float64

}

// 新しい地図を作成
func NewOpenFreeMap() *OpenFreeMap {
	return &OpenFreeMap{
		CenterLat:         35.6762, // 東京
		CenterLng:         139.6503,
		ZoomLevel:         10,
		TileSize:          256,
		IsControlsVisible: true,
		TileProvider:      0, // デフォルトはOSM（確実に動作）
		IsDragging:        false,
		DragStartX:        0,
		DragStartY:        0,
		LastDragX:         0,
		LastDragY:         0,
	}
}

// 動作確認済みのプロバイダーのみ
func (m *OpenFreeMap) getMapEndpoint() string {
	switch m.TileProvider {
	case 1: // CartoDB Positron（軽量・シンプル）
		return "https://cartodb-basemaps-a.global.ssl.fastly.net/light_all"
	default: // 標準OpenStreetMap（デフォルト・確実）
		return "https://tile.openstreetmap.org"
	}
}

// タイル座標計算
func (m *OpenFreeMap) latLngToTile(lat, lng float64, zoom int) (int, int) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, float64(zoom))

	x := int(math.Floor((lng + 180.0) / 360.0 * n))
	y := int(math.Floor((1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n))

	// 範囲制限
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	maxTile := int(n) - 1
	if x > maxTile {
		x = maxTile
	}
	if y > maxTile {
		y = maxTile
	}

	return x, y
}

// ドラッグ終了時にオフセットを緯度経度に変換して中心を更新
func (m *OpenFreeMap) updateCenterFromOffset() {
	if m.mapOffsetX == 0 && m.mapOffsetY == 0 {
		return
	}

	// 現在の中心をワールド座標(0.0-1.0)に変換
	n := math.Pow(2.0, float64(m.ZoomLevel))
	x := (m.CenterLng + 180.0) / 360.0 * n
	latRad := m.CenterLat * math.Pi / 180
	y := (1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n

	// ドラッグによるピクセルオフセットをワールド座標の差分に変換
	newX := x - m.mapOffsetX/float64(m.TileSize)
	newY := y - m.mapOffsetY/float64(m.TileSize)

	// 新しいワールド座標から新しい緯度経度を計算
	m.CenterLng = newX/n*360.0 - 180.0
	latRad = math.Atan(math.Sinh(math.Pi * (1 - 2*newY/n)))
	m.CenterLat = latRad * 180.0 / math.Pi

	// ピクセルオフセットをリセット
	m.mapOffsetX = 0
	m.mapOffsetY = 0
}

// ズーム操作
func (m *OpenFreeMap) zoomIn() {
	if m.ZoomLevel < 18 {
		m.ZoomLevel++
		vecty.Rerender(m)
	}
}

func (m *OpenFreeMap) zoomOut() {
	if m.ZoomLevel > 1 {
		m.ZoomLevel--
		vecty.Rerender(m)
	}
}

// 地図移動（ドラッグ対応）
func (m *OpenFreeMap) panMap(deltaX, deltaY float64) {
	// ピクセル移動量を緯度経度の変化に変換
	n := math.Pow(2.0, float64(m.ZoomLevel))
	
	// 移動量の計算（地図の投影法に応じた変換）
	lngDelta := (deltaX / float64(m.TileSize)) * 360.0 / n
	latDelta := -(deltaY / float64(m.TileSize)) * 360.0 / n // Y軸反転
	
	// 新しい中心座標を設定
	m.CenterLng += lngDelta
	m.CenterLat += latDelta
	
	// 座標の範囲制限
	if m.CenterLat > 85.0511 {
		m.CenterLat = 85.0511
	}
	if m.CenterLat < -85.0511 {
		m.CenterLat = -85.0511
	}
	if m.CenterLng > 180.0 {
		m.CenterLng = 180.0
	}
	if m.CenterLng < -180.0 {
		m.CenterLng = -180.0
	}
}

// ボタンでの地図移動（既存の機能）
func (m *OpenFreeMap) panNorth() { m.CenterLat += 0.01; vecty.Rerender(m) }
func (m *OpenFreeMap) panSouth() { m.CenterLat -= 0.01; vecty.Rerender(m) }
func (m *OpenFreeMap) panEast()  { m.CenterLng += 0.01; vecty.Rerender(m) }
func (m *OpenFreeMap) panWest()  { m.CenterLng -= 0.01; vecty.Rerender(m) }

// コントロール表示切替
func (m *OpenFreeMap) toggleControls() {
	m.IsControlsVisible = !m.IsControlsVisible
	vecty.Rerender(m)
}

// マウス/タッチパッドイベントハンドラー
func (m *OpenFreeMap) handleMouseDown(e *vecty.Event) {
	m.IsDragging = true
	m.DragStartX = e.Get("clientX").Int()
	m.DragStartY = e.Get("clientY").Int()
	m.LastDragX = m.DragStartX
	m.LastDragY = m.DragStartY
	
	// ドラッグ中はテキスト選択を防ぐ
	e.Call("preventDefault")
}

func (m *OpenFreeMap) handleMouseMove(e *vecty.Event) {
	if !m.IsDragging {
		return
	}
	
	currentX := e.Get("clientX").Int()
	currentY := e.Get("clientY").Int()
	
	// 前回位置からの移動量を計算
	deltaX := float64(currentX - m.LastDragX)
	deltaY := float64(currentY - m.LastDragY)
	
	// 地図を移動（ドラッグ方向と逆に移動）
	m.panMap(-deltaX, -deltaY)
	
	// 現在位置を記録
	m.LastDragX = currentX
	m.LastDragY = currentY
	
	// 再描画
	vecty.Rerender(m)
	
	e.Call("preventDefault")
}

func (m *OpenFreeMap) handleMouseUp(e *vecty.Event) {
	m.IsDragging = false
	e.Call("preventDefault")
}

func (m *OpenFreeMap) handleMouseLeave(e *vecty.Event) {
	// マウスが地図エリアから出た場合もドラッグを終了
	m.IsDragging = false
}

// ホイールでのズーム操作
func (m *OpenFreeMap) handleWheel(e *vecty.Event) {
	e.Call("preventDefault")
	
	deltaY := e.Get("deltaY").Float()
	if deltaY < 0 {
		m.zoomIn()
	} else {
		m.zoomOut()
	}
}

// メインレンダリング
func (m *OpenFreeMap) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("padding", "0"),
			vecty.Style("overflow", "hidden"),
			vecty.Style("background", "#1a1a1a"),
			vecty.Style("font-family", "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"),
			vecty.Style("user-select", "none"),
		),

		// 全画面地図コンテナ（ドラッグ操作対応）
		elem.Div(
			vecty.Markup(
				vecty.Style("position", "fixed"),
				vecty.Style("top", "0"),
				vecty.Style("left", "0"),
				vecty.Style("width", "100vw"),
				vecty.Style("height", "100vh"),
				vecty.Style("cursor", func() string {
					if m.IsDragging { return "grabbing" }
					return "grab"
				}()),
				vecty.Style("user-select", "none"), // テキスト選択を防ぐ
				
				// マウス/タッチパッドイベント
				event.MouseDown(m.handleMouseDown),
				event.MouseMove(m.handleMouseMove), 
				event.MouseUp(m.handleMouseUp),
				event.MouseLeave(m.handleMouseLeave),
				event.Wheel(m.handleWheel),
			),

			// OpenFreeMapベースの地図レイヤー
			m.renderOpenFreeMapLayer(),

			// UI コントロール
			m.renderZoomControls(),
			m.renderCoordinateInfo(),
			m.renderControlToggle(),
		),

		// サイドパネル
		m.renderConditionalSidePanel(),
	)
}

// OpenFreeMapレイヤー（ドラッグ＆フルスクリーン対応）
func (m *OpenFreeMap) renderOpenFreeMapLayer() vecty.ComponentOrHTML {
	// 画面サイズを取得
	screenWidth := js.Global().Get("innerWidth").Int()
	screenHeight := js.Global().Get("innerHeight").Int()

	// 画面を埋めるのに必要なタイル数を計算 (少し余分に描画)
	gridWidth := (screenWidth / m.TileSize) + 3
	gridHeight := (screenHeight / m.TileSize) + 3
	// 常に奇数にして中心を明確にする
	if gridWidth%2 == 0 {
		gridWidth++
	}
	if gridHeight%2 == 0 {
		gridHeight++
	}

	// 中心タイル座標
	centerX, centerY := m.latLngToTile(m.CenterLat, m.CenterLng, m.ZoomLevel)
	baseEndpoint := m.getMapEndpoint()

	// タイル要素のスライスを作成
	var tiles vecty.List
	for y := -gridHeight / 2; y <= gridHeight/2; y++ {
		for x := -gridWidth / 2; x <= gridWidth/2; x++ {
			tileX := centerX + x
			tileY := centerY + y
			gridX := x + gridWidth/2
			gridY := y + gridHeight/2
			tiles = append(tiles, m.createTileWithHref(baseEndpoint, tileX, tileY, gridX, gridY))
		}
	}

		// ドラッグイベントハンドラ
	onMouseDown := func(e *vecty.Event) {
		e.Value.Call("preventDefault")
		m.IsDragging = true
		m.DragStartX = e.Value.Get("clientX").Int()
		m.DragStartY = e.Value.Get("clientY").Int()
		// ドラッグ開始時にカーソルを変更
		e.Value.Get("target").Get("style").Set("cursor", "grabbing")
	}

	onMouseMove := func(e *vecty.Event) {
		if m.IsDragging {
			e.Value.Call("preventDefault")
			clientX := e.Value.Get("clientX").Int()
			clientY := e.Value.Get("clientY").Int()
			dx := clientX - m.DragStartX
			dy := clientY - m.DragStartY
			m.mapOffsetX += float64(dx)
			m.mapOffsetY += float64(dy)
			m.DragStartX = clientX
			m.DragStartY = clientY
			vecty.Rerender(m)
		}
	}

	onMouseUpOrLeave := func(e *vecty.Event) {
		if m.IsDragging {
			e.Value.Call("preventDefault")
			m.IsDragging = false
			// ドラッグ終了時にカーソルを戻す
			e.Value.Get("target").Get("style").Set("cursor", "grab")
			m.updateCenterFromOffset()
			vecty.Rerender(m)
		}
	}

	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("top", "0"),
			vecty.Style("left", "0"),
			vecty.Style("width", "100%"),
			vecty.Style("height", "100%"),
			vecty.Style("display", "flex"),
			vecty.Style("justify-content", "center"),
			vecty.Style("align-items", "center"),
			vecty.Style("cursor", "grab"), // 初期カーソル
			event.MouseDown(onMouseDown),
			event.MouseMove(onMouseMove),
			event.MouseUp(onMouseUpOrLeave),
			event.MouseLeave(onMouseUpOrLeave),
		),

		// タイル配置コンテナ
		elem.Div(
			vecty.Markup(
				vecty.Style("position", "relative"),
				vecty.Style("width", strconv.Itoa(m.TileSize*gridWidth)+"px"),
				vecty.Style("height", strconv.Itoa(m.TileSize*gridHeight)+"px"),
				// ドラッグ中のオフセットを適用
				vecty.Style("transform", fmt.Sprintf("translate(%.3fpx, %.3fpx)", m.mapOffsetX, m.mapOffsetY)),
			),
			tiles,
		),
	)
}

// href属性を使用したタイル作成
func (m *OpenFreeMap) createTileWithHref(baseEndpoint string, tileX, tileY, gridX, gridY int) vecty.ComponentOrHTML {
	screenX := gridX * m.TileSize
	screenY := gridY * m.TileSize

	tileHref := fmt.Sprintf("%s/%d/%d/%d.png", baseEndpoint, m.ZoomLevel, tileX, tileY)

	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", strconv.Itoa(screenX)+"px"),
			vecty.Style("top", strconv.Itoa(screenY)+"px"),
			vecty.Style("width", strconv.Itoa(m.TileSize)+"px"),
			vecty.Style("height", strconv.Itoa(m.TileSize)+"px"),
		),

		elem.Image(
			vecty.Markup(
				prop.Src(tileHref),
				prop.Alt(fmt.Sprintf("Tile %d,%d", tileX, tileY)),
				vecty.Style("width", "100%"),
				vecty.Style("height", "100%"),
				vecty.Style("display", "block"),
				vecty.Style("image-rendering", "pixelated"),
				vecty.Style("border", "1px solid rgba(255,255,255,0.1)"),
			),
		),
	)
}

// ズームコントロール
func (m *OpenFreeMap) renderZoomControls() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "20px"),
			vecty.Style("left", "20px"),
			vecty.Style("background", "rgba(26, 26, 26, 0.9)"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("padding", "8px"),
			vecty.Style("display", "flex"),
			vecty.Style("flex-direction", "column"),
			vecty.Style("gap", "4px"),
		),

		elem.Button(
			vecty.Text("+"),
			vecty.Markup(
				vecty.Style("width", "40px"),
				vecty.Style("height", "40px"),
				vecty.Style("background", "rgba(255, 255, 255, 0.1)"),
				vecty.Style("border", "none"),
				vecty.Style("border-radius", "4px"),
				vecty.Style("color", "white"),
				vecty.Style("font-size", "20px"),
				vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) { m.zoomIn() }),
			),
		),

		elem.Button(
			vecty.Text("−"),
			vecty.Markup(
				vecty.Style("width", "40px"),
				vecty.Style("height", "40px"),
				vecty.Style("background", "rgba(255, 255, 255, 0.1)"),
				vecty.Style("border", "none"),
				vecty.Style("border-radius", "4px"),
				vecty.Style("color", "white"),
				vecty.Style("font-size", "20px"),
				vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) { m.zoomOut() }),
			),
		),
	)
}

// 座標情報とデバッグ情報
func (m *OpenFreeMap) renderCoordinateInfo() vecty.ComponentOrHTML {
	centerX, centerY := m.latLngToTile(m.CenterLat, m.CenterLng, m.ZoomLevel)
	endpoint := m.getMapEndpoint()

	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("bottom", "20px"),
			vecty.Style("right", "20px"),
			vecty.Style("background", "rgba(26, 26, 26, 0.9)"),
			vecty.Style("color", "white"),
			vecty.Style("padding", "12px 16px"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("font-size", "11px"),
			vecty.Style("font-family", "Monaco, monospace"),
			vecty.Style("max-width", "300px"),
		),

		elem.Div(vecty.Text(fmt.Sprintf("Lat/Lng: %.6f, %.6f", m.CenterLat, m.CenterLng))),
		elem.Div(vecty.Text(fmt.Sprintf("Zoom: %d", m.ZoomLevel))),
		elem.Div(vecty.Text(fmt.Sprintf("Tile: %d,%d", centerX, centerY))),
		elem.Div(vecty.Text(fmt.Sprintf("Endpoint: %s", endpoint))),
		elem.Div(
			vecty.Text(fmt.Sprintf("Sample URL: %s/%d/%d/%d.png", endpoint, m.ZoomLevel, centerX, centerY)),
			vecty.Markup(
				vecty.Style("margin-top", "8px"),
				vecty.Style("word-break", "break-all"),
				vecty.Style("color", "yellow"),
			),
		),
	)
}

// コントロールトグル
func (m *OpenFreeMap) renderControlToggle() vecty.ComponentOrHTML {
	return elem.Button(
		vecty.Text("☰"),
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "20px"),
			vecty.Style("right", "20px"),
			vecty.Style("width", "48px"),
			vecty.Style("height", "48px"),
			vecty.Style("background", "rgba(26, 26, 26, 0.9)"),
			vecty.Style("border", "1px solid rgba(255, 255, 255, 0.1)"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("color", "white"),
			vecty.Style("font-size", "18px"),
			vecty.Style("cursor", "pointer"),
			event.Click(func(e *vecty.Event) { m.toggleControls() }),
		),
	)
}

// 条件付きサイドパネル
func (m *OpenFreeMap) renderConditionalSidePanel() vecty.ComponentOrHTML {
	if !m.IsControlsVisible {
		return elem.Div()
	}
	return m.renderSidePanel()
}

// サイドパネル
func (m *OpenFreeMap) renderSidePanel() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "0"),
			vecty.Style("right", "0"),
			vecty.Style("width", "320px"),
			vecty.Style("height", "100vh"),
			vecty.Style("background", "rgba(26, 26, 26, 0.95)"),
			vecty.Style("padding", "20px"),
			vecty.Style("overflow-y", "auto"),
			vecty.Style("color", "white"),
		),

		elem.Heading2(vecty.Text("地図設定")),

		// プロバイダー選択
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "rgba(255, 255, 255, 0.05)"),
				vecty.Style("padding", "16px"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("margin-bottom", "20px"),
			),
			elem.Heading3(
				vecty.Text("地図タイル"),
				vecty.Markup(
					vecty.Style("margin", "0 0 12px 0"),
					vecty.Style("font-size", "14px"),
				),
			),

			// OpenStreetMap
			elem.Button(
				vecty.Text("OpenStreetMap（詳細版）"),
				vecty.Markup(
					vecty.Style("width", "100%"),
					vecty.Style("margin-bottom", "12px"),
					vecty.Style("padding", "12px"),
					vecty.Style("background", func() string {
						if m.TileProvider == 0 {
							return "#27AE60"
						}
						return "rgba(255,255,255,0.1)"
					}()),
					vecty.Style("border", "none"),
					vecty.Style("border-radius", "6px"),
					vecty.Style("color", "white"),
					vecty.Style("cursor", "pointer"),
					vecty.Style("font-size", "14px"),
					vecty.Style("transition", "all 0.2s"),
					event.Click(func(e *vecty.Event) {
						m.TileProvider = 0
						vecty.Rerender(m)
					}),
				),
			),
			elem.Paragraph(
				vecty.Text("道路名、建物名など詳細情報を表示"),
				vecty.Markup(
					vecty.Style("font-size", "11px"),
					vecty.Style("color", "rgba(255,255,255,0.7)"),
					vecty.Style("margin", "-8px 0 16px 0"),
				),
			),

			// CartoDB
			elem.Button(
				vecty.Text("CartoDB Light（軽量版）"),
				vecty.Markup(
					vecty.Style("width", "100%"),
					vecty.Style("margin-bottom", "12px"),
					vecty.Style("padding", "12px"),
					vecty.Style("background", func() string {
						if m.TileProvider == 1 {
							return "#3498DB"
						}
						return "rgba(255,255,255,0.1)"
					}()),
					vecty.Style("border", "none"),
					vecty.Style("border-radius", "6px"),
					vecty.Style("color", "white"),
					vecty.Style("cursor", "pointer"),
					vecty.Style("font-size", "14px"),
					vecty.Style("transition", "all 0.2s"),
					event.Click(func(e *vecty.Event) {
						m.TileProvider = 1
						vecty.Rerender(m)
					}),
				),
			),
			elem.Paragraph(
				vecty.Text("軽量で読み込みが高速、シンプルなデザイン"),
				vecty.Markup(
					vecty.Style("font-size", "11px"),
					vecty.Style("color", "rgba(255,255,255,0.7)"),
					vecty.Style("margin", "-8px 0 0 0"),
				),
			),
		),

		
		// 操作説明とゲーム準備セクション
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "rgba(255, 255, 255, 0.05)"),
				vecty.Style("padding", "16px"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("margin-bottom", "20px"),
			),
			elem.Heading3(
				vecty.Text("操作方法"),
				vecty.Markup(
					vecty.Style("margin", "0 0 12px 0"),
					vecty.Style("font-size", "14px"),
				),
			),
			elem.Paragraph(
				vecty.Text("• ドラッグして地図を移動"),
				vecty.Markup(
					vecty.Style("font-size", "12px"),
					vecty.Style("color", "rgba(255,255,255,0.8)"),
					vecty.Style("margin", "4px 0"),
				),
			),
			elem.Paragraph(
				vecty.Text("• マウスホイールでズーム"),
				vecty.Markup(
					vecty.Style("font-size", "12px"),
					vecty.Style("color", "rgba(255,255,255,0.8)"),
					vecty.Style("margin", "4px 0"),
				),
			),
			elem.Paragraph(
				vecty.Text("• +/- ボタンでもズーム可能"),
				vecty.Markup(
					vecty.Style("font-size", "12px"),
					vecty.Style("color", "rgba(255,255,255,0.8)"),
					vecty.Style("margin", "4px 0"),
				),
			),
			elem.Paragraph(
				vecty.Text("• タッチパッド対応"),
				vecty.Markup(
					vecty.Style("font-size", "12px"),
					vecty.Style("color", "#4CAF50"),
					vecty.Style("margin", "4px 0"),
					vecty.Style("font-weight", "bold"),
				),
			),
		),
		
		// ゲーム準備セクション
		elem.Div(
			vecty.Markup(
				vecty.Style("background", "rgba(76, 175, 80, 0.1)"),
				vecty.Style("border", "1px solid rgba(76, 175, 80, 0.3)"),
				vecty.Style("padding", "16px"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("margin-bottom", "20px"),
			),
			elem.Heading3(
				vecty.Text("陣地取りゲーム準備中"),
				vecty.Markup(
					vecty.Style("margin", "0 0 12px 0"),
					vecty.Style("font-size", "14px"),
					vecty.Style("color", "#4CAF50"),
				),
			),
			elem.Paragraph(
				vecty.Text("地図ドラッグ操作が完成！次は地図上でクリックして陣地を塗る機能を実装します。"),
				vecty.Markup(
					vecty.Style("font-size", "12px"),
					vecty.Style("color", "rgba(255,255,255,0.8)"),
					vecty.Style("margin", "0"),
					vecty.Style("line-height", "1.5"),
				),
			),
		),
	)
}

func main() {
	vecty.SetTitle("OpenFreeMap - Drag & Drop")

	openFreeMap := NewOpenFreeMap()

	vecty.RenderBody(openFreeMap)
	select {}
}
