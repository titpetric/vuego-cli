package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"testing/fstest"

	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"
)

// RenderRequest contains template and data for rendering.
type RenderRequest struct {
	Template string            `json:"template"`
	Data     string            `json:"data"`
	Files    map[string]string `json:"files,omitempty"`
}

// RenderResponse contains the rendered HTML or an error.
type RenderResponse struct {
	HTML  string `json:"html,omitempty"`
	Error string `json:"error,omitempty"`
}

// RenderHandler returns an http.HandlerFunc that renders templates via POST /render.
// The optional baseFS provides additional files (like components) available during rendering.
func RenderHandler(baseFS fs.FS, opts ...vuego.LoadOption) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			_ = json.NewEncoder(w).Encode(RenderResponse{
				Error: "method not allowed",
			})
			return
		}

		var req RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = json.NewEncoder(w).Encode(RenderResponse{
				Error: "invalid JSON: " + err.Error(),
			})
			return
		}

		html, err := Render(r.Context(), baseFS, req, opts...)
		if err != nil {
			_ = json.NewEncoder(w).Encode(RenderResponse{
				Error: err.Error(),
			})
			return
		}

		_ = json.NewEncoder(w).Encode(RenderResponse{
			HTML: html,
		})
	}
}

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
	templateFS := buildTemplateFS(baseFS, req.Template, req.Files)

	// Wrap template with injected styles
	templateWithStyles := injectStyleLinksIntoTemplate(req.Template, req.Files)

	// Re-build filesystem with updated template
	templateFS = buildTemplateFS(baseFS, templateWithStyles, req.Files)

	// Render the template
	renderer := vuego.NewFS(templateFS, opts...)
	tpl := renderer.Load("template.html").Fill(data)

	var buf bytes.Buffer

	// Check if template has a layout specified
	if layout := tpl.Get("layout"); layout != "" {
		if err := tpl.Layout(ctx, &buf); err != nil {
			return "", err
		}
	} else {
		if err := tpl.Render(ctx, &buf); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// buildTemplateFS creates a filesystem combining the base FS with request files.
func buildTemplateFS(baseFS fs.FS, template string, files map[string]string) fs.FS {
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

	return &combinedFS{
		primary:   primary,
		secondary: baseFS,
	}
}

// combinedFS implements fs.FS by combining a primary and secondary filesystem.
type combinedFS struct {
	primary   fs.FS
	secondary fs.FS
}

// Open implements fs.FS by trying the primary filesystem first, then falling back to secondary.
func (cfs *combinedFS) Open(name string) (fs.File, error) {
	f, err := cfs.primary.Open(name)
	if err == nil {
		return f, nil
	}
	return cfs.secondary.Open(name)
}

// ReadDir implements fs.ReadDirFS for component discovery.
func (cfs *combinedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(cfs.secondary, name)
}

// injectStyleLinksIntoTemplate injects style tags directly into the template string
func injectStyleLinksIntoTemplate(template string, files map[string]string) string {
	// For API requests, look for style files in the files map
	var styleFiles []string
	for name := range files {
		if strings.HasSuffix(name, ".less") || strings.HasSuffix(name, ".css") {
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
			// For LESS files, use type="text/css+less" so the LessProcessor compiles it
			if strings.HasSuffix(name, ".less") {
				styleTags.WriteString("\n<style type=\"text/css+less\">")
				styleTags.WriteString(content)
				styleTags.WriteString("</style>")
			} else {
				// For plain CSS files, use regular style tag
				styleTags.WriteString("\n<style>")
				styleTags.WriteString(content)
				styleTags.WriteString("</style>")
			}
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

// injectStyleLinksForRequest injects style links for adjacent .less/.css files in the request files.
func injectStyleLinksForRequest(buf *bytes.Buffer, files map[string]string) (string, error) {
	// For API requests, look for style files in the files map
	var styleFiles []string
	for name := range files {
		if strings.HasSuffix(name, ".less") || strings.HasSuffix(name, ".css") {
			styleFiles = append(styleFiles, name)
		}
	}

	if len(styleFiles) == 0 {
		return buf.String(), nil
	}

	// Inject style files as <style> tags or <link> tags
	htmlContent := buf.String()
	var styleTags strings.Builder

	for _, name := range styleFiles {
		if content, ok := files[name]; ok {
			// For LESS files, use type="text/css+less" so the LessProcessor compiles it
			if strings.HasSuffix(name, ".less") {
				styleTags.WriteString("\n<style type=\"text/css+less\">")
				styleTags.WriteString(content)
				styleTags.WriteString("</style>")
			} else {
				// For plain CSS files, use regular style tag
				styleTags.WriteString("\n<style>")
				styleTags.WriteString(content)
				styleTags.WriteString("</style>")
			}
		}
	}

	// Inject before </head> or </body>
	if strings.Contains(htmlContent, "</head>") {
		htmlContent = strings.Replace(htmlContent, "</head>", styleTags.String()+"\n</head>", 1)
	} else if strings.Contains(htmlContent, "</body>") {
		htmlContent = strings.Replace(htmlContent, "</body>", styleTags.String()+"\n</body>", 1)
	} else {
		htmlContent = htmlContent + styleTags.String()
	}

	return htmlContent, nil
}
