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

// 軽量な地図コンポーネント（HTML版の機能を再現）
type LightweightMap struct {
	vecty.Core
	
	// 地図状態
	Lat        float64
	Lng        float64
	Zoom       int
	IsDragging bool
	LastX      int
	LastY      int
}

// 新しい軽量地図を作成
func NewLightweightMap() *LightweightMap {
	return &LightweightMap{
		Lat:        35.676200,
		Lng:        139.650300,
		Zoom:       10,
		IsDragging: false,
		LastX:      0,
		LastY:      0,
	}
}

// 緯度経度からタイル座標に変換
func (m *LightweightMap) latLngToTile(lat, lng float64, zoom int) (int, int) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, float64(zoom))
	x := int(math.Floor((lng + 180.0) / 360.0 * n))
	y := int(math.Floor((1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n))
	return x, y
}

// ズーム操作
func (m *LightweightMap) zoomIn() {
	if m.Zoom < 18 {
		m.Zoom++
		vecty.Rerender(m)
	}
}

func (m *LightweightMap) zoomOut() {
	if m.Zoom > 1 {
		m.Zoom--
		vecty.Rerender(m)
	}
}

// 地図移動
func (m *LightweightMap) panMap(deltaX, deltaY float64) {
	n := math.Pow(2.0, float64(m.Zoom))
	m.Lng += -deltaX * 360.0 / (256.0 * n)
	m.Lat += deltaY * 360.0 / (256.0 * n)
	
	// 範囲制限
	if m.Lat > 85.0511 { m.Lat = 85.0511 }
	if m.Lat < -85.0511 { m.Lat = -85.0511 }
	if m.Lng > 180.0 { m.Lng = 180.0 }
	if m.Lng < -180.0 { m.Lng = -180.0 }
}

// マウスイベントハンドラ
func (m *LightweightMap) handleMouseDown(e *vecty.Event) {
	m.IsDragging = true
	m.LastX = e.Get("clientX").Int()
	m.LastY = e.Get("clientY").Int()
	e.Call("preventDefault")
}

func (m *LightweightMap) handleMouseMove(e *vecty.Event) {
	if !m.IsDragging { return }
	
	currentX := e.Get("clientX").Int()
	currentY := e.Get("clientY").Int()
	
	deltaX := float64(currentX - m.LastX)
	deltaY := float64(currentY - m.LastY)
	
	m.panMap(deltaX, deltaY)
	
	m.LastX = currentX
	m.LastY = currentY
	
	vecty.Rerender(m)
	e.Call("preventDefault")
}

func (m *LightweightMap) handleMouseUp(e *vecty.Event) {
	m.IsDragging = false
	e.Call("preventDefault")
}

func (m *LightweightMap) handleWheel(e *vecty.Event) {
	e.Call("preventDefault")
	deltaY := e.Get("deltaY").Float()
	if deltaY < 0 {
		m.zoomIn()
	} else {
		m.zoomOut()
	}
}

// メインレンダリング
func (m *LightweightMap) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("padding", "0"),
			vecty.Style("overflow", "hidden"),
			vecty.Style("background", "#1a1a1a"),
			vecty.Style("font-family", "Arial, sans-serif"),
			vecty.Style("user-select", "none"),
		),
		
		// 地図コンテナ
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
				
				// イベントハンドラ
				event.MouseDown(m.handleMouseDown),
				event.MouseMove(m.handleMouseMove),
				event.MouseUp(m.handleMouseUp),
				event.Wheel(m.handleWheel),
			),
			
			// 地図グリッド
			m.renderMapGrid(),
		),
		
		// コントロール
		m.renderControls(),
		
		// 情報表示
		m.renderInfo(),
	)
}

// 地図グリッド（HTML版と同じ3x3タイル）
func (m *LightweightMap) renderMapGrid() vecty.ComponentOrHTML {
	centerX, centerY := m.latLngToTile(m.Lat, m.Lng, m.Zoom)
	
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("width", "768px"),
			vecty.Style("height", "768px"),
			vecty.Style("top", "50%"),
			vecty.Style("left", "50%"),
			vecty.Style("transform", "translate(-50%, -50%)"),
		),
		
		// 3x3のタイル配置
		m.createTile(centerX-1, centerY-1, 0, 0),
		m.createTile(centerX, centerY-1, 1, 0),
		m.createTile(centerX+1, centerY-1, 2, 0),
		
		m.createTile(centerX-1, centerY, 0, 1),
		m.createTile(centerX, centerY, 1, 1),
		m.createTile(centerX+1, centerY, 2, 1),
		
		m.createTile(centerX-1, centerY+1, 0, 2),
		m.createTile(centerX, centerY+1, 1, 2),
		m.createTile(centerX+1, centerY+1, 2, 2),
	)
}

// 個別タイル作成
func (m *LightweightMap) createTile(tileX, tileY, gridX, gridY int) vecty.ComponentOrHTML {
	screenX := gridX * 256
	screenY := gridY * 256
	
	// CartoDB Light URL
	tileUrl := fmt.Sprintf("https://cartodb-basemaps-a.global.ssl.fastly.net/light_all/%d/%d/%d.png", 
		m.Zoom, tileX, tileY)
	
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", strconv.Itoa(screenX)+"px"),
			vecty.Style("top", strconv.Itoa(screenY)+"px"),
			vecty.Style("width", "256px"),
			vecty.Style("height", "256px"),
			vecty.Style("background", "#2c3e50"),
			vecty.Style("border", "1px solid #34495e"),
		),
		elem.Image(
			vecty.Markup(
				prop.Src(tileUrl),
				prop.Alt(fmt.Sprintf("Tile %d,%d", tileX, tileY)),
				vecty.Style("width", "100%"),
				vecty.Style("height", "100%"),
				vecty.Style("display", "block"),
			),
		),
	)
}

// コントロール（HTML版と同じ）
func (m *LightweightMap) renderControls() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "20px"),
			vecty.Style("left", "20px"),
			vecty.Style("background", "rgba(0, 0, 0, 0.8)"),
			vecty.Style("padding", "10px"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("z-index", "1000"),
		),
		
		elem.Button(
			vecty.Text("+"),
			vecty.Markup(
				vecty.Style("background", "rgba(255, 255, 255, 0.2)"),
				vecty.Style("border", "none"),
				vecty.Style("color", "white"),
				vecty.Style("padding", "10px 15px"),
				vecty.Style("margin", "2px"),
				vecty.Style("border-radius", "4px"),
				vecty.Style("cursor", "pointer"),
				vecty.Style("font-size", "16px"),
				event.Click(func(e *vecty.Event) {
					m.zoomIn()
				}),
			),
		),
		
		elem.Button(
			vecty.Text("-"),
			vecty.Markup(
				vecty.Style("background", "rgba(255, 255, 255, 0.2)"),
				vecty.Style("border", "none"),
				vecty.Style("color", "white"),
				vecty.Style("padding", "10px 15px"),
				vecty.Style("margin", "2px"),
				vecty.Style("border-radius", "4px"),
				vecty.Style("cursor", "pointer"),
				vecty.Style("font-size", "16px"),
				event.Click(func(e *vecty.Event) {
					m.zoomOut()
				}),
			),
		),
	)
}

// 情報表示
func (m *LightweightMap) renderInfo() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("bottom", "20px"),
			vecty.Style("right", "20px"),
			vecty.Style("background", "rgba(0, 0, 0, 0.8)"),
			vecty.Style("color", "white"),
			vecty.Style("padding", "10px"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("font-size", "12px"),
			vecty.Style("z-index", "1000"),
		),
		
		elem.Div(vecty.Text(fmt.Sprintf("緯度: %.3f", m.Lat))),
		elem.Div(vecty.Text(fmt.Sprintf("経度: %.3f", m.Lng))),
		elem.Div(vecty.Text(fmt.Sprintf("ズーム: %d", m.Zoom))),
		elem.Div(vecty.Text("CartoDB Light")),
	)
}

func main() {
	vecty.SetTitle("軽量地図表示")
	
	lightMap := NewLightweightMap()
	
	vecty.RenderBody(lightMap)
	select {}
}