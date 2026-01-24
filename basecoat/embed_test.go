package basecoat_test

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/titpetric/vuego-cli/basecoat"
)

func PrintTree(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root itself
		if p == "." {
			return nil
		}

		depth := strings.Count(p, "/")
		indent := strings.Repeat("│   ", depth)

		name := path.Base(p)
		if d.IsDir() {
			fmt.Printf("%s├── %s/\n", indent, name)
		} else {
			fmt.Printf("%s├── %s\n", indent, name)
		}

		return nil
	})
}

func TestFS(t *testing.T) {
	PrintTree(basecoat.FS)
}
