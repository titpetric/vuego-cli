package tour

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"github.com/titpetric/platform"
	"github.com/titpetric/vuego"

	"github.com/titpetric/vuego-cli/internal/codeblock"
	"github.com/titpetric/vuego-cli/internal/model"
	"github.com/titpetric/vuego-cli/internal/sqlite"
	"github.com/titpetric/vuego-cli/server"
)

// DefaultConfig enables the evaluators the tour relies on. Shell execution is
// intentionally left disabled.
var DefaultConfig = codeblock.Config{
	EnablePHP:    true,
	EnableVuego:  true,
	EnableSQLite: true,
}

//go:embed all:app
var embeddedApp embed.FS

// Module represents the tour module for the platform.
type Module struct {
	platform.UnimplementedModule

	tour       *Tour
	FS         fs.FS
	readmeHTML string
	doneHTML   string
	eval       *codeblock.Service
}

// NewModule creates a new tour module using embedded content.
func NewModule(contentFS fs.FS) *Module {
	appFS, _ := fs.Sub(embeddedApp, "app")
	return &Module{
		FS: vuego.NewOverlayFS(contentFS, appFS),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "vuego-tour"
}

func (m *Module) indexTmpl() (string, error) {
	tmplData, err := fs.ReadFile(m.FS, "templates/index.vuego")
	if err != nil {
		return "", err
	}
	return string(tmplData), nil
}

// Mount registers the tour routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
	tour, err := ParseTour(m.FS)
	if err != nil {
		return err
	}

	m.tour = tour
	m.eval = codeblock.New(DefaultConfig, m.FS, vuego.WithLessProcessor())

	// Load and render README
	readmeData, _ := fs.ReadFile(m.FS, "README.md")
	if len(readmeData) > 0 {
		m.readmeHTML = string(readmeData)
	}

	// Load DONE.md
	doneData, _ := fs.ReadFile(m.FS, "DONE.md")
	if len(doneData) > 0 {
		m.doneHTML = string(doneData)
	}

	r.Get("/", m.serveIndex)
	r.Get("/done", m.serveDone)
	r.Get("/lesson/{chapter}/{lesson}", m.serveLessonPage)
	r.Get("/static/tour.js", m.serveJS)
	r.Get("/static/tour.css", m.serveCSS)
	r.Post("/render", m.handleRender)
	r.Post(codeblock.EvalPath, m.eval.Handler())

	return nil
}

func (m *Module) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	m.renderTourPage(w, nil, m.readmeHTML, false)
}

func (m *Module) serveDone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	m.renderTourPage(w, nil, m.doneHTML, true)
}

func (m *Module) serveLessonPage(w http.ResponseWriter, r *http.Request) {
	chapterName := chi.URLParam(r, "chapter")
	lessonIdx := chi.URLParam(r, "lesson")

	lesson := m.tour.GetLessonByName(chapterName, lessonIdx)

	// Check Accept header - return JSON for API requests
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		if lesson == nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "lesson not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(lesson)
		return
	}

	// Return HTML page
	if lesson == nil {
		lesson = m.tour.FirstLesson()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	m.renderTourPage(w, lesson, "", false)
}

// handleRender preserves the tour's legacy {template,data,files} request shape
// while delegating the actual evaluation to the shared codeblock service.
func (m *Module) handleRender(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req server.RenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(server.RenderResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	resp := m.eval.Eval(r.Context(), toEvalRequest(req))
	_ = json.NewEncoder(w).Encode(server.RenderResponse{HTML: resp.Content, Error: resp.Error})
}

// toEvalRequest maps a legacy RenderRequest to an EvalRequest, picking the
// evaluator from the files present. The .vuego template and its data are folded
// into the file set under index.vuego / index.yaml.
func toEvalRequest(req server.RenderRequest) model.EvalRequest {
	files := make(map[string]string, len(req.Files)+2)
	for name, content := range req.Files {
		files[name] = content
	}

	switch {
	case sqlite.HasSQL(files):
		return model.EvalRequest{Language: codeblock.LangSQL, Files: files}
	case req.Template == "":
		return model.EvalRequest{Language: codeblock.LangPHP, Files: files}
	default:
		files["index.vuego"] = req.Template
		if req.Data != "" {
			files["index.yaml"] = req.Data
		}
		return model.EvalRequest{Language: codeblock.LangVuego, Entry: "index.vuego", Files: files}
	}
}

func (m *Module) serveJS(w http.ResponseWriter, _ *http.Request) {
	data, _ := fs.ReadFile(m.FS, "public/tour.js")
	w.Header().Set("Content-Type", "application/javascript")
	_, _ = w.Write(data)
}

func (m *Module) serveCSS(w http.ResponseWriter, _ *http.Request) {
	data, _ := fs.ReadFile(m.FS, "public/tour.css")
	w.Header().Set("Content-Type", "text/css")
	_, _ = w.Write(data)
}

func (m *Module) renderTourPage(w http.ResponseWriter, lesson *Lesson, markdown string, isDone bool) {
	data := map[string]any{
		"lesson":   lesson,
		"chapters": m.tour.Chapters,
		"total":    m.tour.LessonCount(),
		"isDone":   isDone,
	}
	if markdown != "" {
		data["readme"] = markdown
	}

	tmpl := vuego.NewFS(m.FS, vuego.WithLessProcessor())
	var buf bytes.Buffer
	if err := tmpl.Load("templates/index.vuego").Fill(data).Render(context.Background(), &buf); err != nil {
		log.Printf("error rendering tour page: %v", err)
	}
	_, _ = buf.WriteTo(w)
}
