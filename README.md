# vuego-cli - CLI for the vuego template engine

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
- `host`: Serve multiple sites by domain from a config file
- `version`: Show version/build information

The `serve`, `docs` and `tour` commands take an optional path argument after
the command. If provided, the contents are loaded from that location. If
omitted, `serve` and `docs` load files from the current directory (`.`), and
`tour` serves an empty tour, so it needs a path to a tour folder such as
`content/vuego-tour/`. Writing one is covered in
[docs/tour-guide.md](docs/tour-guide.md).

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

## Virtual hosting

The `host` command serves several sites from one process, picking the site
by the request host name. It is meant to run behind a reverse proxy that
passes the client's `Host` header through, so one container can back every
domain you point at it.

Sites are declared in `vuego-cli.yml`:

```yaml
vhost:
  - domain: docs.example.com docs.localhost
    path: ./content/docs
    mode: docs
  - domain: tour.example.com
    path: ./content/vuego-tour
    mode: tour
  - domain: static.example.com
    path: ./content/assets
    mode: serve
```

A vhost entry has three fields:

- `domain`: the host name to match, or several of them separated by spaces,
  which serves the same content on every name listed. Matching ignores case
  and the port. A name may only appear in one vhost entry.
- `path`: the content root. This is the full path to the folder holding the
  content, so point it at the nested folder when the documents do not sit at
  the top of a checkout. Relative paths resolve against the directory holding
  the config file, not the working directory.
- `mode`: `docs`, `tour` or `serve`, selecting the same server the command of
  that name runs.

Two optional top level keys are available. `addr` sets the listen address,
and `trust_forwarded_host` routes by the `X-Forwarded-Host` header for
proxies that rewrite `Host` on the way through. Leave the latter off unless
the process is only reachable through such a proxy, because any client that
can reach the server can set the header.

```bash
vuego-cli host --config vuego-cli.yml --addr :8080
```

Without `--config`, the current directory is searched for `vuego-cli.yml`
and `vuego-cli.yaml`. When neither exists the current directory is served as
`vuego-cli serve .` would. A config that is present but unusable, such as one
naming a path that does not exist, stops the server from starting rather than
leaving a site silently missing.

### The sites in this repository

The `vuego-cli.yml` at the root declares a vhost for every content tree here,
and `compose.yml` runs one container serving all of them behind the Caddy
proxy on the `projects-ingress` network:

```bash
atkins up      # build the image and start the container
atkins down    # stop it
```

| Host                  | Tree                  | Mode |
|-----------------------|-----------------------|------|
| `docs.localhost`      | `docs/`               | docs |
| `basecoat.localhost`  | `basecoat/`           | docs |
| `bootstrap.localhost` | `basecoat/docs/`      | docs |
| `tour.localhost`      | `content/vuego-tour/` | tour |

The hostnames appear in three places: in `vuego-cli.yml`, which decides what
the server answers for; in the `caddy` label in `compose.yml`, which decides
what the proxy routes, with several hostnames in one label separated by
spaces; and in the healthcheck of the same file, which asks for one of them by
name. Adding a site means editing the first two, and renaming or dropping
`docs.localhost` means editing the healthcheck as well, or the container is
reported unhealthy while serving fine.

A docs tree needs a `menu` in its data, because the basecoat sidebar partial
declares it as a required attribute. Vuego loads `theme.yml` at the root of
the tree and then every `data/*.yml`, so the key can come from either; the
trees here use both. A tree that defines `menu` nowhere renders a 500 with
`required attribute 'menu' not provided` instead of a page.

Writing tour content is covered in [docs/tour-guide.md](docs/tour-guide.md).

The platform telemetry dashboard mounts on the shared router before the
dispatcher, so when it is enabled it answers on every configured hostname,
ahead of any vhost route with the same path. It is unauthenticated. The
compose service asks for it with `PLATFORM_TELEMETRY_ENABLED` and names itself
with `PLATFORM_TELEMETRY_SERVICE`; `PLATFORM_TELEMETRY_PATH` moves it off
`/debug/oida`. The pinned `platform` v0.6.0 still enables it by default, so the
variable only documents the intent until the dependency is bumped past the
release that makes recording opt-in.

## Docker

You can also use the `titpetric/vuego-cli` docker image.

```bash
docker run --rm -p 8080:8080 -v $PWD:/app titpetric/vuego-cli
```

The image runs `host` against `/app`, which is the working directory. Mount
a tree holding `vuego-cli.yml` and the content it names, and one container
serves every site in it on
[http://localhost:8080](http://localhost:8080). With no config file in the
tree the folder is served as `serve .` would, so a templates folder works
without any configuration.

Name a command to run one of the single site servers instead:

```bash
docker run --rm -p 8080:8080 -v $PWD:/app titpetric/vuego-cli serve .
docker run --rm -p 8080:8080 -v $PWD:/app titpetric/vuego-cli docs .
docker run --rm -p 8080:8080 -v $PWD:/app titpetric/vuego-cli tour .
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
