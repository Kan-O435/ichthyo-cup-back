package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

// apiBaseURL is defined here as it might not be in signup_page
const apiBaseURL = "https://hack-s-ikuthio-2025.vercel.app"

// LoginPage is a component that displays a login form.
type LoginPage struct {
	vecty.Core
	username string
	password string
	message  string
	OnLogin  func()
}

// onLoginAttempt handles the login attempt by calling the backend API.
func (p *LoginPage) onLoginAttempt(e *vecty.Event) {
	loginURL := apiBaseURL + "/api/auth/login" // Assuming this endpoint exists

	authRequest(loginURL, p.username, p.password,
		func(response string) {
			// Parse JWT response
			var loginResp LoginResponse
			if err := json.Unmarshal([]byte(response), &loginResp); err != nil {
				p.message = "Failed to parse login response: " + err.Error()
				vecty.Rerender(p)
				return
			}

			// Parse JWT token to get user ID
			payload, err := parseJWT(loginResp.Token)
			if err != nil {
				p.message = "Failed to parse JWT token: " + err.Error()
				vecty.Rerender(p)
				return
			}

			// Store token and user data
			storeUserData(loginResp.Token, payload.UserID)

			// Login successful
			p.message = "Login successful!"
			vecty.Rerender(p)

			if p.OnLogin != nil {
				p.OnLogin() // This will trigger navigation to /wplace
			}
		},
		func(err string) {
			p.message = "Login failed: " + err
			vecty.Rerender(p)
		},
	)
}

// Render renders the component.
func (p *LoginPage) Render() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("login-container"),
		),
		elem.Form(
			vecty.Markup(
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
		elem.Button(
			vecty.Text("Don't have an account? Sign Up"),
			vecty.Markup(event.Click(func(e *vecty.Event) {
				js.Global().Get("location").Set("hash", "#/signup")
			})),
		),
	)
}

func (p *LoginPage) renderMessage() vecty.ComponentOrHTML {
	if p.message != "" {
		return elem.Paragraph(vecty.Text(p.message))
	}
	return nil
}
