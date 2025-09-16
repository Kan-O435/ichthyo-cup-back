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

// 軽量CartoDBマップ
type CartoDBMap struct {
	vecty.Core
	
	// 地図状態（必要最小限）
	Lat        float64
	Lng        float64
	Zoom       int
	
	// ドラッグ状態
	IsDragging bool
	LastX      int
	LastY      int
}

func NewCartoDBMap() *CartoDBMap {
	return &CartoDBMap{
		Lat:  35.676200, // 東京
		Lng: 139.650300,
		Zoom: 10,
	}
}

// CartoDB Light専用URL（単一タイルサーバー）
func (m *CartoDBMap) getTileURL(x, y, z int) string {
	return fmt.Sprintf("https://cartodb-basemaps-a.global.ssl.fastly.net/light_all/%d/%d/%d.png", z, x, y)
}

// 高速化された座標変換
func (m *CartoDBMap) latLngToTile(lat, lng float64, zoom int) (int, int) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, float64(zoom))
	
	x := int((lng + 180.0) / 360.0 * n)
	y := int((1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n)
	
	// 範囲制限（簡素化）
	max := int(n) - 1
	if x < 0 { x = 0 }
	if x > max { x = max }
	if y < 0 { y = 0 }
	if y > max { y = max }
	
	return x, y
}

// 軽量パン処理
func (m *CartoDBMap) panMap(deltaX, deltaY float64) {
	scale := 360.0 / (256.0 * math.Pow(2.0, float64(m.Zoom)))
	
	m.Lng -= deltaX * scale
	m.Lat += deltaY * scale
	
	// 範囲制限
	if m.Lat > 85 { m.Lat = 85 }
	if m.Lat < -85 { m.Lat = -85 }
	if m.Lng > 180 { m.Lng = 180 }
	if m.Lng < -180 { m.Lng = -180 }
}

// シンプルイベントハンドラ
func (m *CartoDBMap) onMouseDown(e *vecty.Event) {
	m.IsDragging = true
	m.LastX = e.Get("clientX").Int()
	m.LastY = e.Get("clientY").Int()
	e.Call("preventDefault")
}

func (m *CartoDBMap) onMouseMove(e *vecty.Event) {
	if !m.IsDragging { return }
	
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	m.panMap(float64(x-m.LastX), float64(y-m.LastY))
	m.LastX, m.LastY = x, y
	
	vecty.Rerender(m)
}

func (m *CartoDBMap) onMouseUp(e *vecty.Event) {
	m.IsDragging = false
}

func (m *CartoDBMap) onWheel(e *vecty.Event) {
	e.Call("preventDefault")
	if e.Get("deltaY").Float() < 0 {
		if m.Zoom < 18 { m.Zoom++ }
	} else {
		if m.Zoom > 1 { m.Zoom-- }
	}
	vecty.Rerender(m)
}

// メインレンダー（最小構成）
func (m *CartoDBMap) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("padding", "0"),
			vecty.Style("overflow", "hidden"),
			vecty.Style("background", "#34495e"),
			vecty.Style("font-family", "Arial, sans-serif"),
		),
		
		// 地図コンテナ
		elem.Div(
			vecty.Markup(
				vecty.Style("position", "fixed"),
				vecty.Style("inset", "0"),
				vecty.Style("cursor", map[bool]string{true: "grabbing", false: "grab"}[m.IsDragging]),
				
				event.MouseDown(m.onMouseDown),
				event.MouseMove(m.onMouseMove),
				event.MouseUp(m.onMouseUp),
				event.Wheel(m.onWheel),
			),
			
			m.renderTiles(),
		),
		
		// 最小コントロール
		m.renderControls(),
	)
}

// タイル描画（3x3グリッド）
func (m *CartoDBMap) renderTiles() vecty.ComponentOrHTML {
	centerX, centerY := m.latLngToTile(m.Lat, m.Lng, m.Zoom)
	
	// 直接子要素として追加
	children := []vecty.MarkupOrChild{
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", "50%"),
			vecty.Style("top", "50%"),
			vecty.Style("transform", "translate(-50%, -50%)"),
			vecty.Style("width", "768px"),
			vecty.Style("height", "768px"),
		),
	}
	
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			children = append(children, m.renderTile(
				centerX+dx, centerY+dy, 
				(dx+1)*256, (dy+1)*256,
			))
		}
	}
	
	return elem.Div(children...)
}

// 単一タイル
func (m *CartoDBMap) renderTile(tileX, tileY, screenX, screenY int) vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", strconv.Itoa(screenX)+"px"),
			vecty.Style("top", strconv.Itoa(screenY)+"px"),
			vecty.Style("width", "256px"),
			vecty.Style("height", "256px"),
			vecty.Style("background", "#2c3e50"),
		),
		
		elem.Image(
			vecty.Markup(
				prop.Src(m.getTileURL(tileX, tileY, m.Zoom)),
				prop.Alt(""),
				vecty.Style("width", "100%"),
				vecty.Style("height", "100%"),
				vecty.Style("display", "block"),
			),
		),
	)
}

// 最小コントロール
func (m *CartoDBMap) renderControls() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "20px"),
			vecty.Style("left", "20px"),
			vecty.Style("background", "rgba(0,0,0,0.7)"),
			vecty.Style("border-radius", "6px"),
			vecty.Style("padding", "5px"),
		),
		
		// ズームイン
		elem.Button(
			vecty.Text("+"),
			vecty.Markup(
				vecty.Style("background", "rgba(255,255,255,0.2)"),
				vecty.Style("border", "none"),
				vecty.Style("color", "white"),
				vecty.Style("padding", "8px 12px"),
				vecty.Style("margin", "2px"),
				vecty.Style("border-radius", "3px"),
				vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) {
					if m.Zoom < 18 {
						m.Zoom++
						vecty.Rerender(m)
					}
				}),
			),
		),
		
		// ズームアウト  
		elem.Button(
			vecty.Text("-"),
			vecty.Markup(
				vecty.Style("background", "rgba(255,255,255,0.2)"),
				vecty.Style("border", "none"),
				vecty.Style("color", "white"),
				vecty.Style("padding", "8px 12px"),
				vecty.Style("margin", "2px"),
				vecty.Style("border-radius", "3px"),
				vecty.Style("cursor", "pointer"),
				event.Click(func(e *vecty.Event) {
					if m.Zoom > 1 {
						m.Zoom--
						vecty.Rerender(m)
					}
				}),
			),
		),
		
		// 座標表示（簡素版）
		elem.Div(
			vecty.Text(fmt.Sprintf("%.3f,%.3f Z%d", m.Lat, m.Lng, m.Zoom)),
			vecty.Markup(
				vecty.Style("color", "white"),
				vecty.Style("font-size", "10px"),
				vecty.Style("margin", "5px 2px 2px"),
				vecty.Style("font-family", "monospace"),
			),
		),
	)
}

func main() {
	vecty.SetTitle("軽量CartoDB地図")
	vecty.RenderBody(NewCartoDBMap())
	select {}
}