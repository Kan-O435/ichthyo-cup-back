package components

import (
    "syscall/js"

    "github.com/hexops/vecty"
    "github.com/hexops/vecty/elem"
    "github.com/hexops/vecty/event"
)

type Signup struct {
    vecty.Core
    username string
    password string
    message  string
}

// サインアップ処理
func (s *Signup) Render() vecty.ComponentOrHTML {
    return elem.Div(
        elem.Heading1(vecty.Text("Sign Up")),
        elem.Form(
            vecty.Markup(event.Submit(func(e *vecty.Event) {
                // まず /api/signup を叩く
                apiURL := "https://hack-s-ikuthio-2025.vercel.app/api/auth/signup"
                authRequest(apiURL, s.username, s.password,
                    func(_ string) {
                        // サインアップ成功したら続けてログイン
                        authRequest("https://hack-s-ikuthio-2025.vercel.app/api/auth/callback/credentials", s.username, s.password,
                            func(_ string) {
                                s.message = "Signup + Login successful"
                                js.Global().Get("window").Get("location").Set("href", "/home")
                                vecty.Rerender(s)
                            },
                            func(err string) {
                                s.message = "Signup OK, but login failed: " + err
                                vecty.Rerender(s)
                            },
                        )
                    },
                    func(err string) {
                        s.message = "Signup failed: " + err
                        vecty.Rerender(s)
                    },
                )
            })),
            elem.Div(
                elem.Label(vecty.Text("username : ")),
                elem.Input(
                    vecty.Markup(
                        vecty.Property("type", "text"),
                        event.Input(func(e *vecty.Event) {
                            s.username = e.Target.Get("value").String()
                        }),
                    ),
                ),
            ),
            elem.Div(
                elem.Label(vecty.Text("password : ")),
                elem.Input(
                    vecty.Markup(
                        vecty.Property("type", "password"),
                        event.Input(func(e *vecty.Event) {
                            s.password = e.Target.Get("value").String()
                        }),
                    ),
                ),
            ),
            elem.Button(
                vecty.Text("Sign Up"),
                vecty.Markup(vecty.Property("type", "submit")),
            ),
        ),
        elem.Button(
            vecty.Text("ログインページへ戻る"),
            vecty.Markup(
                vecty.Style("margin-top", "10px"),
                vecty.Style("background-color", "#6c757d"),
                vecty.Style("color", "white"),
                vecty.Style("border", "none"),
                vecty.Style("padding", "10px 20px"),
                vecty.Style("border-radius", "4px"),
                vecty.Style("cursor", "pointer"),
                event.Click(func(e *vecty.Event) {
                    js.Global().Get("location").Set("hash", "#/login")
                }),
            ),
        ),
        elem.Paragraph(vecty.Text(s.message)),
    )
}
