package content

import (
	"embed"
	"io/fs"
)

//go:embed all:vuego-tour
var vuegoTour embed.FS

// VuegoTour is the tour content as a filesystem rooted at the tour itself,
// so a caller reads "01-intro/lesson.md" rather than carrying the embed
// directory in every path.
func VuegoTour() fs.FS {
	f, _ := fs.Sub(vuegoTour, "vuego-tour")
	return f
}
