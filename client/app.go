package main

import (
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
)

// App is the main application component, acting as a router.
type App struct {
	vecty.Core
	currentRoute string
	mapView      *IchthyoMapView
	uiView       *UIView
}

// NewApp creates a new App component.
func NewApp() *App {
	app := &App{}
	app.mapView = NewIchthyoMapView()
	app.uiView = NewUIView(app.mapView)
	return app
}

// Mount handles component mounting and sets up routing.
func (a *App) Mount() {
	a.handleRouteChange(js.Undefined(), nil)

	js.Global().Set("onhashchange", js.FuncOf(a.handleRouteChange))
}

func (a *App) handleRouteChange(this js.Value, args []js.Value) interface{} {
	newRoute := js.Global().Get("location").Get("hash").String()
	if newRoute == "" {
		newRoute = "#/login" // Default route
	}
	a.currentRoute = newRoute
	vecty.Rerender(a)
	return nil
}

// Render renders the component based on the current route.
func (a *App) Render() vecty.ComponentOrHTML {
	switch a.currentRoute {
	case "#/map":
		// 新しいWplaceアプリ（wplace_leaflet.html）にリダイレクト
		js.Global().Get("location").Set("href", "/wplace")
		return elem.Body(
			elem.Div(
				vecty.Text("リダイレクト中..."),
			),
		)
	case "#/signup":
		return elem.Body(&SignupPage{})
	case "#/login":
		fallthrough
	default:
		loginPage := &LoginPage{
			OnLogin: func() {
				js.Global().Get("location").Set("href", "/map")
			},
		}
		return elem.Body(loginPage)
	}
}
