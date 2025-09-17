package main

import (
	"fmt"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// UIView is a component that displays the map's UI controls.
type UIView struct {
	vecty.Core
	MapView *IchthyoMapView `vecty:"prop"`

	// UI State
	isControlsVisible bool
	tileProvider      int
}

// NewUIView creates a new UIView
func NewUIView(mapView *IchthyoMapView) *UIView {
	return &UIView{
		MapView:           mapView,
		isControlsVisible: true,
	}
}

// Render renders the UI components.
func (u *UIView) Render() vecty.ComponentOrHTML {
	return elem.Div(
		u.renderZoomControls(),
		u.renderCoordinateInfo(),
		u.renderControlToggle(),
		u.renderConditionalSidePanel(),
	)
}

func (u *UIView) renderZoomControls() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("top", "20px"), vecty.Style("left", "20px"), vecty.Style("zIndex", "1001")),
		elem.Button(vecty.Text("+"), vecty.Markup(event.Click(func(e *vecty.Event) { 
			u.MapView.Zoom += 0.5 
			u.MapView.DrawMap()
		}))),
		elem.Button(vecty.Text("-"), vecty.Markup(event.Click(func(e *vecty.Event) { 
			u.MapView.Zoom -= 0.5
			u.MapView.DrawMap()
		}))),
	)
}

func (u *UIView) renderCoordinateInfo() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("bottom", "20px"), vecty.Style("right", "20px"), vecty.Style("background", "rgba(0,0,0,0.7)"), vecty.Style("color", "white"), vecty.Style("padding", "5px 10px"), vecty.Style("borderRadius", "3px"), vecty.Style("zIndex", "1001")),
		vecty.Text(fmt.Sprintf("Lat: %.4f, Lng: %.4f, Zoom: %.2f", u.MapView.CenterLat, u.MapView.CenterLng, u.MapView.Zoom)),
	)
}

func (u *UIView) renderControlToggle() vecty.ComponentOrHTML {
	return elem.Button(
		vecty.Text("☰"),
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("top", "20px"), vecty.Style("right", "20px"), vecty.Style("zIndex", "1001"), event.Click(u.toggleControls)),
	)
}

func (u *UIView) toggleControls(e *vecty.Event) {
	u.isControlsVisible = !u.isControlsVisible
	vecty.Rerender(u)
}

func (u *UIView) renderConditionalSidePanel() vecty.ComponentOrHTML {
	if !u.isControlsVisible {
		return nil
	}
	return u.renderSidePanel()
}

func (u *UIView) renderSidePanel() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Style("position", "fixed"), vecty.Style("top", "0"), vecty.Style("right", "0"), vecty.Style("width", "250px"), vecty.Style("height", "100%"), vecty.Style("background", "rgba(26, 26, 26, 0.95)"), vecty.Style("color", "white"), vecty.Style("padding", "10px"), vecty.Style("zIndex", "1000"), vecty.Style("overflowY", "auto")),
		elem.Heading4(vecty.Text("Map Settings")),
		elem.Button(vecty.Text("OSM"), vecty.Markup(event.Click(func(e *vecty.Event) { u.tileProvider = 0; u.MapView.DrawMap() }))),
		elem.Button(vecty.Text("CartoDB"), vecty.Markup(event.Click(func(e *vecty.Event) { u.tileProvider = 1; u.MapView.DrawMap() }))),
	)
}
