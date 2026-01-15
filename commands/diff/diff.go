package diff

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	flag "github.com/spf13/pflag"
	"github.com/titpetric/cli"
	"github.com/titpetric/vuego/diff"
)

// Name is the command title.
const Name = "Compare two HTML/vuego files using DOM comparison"

// New creates a new diff command.
func New() *cli.Command {
	var outputFormat string

	return &cli.Command{
		Name:  "diff",
		Title: Name,
		Bind: func(fs *flag.FlagSet) {
			flag.StringVar(&outputFormat, "output", "unified", "output format: simple, unified, yaml")
			flag.StringVar(&outputFormat, "format", "unified", "deprecated: use --output instead")
		},
		Run: func(ctx context.Context, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("diff: requires exactly 2 file arguments")
			}

			file1 := args[0]
			file2 := args[1]

			// Read both files
			content1, err := os.ReadFile(file1)
			if err != nil {
				return fmt.Errorf("reading %s: %w", file1, err)
			}

			content2, err := os.ReadFile(file2)
			if err != nil {
				return fmt.Errorf("reading %s: %w", file2, err)
			}

			// Compare using DOM-aware comparison
			isEqual := diff.CompareHTML(content1, content2)

			switch outputFormat {
			case "simple":
				return runSimple(file1, file2, content1, content2, isEqual)
			case "unified":
				return runUnified(file1, file2, content1, content2, isEqual)
			case "yaml":
				return runYAML(content1, content2, isEqual)
			default:
				return fmt.Errorf("unknown output format: %s (use: simple, unified, yaml)", outputFormat)
			}
		},
	}
}

func runSimple(file1, file2 string, content1, content2 []byte, isEqual bool) error {
	if isEqual {
		fmt.Printf("✓ %s and %s have equivalent DOM trees\n", file1, file2)
		return nil
	}
	fmt.Printf("✗ %s and %s have different DOM trees\n", file1, file2)
	fmt.Printf("\nExpected (%s):\n%s\n", file1, content1)
	fmt.Printf("\nActual (%s):\n%s\n", file2, content2)
	return fmt.Errorf("DOM trees do not match")
}

func runUnified(file1, file2 string, content1, content2 []byte, isEqual bool) error {
	formatted1, err := diff.FormatToNormalizedHTML(content1)
	if err != nil {
		return fmt.Errorf("formatting %s: %w", file1, err)
	}
	formatted2, err := diff.FormatToNormalizedHTML(content2)
	if err != nil {
		return fmt.Errorf("formatting %s: %w", file2, err)
	}

	unifiedDiff := diff.GenerateUnifiedDiff(file1, file2, formatted1, formatted2)
	fmt.Print(unifiedDiff)

	if !isEqual {
		return fmt.Errorf("DOM trees do not match")
	}
	return nil
}

func runYAML(content1, content2 []byte, isEqual bool) error {
	yaml1 := diff.DomToYAML(content1)
	yaml2 := diff.DomToYAML(content2)

	// Create temp files for dyff
	tmp1, err := os.CreateTemp("", "vuego-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp1.Name())

	tmp2, err := os.CreateTemp("", "vuego-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp2.Name())

	if _, err := tmp1.WriteString(yaml1); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmp1.Close()

	if _, err := tmp2.WriteString(yaml2); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmp2.Close()

	// Run dyff
	cmd := exec.Command("dyff", "between", tmp1.Name(), tmp2.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Ignore error as dyff returns 1 when files differ

	if !isEqual {
		return fmt.Errorf("DOM trees do not match")
	}
	return nil
}
