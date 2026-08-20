// Package config defines the vuego-cli configuration file format.
//
// A configuration file declares the virtual hosts a single vuego-cli process
// serves. Each entry binds a DNS host name to a content folder and the mode
// used to render it, which lets one process sit behind a reverse proxy and
// serve several documentation sites from one folder tree.
//
//	vhost:
//	  - domain: docs.example.com
//	    path: ./content/docs
//	    mode: docs
//	  - domain: tour.example.com
//	    path: ./content/vuego-tour
//	    mode: tour
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// DefaultFilenames lists the configuration files looked up by Find, in order.
var DefaultFilenames = []string{"vuego-cli.yml", "vuego-cli.yaml"}

// Mode selects the server implementation used for a virtual host.
type Mode string

// The available modes. Each corresponds to an existing vuego-cli command.
const (
	// ModeDocs renders a markdown documentation tree with the basecoat theme.
	ModeDocs Mode = "docs"

	// ModeTour renders a tour: numbered chapter files split into lessons.
	ModeTour Mode = "tour"

	// ModeServe is the development server: vuego templates, LESS stylesheets
	// and static files served straight from the folder.
	ModeServe Mode = "serve"
)

// Modes returns every valid mode.
func Modes() []Mode {
	return []Mode{ModeDocs, ModeTour, ModeServe}
}

// Valid reports whether the mode is one returned by Modes.
func (m Mode) Valid() bool {
	switch m {
	case ModeDocs, ModeTour, ModeServe:
		return true
	}
	return false
}

// String returns the mode as written in the configuration file.
func (m Mode) String() string {
	return string(m)
}

// VHost binds a domain name to a content folder and a rendering mode.
type VHost struct {
	// Domain is the host name matched against the request. Matching is
	// case insensitive and ignores the port.
	Domain string `yaml:"domain"`

	// Path is the folder served for this domain. It is the full content
	// root: point it at the nested folder if the documents do not live at
	// the top of a checkout. Relative paths resolve against the directory
	// holding the configuration file.
	Path string `yaml:"path"`

	// Mode selects the server: docs, tour or serve.
	Mode Mode `yaml:"mode"`
}

// Config is the vuego-cli configuration document.
type Config struct {
	// Addr is the address the HTTP server listens on. When empty the
	// command line default applies.
	Addr string `yaml:"addr"`

	// TrustForwardedHost routes by the X-Forwarded-Host header when it is
	// present, for proxies that rewrite Host on the way through. Leave it
	// off unless the process is only reachable through such a proxy: any
	// client that can reach the server can set the header.
	TrustForwardedHost bool `yaml:"trust_forwarded_host"`

	// VHosts are the virtual hosts this process serves.
	VHosts []VHost `yaml:"vhost"`
}

// ErrNotFound is returned by Find when no default configuration file exists.
var ErrNotFound = errors.New("no configuration file found")

// Find returns the path of the first default configuration file present in
// dir. It returns ErrNotFound when none of DefaultFilenames exists.
func Find(dir string) (string, error) {
	name, err := FindFS(os.DirFS(dir))
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// FindFS returns the name of the first default configuration file present in
// fsys. It returns ErrNotFound when none of DefaultFilenames exists.
func FindFS(fsys fs.FS) (string, error) {
	for _, name := range DefaultFilenames {
		info, err := fs.Stat(fsys, name)
		if err != nil || info.IsDir() {
			continue
		}
		return name, nil
	}
	return "", ErrNotFound
}

// Load reads, decodes and validates the configuration file at path. Relative
// vhost paths are resolved against the directory holding the file, so a
// configuration is independent of the process working directory.
func Load(path string) (*Config, error) {
	dir := filepath.Dir(path)

	base, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving config directory: %w", err)
	}

	cfg, err := LoadFS(os.DirFS(dir), filepath.Base(path), base)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// LoadFS reads, decodes and validates the configuration file named name in
// fsys. Relative vhost paths are resolved against base, which is the
// directory the configuration is considered to live in.
func LoadFS(fsys fs.FS, name string, base string) (*Config, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg, err := Decode(data)
	if err != nil {
		return nil, err
	}
	cfg.Resolve(base)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Decode parses a configuration document without resolving paths or
// validating it. Load is the usual entry point; Decode exists for callers
// that supply the document from somewhere other than the filesystem.
func Decode(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Resolve rewrites relative vhost paths so they are anchored at base, and
// normalizes domain names for matching. It is idempotent.
func (c *Config) Resolve(base string) {
	for i := range c.VHosts {
		vhost := &c.VHosts[i]
		vhost.Domain = NormalizeHost(vhost.Domain)
		if vhost.Path != "" && !filepath.IsAbs(vhost.Path) {
			vhost.Path = filepath.Join(base, vhost.Path)
		}
	}
}

// Validate reports the first problem found in the configuration document. It
// checks the document only and does not touch the filesystem; whether a
// vhost's path exists is settled when its content filesystem is opened.
//
// A configuration with no vhost entries is an error: an empty file is a
// mistake, and the caller falls back to serving the working directory only
// when no file exists at all.
func (c *Config) Validate() error {
	if len(c.VHosts) == 0 {
		return errors.New("no vhost entries configured")
	}

	seen := make(map[string]int, len(c.VHosts))
	for i, vhost := range c.VHosts {
		if vhost.Domain == "" {
			return fmt.Errorf("vhost[%d]: domain is required", i)
		}
		if first, ok := seen[vhost.Domain]; ok {
			return fmt.Errorf("vhost[%d]: domain %q already declared by vhost[%d]", i, vhost.Domain, first)
		}
		seen[vhost.Domain] = i

		if vhost.Path == "" {
			return fmt.Errorf("vhost[%d] (%s): path is required", i, vhost.Domain)
		}

		if !vhost.Mode.Valid() {
			return fmt.Errorf("vhost[%d] (%s): unknown mode %q, want one of %s", i, vhost.Domain, vhost.Mode, modeList())
		}
	}
	return nil
}

// NormalizeHost reduces a host name to the form used for matching: lower
// case, without a port, and without the trailing dot of a fully qualified
// name. IPv6 literals keep their brackets.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.ToLower(host)

	// Strip the port. An IPv6 literal is bracketed, so only a colon after
	// the closing bracket is a port separator.
	if idx := strings.LastIndex(host, "]"); idx >= 0 {
		if colon := strings.Index(host[idx:], ":"); colon >= 0 {
			host = host[:idx+colon]
		}
	} else if colon := strings.LastIndex(host, ":"); colon >= 0 {
		host = host[:colon]
	}

	return strings.TrimSuffix(host, ".")
}

// modeList renders the valid modes for an error message.
func modeList() string {
	modes := Modes()
	names := make([]string, len(modes))
	for i, mode := range modes {
		names[i] = mode.String()
	}
	return strings.Join(names, ", ")
}
