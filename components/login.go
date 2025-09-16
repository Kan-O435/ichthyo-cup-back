package components

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

type Login struct {
	vecty.Core
	username string
	password string
	message string
}

func (l *Login) Render() vecty.ComponentOrHTML {
	return elem.Div(
		elem.Div(
			elem.Heading1(vecty.Text("Login")),

			elem.Form(
				vecty.Markup(event.Submit(func(e *vecty.Event) {
					// PreventDefault is handled by Vecty automatically
					if l.username == "admin" && l.password == "admin1234" {
						l.message = "Login successful"
					} else {
						l.message = "Invalid credentials"
					}
					vecty.Rerender(l)
				})),

				elem.Div(
					elem.Label(
						vecty.Text("username : "),
					),
					elem.Input(
						vecty.Markup(
							vecty.Property("type", "text"),
							event.Input(func(e *vecty.Event) {
								l.username = e.Target.Get("value").String()
							}),
						),
					),
				),

				elem.Div(
					elem.Label(
						vecty.Text("password : "),
					),
					elem.Input(
						vecty.Markup(
							vecty.Property("type", "password"),
							event.Input(func(e *vecty.Event) {
								l.password = e.Target.Get("value").String()
							}),
						),
					),
				),
				
				elem.Button(
					vecty.Text("Login"),
					vecty.Markup(vecty.Property("type", "submit")),
				),
			),
		),
		
		elem.Paragraph(
			vecty.Text(l.message),
		),
	)
}