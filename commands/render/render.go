package render

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"
)

// Name is the command title.
const Name = "Render templates with data"

// New creates a new render command.
func New() *cli.Command {
	return &cli.Command{
		Name:  "render",
		Title: Name,
		Bind: func(fs *flag.FlagSet) {
			// No flags for render command
		},
		Run: func(ctx context.Context, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("render: requires 1 or 2 arguments")
			}

			tplFile := args[0]
			var dataFile string

			// Determine data file
			if len(args) == 2 {
				// Explicitly provided data file
				dataFile = args[1]
			} else {
				// Auto-discover data file from template name
				dataFile = findDataFile(tplFile)
			}

			// Load data
			var data map[string]any
			if dataFile != "" {
				dataContent, err := os.ReadFile(dataFile)
				if err != nil {
					return fmt.Errorf("reading data file: %w", err)
				}

				if err := yaml.Unmarshal(dataContent, &data); err != nil {
					return fmt.Errorf("parsing data file: %w", err)
				}
			}

			// Initialize with empty map if no data
			if data == nil {
				data = make(map[string]any)
			}

			// Open template file for efficient streaming
			tplReader, err := os.Open(tplFile)
			if err != nil {
				return fmt.Errorf("opening template file: %w", err)
			}
			defer tplReader.Close()

			// Load template with data and render directly from file reader
			templateFS := os.DirFS(filepath.Dir(tplFile))
			tmpl := vuego.NewFS(templateFS).Fill(data)

			if err := tmpl.RenderReader(ctx, os.Stdout, tplReader); err != nil {
				return fmt.Errorf("rendering template: %w", err)
			}

			return nil
		},
	}
}

// findDataFile looks for a data file with the same base name as the template.
// Returns the path if found (checking .yaml first, then .yml, then .json), or empty string if not found.
func findDataFile(tplFile string) string {
	base := tplFile[:len(tplFile)-len(filepath.Ext(tplFile))]
	dir := filepath.Dir(tplFile)

	// Check for .yaml
	candidates := []string{
		filepath.Join(dir, filepath.Base(base)+".yaml"),
		filepath.Join(dir, filepath.Base(base)+".yml"),
		filepath.Join(dir, filepath.Base(base)+".json"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
