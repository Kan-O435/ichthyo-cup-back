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
}

// NewUIView creates a new UIView
func NewUIView(mapView *IchthyoMapView) *UIView {
	return &UIView{
		MapView: mapView,
	}
}

// Render renders the UI components.
func (u *UIView) Render() vecty.ComponentOrHTML {
	return elem.Div(
		u.renderZoomControls(),
		u.renderCoordinateInfo(),
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
