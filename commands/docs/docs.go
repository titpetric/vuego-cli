package docs

import (
	"context"
	"log"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/platform"
)

// Name is the command title.
const Name = "Start documentation server for markdown files"

// New creates a new docs command.
func New() *cli.Command {
	var addr string

	return &cli.Command{
		Name:  "docs",
		Title: Name,
		Bind: func(fs *flag.FlagSet) {
			fs.StringVar(&addr, "addr", ":8080", "HTTP server address")
		},
		Run: func(ctx context.Context, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return Serve(ctx, addr, dir)
		},
	}
}

// Serve starts the docs server using the platform.
func Serve(ctx context.Context, addr string, contentPath string) error {
	opts := platform.NewOptions()
	opts.ServerAddr = addr

	log.Printf("Serving docs from: %s", contentPath)
	contentFS := os.DirFS(contentPath)
	docsModule := NewModule(contentFS)

	p := platform.New(opts)
	p.Register(docsModule)

	if err := p.Start(context.Background()); err != nil {
		return err
	}

	p.Wait()
	return nil
}
