package basecoat

import "embed"

// FS is an importable symbol to use as a base layer. It provides
// a layout system with partials, components and an area for
// assets. Some of the assets are generated out of process with
// tailwindcss, the step should be done before build.
//
//go:embed all:assets all:components all:partials all:layouts
var FS embed.FS
