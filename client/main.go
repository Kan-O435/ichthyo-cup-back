package main

import (
	"fmt"
	"math"
	"strconv"
	"syscall/js" // JavaScriptのグローバルオブジェクトにアクセスするため、syscall/jsをインポート

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

// シンプルなズーム最適化地図
type ZoomOptimizedMap struct {
	vecty.Core
	
	// 地図状態
	Lat   float64
	Lng   float64
	Zoom  float64
	Scale float64
	
	// ドラッグ状態
	IsDragging bool
	LastX      int
	LastY      int
	
	// 設定
	TileSize int
	GridSize int
	MaxTiles int
	LoadRadius int
}

func NewZoomOptimizedMap() *ZoomOptimizedMap {
	m := &ZoomOptimizedMap{
		Lat:        35.676200,
		Lng:        139.650300,
		Zoom:       8.0,
		Scale:      1.0,
		TileSize:   256,
		MaxTiles:   2500,
		LoadRadius: 25,
	}

	// JavaScriptのウィンドウリサイズイベントを購読
	js.Global().Get("window").Call("addEventListener", "resize", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		vecty.Rerender(m)
		return nil
	}))
	
	return m
}

func (m *ZoomOptimizedMap) getMovementSensitivity() float64 {
	baseSensitivity := 0.005
	switch {
	case m.Zoom <= 5:
		return baseSensitivity * 2.0
	case m.Zoom <= 8:
		return baseSensitivity * 1.0
	case m.Zoom <= 12:
		return baseSensitivity * 0.3
	case m.Zoom <= 16:
		return baseSensitivity * 0.1
	default:
		return baseSensitivity * 0.05
	}
}

func (m *ZoomOptimizedMap) getTileURL(x, y, z int) string {
	return fmt.Sprintf("https://cartodb-basemaps-a.global.ssl.fastly.net/light_all/%d/%d/%d.png", z, x, y)
}

// 座標変換
func (m *ZoomOptimizedMap) latLngToTile(lat, lng float64, zoom int) (int, int) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, float64(zoom))
	
	// 経度を常に-180から180の範囲に正規化する
	// このロジックによって、地図が横方向にループする
	lng = math.Mod(lng+180.0, 360.0) - 180.0
	
	x := int((lng + 180.0) / 360.0 * n)
	y := int((1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n)
	
	// 緯度方向のワールドラップは行わないため、範囲を制限
	max := int(n) - 1
	if x < 0 { x = 0 }
	if x > max { x = max }
	if y < 0 { y = 0 }
	if y > max { y = max }
	return x, y
}

func (m *ZoomOptimizedMap) shouldShowTile(tileX, tileY, centerX, centerY int) bool {
	dx := tileX - centerX
	dy := tileY - centerY
	distance := int(math.Sqrt(float64(dx*dx + dy*dy)))
	return distance <= m.LoadRadius
}

func (m *ZoomOptimizedMap) onMouseDown(e *vecty.Event) {
	m.IsDragging = true
	m.LastX = e.Get("clientX").Int()
	m.LastY = e.Get("clientY").Int()
	e.Call("preventDefault")
}

func (m *ZoomOptimizedMap) onMouseMove(e *vecty.Event) {
	if !m.IsDragging { return }
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	deltaX := float64(x - m.LastX)
	deltaY := float64(y - m.LastY)
	sensitivity := m.getMovementSensitivity()
	m.Lng -= deltaX * sensitivity
	m.Lat += deltaY * sensitivity
	if m.Lat > 85 { m.Lat = 85 }
	if m.Lat < -85 { m.Lat = -85 }
	// ワールドラップのための経度制限を削除
	m.LastX, m.LastY = x, y
	vecty.Rerender(m)
}

func (m *ZoomOptimizedMap) onMouseUp(e *vecty.Event) {
	m.IsDragging = false
}

func (m *ZoomOptimizedMap) onWheel(e *vecty.Event) {
	e.Call("preventDefault")
	deltaY := e.Get("deltaY").Float()
	zoomSensitivity := 0.05
	if deltaY < 0 {
		m.Scale += zoomSensitivity
	} else {
		m.Scale -= zoomSensitivity
	}
	m.Scale = math.Max(1.0, math.Min(1.99, m.Scale))
	vecty.Rerender(m)
}

type touchState struct {
	PinchInitialDistance float64
	IsPinched bool
}

var touch = &touchState{}

func (m *ZoomOptimizedMap) onTouchStart(e *vecty.Event) {
	e.Call("preventDefault")
	touches := e.Get("touches")
	if touches.Length() == 1 {
		m.IsDragging = true
		m.LastX = touches.Index(0).Get("clientX").Int()
		m.LastY = touches.Index(0).Get("clientY").Int()
		touch.IsPinched = false
	} else if touches.Length() == 2 {
		touch1 := touches.Index(0)
		touch2 := touches.Index(1)
		dx := touch2.Get("clientX").Float() - touch1.Get("clientX").Float()
		dy := touch2.Get("clientY").Float() - touch1.Get("clientY").Float()
		touch.PinchInitialDistance = math.Hypot(dx, dy)
		touch.IsPinched = true
	}
}

func (m *ZoomOptimizedMap) onTouchMove(e *vecty.Event) {
	e.Call("preventDefault")
	touches := e.Get("touches")
	if touch.IsPinched && touches.Length() == 2 {
		touch1 := touches.Index(0)
		touch2 := touches.Index(1)
		dx := touch2.Get("clientX").Float() - touch1.Get("clientX").Float()
		dy := touch2.Get("clientY").Float() - touch1.Get("clientY").Float()
		currentDistance := math.Hypot(dx, dy)
		if touch.PinchInitialDistance > 0 {
			pinchSensitivity := 0.001
			delta := currentDistance - touch.PinchInitialDistance
			m.Scale += delta * pinchSensitivity
			m.Scale = math.Max(1.0, math.Min(1.99, m.Scale))
			touch.PinchInitialDistance = currentDistance
			vecty.Rerender(m)
		}
	} else if m.IsDragging && touches.Length() == 1 {
		x, y := touches.Index(0).Get("clientX").Int(), touches.Index(0).Get("clientY").Int()
		deltaX := float64(x - m.LastX)
		deltaY := float64(y - m.LastY)
		sensitivity := m.getMovementSensitivity()
		m.Lng -= deltaX * sensitivity
		m.Lat += deltaY * sensitivity
		m.Lat = math.Max(-85, math.Min(85, m.Lat))
		m.Lng = math.Max(-180, math.Min(180, m.Lng))
		m.LastX, m.LastY = x, y
		vecty.Rerender(m)
	}
}

func (m *ZoomOptimizedMap) onTouchEnd(e *vecty.Event) {
	m.IsDragging = false
	touch.IsPinched = false
	if m.Scale >= 1.9 {
		m.Zoom++
		m.Scale = 1.0
		vecty.Rerender(m)
	} else if m.Scale <= 1.1 && m.Zoom > 2 {
		m.Zoom--
		m.Scale = 1.0
		vecty.Rerender(m)
	}
}

func (m *ZoomOptimizedMap) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("padding", "0"),
			vecty.Style("overflow", "hidden"), // body全体のスクロールバーを非表示
			vecty.Style("background", "#2c3e50"),
			vecty.Style("font-family", "Arial, sans-serif"),
		),
		elem.Div(
			vecty.Markup(
				vecty.Style("position", "fixed"),
				vecty.Style("inset", "0"),
				vecty.Style("cursor", map[bool]string{true: "grabbing", false: "grab"}[m.IsDragging]),
				vecty.Style("touch-action", "none"),
				vecty.Style("overflow", "hidden"), // このDivで地図の表示エリアをクリップ
				event.MouseDown(m.onMouseDown),
				event.MouseMove(m.onMouseMove),
				event.MouseUp(m.onMouseUp),
				event.Wheel(m.onWheel),
				event.TouchStart(m.onTouchStart),
				event.TouchMove(m.onTouchMove),
				event.TouchEnd(m.onTouchEnd),
			),
			m.renderTileGrid(),
		),
		m.renderControls(),
	)
}

func (m *ZoomOptimizedMap) renderTileGrid() vecty.ComponentOrHTML {
	if m.Scale >= 2.0 {
		m.Zoom++
		m.Scale = 1.0
	} else if m.Scale < 1.0 && m.Zoom > 2 {
		m.Zoom--
		m.Scale = 1.99
	}
	
	baseZoom := int(math.Floor(m.Zoom))
	centerX, centerY := m.latLngToTile(m.Lat, m.Lng, baseZoom)
	
	window := js.Global().Get("window")
	screenWidth := window.Get("innerWidth").Float()
	screenHeight := window.Get("innerHeight").Float()
	
	numTilesX := int(math.Ceil(screenWidth / float64(m.TileSize))) + 2
	numTilesY := int(math.Ceil(screenHeight / float64(m.TileSize))) + 2
	
	children := []vecty.MarkupOrChild{
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", "50%"),
			vecty.Style("top", "50%"),
			vecty.Style("transform", fmt.Sprintf("translate(-50%%, -50%%) scale(%f)", m.Scale)),
			vecty.Style("width", strconv.Itoa(numTilesX*m.TileSize)+"px"),
			vecty.Style("height", strconv.Itoa(numTilesY*m.TileSize)+"px"),
		),
	}
	
	halfWidth := numTilesX / 2
	halfHeight := numTilesY / 2
	maxTilesAtZoom := int(math.Pow(2, float64(baseZoom)))
	
	tileCount := 0
	
	for dy := -halfHeight; dy <= halfHeight && tileCount < m.MaxTiles; dy++ {
		for dx := -halfWidth; dx <= halfWidth && tileCount < m.MaxTiles; dx++ {
			tileX := (centerX + dx) % maxTilesAtZoom
			if tileX < 0 {
				tileX += maxTilesAtZoom
			}
			
			tileY := centerY + dy
			
			if tileY < 0 || tileY >= maxTilesAtZoom {
				continue
			}
			
			screenX := (dx + halfWidth) * m.TileSize
			screenY := (dy + halfHeight) * m.TileSize
			
			children = append(children, m.renderTile(tileX, tileY, screenX, screenY))
			tileCount++
		}
	}
	
	return elem.Div(children...)
}

func (m *ZoomOptimizedMap) renderTile(tileX, tileY, screenX, screenY int) vecty.ComponentOrHTML {
	tileSizeStr := strconv.Itoa(m.TileSize) + "px"
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "absolute"),
			vecty.Style("left", strconv.Itoa(screenX)+"px"),
			vecty.Style("top", strconv.Itoa(screenY)+"px"),
			vecty.Style("width", tileSizeStr),
			vecty.Style("height", tileSizeStr),
			vecty.Style("background", "#34495e"),
			vecty.Style("box-sizing", "border-box"),
		),
		elem.Image(
			vecty.Markup(
				prop.Src(m.getTileURL(tileX, tileY, int(math.Floor(m.Zoom)))),
				prop.Alt(""),
				vecty.Style("width", "100%"),
				vecty.Style("height", "100%"),
				vecty.Style("display", "block"),
			),
		),
	)
}

func (m *ZoomOptimizedMap) renderControls() vecty.ComponentOrHTML {
	sensitivity := m.getMovementSensitivity()
	return elem.Div(
		vecty.Markup(
			vecty.Style("position", "fixed"),
			vecty.Style("top", "20px"),
			vecty.Style("left", "20px"),
			vecty.Style("background", "rgba(0,0,0,0.85)"),
			vecty.Style("border-radius", "8px"),
			vecty.Style("padding", "16px"),
			vecty.Style("color", "white"),
			vecty.Style("backdrop-filter", "blur(10px)"),
		),
		elem.Div(
			vecty.Text("ズーム適応地図"),
			vecty.Markup(
				vecty.Style("font-weight", "600"),
				vecty.Style("color", "#4CAF50"),
				vecty.Style("margin-bottom", "12px"),
				vecty.Style("font-size", "16px"),
			),
		),
		elem.Div(
			vecty.Text(fmt.Sprintf("ズーム: %.2f/16", m.Zoom)),
			vecty.Markup(
				vecty.Style("margin-bottom", "4px"),
				vecty.Style("color", "#FFD700"),
				vecty.Style("font-family", "monospace"),
			),
		),
		elem.Div(
			vecty.Text(fmt.Sprintf("感度: %.4f", sensitivity)),
			vecty.Markup(
				vecty.Style("margin-bottom", "12px"),
				vecty.Style("color", "#87CEEB"),
				vecty.Style("font-family", "monospace"),
				vecty.Style("font-size", "12px"),
			),
		),
		elem.Div(
			vecty.Markup(
				vecty.Style("display", "flex"),
				vecty.Style("gap", "8px"),
				vecty.Style("margin-bottom", "12px"),
			),
			elem.Button(
				vecty.Text("+ 拡大"),
				vecty.Markup(
					vecty.Style("background", "#4CAF50"),
					vecty.Style("border", "none"),
					vecty.Style("color", "white"),
					vecty.Style("padding", "10px 15px"),
					vecty.Style("border-radius", "6px"),
					vecty.Style("cursor", "pointer"),
					event.Click(func(e *vecty.Event) {
						if m.Zoom < 16 {
							m.Zoom++
							vecty.Rerender(m)
						}
					}),
				),
			),
			elem.Button(
				vecty.Text("- 縮小"),
				vecty.Markup(
					vecty.Style("background", "#f44336"),
					vecty.Style("border", "none"),
					vecty.Style("color", "white"),
					vecty.Style("padding", "10px 15px"),
					vecty.Style("border-radius", "6px"),
					vecty.Style("cursor", "pointer"),
					event.Click(func(e *vecty.Event) {
						if m.Zoom > 2 {
							m.Zoom--
							vecty.Rerender(m)
						}
					}),
				),
			),
		),
		elem.Div(
			vecty.Text(m.getSensitivityLabel()),
			vecty.Markup(
				vecty.Style("background", "rgba(255,255,255,0.1)"),
				vecty.Style("padding", "6px 10px"),
				vecty.Style("border-radius", "4px"),
				vecty.Style("font-size", "11px"),
				vecty.Style("text-align", "center"),
				vecty.Style("margin-bottom", "12px"),
			),
		),
		elem.Div(
			vecty.Text(fmt.Sprintf("%.5f, %.5f", m.Lat, m.Lng)),
			vecty.Markup(
				vecty.Style("color", "rgba(255,255,255,0.7)"),
				vecty.Style("font-family", "monospace"),
				vecty.Style("font-size", "11px"),
			),
		),
		elem.Div(
			vecty.Markup(
				vecty.Style("margin-top", "12px"),
				vecty.Style("font-size", "10px"),
				vecty.Style("color", "rgba(255,255,255,0.6)"),
				vecty.Style("line-height", "1.4"),
			),
			elem.Div(vecty.Text("• ドラッグ: 地図移動（ズーム適応）")),
			elem.Div(vecty.Text("• ホイール/ピンチ: ズーム変更")),
		),
	)
}

func (m *ZoomOptimizedMap) getSensitivityLabel() string {
	switch {
	case m.Zoom <= 5:
		return "📍 広域ビュー（感度: 高）"
	case m.Zoom <= 8:
		return "🗺️ 中域ビュー（感度: 標準）"
	case m.Zoom <= 12:
		return "🔍 詳細ビュー（感度: 低）"
	case m.Zoom <= 16:
		return "🔬 超詳細ビュー（感度: かなり低）"
	default:
		return "🎯 最大ズーム（感度: 最低）"
	}
}

func main() {
	vecty.SetTitle("ズーム適応CartoDB地図")
	vecty.RenderBody(NewZoomOptimizedMap())
	select {}
}