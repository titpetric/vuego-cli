package host

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/titpetric/cli"
	"github.com/titpetric/platform"

	"github.com/titpetric/vuego-cli/commands/serve"
	"github.com/titpetric/vuego-cli/config"
)

// Name is the command title.
const Name = "Serve multiple sites by domain from a config file"

// Usage describes the configuration file the command reads.
const usage = `Serves several documentation sites from one process, choosing
the site by the request host name. Intended to run behind a reverse proxy
that passes the client's Host header through.

The configuration lists the virtual hosts:

  vhost:
    - domain: docs.example.com docs.localhost
      path: ./content/docs
      mode: docs
    - domain: tour.example.com
      path: ./content/vuego-tour
      mode: tour

A domain field holding several space separated names serves the same content
on each of them, which is how one site answers on a public and a local name.

A path is the full content root, so point it at a nested folder when the
documents do not sit at the top of a checkout. Relative paths resolve
against the directory holding the configuration file. Modes are docs, tour
and serve.

With no --config flag the current directory is searched for vuego-cli.yml
and vuego-cli.yaml. When neither exists the current directory is served as
'vuego-cli serve .' would.`

// New creates a new host command.
func New() *cli.Command {
	var addr string
	var configPath string

	return &cli.Command{
		Name:  "host",
		Title: Name,
		Usage: func() string { return usage },
		Bind: func(fs *cli.FlagSet) {
			fs.StringVar(&addr, "addr", "", "HTTP server address (default: the config's addr, else :8080)")
			fs.StringVar(&configPath, "config", "", "Path to the configuration file (default: vuego-cli.yml in the current directory)")
		},
		Run: func(ctx context.Context, args []string) error {
			return Serve(ctx, addr, configPath)
		},
	}
}

// DefaultAddr is the address used when neither the command line nor the
// configuration file names one.
const DefaultAddr = ":8080"

// Serve starts the virtual hosting server. An empty configPath looks for a
// default configuration file in the current directory and falls back to
// serving that directory when none is found. An empty addr defers to the
// configuration file, then to DefaultAddr.
func Serve(ctx context.Context, addr string, configPath string) error {
	if configPath == "" {
		found, err := config.Find(".")
		if err != nil {
			if !errors.Is(err, config.ErrNotFound) {
				return err
			}
			log.Printf("no configuration file found, serving the current directory")
			return serve.Serve(ctx, ".", cmp.Or(addr, DefaultAddr))
		}
		configPath = found
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	log.Printf("Loaded configuration from: %s", configPath)

	module, err := NewModule(cfg)
	if err != nil {
		return err
	}

	opts := platform.NewOptions()
	opts.ServerAddr = cmp.Or(addr, cfg.Addr, DefaultAddr)

	p := platform.New(opts)
	p.Register(module)

	if err := p.Start(ctx); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	p.Wait()
	return nil
}
