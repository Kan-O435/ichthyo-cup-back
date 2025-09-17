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
		return elem.Body(
			a.mapView,
			a.uiView,
		)
	case "#/signup":
		return elem.Body(&SignupPage{})
	case "#/login":
		fallthrough
	default:
		loginPage := &LoginPage{
			OnLogin: func() {
				js.Global().Get("location").Set("hash", "#/map")
			},
		}
		// Add a link to the signup page
		return elem.Body(
			loginPage,
			elem.Div(
				vecty.Markup(vecty.Style("text-align", "center"), vecty.Style("margin-top", "20px")),
				elem.Anchor(vecty.Text("Don't have an account? Sign Up"),
					vecty.Markup(vecty.Property("href", "#/signup")),
				),
			),
		)
	}
}
