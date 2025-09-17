package main

import (
	"github.com/hexops/vecty"
)

func main() {
	vecty.SetTitle("Ichthyo Cup")
	vecty.RenderBody(NewApp())
	select {}
}
