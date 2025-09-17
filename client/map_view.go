package main

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// Constants
const (
	minZoom   = 1
	maxZoom   = 18
	tileSize  = 256
	zoomSpeed = 0.01
)

// Point represents a 2D point
type Point struct {
	X, Y int
}

// IchthyoMapView is the main map component
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
}

// NewIchthyoMapView creates a new map view
func NewIchthyoMapView() *IchthyoMapView {
	return &IchthyoMapView{
		CenterLat: 35.6762, // Tokyo
		CenterLng: 139.6503,
		Zoom:      10,
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
	// The parent component will render the UI, so we just need the viewport here.
	return elem.Div(
		vecty.Markup(
			vecty.Class("map-viewport"),
			vecty.Style("position", "fixed"),
			vecty.Style("width", "100vw"),
			vecty.Style("height", "100vh"),
			vecty.Style("cursor", "grab"),
			event.MouseDown(m.onMouseDown),
			event.MouseMove(m.onMouseMove),
			event.MouseUp(m.onMouseUp),
			event.MouseLeave(m.onMouseUp),
			event.Wheel(m.onWheel),
		),
	)
}

// DrawMap redraws the map tiles based on the current state.
func (m *IchthyoMapView) DrawMap() {
	if !m.isMounted {
		return
	}

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

	scale := math.Pow(2, m.Zoom-float64(baseZoom))

	centerX, centerY := m.latLngToPixel(m.CenterLat, m.CenterLng, float64(baseZoom))

	tx := centerX - (float64(screenWidth)/2)/scale
	ty := centerY - (float64(screenHeight)/2)/scale

	startTileX := int(math.Floor(tx / tileSize))
	startTileY := int(math.Floor(ty / tileSize))

	numTilesX := int(math.Ceil(float64(screenWidth)/(tileSize*scale))) + 1
	numTilesY := int(math.Ceil(float64(screenHeight)/(tileSize*scale))) + 1

	containerStyle := m.tileContainer.Get("style")
	containerStyle.Set("transform", fmt.Sprintf("scale(%.6f)", scale))
	containerStyle.Set("transform-origin", "top left")

	for y := 0; y <= numTilesY; y++ {
		for x := 0; x <= numTilesX; x++ {
			tileX := startTileX + x
			tileY := startTileY + y

			img := js.Global().Get("document").Call("createElement", "img")
			img.Set("src", m.getTileURL(tileX, tileY, baseZoom))
			img.Set("draggable", false)
			style := img.Get("style")
			style.Set("position", "absolute")
			style.Set("left", fmt.Sprintf("%.3fpx", float64(tileX*tileSize)-tx))
			style.Set("top", fmt.Sprintf("%.3fpx", float64(tileY*tileSize)-ty))
			m.tileContainer.Call("appendChild", img)
		}
	}
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

	scale := math.Pow(2, m.Zoom)
	lngPerPixel := 360.0 / (scale * tileSize)
	latPerPixel := 180.0 / (scale * tileSize) // Approximation

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

	cursorX := e.Get("clientX").Float()
	cursorY := e.Get("clientY").Float()

	lat1, lng1 := m.pixelToLatLng(cursorX, cursorY, m.Zoom)

	delta := e.Get("deltaY").Float()
	m.Zoom -= delta * zoomSpeed
	if m.Zoom < minZoom {
		m.Zoom = minZoom
	}
	if m.Zoom > maxZoom {
		m.Zoom = maxZoom
	}

	lat2, lng2 := m.pixelToLatLng(cursorX, cursorY, m.Zoom)

	m.CenterLat += lat1 - lat2
	m.CenterLng += lng1 - lng2

	m.DrawMap()
}

// --- Coordinate Conversion Utilities ---

func (m *IchthyoMapView) latLngToPixel(lat, lng, zoom float64) (float64, float64) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, zoom)
	x := (lng + 180.0) / 360.0 * n * tileSize
	y := (1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n * tileSize
	return x, y
}

func (m *IchthyoMapView) pixelToLatLng(px, py, zoom float64) (float64, float64) {
	screenWidth := js.Global().Get("innerWidth").Float()
	screenHeight := js.Global().Get("innerHeight").Float()

	centerX, centerY := m.latLngToPixel(m.CenterLat, m.CenterLng, zoom)

	tx := centerX - screenWidth/2
	ty := centerY - screenHeight/2

	worldX := px + tx
	worldY := py + ty

	n := math.Pow(2.0, zoom)
	lng := (worldX/tileSize)/n*360.0 - 180.0
	latRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*worldY/(n*tileSize))))
	lat := latRad * 180.0 / math.Pi
	return lat, lng
}

func (m *IchthyoMapView) getTileURL(x, y, z int) string {
	// Assuming tileProvider is handled by a parent component or is hardcoded for now
	return fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", z, x, y)
}
