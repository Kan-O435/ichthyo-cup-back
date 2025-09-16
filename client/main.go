package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

// OpenFreeMapベースの地図コンポーネント
type OpenFreeMap struct {
	vecty.Core
	
	// 地図設定
	CenterLat   float64
	CenterLng   float64
	ZoomLevel   int
	TileSize    int
	
	// UI状態
	IsControlsVisible bool
	
	// タイルプロバイダー
	TileProvider int // 0: OSM, 1: OpenFreeMap, 2: CartoDB
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
	if x < 0 { x = 0 }
	if y < 0 { y = 0 }
	maxTile := int(n) - 1
	if x > maxTile { x = maxTile }
	if y > maxTile { y = maxTile }
	
	return x, y
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

// 地図移動
func (m *OpenFreeMap) panNorth() { m.CenterLat += 0.01; vecty.Rerender(m) }
func (m *OpenFreeMap) panSouth() { m.CenterLat -= 0.01; vecty.Rerender(m) }
func (m *OpenFreeMap) panEast()  { m.CenterLng += 0.01; vecty.Rerender(m) }
func (m *OpenFreeMap) panWest()  { m.CenterLng -= 0.01; vecty.Rerender(m) }

// コントロール表示切替
func (m *OpenFreeMap) toggleControls() {
	m.IsControlsVisible = !m.IsControlsVisible
	vecty.Rerender(m)
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
		
		// 全画面地図コンテナ
		elem.Div(
			vecty.Markup(
				vecty.Style("position", "fixed"),
				vecty.Style("top", "0"),
				vecty.Style("left", "0"),
				vecty.Style("width", "100vw"),
				vecty.Style("height", "100vh"),
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

// OpenFreeMapレイヤー（href属性使用）
func (m *OpenFreeMap) renderOpenFreeMapLayer() vecty.ComponentOrHTML {
	centerX, centerY := m.latLngToTile(m.CenterLat, m.CenterLng, m.ZoomLevel)
	baseEndpoint := m.getMapEndpoint()
	
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
		),
		
		// 3x3のタイル配置（href属性使用）
		elem.Div(
			vecty.Markup(
				vecty.Style("position", "relative"),
				vecty.Style("width", strconv.Itoa(m.TileSize*3)+"px"),
				vecty.Style("height", strconv.Itoa(m.TileSize*3)+"px"),
			),
			
			// Reactスタイルでhref属性を使用してタイルを表示
			m.createTileWithHref(baseEndpoint, centerX-1, centerY-1, 0, 0),
			m.createTileWithHref(baseEndpoint, centerX, centerY-1, 1, 0),
			m.createTileWithHref(baseEndpoint, centerX+1, centerY-1, 2, 0),
			
			m.createTileWithHref(baseEndpoint, centerX-1, centerY, 0, 1),
			m.createTileWithHref(baseEndpoint, centerX, centerY, 1, 1),
			m.createTileWithHref(baseEndpoint, centerX+1, centerY, 2, 1),
			
			m.createTileWithHref(baseEndpoint, centerX-1, centerY+1, 0, 2),
			m.createTileWithHref(baseEndpoint, centerX, centerY+1, 1, 2),
			m.createTileWithHref(baseEndpoint, centerX+1, centerY+1, 2, 2),
		),
	)
}

// href属性を使用したタイル作成（Reactスタイル）
func (m *OpenFreeMap) createTileWithHref(baseEndpoint string, tileX, tileY, gridX, gridY int) vecty.ComponentOrHTML {
	screenX := gridX * m.TileSize
	screenY := gridY * m.TileSize
	
	// OpenFreeMapのタイルURL（Reactのhref方式と同様）
	tileHref := fmt.Sprintf("%s/%d/%d/%d.png", baseEndpoint, m.ZoomLevel, tileX, tileY)
	
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", strconv.Itoa(screenX)+"px"),
			vecty.Style("top", strconv.Itoa(screenY)+"px"),
			vecty.Style("width", strconv.Itoa(m.TileSize)+"px"),
			vecty.Style("height", strconv.Itoa(m.TileSize)+"px"),
		),
		
		// Reactのhref属性と同様にVectyでも実装
		elem.Image(
			vecty.Markup(
				prop.Src(tileHref), // href属性の代わりにsrc属性
				prop.Alt(fmt.Sprintf("Tile %d,%d", tileX, tileY)),
				vecty.Style("width", "100%"),
				vecty.Style("height", "100%"),
				vecty.Style("display", "block"),
				vecty.Style("image-rendering", "pixelated"),
				vecty.Style("border", "1px solid rgba(255,255,255,0.1)"), // デバッグ用
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

// サイドパネル（動作確認済み2つのプロバイダー）
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
		
		// 動作確認済みプロバイダー選択
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
			
			// OpenStreetMap（デフォルト・詳細）
			elem.Button(
				vecty.Text("OpenStreetMap（詳細版）"),
				vecty.Markup(
					vecty.Style("width", "100%"),
					vecty.Style("margin-bottom", "12px"),
					vecty.Style("padding", "12px"),
					vecty.Style("background", func() string {
						if m.TileProvider == 0 { return "#27AE60" }
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
					event.MouseOver(func(e *vecty.Event) {
						if m.TileProvider != 0 {
							e.Target.Get("style").Set("background", "rgba(255,255,255,0.2)")
						}
					}),
					event.MouseOut(func(e *vecty.Event) {
						if m.TileProvider != 0 {
							e.Target.Get("style").Set("background", "rgba(255,255,255,0.1)")
						}
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
			
			// CartoDB（軽量・シンプル）  
			elem.Button(
				vecty.Text("CartoDB Light（軽量版）"),
				vecty.Markup(
					vecty.Style("width", "100%"),
					vecty.Style("margin-bottom", "12px"),
					vecty.Style("padding", "12px"),
					vecty.Style("background", func() string {
						if m.TileProvider == 1 { return "#3498DB" }
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
					event.MouseOver(func(e *vecty.Event) {
						if m.TileProvider != 1 {
							e.Target.Get("style").Set("background", "rgba(255,255,255,0.2)")
						}
					}),
					event.MouseOut(func(e *vecty.Event) {
						if m.TileProvider != 1 {
							e.Target.Get("style").Set("background", "rgba(255,255,255,0.1)")
						}
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
				vecty.Text("地図表示が完成しました！次は地図上でクリックして陣地を塗る機能を実装します。"),
				vecty.Markup(
					vecty.Style("font-size", "12px"),
					vecty.Style("color", "rgba(255,255,255,0.8)"),
					vecty.Style("margin", "0"),
					vecty.Style("line-height", "1.5"),
				),
			),
		),
		
		// 移動ボタン
		m.renderNavigationButtons(),
	)
}

// ナビゲーションボタン
func (m *OpenFreeMap) renderNavigationButtons() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("background", "rgba(255, 255, 255, 0.05)"),
			vecty.Style("padding", "16px"),
			vecty.Style("border-radius", "8px"),
		),
		
		elem.Heading3(vecty.Text("地図移動")),
		
		elem.Div(
			vecty.Markup(vecty.Style("text-align", "center"), vecty.Style("margin-bottom", "8px")),
			elem.Button(vecty.Text("↑"), vecty.Markup(
				vecty.Style("width", "40px"), vecty.Style("height", "40px"),
				vecty.Style("background", "rgba(255,255,255,0.1)"), vecty.Style("border", "none"),
				vecty.Style("color", "white"), vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) { m.panNorth() }),
			)),
		),
		
		elem.Div(
			vecty.Markup(vecty.Style("display", "flex"), vecty.Style("justify-content", "space-between"), vecty.Style("margin-bottom", "8px")),
			elem.Button(vecty.Text("←"), vecty.Markup(
				vecty.Style("width", "40px"), vecty.Style("height", "40px"),
				vecty.Style("background", "rgba(255,255,255,0.1)"), vecty.Style("border", "none"),
				vecty.Style("color", "white"), vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) { m.panWest() }),
			)),
			elem.Button(vecty.Text("→"), vecty.Markup(
				vecty.Style("width", "40px"), vecty.Style("height", "40px"),
				vecty.Style("background", "rgba(255,255,255,0.1)"), vecty.Style("border", "none"),
				vecty.Style("color", "white"), vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) { m.panEast() }),
			)),
		),
		
		elem.Div(
			vecty.Markup(vecty.Style("text-align", "center")),
			elem.Button(vecty.Text("↓"), vecty.Markup(
				vecty.Style("width", "40px"), vecty.Style("height", "40px"),
				vecty.Style("background", "rgba(255,255,255,0.1)"), vecty.Style("border", "none"),
				vecty.Style("color", "white"), vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) { m.panSouth() }),
			)),
		),
	)
}

func main() {
	vecty.SetTitle("OpenFreeMap - href方式")
	
	openFreeMap := NewOpenFreeMap()
	
	vecty.RenderBody(openFreeMap)
	select {}
}