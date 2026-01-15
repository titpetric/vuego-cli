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
- `version`: Show version/build information

## Testing

Tests are implemented using [titpetric/atkins](https://github.com/titpetric/atkins).
