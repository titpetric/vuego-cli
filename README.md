# vuego-cli

Command-line interface for the vuego template engine. Vuego is a lightweight, expression-based templating system for generating content from templates and data files.

## Installation

```bash
go install github.com/titpetric/vuego-cli@latest
```

## Usage

Usage: `vuego-cli (command) [--flags]`

Available commands:

- `fmt`: Format vuego template files
- `render`: Render templates with data
- `diff`: Compare two HTML/vuego files using DOM comparison
- `serve`: Start development server for templates and assets
- `tour`: Start the vuego tour server
- `docs`: Start a vuego centric docs server
- `version`: Show version/build information

The `serve` and `tour` take an optional path argument after the command.
If provided, the contents will be loaded from that location. If omitted,
the `tour` command will load the embedded tour, and the `serve` command
will load files from the current directory (`.`).

## Basecoat

You can use the basecoat as a package to provide a "theme" for your
vuego templates. The templates are embedded and available for use.

```go
import (
	"github.com/titpetric/vuego"
	"github.com/titpetric/vuego-cli/basecoat"
)

// NewModule creates a new docs module with a filesystem.
func NewModule(contentFS fs.FS) *Module {
        ofs := vuego.NewOverlayFS(contentFS, basecoat.FS)
        return &Module{
                FS:    ofs,
                vuego: vuego.NewFS(ofs, vuego.WithLessProcessor()),
        }
}
```

The `docs` command uses the basecoat as an imported package. The
tool then scans a folder with content to overlay it on top of the
base filesystem. With this approach you're enabled to:

- change or replace components, assets
- use basecoat as a theme to set up front end applications
- use existing `vuego-cli` with custom content

## Docker

You can also use the `titpetric/vuego-cli` docker image.

```bash
docker run --rm -p 8080:8080 titpetric/vuego-cli
```

By default it starts the tour server on [http://localhost:8080](http://localhost:8080).
From your templates folder, run:

```bash
docker run --rm -p 8080:8080 -v $PWD:/app titpetric/vuego-cli serve .
```

You may produce a folder tree for `serve`, `docs` or `tour`, providing
your own content for your projects.

When you navigate to any of the following files:

- `.json/yml` - data for the template is displayed,
- `.less` - LESS CSS is rendered to CSS on the fly
- `.vuego` - will render the template with the json data

With `docs` and `tour`, additional rendering is implemented around `.md` files.

When you edit the data and template in your editor of choice, you have
to refresh your browser to see the changes (similar to PHP development).
No server restart is necessary.

## Testing

Tests are implemented using [titpetric/atkins](https://github.com/titpetric/atkins).
