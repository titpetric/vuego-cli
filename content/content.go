package content

import (
	"embed"
	"io/fs"
)

//go:embed all:vuego-tour
var vuegoTour embed.FS

func VuegoTour() fs.FS {
	f, _ := fs.Sub(vuegoTour, "vuego-tour")
	return f
}
