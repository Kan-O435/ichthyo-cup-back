package components

import (

	"syscall/js"

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
			elem.Heading1(vecty.Text("Log In")),
			elem.Form(
				vecty.Markup(event.Submit(func(e *vecty.Event) {
				authRequest("/api/auth/signup", l.username, l.password,
					func(_ string) {
						l.message = "Login successful!!"
						// cookie will be stored on NextAuth
						js.Global().Get("window").Get("location").Set("href","/home")
						vecty.Rerender(l) // it is like re-directin'
					},
					/* error handling */
					func(err string) {
						l.message = err
						vecty.Rerender(l)
					},
				)
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
					vecty.Text("Log in"),
					vecty.Markup(vecty.Property("type", "submit")),
				),
			),
		),
		
		elem.Paragraph(
			vecty.Text(l.message),
		),
	)
}
