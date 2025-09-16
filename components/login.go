package components

import (
	"github.com/hexops/vecty"
  "github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/elem"
)

type Login struct {
	vecty.Core
	username string
	password string
	message string
}

func (l *Login) Render() vecty.ComponentOrHTML {
	return elem.Div (
		elem.Div(
			elem.Heading1(vecty.Text("Login")),

			elem.Form(
				vecty.Markup(event.Submit(func(e *vecty.event) {
					e.PreventDefault()
					if l.username == "admin" && l.password == "admin1234" {
						l.message = "Login successfull"
					} else {
						l.message = "invalid request"
					}
					vecty.Render(1)
				})),

				elem.Div(
					elem.Label(
						vecty.Text("username : "),
					),
					elem.Input(
						vecty.Markup(
							vecty.Property("type", "password"),
							even.Input(func(e *vecty.event) {
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
		)
		
		elem.Paragraph(
			vecty.Text(l.message),
		),
	)
}