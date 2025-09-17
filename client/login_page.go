package main

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// LoginPage is a component that displays a login form.
type LoginPage struct {
	vecty.Core
	username   string
	password   string
	message    string
	OnLogin    func()
}

// Render renders the component.
func (p *LoginPage) Render() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Style("display", "flex"),
			vecty.Style("justify-content", "center"),
			vecty.Style("align-items", "center"),
			vecty.Style("height", "100vh"),
			vecty.Style("background-color", "#282c34"),
			vecty.Style("color", "white"),
		),
		elem.Form(
			vecty.Markup(
				vecty.Style("padding", "40px"),
				vecty.Style("background-color", "#20232a"),
				vecty.Style("border-radius", "8px"),
				vecty.Style("box-shadow", "0 4px 8px rgba(0,0,0,0.3)"),
				event.Submit(p.onLoginAttempt).PreventDefault(),
			),
			elem.Heading1(vecty.Text("Login")),
			elem.Div(
				elem.Label(vecty.Text("Username:")),
				elem.Input(vecty.Markup(
					vecty.Property("type", "text"),
					event.Input(func(e *vecty.Event) {
						p.username = e.Target.Get("value").String()
					}),
				)),
			),
			elem.Div(
				elem.Label(vecty.Text("Password:")),
				elem.Input(vecty.Markup(
					vecty.Property("type", "password"),
					event.Input(func(e *vecty.Event) {
						p.password = e.Target.Get("value").String()
					}),
				)),
			),
			elem.Button(vecty.Text("Login"), vecty.Markup(vecty.Property("type", "submit"))),
			p.renderMessage(),
		),
	)
}

func (p *LoginPage) renderMessage() vecty.ComponentOrHTML {
	if p.message != "" {
		return elem.Paragraph(vecty.Text(p.message))
	}
	return nil
}

func (p *LoginPage) onLoginAttempt(e *vecty.Event) {
	if p.username == "admin" && p.password == "admin1234" {
		p.message = "Login successful!"
		if p.OnLogin != nil {
			p.OnLogin()
		}
	} else {
		p.message = "Invalid credentials"
	}
	vecty.Rerender(p)
}
