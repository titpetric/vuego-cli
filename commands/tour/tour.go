package tour

import (
	"context"
	"log"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/platform"

	"github.com/titpetric/vuego-cli/tour"
)

// Name is the command title.
const Name = "Start the vuego tour server"

// New creates a new tour command.
func New() *cli.Command {
	var addr string
	var contentPath string

	return &cli.Command{
		Name:  "tour",
		Title: Name,
		Bind: func(fs *flag.FlagSet) {
			flag.StringVar(&addr, "addr", ":8080", "HTTP server address")
			flag.StringVar(&contentPath, "content", "", "Path to tour content directory (uses embedded if not specified)")
		},
		Run: func(ctx context.Context, args []string) error {
			dir := ""
			if len(args) > 0 {
				dir = args[0]
			}
			return Serve(ctx, addr, dir)
		},
	}
}

// Serve starts the tour server using the platform.
func Serve(ctx context.Context, addr string, contentPath string) error {
	opts := platform.NewOptions()
	opts.ServerAddr = addr

	var tourModule *tour.Module
	if contentPath != "" {
		log.Printf("Serving tour from: %s", contentPath)
		contentFS := os.DirFS(contentPath)
		tourModule = tour.NewModuleWithFS(contentFS)
	} else {
		log.Print("Serving embedded tour")
		tourModule = tour.NewModule()
	}

	p := platform.New(opts)
	p.Register(tourModule)

	if err := p.Start(context.Background()); err != nil {
		return err
	}

	p.Wait()
	return nil
}
