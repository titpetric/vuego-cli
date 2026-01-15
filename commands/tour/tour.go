package tour

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/titpetric/platform"

	"github.com/titpetric/vuego-cli/tour"
)

// Run executes the tour command with the given arguments.
func Run(args []string) error {
	fs := flag.NewFlagSet("tour", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "HTTP server address")
	contentPath := fs.String("content", "", "Path to tour content directory (uses embedded if not specified)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	return Serve(*addr, *contentPath)
}

// Serve starts the tour server using the platform.
func Serve(addr string, contentPath string) error {
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

// Usage returns the usage string for the tour command.
func Usage() string {
	return `vuego tour [options]

Start the vuego tour server.

Options:
   -addr string      HTTP server address (default ":8080")
   -content string   Path to tour content directory (uses embedded if not specified)

Examples:
   vuego tour                                          # Start tour on default port with embedded content
   vuego tour -addr :3000                              # Start tour on port 3000
   vuego tour -content ./cmd/vuego/server/tour/content # Start tour with local content directory`
}
