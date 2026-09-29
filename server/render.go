package server

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing/fstest"

	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"
)

// Render processes a RenderRequest and returns rendered HTML.
// This function can be used by both HTTP handlers and CLI commands.
// If the template specifies a layout in frontmatter, Layout() is used instead of Render().
// This function automatically injects style links for adjacent .less/.css files.
func Render(ctx context.Context, baseFS fs.FS, req RenderRequest, opts ...vuego.LoadOption) (string, error) {
	// Parse data (supports both JSON and YAML)
	var data map[string]any
	if req.Data != "" {
		if err := yaml.Unmarshal([]byte(req.Data), &data); err != nil {
			return "", err
		}
	}
	if data == nil {
		data = make(map[string]any)
	}

	// Build filesystem with template and any additional files
	templateFS := buildTemplateFS(baseFS, req.Files, req.Template)

	// Wrap template with injected styles
	templateWithStyles := injectStyleLinksIntoTemplate(req.Files, req.Template)

	// Re-build filesystem with updated template
	templateFS = buildTemplateFS(baseFS, req.Files, templateWithStyles)

	// Render the template
	renderer := vuego.NewFS(templateFS, opts...)
	tpl := renderer.Load("template.html").Fill(data)

	var buf bytes.Buffer

	if err := tpl.Render(ctx, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// buildTemplateFS creates a filesystem combining the base FS with request files.
func buildTemplateFS(baseFS fs.FS, files map[string]string, template string) fs.FS {
	primary := fstest.MapFS{
		"template.html": &fstest.MapFile{Data: []byte(template)},
	}

	// Add any additional files from the request
	for name, content := range files {
		primary[name] = &fstest.MapFile{Data: []byte(content)}
	}

	if baseFS == nil {
		return primary
	}

	return vuego.NewOverlayFS(primary, baseFS)
}

// injectStyleLinksIntoTemplate injects style tags directly into the template string.
// Assumes all files have already been processed (LESS compiled to CSS).
func injectStyleLinksIntoTemplate(files map[string]string, template string) string {
	// For API requests, look for style files in the files map
	var styleFiles []string
	for name := range files {
		if strings.HasSuffix(name, ".css") {
			styleFiles = append(styleFiles, name)
		}
	}

	if len(styleFiles) == 0 {
		return template
	}

	// Build style tags
	var styleTags strings.Builder
	for _, name := range styleFiles {
		if content, ok := files[name]; ok {
			// All files at this point are CSS (LESS has been compiled)
			styleTags.WriteString("\n<style>")
			styleTags.WriteString(content)
			styleTags.WriteString("</style>")
		}
	}

	// Inject before </head> or </body>
	if strings.Contains(template, "</head>") {
		return strings.Replace(template, "</head>", styleTags.String()+"\n</head>", 1)
	} else if strings.Contains(template, "</body>") {
		return strings.Replace(template, "</body>", styleTags.String()+"\n</body>", 1)
	}

	return template + styleTags.String()
}
