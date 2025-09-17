package main

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// Constants for division
const (
	minZoom       = 1
	maxZoom       = 18
	baseTileSize  = 256      // 元のタイルサイズ
	subTileSize   = 16       // 256を16で割って16px
	gridSize      = 100      // 100×100グリッド
	totalSubTiles = 10000    // 100×100 = 10,000サブタイル
	zoomSpeed     = 0.01
)

// Point represents a 2D point
type Point struct {
	X, Y int
}

// IchthyoMapView is the main map component with 100x100 division
type IchthyoMapView struct {
	vecty.Core

	// --- State ---
	CenterLat float64 `vecty:"prop"`
	CenterLng float64 `vecty:"prop"`
	Zoom      float64 `vecty:"prop"`

	// --- Internal Animation & Interaction State ---
	tileContainer js.Value
	isMounted     bool
	isDragging    bool
	dragStart     Point
	lastDrag      Point
	
	// --- Performance Settings ---
	maxVisibleTiles int  // 表示タイル数制限
	loadRadius      int  // 読み込み半径
}

// NewIchthyoMapView creates a new map view
func NewIchthyoMapView() *IchthyoMapView {
	return &IchthyoMapView{
		CenterLat:       35.6762, // Tokyo
		CenterLng:       139.6503,
		Zoom:            10,
		maxVisibleTiles: 5000,  // 最大5000タイル表示
		loadRadius:      50,    // 中心から50タイル以内
	}
}

// --- Component Lifecycle ---

func (m *IchthyoMapView) Mount() {
	m.tileContainer = js.Global().Get("document").Call("createElement", "div")
	style := m.tileContainer.Get("style")
	style.Set("position", "absolute")
	style.Set("top", "0")
	style.Set("left", "0")
	style.Set("will-change", "transform")
	style.Set("pointer-events", "none") // パフォーマンス向上

	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		viewport := js.Global().Get("document").Call("querySelector", ".map-viewport")
		if !viewport.IsUndefined() {
			viewport.Call("appendChild", m.tileContainer)
			m.isMounted = true
			m.DrawMap()
		}
		return nil
	}), 0)
}

func (m *IchthyoMapView) Unmount() {
	if m.isMounted {
		m.tileContainer.Call("remove")
	}
	m.isMounted = false
}

// Render renders the map viewport
func (m *IchthyoMapView) Render() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("map-viewport"),
			vecty.Style("position", "fixed"),
			vecty.Style("width", "100vw"),
			vecty.Style("height", "100vh"),
			vecty.Style("cursor", "grab"),
			vecty.Style("overflow", "hidden"),
			event.MouseDown(m.onMouseDown),
			event.MouseMove(m.onMouseMove),
			event.MouseUp(m.onMouseUp),
			event.MouseLeave(m.onMouseUp),
			event.Wheel(m.onWheel),
		),
		
		// 情報表示
		m.renderInfo(),
	)
}

// DrawMap draws the 100x100 divided map
func (m *IchthyoMapView) DrawMap() {
	if !m.isMounted {
		return
	}

	// 既存のタイルをクリア
	m.tileContainer.Set("innerHTML", "")

	screenWidth := js.Global().Get("innerWidth").Int()
	screenHeight := js.Global().Get("innerHeight").Int()

	baseZoom := int(math.Ceil(m.Zoom))
	if baseZoom < minZoom {
		baseZoom = minZoom
	}
	if baseZoom > maxZoom {
		baseZoom = maxZoom
	}

	// 中心座標を取得
	centerX, centerY := m.latLngToPixel(m.CenterLat, m.CenterLng, float64(baseZoom))
	
	// 100×100グリッドの中心タイル座標
	centerSubTileX := int(centerX / float64(subTileSize))
	centerSubTileY := int(centerY / float64(subTileSize))

	// 表示範囲計算
	halfGrid := gridSize / 2
	tilesDrawn := 0

	// 100×100グリッドを描画
	for dy := -halfGrid; dy <= halfGrid && tilesDrawn < m.maxVisibleTiles; dy++ {
		for dx := -halfGrid; dx <= halfGrid && tilesDrawn < m.maxVisibleTiles; dx++ {
			subTileX := centerSubTileX + dx
			subTileY := centerSubTileY + dy
			
			// 距離チェック（パフォーマンス最適化）
			distance := int(math.Sqrt(float64(dx*dx + dy*dy)))
			if distance > m.loadRadius {
				continue
			}
			
			// 元のタイル座標を計算
			baseTileX := subTileX * subTileSize / baseTileSize
			baseTileY := subTileY * subTileSize / baseTileSize
			
			// サブタイル内でのオフセット
			offsetX := (subTileX * subTileSize) % baseTileSize
			offsetY := (subTileY * subTileSize) % baseTileSize
			
			// 画面上の位置
			screenX := float64((dx + halfGrid) * subTileSize)
			screenY := float64((dy + halfGrid) * subTileSize)
			
			m.createSubTile(baseTileX, baseTileY, baseZoom, offsetX, offsetY, screenX, screenY)
			tilesDrawn++
		}
	}
	
	// コンテナの位置調整
	offsetX := centerX - float64(screenWidth)/2
	offsetY := centerY - float64(screenHeight)/2
	
	containerStyle := m.tileContainer.Get("style")
	containerStyle.Set("transform", fmt.Sprintf("translate(%.2fpx, %.2fpx)", -offsetX, -offsetY))
	
	// デバッグ情報をコンソールに出力
	js.Global().Get("console").Call("log", fmt.Sprintf("Drew %d tiles in 100x100 grid", tilesDrawn))
}

// createSubTile creates a single sub-tile from a larger tile
func (m *IchthyoMapView) createSubTile(baseTileX, baseTileY, zoom, offsetX, offsetY int, screenX, screenY float64) {
	// サブタイル要素を作成
	subTile := js.Global().Get("document").Call("createElement", "div")
	
	style := subTile.Get("style")
	style.Set("position", "absolute")
	style.Set("left", fmt.Sprintf("%.2fpx", screenX))
	style.Set("top", fmt.Sprintf("%.2fpx", screenY))
	style.Set("width", fmt.Sprintf("%dpx", subTileSize))
	style.Set("height", fmt.Sprintf("%dpx", subTileSize))
	style.Set("overflow", "hidden")
	style.Set("background-color", "#2c3e50")
	style.Set("border", "1px solid rgba(255,255,255,0.1)")
	style.Set("box-sizing", "border-box")
	
	// 背景画像として元のタイルを設定
	tileURL := m.getTileURL(baseTileX, baseTileY, zoom)
	style.Set("background-image", fmt.Sprintf("url(%s)", tileURL))
	style.Set("background-size", fmt.Sprintf("%dpx %dpx", baseTileSize, baseTileSize))
	style.Set("background-position", fmt.Sprintf("-%dpx -%dpx", offsetX, offsetY))
	style.Set("background-repeat", "no-repeat")
	
	m.tileContainer.Call("appendChild", subTile)
}

// --- Event Handlers ---

func (m *IchthyoMapView) onMouseDown(e *vecty.Event) {
	e.Call("preventDefault")
	m.isDragging = true
	m.dragStart = Point{X: e.Get("clientX").Int(), Y: e.Get("clientY").Int()}
	m.lastDrag = m.dragStart
	js.Global().Get("document").Get("body").Get("style").Set("cursor", "grabbing")
}

func (m *IchthyoMapView) onMouseMove(e *vecty.Event) {
	if !m.isDragging {
		return
	}
	e.Call("preventDefault")
	currentPos := Point{X: e.Get("clientX").Int(), Y: e.Get("clientY").Int()}
	dx := float64(m.lastDrag.X - currentPos.X)
	dy := float64(m.lastDrag.Y - currentPos.Y)

	// 感度を大幅に下げる（100分割用）
	sensitivity := 0.001
	scale := math.Pow(2, m.Zoom)
	lngPerPixel := (360.0 / (scale * float64(baseTileSize))) * sensitivity
	latPerPixel := (180.0 / (scale * float64(baseTileSize))) * sensitivity

	m.CenterLng += dx * lngPerPixel
	m.CenterLat -= dy * latPerPixel

	m.lastDrag = currentPos
	m.DrawMap()
}

func (m *IchthyoMapView) onMouseUp(e *vecty.Event) {
	if !m.isDragging {
		return
	}
	m.isDragging = false
	js.Global().Get("document").Get("body").Get("style").Set("cursor", "default")
}

func (m *IchthyoMapView) onWheel(e *vecty.Event) {
	e.Call("preventDefault")

	delta := e.Get("deltaY").Float()
	zoomThreshold := 50.0 // ズーム感度を下げる
	
	if math.Abs(delta) > zoomThreshold {
		if delta < 0 {
			if m.Zoom < maxZoom {
				m.Zoom += 0.5 // ズーム変化量を小さく
			}
		} else {
			if m.Zoom > minZoom {
				m.Zoom -= 0.5
			}
		}
		m.DrawMap()
	}
}

// --- Coordinate Conversion Utilities ---

func (m *IchthyoMapView) latLngToPixel(lat, lng, zoom float64) (float64, float64) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, zoom)
	x := (lng + 180.0) / 360.0 * n * float64(baseTileSize)
	y := (1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n * float64(baseTileSize)
	return x, y
}

func (m *IchthyoMapView) getTileURL(x, y, z int) string {
	// OpenStreetMapタイルを使用
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", z, x, y)
}

// 情報表示
func (m *IchthyoMapView) renderInfo() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "20px"),
			vecty.Style("left", "20px"),
			vecty.Style("background", "rgba(0,0,0,0.8)"),
			vecty.Style("color", "white"),
			vecty.Style("padding", "15px"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("font-family", "monospace"),
			vecty.Style("font-size", "12px"),
			vecty.Style("z-index", "1000"),
		),
		
		elem.Div(
			vecty.Text("🗺️ 100×100分割地図"),
			vecty.Markup(
				vecty.Style("font-weight", "bold"),
				vecty.Style("color", "#4CAF50"),
				vecty.Style("margin-bottom", "10px"),
			),
		),
		
		elem.Div(
			vecty.Text(fmt.Sprintf("グリッド: %d×%d", gridSize, gridSize)),
			vecty.Markup(vecty.Style("margin", "3px 0")),
		),
		
		elem.Div(
			vecty.Text(fmt.Sprintf("サブタイル: %dpx", subTileSize)),
			vecty.Markup(vecty.Style("margin", "3px 0")),
		),
		
		elem.Div(
			vecty.Text(fmt.Sprintf("最大表示: %d", m.maxVisibleTiles)),
			vecty.Markup(vecty.Style("margin", "3px 0")),
		),
		
		elem.Div(
			vecty.Text(fmt.Sprintf("ズーム: %.1f", m.Zoom)),
			vecty.Markup(
				vecty.Style("margin", "3px 0"),
				vecty.Style("color", "#FFD700"),
			),
		),
		
		elem.Div(
			vecty.Text(fmt.Sprintf("緯度: %.5f", m.CenterLat)),
			vecty.Markup(
				vecty.Style("margin", "3px 0"),
				vecty.Style("color", "rgba(255,255,255,0.7)"),
			),
		),
		
		elem.Div(
			vecty.Text(fmt.Sprintf("経度: %.5f", m.CenterLng)),
			vecty.Markup(
				vecty.Style("margin", "3px 0"),
				vecty.Style("color", "rgba(255,255,255,0.7)"),
			),
		),
		
		elem.Div(
			vecty.Markup(
				vecty.Style("margin-top", "10px"),
				vecty.Style("font-size", "10px"),
				vecty.Style("color", "rgba(255,255,255,0.5)"),
			),
			elem.Div(vecty.Text("ドラッグ: 地図移動")),
			elem.Div(vecty.Text("ホイール: ズーム")),
		),
	)
}