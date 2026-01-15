package format

import (
	"context"
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/vuego/formatter"
)

// Name is the command title.
const Name = "Format vuego template files"

// New creates a new format command.
func New() *cli.Command {
	return &cli.Command{
		Name:  "fmt",
		Title: Name,
		Bind: func(fs *flag.FlagSet) {
			// No flags for format command
		},
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("fmt: missing file argument")
			}

			f := formatter.NewFormatter()
			var lastErr error

			for _, file := range args {
				content, err := os.ReadFile(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
					lastErr = err
					continue
				}

				formatted, err := f.Format(string(content))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", file, err)
					lastErr = err
					continue
				}

				// Only write and report if content changed
				if string(content) != formatted {
					if err := os.WriteFile(file, []byte(formatted), 0o644); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
						lastErr = err
						continue
					}
					fmt.Println(file)
				}
			}

			return lastErr
		},
	}
}
