package host

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"github.com/titpetric/platform"

	"github.com/titpetric/vuego-cli/commands/docs"
	"github.com/titpetric/vuego-cli/commands/serve"
	"github.com/titpetric/vuego-cli/config"
	"github.com/titpetric/vuego-cli/tour"
)

// site is one configured virtual host: the module rendering its content and
// the router that module mounted itself on.
type site struct {
	vhost  config.VHost
	module platform.Module
	router chi.Router
}

// Resolver opens the filesystem holding a virtual host's content. DirFS is
// the default; tests supply an in-memory filesystem instead.
type Resolver func(path string) (fs.FS, error)

// DirFS opens a vhost path as a directory on disk, reporting a path that is
// missing or is not a directory.
func DirFS(path string) (fs.FS, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid directory: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("directory not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absPath)
	}

	return os.DirFS(absPath), nil
}

// Option configures a Module.
type Option func(*Module)

// WithResolver replaces the filesystem resolver used to open vhost content.
func WithResolver(resolver Resolver) Option {
	return func(m *Module) {
		m.resolver = resolver
	}
}

// Module routes requests to a per-domain module. Every virtual host gets its
// own router, so the modules keep mounting at their absolute paths and stay
// unaware of each other.
type Module struct {
	platform.UnimplementedModule

	sites map[string]*site
	order []string

	// resolver opens each vhost's content filesystem.
	resolver Resolver

	// trustForwardedHost makes the module prefer the X-Forwarded-Host
	// header over the request Host.
	trustForwardedHost bool
}

// NewModule builds a virtual hosting module from a configuration. The
// configuration is expected to be valid, as config.Load returns it. Each
// vhost's content filesystem is opened here, so a path that does not exist
// is reported before the server starts.
func NewModule(cfg *config.Config, opts ...Option) (*Module, error) {
	m := &Module{
		sites:              make(map[string]*site, len(cfg.VHosts)),
		order:              make([]string, 0, len(cfg.VHosts)),
		resolver:           DirFS,
		trustForwardedHost: cfg.TrustForwardedHost,
	}
	for _, opt := range opts {
		opt(m)
	}

	for _, vhost := range cfg.VHosts {
		contentFS, err := m.resolver(vhost.Path)
		if err != nil {
			return nil, fmt.Errorf("vhost %s: %w", vhost.Domain, err)
		}

		module, err := newSiteModule(vhost.Mode, contentFS)
		if err != nil {
			return nil, fmt.Errorf("vhost %s: %w", vhost.Domain, err)
		}

		m.sites[vhost.Domain] = &site{vhost: vhost, module: module}
		m.order = append(m.order, vhost.Domain)
	}

	return m, nil
}

// newSiteModule creates the module implementing a mode over a content
// filesystem.
func newSiteModule(mode config.Mode, contentFS fs.FS) (platform.Module, error) {
	switch mode {
	case config.ModeServe:
		return serve.NewModuleFS(contentFS), nil
	case config.ModeDocs:
		return docs.NewModule(contentFS), nil
	case config.ModeTour:
		return tour.NewModule(contentFS), nil
	}
	return nil, fmt.Errorf("unknown mode %q", mode)
}

// Name returns the module name.
func (m *Module) Name() string {
	return "vuego-host"
}

// Mount gives every virtual host its own router and installs the dispatcher
// as a catch all on the shared router. A vhost that fails to mount aborts
// startup: a site silently missing from a running server is worse than a
// server that refuses to start.
func (m *Module) Mount(ctx context.Context, r platform.Router) error {
	for _, domain := range m.order {
		s := m.sites[domain]

		router := chi.NewRouter()
		if err := s.module.Mount(ctx, router); err != nil {
			return fmt.Errorf("vhost %s: mounting %s: %w", domain, s.vhost.Mode, err)
		}
		s.router = router

		log.Printf("vhost %s: serving %s from %s", domain, s.vhost.Mode, s.vhost.Path)
	}

	r.Handle("/*", http.HandlerFunc(m.dispatch))
	return nil
}

// Start starts every virtual host's module.
func (m *Module) Start(ctx context.Context) error {
	for _, domain := range m.order {
		if err := m.sites[domain].module.Start(ctx); err != nil {
			return fmt.Errorf("vhost %s: %w", domain, err)
		}
	}
	return nil
}

// Stop stops every virtual host's module in reverse order. All modules are
// given a chance to stop; the first error is returned.
func (m *Module) Stop(ctx context.Context) error {
	var firstErr error
	for i := len(m.order) - 1; i >= 0; i-- {
		domain := m.order[i]
		if err := m.sites[domain].module.Stop(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("vhost %s: %w", domain, err)
		}
	}
	return firstErr
}

// dispatch routes a request to the router of the virtual host matching its
// host name.
func (m *Module) dispatch(w http.ResponseWriter, r *http.Request) {
	host := config.NormalizeHost(m.requestHost(r))

	s, ok := m.sites[host]
	if !ok {
		http.Error(w, fmt.Sprintf("no site configured for host %q", host), http.StatusNotFound)
		return
	}

	s.router.ServeHTTP(w, detachRouteContext(r))
}

// requestHost returns the host name a request should be routed by. Behind a
// proxy that rewrites Host, X-Forwarded-Host carries the name the client
// asked for; it is only consulted when the configuration opts in, since any
// client can set it.
func (m *Module) requestHost(r *http.Request) string {
	if m.trustForwardedHost {
		if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
			return firstHost(forwarded)
		}
	}
	return r.Host
}

// firstHost returns the first entry of a comma separated header value. A
// chain of proxies appends to X-Forwarded-Host, and the first entry is the
// name the original client used.
func firstHost(value string) string {
	first, _, _ := strings.Cut(value, ",")
	return first
}

// detachRouteContext removes the routing context of the outer router from
// the request, so the vhost's router allocates its own.
//
// A chi router reuses an existing context rather than allocating one, and its
// route lookup appends to that context instead of resetting it. Sharing one
// context across both routers therefore leaves the dispatcher's own "/*"
// parameter and route pattern in place: a vhost route that declares no
// wildcard still reads one back, and the recorded route pattern is the
// dispatcher's concatenated with the vhost's. Detaching keeps each vhost
// routing exactly as it would if it were the only router in the process.
func detachRouteContext(r *http.Request) *http.Request {
	if r.Context().Value(chi.RouteCtxKey) == nil {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, nil))
}
