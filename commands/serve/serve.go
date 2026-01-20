package serve

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/lessgo"
	"github.com/titpetric/platform"
	"github.com/titpetric/vuego"

	"github.com/titpetric/vuego-cli/server"
)

// Name is the command title.
const Name = "Start development server for templates and assets"

// New creates a new serve command.
func New() *cli.Command {
	var addr string

	return &cli.Command{
		Name:  "serve",
		Title: Name,
		Bind: func(fs *flag.FlagSet) {
			flag.StringVar(&addr, "addr", ":8080", "HTTP server address")
		},
		Run: func(ctx context.Context, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return Serve(ctx, dir, addr)
		},
	}
}

// Module represents the serve module for the platform.
type Module struct {
	platform.UnimplementedModule

	dir    string
	absDir string
	dirFS  fs.FS
}

// NewModule creates a new serve module for the given directory.
func NewModule(dir string) (*Module, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid directory: %w", err)
	}

	if _, err := os.Stat(absDir); err != nil {
		return nil, fmt.Errorf("directory not accessible: %w", err)
	}

	return &Module{
		dir:    dir,
		absDir: absDir,
		dirFS:  os.DirFS(absDir),
	}, nil
}

// Name returns the module name.
func (m *Module) Name() string {
	return "vuego-serve"
}

// Mount registers the serve routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
	fileServer := http.FileServer(http.FS(m.dirFS))

	r.Use(lessgo.NewMiddleware(m.dirFS, "/"))

	r.Use(func(next http.Handler) http.Handler {
		return &vuegoMiddleware{
			vuegoHandler: server.Middleware(m.dirFS, server.WithLoadOption(vuego.WithLessProcessor(), vuego.WithComponents())),
			next:         next,
		}
	})

	r.Handle("/*", fileServer)

	return nil
}

// Serve starts an HTTP server that serves templates and assets from the given directory.
// It uses os.DirFS to create a filesystem rooted at the specified directory.
// The server provides:
// - .vuego file rendering via server middleware
// - .less file compilation via lessgo middleware
// - Directory listing and file serving for all other files.
func Serve(ctx context.Context, dir string, addr string) error {
	module, err := NewModule(dir)
	if err != nil {
		return err
	}

	opts := platform.NewOptions()
	opts.ServerAddr = addr

	p := platform.New(opts)
	p.Register(module)

	if err := p.Start(context.Background()); err != nil {
		return err
	}

	p.Wait()
	return nil
}

// vuegoMiddleware wraps the vuego handler and falls through to the next handler
// if the vuego handler doesn't handle the request (404).
type vuegoMiddleware struct {
	vuegoHandler http.Handler
	next         http.Handler
}

func (m *vuegoMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Buffer the response to check if it's a 404
	buf := &bytes.Buffer{}
	wrapped := &bufferingResponseWriter{
		buf:     buf,
		headers: make(http.Header),
	}
	m.vuegoHandler.ServeHTTP(wrapped, r)

	// If it was a 404, pass to next handler instead
	if wrapped.statusCode == http.StatusNotFound {
		m.next.ServeHTTP(w, r)
		return
	}

	// Write buffered response to actual response writer
	for k, v := range wrapped.headers {
		w.Header()[k] = v
	}
	w.WriteHeader(wrapped.statusCode)
	_, _ = w.Write(buf.Bytes())
}

// bufferingResponseWriter buffers response to check if it's a 404
type bufferingResponseWriter struct {
	buf        *bytes.Buffer
	headers    http.Header
	statusCode int
}

func (w *bufferingResponseWriter) Header() http.Header {
	return w.headers
}

func (w *bufferingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *bufferingResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.buf.Write(b)
}
