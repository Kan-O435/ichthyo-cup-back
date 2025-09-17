package main

import (
	"encoding/json"
	"syscall/js"

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

// onLoginAttempt handles the login attempt by calling the backend API.
func (p *LoginPage) onLoginAttempt(e *vecty.Event) {
	// APIに送信するデータを作成
	requestBody := map[string]string{
		"username":    p.username, 
		"password": p.password,
	}
	// GoのmapをJSONに変換
	body, err := json.Marshal(requestBody)
	if err != nil {
		p.message = "Error creating request"
		vecty.Rerender(p)
		return
	}
	apiURL := "https://hack-s-ikuthio-2025.vercel.app/api/auth/login"
	// JavaScriptのfetch APIを呼び出す準備
	promise := js.Global().Get("fetch").Invoke(apiURL, map[string]interface{}{
		"method": "POST",
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
		"body": string(body),
	})

	// 成功時の処理
	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		// レスポンスが成功でなければエラーメッセージを表示
		if !response.Get("ok").Bool() {
			p.message = "Invalid credentials"
			vecty.Rerender(p)
			return nil
		}

		// レスポンスボディをJSONとして取得するPromise
		jsonPromise := response.Call("json")
		jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// JSONからトークンを取得
			token := args[0].Get("token").String()
			// ここでトークンをlocalStorageなどに保存することができる
			js.Global().Get("localStorage").Call("setItem", "jwt_token", token)

			p.message = "Login successful!"
			// OnLoginコールバックを呼び出して画面遷移
			if p.OnLogin != nil {
				p.OnLogin()
			}
			vecty.Rerender(p)
			return nil
		}))

		return nil
	})

	// 失敗時の処理
	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		p.message = "Network error"
		vecty.Rerender(p)
		return nil
	})

	// Promiseの実行
	promise.Call("then", success).Call("catch", failure)
}


// Render renders the component.
// (Render関数は変更なしのため省略)
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
				elem.Label(vecty.Text("Username:")), // ラベルを分かりやすく変更
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
			elem.Button(
				vecty.Text("Mapへ移動（スキップ）"),
				vecty.Markup(
					vecty.Style("margin-top", "10px"),
					vecty.Style("background-color", "#007bff"),
					vecty.Style("color", "white"),
					vecty.Style("border", "none"),
					vecty.Style("padding", "10px 20px"),
					vecty.Style("border-radius", "4px"),
					vecty.Style("cursor", "pointer"),
					vecty.Style("width", "100%"),
					event.Click(func(e *vecty.Event) {
						if p.OnLogin != nil {
							p.OnLogin()
						}
					}),
				),
			),
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