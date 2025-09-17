package main

import (
	"fmt"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// apiBaseURL is defined in login_page.go

type SignupPage struct {
	vecty.Core
	username string
	password string
	message  string
}

func (s *SignupPage) Render() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("signup-container"), // Add a class for styling
		),
		elem.Form(
			vecty.Markup(
				event.Submit(s.handleSignup).PreventDefault(),
			),
			elem.Heading1(vecty.Text("Sign Up")),
			elem.Div(
				elem.Label(vecty.Text("Username:")),
				elem.Input(vecty.Markup(
					vecty.Property("type", "text"),
					event.Input(func(e *vecty.Event) {
						s.username = e.Target.Get("value").String()
					}),
				)),
			),
			elem.Div(
				elem.Label(vecty.Text("Password:")),
				elem.Input(vecty.Markup(
					vecty.Property("type", "password"),
					event.Input(func(e *vecty.Event) {
						s.password = e.Target.Get("value").String()
					}),
				)),
			),
			elem.Button(vecty.Text("Sign Up"), vecty.Markup(vecty.Property("type", "submit"))),
			s.renderMessage(),
		),
		elem.Button(
			vecty.Text("Back to Login"),
			vecty.Markup(event.Click(func(e *vecty.Event) {
				js.Global().Get("location").Set("hash", "#/login")
			})),
		),
	)
}

func (s *SignupPage) handleSignup(e *vecty.Event) {
	signupURL := apiBaseURL + "/api/auth/signup"

	authRequest(signupURL, s.username, s.password,
		func(response string) {
			// Signup successful, now try to login
			s.message = "Signup successful! Please log in."
			vecty.Rerender(s)

			// NOTE: NextAuth credentials login is complex from an external client.
			// For now, we just redirect to the login page to have the user log in manually.
			js.Global().Get("location").Set("hash", "#/login")
		},
		func(err string) {
			s.message = fmt.Sprintf("Signup failed: %s", err)
			vecty.Rerender(s)
		},
	)
}

func (s *SignupPage) renderMessage() vecty.ComponentOrHTML {
	if s.message != "" {
		return elem.Paragraph(vecty.Text(s.message))
	}
	return nil
}