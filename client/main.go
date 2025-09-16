package main

import (
	"fmt"
	"math"
	"strconv"
	"syscall/js"
	"time"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

const (
	minZoom     = 1
	maxZoom     = 18
	tileSize    = 256
	zoomSpeed   = 0.065
	lerpFactor  = 0.15 // For smooth animation
)

// MapView is the main map component
type MapView struct {
	vecty.Core

	// --- State ---
	centerLat float64
	centerLng float64
	zoom      float64 // The actual, fractional zoom level

	// --- UI State ---
	isControlsVisible bool
	tileProvider      int

	// --- Internal Animation & Interaction State ---
	tileContainer     js.Value
	isMounted         bool
	isDragging        bool
	dragStart         Point
	lastDrag          Point
	animationFrame    int
	lastWheelTime     time.Time
}

// Point represents a 2D integer point
type Point struct {
	X, Y int
}

// NewMapView creates a new map view
func NewMapView() *MapView {
	return &MapView{
		centerLat:         35.6762, // Tokyo
		centerLng:         139.6503,
		zoom:              10,
		isControlsVisible: true,
	}
}

// --- Component Lifecycle ---

func (m *MapView) Mount() {
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
			m.startAnimationLoop()
		}
		return nil
	}), 0)
}

func (m *MapView) Unmount() {
	if m.isMounted {
		m.tileContainer.Call("remove")
	}
	m.isMounted = false
	if m.animationFrame != 0 {
		js.Global().Call("cancelAnimationFrame", m.animationFrame)
	}
}

// --- Rendering & Animation Loop ---

func (m *MapView) Render() vecty.ComponentOrHTML {
	return elem.Body(
		vecty.Markup(
			vecty.Style("margin", "0"),
			vecty.Style("overflow", "hidden"),
			vecty.Style("background", "#1a1a1a"),
		),
		elem.Div(
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
		),
		m.renderUIs(),
	)
}

func (m *MapView) startAnimationLoop() {
	var renderFrame js.Func
	renderFrame = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		m.drawMap()
		m.animationFrame = js.Global().Call("requestAnimationFrame", renderFrame).Int()
		return nil
	})
	m.animationFrame = js.Global().Call("requestAnimationFrame", renderFrame).Int()
}

func (m *MapView) drawMap() {
	if !m.isMounted {
		return
	}

	// Clear the container
	m.tileContainer.Set("innerHTML", "")

	// Get viewport dimensions
	screenWidth := js.Global().Get("innerWidth").Int()
	screenHeight := js.Global().Get("innerHeight").Int()

	// Determine the integer zoom level for which to fetch tiles
	baseZoom := int(math.Ceil(m.zoom))
	if baseZoom < minZoom {
		baseZoom = minZoom
	}
	if baseZoom > maxZoom {
		baseZoom = maxZoom
	}

	// Calculate the scale factor from the base zoom
	scale := math.Pow(2, m.zoom-float64(baseZoom))

	// Calculate the pixel coordinates of the center of the screen in the world map
	centerX, centerY := m.latLngToPixel(m.centerLat, m.centerLng, float64(baseZoom))

	// Calculate the top-left corner of the visible map area in world pixels
	tx := centerX - (float64(screenWidth)/2)/scale
	ty := centerY - (float64(screenHeight)/2)/scale

	// Calculate the tile coordinates of the top-left corner
	startTileX := int(math.Floor(tx / tileSize))
	startTileY := int(math.Floor(ty / tileSize))

	// Calculate how many tiles we need to draw
	numTilesX := int(math.Ceil(float64(screenWidth)/(tileSize*scale))) + 1
	numTilesY := int(math.Ceil(float64(screenHeight)/(tileSize*scale))) + 1

	// Set the transform on the container to position and scale the entire tile set
	containerStyle := m.tileContainer.Get("style")
	containerStyle.Set("transform", fmt.Sprintf("scale(%.6f)", scale))
	containerStyle.Set("transform-origin", "top left")

	// Draw the tiles
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

func (m *MapView) onMouseDown(e *vecty.Event) {
	e.Call("preventDefault")
	m.isDragging = true
	m.dragStart = Point{X: e.Get("clientX").Int(), Y: e.Get("clientY").Int()}
	m.lastDrag = m.dragStart
	js.Global().Get("document").Get("body").Get("style").Set("cursor", "grabbing")
}

func (m *MapView) onMouseMove(e *vecty.Event) {
	if !m.isDragging {
		return
	}
	e.Call("preventDefault")
	currentPos := Point{X: e.Get("clientX").Int(), Y: e.Get("clientY").Int()}
	dx := float64(m.lastDrag.X - currentPos.X)
	dy := float64(m.lastDrag.Y - currentPos.Y)

	// Convert pixel delta to lat/lng delta
	scale := math.Pow(2, m.zoom)
	lngPerPixel := 360.0 / (scale * tileSize)
	latPerPixel := 180.0 / (scale * tileSize) // Approximation

	m.centerLng += dx * lngPerPixel
	m.centerLat -= dy * latPerPixel

	m.lastDrag = currentPos
	vecty.Rerender(m) // For coordinate display
}

func (m *MapView) onMouseUp(e *vecty.Event) {
	if !m.isDragging {
		return
	}
	m.isDragging = false
	js.Global().Get("document").Get("body").Get("style").Set("cursor", "default")
}

func (m *MapView) onWheel(e *vecty.Event) {
	e.Call("preventDefault")

	// Get mouse position relative to the viewport
	cursorX := e.Get("clientX").Float()
	cursorY := e.Get("clientY").Float()

	// Get the lat/lng of the point under the cursor before the zoom
	lat1, lng1 := m.pixelToLatLng(cursorX, cursorY, m.zoom)

	// Update the zoom level
	delta := e.Get("deltaY").Float()
	m.zoom -= delta * zoomSpeed
	if m.zoom < minZoom {
		m.zoom = minZoom
	}
	if m.zoom > maxZoom {
		m.zoom = maxZoom
	}

	// Get the lat/lng of the point under the cursor after the zoom
	lat2, lng2 := m.pixelToLatLng(cursorX, cursorY, m.zoom)

	// Adjust the map center to keep the point under the cursor stationary
	m.centerLat += lat1 - lat2
	m.centerLng += lng1 - lng2

	vecty.Rerender(m) // For coordinate display
}

// --- Coordinate Conversion Utilities ---

func (m *MapView) latLngToPixel(lat, lng, zoom float64) (float64, float64) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2.0, zoom)
	x := (lng + 180.0) / 360.0 * n * tileSize
	y := (1.0 - math.Asinh(math.Tan(latRad))/math.Pi) / 2.0 * n * tileSize
	return x, y
}

func (m *MapView) pixelToLatLng(px, py, zoom float64) (float64, float64) {
	screenWidth := js.Global().Get("innerWidth").Float()
	screenHeight := js.Global().Get("innerHeight").Float()

	// Calculate the world pixel coordinates of the screen center
	centerX, centerY := m.latLngToPixel(m.centerLat, m.centerLng, zoom)

	// Calculate the world pixel coordinates of the top-left corner of the screen
	tx := centerX - screenWidth/2
	ty := centerY - screenHeight/2

	// Convert the given screen pixel to world pixel
	worldX := px + tx
	worldY := py + ty

	// Convert world pixel to lat/lng
	n := math.Pow(2.0, zoom)
	lng := (worldX/tileSize)/n*360.0 - 180.0
	latRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*worldY/(n*tileSize))))
	lat := latRad * 180.0 / math.Pi
	return lat, lng
}

func (m *MapView) getTileURL(x, y, z int) string {
	endpoint := "https://tile.openstreetmap.org"
	if m.tileProvider == 1 {
		endpoint = "https://cartodb-basemaps-a.global.ssl.fastly.net/light_all"
	}
	return fmt.Sprintf("%s/%d/%d/%d.png", endpoint, z, x, y)
}

// --- UI Components ---

func (m *MapView) renderUIs() vecty.ComponentList {
	return vecty.ComponentList{
		m.renderZoomControls(),
		m.renderCoordinateInfo(),
		m.renderControlToggle(),
		m.renderConditionalSidePanel(),
	}
}

func (m *MapView) renderZoomControls() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("top", "20px"), vecty.Style("left", "20px"), vecty.Style("zIndex", "1001")),
		elem.Button(vecty.Text("+"), vecty.Markup(event.Click(func(e *vecty.Event) { m.zoom += 0.5 }))),
		elem.Button(vecty.Text("-"), vecty.Markup(event.Click(func(e *vecty.Event) { m.zoom -= 0.5 }))),
	)
}

func (m *MapView) renderCoordinateInfo() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("bottom", "20px"), vecty.Style("right", "20px"), vecty.Style("background", "rgba(0,0,0,0.7)"), vecty.Style("color", "white"), vecty.Style("padding", "5px 10px"), vecty.Style("borderRadius", "3px"), vecty.Style("zIndex", "1001")),
		vecty.Text(fmt.Sprintf("Lat: %.4f, Lng: %.4f, Zoom: %.2f", m.centerLat, m.centerLng, m.zoom)),
	)
}

func (m *MapView) renderControlToggle() vecty.ComponentOrHTML {
	return elem.Button(
		vecty.Text("☰"),
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("top", "20px"), vecty.Style("right", "20px"), vecty.Style("zIndex", "1001"), event.Click(func(e *vecty.Event) { m.toggleControls() })),
	)
}

func (m *MapView) toggleControls() {
	m.isControlsVisible = !m.isControlsVisible
	vecty.Rerender(m)
}

func (m *MapView) renderConditionalSidePanel() vecty.ComponentOrHTML {
	if !m.isControlsVisible {
		return nil
	}
	return m.renderSidePanel()
}

func (m *MapView) renderSidePanel() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("top", "0"), vecty.Style("right", "0"), vecty.Style("width", "250px"), vecty.Style("height", "100%"), vecty.Style("background", "rgba(26, 26, 26, 0.95)"), vecty.Style("color", "white"), vecty.Style("padding", "10px"), vecty.Style("zIndex", "1000"), vecty.Style("overflowY", "auto")),
		elem.Heading4(vecty.Text("Map Settings")),
		elem.Button(vecty.Text("OSM"), vecty.Markup(event.Click(func(e *vecty.Event) { m.tileProvider = 0 }))),
		elem.Button(vecty.Text("CartoDB"), vecty.Markup(event.Click(func(e *vecty.Event) { m.tileProvider = 1 }))),
	)
}

func main() {
	vecty.SetTitle("Vecty Map")
	vecty.RenderBody(NewMapView())
	select {}
}
