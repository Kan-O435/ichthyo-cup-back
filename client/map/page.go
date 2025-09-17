package home

import (
    "github.com/hexops/vecty"
    "github.com/hexops/vecty/elem"
)

type HomePage struct {
    vecty.Core
}

func (h *HomePage) Render() vecty.ComponentOrHTML {
    return elem.Div(
        vecty.Text("これはホームページです"),
    )
}