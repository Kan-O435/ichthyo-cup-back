package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// Constants
const (
	minZoom       = 1
	maxZoom       = 18
	tileSize      = 256
	zoomSpeed     = 0.01
	cellGridSize  = 16 // Each tile is a 16x16 grid of cells
	paintMinZoom  = 15 // Minimum zoom level to load/paint cells
	selectionColor = "rgba(255, 0, 0, 0.5)" // Semi-transparent red for selection
)

// --- Structs for API communication ---

// TileCell represents a single colored cell from the API (for GET response)
type TileCell struct {
	CellX  int    `json:"cellX"`
	CellY  int    `json:"cellY"`
	Color  string `json:"color"`
	UserID string `json:"userId"`
}

// PaintGetResponse is the structure for the response from GET /api/paint
type PaintGetResponse struct {
	Zoom   int        `json:"zoom"`
	TileX  int        `json:"tile_x"`
	TileY  int        `json:"tile_y"`
	Cells  []TileCell `json:"cells"`
}

// PaintCellPayload is a single cell sent in a POST request
type PaintCellPayload struct {
	CellX int    `json:"cell_x"`
	CellY int    `json:"cell_y"`
	Color string `json:"color"`
}

// PaintPostRequest is the structure for the body of POST /api/paint
type PaintPostRequest struct {
	UserID string             `json:"user_id"`
	Zoom   int                `json:"zoom"`
	TileX  int                `json:"tile_x"`
	TileY  int                `json:"tile_y"`
	Cells  []PaintCellPayload `json:"cells"`
}

// --- Component-specific Structs ---

// Point represents a 2D point
type Point struct {
	X, Y int
}

// SelectedCellInfo holds all context for a cell the user has selected.
type SelectedCellInfo struct {
	TileX   int
	TileY   int
	Payload PaintCellPayload
}

// IchthyoMapView is the main map component
type IchthyoMapView struct {
	vecty.Core

	CenterLat float64 `vecty:"prop"`
	CenterLng float64 `vecty:"prop"`
	Zoom      float64 `vecty:"prop"`

	tileContainer         js.Value
	isMounted             bool
	isDragging            bool
	dragStart             Point
	lastDrag              Point
	lastDragForClickCheck Point

	SelectedCells map[string]SelectedCellInfo `vecty:"prop"`
	OnSelectionChange func() `vecty:"prop"` // Callback to trigger re-render of UIView
	CurrentUserID string `vecty:"prop"` // The ID of the currently logged-in user
	SelectedColor string `vecty:"prop"` // The currently selected color for painting

	paintCache map[string][]TileCell // key: z-x-y, value: cells for the tile

	isRedrawScheduled bool
	lastRedrawMs      int
}

// NewIchthyoMapViewWithOptions creates a map view with explicit options.
func NewIchthyoMapViewWithOptions(onSelectionChange func(), userID string, selectedColor string) *IchthyoMapView {
	if onSelectionChange == nil {
		onSelectionChange = func() {}
	}
	return &IchthyoMapView{
		CenterLat:         35.6762,
		CenterLng:         139.6503,
		Zoom:              16,
		SelectedCells:     make(map[string]SelectedCellInfo),
		OnSelectionChange: onSelectionChange,
		CurrentUserID:     userID,
		SelectedColor:     selectedColor,
		paintCache:        make(map[string][]TileCell),
	}
}

// NewIchthyoMapView provides a zero-arg constructor for existing call sites.
func NewIchthyoMapView() *IchthyoMapView {
	return NewIchthyoMapViewWithOptions(func() {}, "", "#FF0000")
}

// --- Component Lifecycle & Rendering ---

func (m *IchthyoMapView) Mount() {
	m.tileContainer = js.Global().Get("document").Call("createElement", "div")
	style := m.tileContainer.Get("style")
	style.Set("position", "absolute")
	style.Set("top", "0")
	style.Set("left", "0")
	style.Set("will-change", "transform")

	// ビューポート要素が確実にDOMに現れるまでリトライ
	retries := 0
	var attachFunc js.Func
	attachFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		viewport := js.Global().Get("document").Call("querySelector", ".map-viewport")
		if !viewport.IsUndefined() && !viewport.IsNull() {
			viewport.Call("appendChild", m.tileContainer)
			m.isMounted = true
			m.DrawMap()
			// 停止
			return nil
		}
		retries++
		if retries < 30 { // 約1秒（30*~33ms）
			js.Global().Call("requestAnimationFrame", attachFunc)
		}
		return nil
	})
	js.Global().Call("requestAnimationFrame", attachFunc)
}

func (m *IchthyoMapView) Unmount() {
	if m.isMounted {
		m.tileContainer.Call("remove")
	}
	m.isMounted = false
}

func (m *IchthyoMapView) Render() vecty.ComponentOrHTML {
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

// --- Map Drawing ---

func (m *IchthyoMapView) DrawMap() {
	if !m.isMounted {
		return
	}

	m.isRedrawScheduled = false
	m.lastRedrawMs = js.Global().Get("Date").New().Call("getTime").Int()

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

			canvas := js.Global().Get("document").Call("createElement", "canvas")
			canvas.Set("id", fmt.Sprintf("tile-canvas-%d-%d-%d", baseZoom, tileX, tileY))
			canvas.Set("width", tileSize)
			canvas.Set("height", tileSize)

			style := canvas.Get("style")
			style.Set("position", "absolute")
			style.Set("left", fmt.Sprintf("%.3fpx", float64(tileX*tileSize)-tx))
			style.Set("top", fmt.Sprintf("%.3fpx", float64(tileY*tileSize)-ty))

			m.tileContainer.Call("appendChild", canvas)
			m.drawTile(canvas, tileX, tileY, baseZoom, scale)
		}
	}
}

func (m *IchthyoMapView) scheduleDraw() {
	if m.isRedrawScheduled {
		return
	}
	minInterval := 80 // ms
	now := js.Global().Get("Date").New().Call("getTime").Int()
	delay := minInterval - (now - m.lastRedrawMs)
	if delay < 0 {
		delay = 0
	}
	m.isRedrawScheduled = true
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		js.Global().Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			m.DrawMap()
			return nil
		}))
		return nil
	}), delay)
}

func (m *IchthyoMapView) drawTile(canvas js.Value, tileX, tileY, zoom int, scale float64) {
	ctx := canvas.Call("getContext", "2d")

	// まずプレースホルダ背景
	ctx.Set("fillStyle", "#f2f2f2")
	ctx.Call("fillRect", 0, 0, tileSize, tileSize)

	// すぐにキャッシュと選択を描画（タイル画像が遅れても見えるように）
	m.drawCachedCellsForTile(ctx, zoom, tileX, tileY)
	m.drawSelectionsForTile(ctx, tileX, tileY, scale)

	// 背景タイル読み込み（成功時に下地として描く）
	img := js.Global().Get("Image").New()
	img.Set("src", m.getTileURL(tileX, tileY, zoom))

	img.Call("addEventListener", "load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// 背景を先に描いて、その上に再度オーバーレイ
		ctx.Call("drawImage", img, 0, 0)
		m.drawCachedCellsForTile(ctx, zoom, tileX, tileY)
		m.drawSelectionsForTile(ctx, tileX, tileY, scale)
		return nil
	}))

	img.Call("addEventListener", "error", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// タイル画像が読めなくてもキャッシュと選択は表示済み
		return nil
	}))
}

func (m *IchthyoMapView) fetchAndDrawCells(ctx js.Value, zoom, tileX, tileY int, scale float64) {
	if zoom < paintMinZoom {
		return
	}

	url := fmt.Sprintf("https://hack-s-ikuthio-2025.vercel.app/api/paint?zoom=%d&tile_x=%d&tile_y=%d", zoom, tileX, tileY)

	getRequest(url, func(responseBody string) {
		var data PaintGetResponse
		if err := json.Unmarshal([]byte(responseBody), &data); err != nil {
			fmt.Println("Failed to unmarshal paint data:", err)
			return
		}

		// Update cache
		key := m.tileKey(zoom, tileX, tileY)
		m.paintCache[key] = data.Cells

		// Fixed cell pixel size regardless of zoom scale
		cellPixelSize := float64(tileSize) / float64(cellGridSize)

		for _, cell := range data.Cells {
			ctx.Set("fillStyle", cell.Color)
			ctx.Call("fillRect", float64(cell.CellX)*cellPixelSize, float64(cell.CellY)*cellPixelSize, cellPixelSize, cellPixelSize)
		}

		m.drawSelectionsForTile(ctx, tileX, tileY, scale)

	}, func(errText string) {
		fmt.Println("Failed to fetch paint data:", errText)
	})
}

func (m *IchthyoMapView) drawCachedCellsForTile(ctx js.Value, zoom, tileX, tileY int) {
	cells, ok := m.paintCache[m.tileKey(zoom, tileX, tileY)]
	if !ok {
		return
	}
	cellPixelSize := float64(tileSize) / float64(cellGridSize)
	for _, cell := range cells {
		ctx.Set("fillStyle", cell.Color)
		ctx.Call("fillRect", float64(cell.CellX)*cellPixelSize, float64(cell.CellY)*cellPixelSize, cellPixelSize, cellPixelSize)
	}
}

func (m *IchthyoMapView) drawSelectionsForTile(ctx js.Value, tileX, tileY int, scale float64) {
	cellPixelSize := float64(tileSize) / float64(cellGridSize)
	
	for _, selection := range m.SelectedCells {
		if selection.TileX == tileX && selection.TileY == tileY {
			ctx.Set("fillStyle", selection.Payload.Color)
			ctx.Call("fillRect", float64(selection.Payload.CellX)*cellPixelSize, float64(selection.Payload.CellY)*cellPixelSize, cellPixelSize, cellPixelSize)
		}
	}
}

// --- Event Handlers & Painting Logic ---

func (m *IchthyoMapView) onMouseDown(e *vecty.Event) {
	e.Call("preventDefault")
	m.isDragging = true
	pos := Point{X: e.Get("clientX").Int(), Y: e.Get("clientY").Int()}
	m.dragStart = pos
	m.lastDrag = pos
	m.lastDragForClickCheck = pos
	js.Global().Get("document").Get("body").Get("style").Set("cursor", "grabbing")
}

func (m *IchthyoMapView) onMouseMove(e *vecty.Event) {
	if !m.isDragging {
		return
	}
	e.Call("preventDefault")
	currentPos := Point{X: e.Get("clientX").Int(), Y: e.Get("clientY").Int()}
	m.lastDragForClickCheck = currentPos

	dx := float64(m.lastDrag.X - currentPos.X)
	dy := float64(m.lastDrag.Y - currentPos.Y)

	scale := math.Pow(2, m.Zoom)
	lngPerPixel := 360.0 / (scale * tileSize)
	latPerPixel := 180.0 / (scale * tileSize)

	m.CenterLng += dx * lngPerPixel
	m.CenterLat -= dy * latPerPixel

	m.lastDrag = currentPos
	m.scheduleDraw()
}

func (m *IchthyoMapView) onMouseUp(e *vecty.Event) {
	if !m.isDragging {
		return
	}
	m.isDragging = false
	js.Global().Get("document").Get("body").Get("style").Set("cursor", "grab")

	dx := math.Abs(float64(m.lastDragForClickCheck.X - m.dragStart.X))
	dy := math.Abs(float64(m.lastDragForClickCheck.Y - m.dragStart.Y))

	if dx < 5 && dy < 5 {
		m.handleClick(e)
	}
}

func (m *IchthyoMapView) handleClick(e *vecty.Event) {
	baseZoom := int(math.Ceil(m.Zoom))
	if baseZoom < paintMinZoom {
		fmt.Println("Zoom in further to paint!")
		return
	}

	cursorX := e.Get("clientX").Float()
	cursorY := e.Get("clientY").Float()

	lat, lng := m.pixelToLatLng(cursorX, cursorY, m.Zoom)
	worldX, worldY := m.latLngToPixel(lat, lng, float64(baseZoom))

	tileX := int(math.Floor(worldX / tileSize))
	tileY := int(math.Floor(worldY / tileSize))

	cellPixelSize := float64(tileSize) / float64(cellGridSize)
	cellX := int(math.Floor(math.Mod(worldX, tileSize) / cellPixelSize))
	cellY := int(math.Floor(math.Mod(worldY, tileSize) / cellPixelSize))

	cellKey := fmt.Sprintf("%d-%d-%d-%d", tileX, tileY, cellX, cellY)
	wasSelected := false
	if _, exists := m.SelectedCells[cellKey]; exists {
		delete(m.SelectedCells, cellKey)
		wasSelected = true
	} else {
		color := m.SelectedColor // Use the selected color from the UI
		m.SelectedCells[cellKey] = SelectedCellInfo{
			TileX: tileX,
			TileY: tileY,
			Payload: PaintCellPayload{CellX: cellX, CellY: cellY, Color: color},
		}
	}

	canvasID := fmt.Sprintf("tile-canvas-%d-%d-%d", baseZoom, tileX, tileY)
	canvas := js.Global().Get("document").Call("getElementById", canvasID)
	if !canvas.IsUndefined() && !canvas.IsNull() {
		scale := math.Pow(2, m.Zoom-float64(baseZoom))
		
		// Draw selection immediately on the canvas
		ctx := canvas.Call("getContext", "2d")
		cellPixelSize := float64(tileSize) / float64(cellGridSize)
		
		if wasSelected {
			// Clear and redraw the tile when deselecting
			ctx.Call("clearRect", 0, 0, tileSize, tileSize)
			// Redraw the background tile
			img := js.Global().Get("Image").New()
			img.Set("crossOrigin", "anonymous")
			img.Set("src", m.getTileURL(tileX, tileY, baseZoom))
			img.Call("addEventListener", "load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				ctx.Call("drawImage", img, 0, 0)
				m.drawSelectionsForTile(ctx, tileX, tileY, scale)
				return nil
			}))
		} else {
			// Draw the new selection immediately with the selected color
			ctx.Set("fillStyle", m.SelectedColor)
			ctx.Call("fillRect", float64(cellX)*cellPixelSize, float64(cellY)*cellPixelSize, cellPixelSize, cellPixelSize)
		}
	}
	if m.OnSelectionChange != nil {
		m.OnSelectionChange()
	}
}

func (m *IchthyoMapView) CommitSelection() {
	if len(m.SelectedCells) == 0 {
		return
	}

	baseZoom := int(math.Ceil(m.Zoom))
	userID := m.CurrentUserID // Use the actual user ID

	if userID == "" {
		fmt.Println("Error: User not logged in. Cannot paint.")
		return
	}

	groups := make(map[string][]PaintCellPayload)
	for _, selection := range m.SelectedCells {
		tileKey := fmt.Sprintf("%d-%d", selection.TileX, selection.TileY)
		groups[tileKey] = append(groups[tileKey], selection.Payload)
	}

	for tileKey, cells := range groups {
		parts := strings.Split(tileKey, "-")
		tileX, _ := strconv.Atoi(parts[0])
		tileY, _ := strconv.Atoi(parts[1])

		payload := PaintPostRequest{
			UserID: userID,
			Zoom:   baseZoom,
			TileX:  tileX,
			TileY:  tileY,
			Cells:  cells,
		}

		requestBody, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("Failed to create paint request for tile %s: %v\n", tileKey, err)
			continue
		}

		postRequest("https://hack-s-ikuthio-2025.vercel.app/api/paint", requestBody, func(responseBody string) {
			fmt.Printf("Paint successful for tile %s: %s\n", tileKey, responseBody)
		}, func(errText string) {
			fmt.Printf("Paint failed for tile %s: %s\n", tileKey, errText)
		})
	}

	m.SelectedCells = make(map[string]SelectedCellInfo)
	m.DrawMap()
	if m.OnSelectionChange != nil {
		m.OnSelectionChange()
	}
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

	m.scheduleDraw()
}

// --- Coordinate Conversion & Helpers ---

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

	baseZoom := int(math.Ceil(zoom))
	scale := math.Pow(2, zoom-float64(baseZoom))

	centerX, centerY := m.latLngToPixel(m.CenterLat, m.CenterLng, float64(baseZoom))

	tx := centerX - (screenWidth/2)/scale
	ty := centerY - (screenHeight/2)/scale

	worldX := (px / scale) + tx
	worldY := (py / scale) + ty

	n := math.Pow(2.0, float64(baseZoom))
	lng := (worldX/tileSize)/n*360.0 - 180.0
	latRad := math.Atan(math.Sinh(math.Pi * (1.0 - 2.0*worldY/(n*tileSize))))
	lat := latRad * 180.0 / math.Pi
	return lat, lng
}

func (m *IchthyoMapView) getTileURL(x, y, z int) string {
	// Nginxの/tiles/経由でOSMにプロキシ（CORS回避）
	return fmt.Sprintf("/tiles/%d/%d/%d.png", z, x, y)
}

func (m *IchthyoMapView) tileKey(zoom, tileX, tileY int) string {
	return fmt.Sprintf("%d-%d-%d", zoom, tileX, tileY)
}

// --- HTTP Helpers ---

func getRequest(url string, onSuccess, onError func(string)) {
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			if onError != nil {
				onError("Request failed: " + err.Error())
			}
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if onError != nil {
				onError("Failed to read response body: " + err.Error())
			}
			return
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if onSuccess != nil {
				onSuccess(string(body))
			}
		} else {
			if onError != nil {
				onError(fmt.Sprintf("API Error (status %d): %s", resp.StatusCode, string(body)))
			}
		}
	}()
}

func postRequest(url string, requestBody []byte, onSuccess, onError func(string)) {
	go func() {
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			if onError != nil {
				onError("Request failed: " + err.Error())
			}
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if onError != nil {
				onError("Failed to read response body: " + err.Error())
			}
			return
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if onSuccess != nil {
				onSuccess(string(body))
			}
		} else {
			if onError != nil {
				onError(fmt.Sprintf("API Error (status %d): %s", resp.StatusCode, string(body)))
			}
		}
	}()
}
