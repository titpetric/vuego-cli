# Creating a tour

A tour is a folder of Markdown chapters. Each chapter is split into lessons,
and each lesson attaches a set of files that the reader edits in the browser
and runs against the server. The tour server renders the lesson text on the
left and one editor per file on the right; **Render** posts the current
editor contents back and shows the result in an iframe.

Serve a tour from a folder:

```bash
vuego-cli tour ./content/my-tour
```

Or as a virtual host, which runs the same server:

```yaml
vhost:
  - domain: tour.example.com
    path: ./content/my-tour
    mode: tour
```

Called without a path, `vuego-cli tour` serves the application shell with no
chapters in it. The tour in this repository lives in `content/vuego-tour/`.

## Folder layout

```
content/my-tour/
  README.md            landing page, shown at /
  DONE.md              closing page, shown at /done
  01-interpolation.md  chapter, lessons separated by ---
  02-filters.md
  interpolation/       files for the lessons of 01-interpolation.md
    basic.vuego
    basic.yaml
  filters/
    basic.vuego
    basic.yaml
```

Every `.md` file in the root except `README.md` and `DONE.md` is a chapter.
Chapters are sorted by filename, which is what the numeric prefix is for. The
chapter title is the first `# ` heading in the file, falling back to the
filename.

Stripping the numeric prefix gives the chapter slug: `01-interpolation.md`
becomes `interpolation`. The slug is both the URL segment
(`/lesson/interpolation/0`) and the name of the directory holding that
chapter's files.

## Chapters and lessons

Within a chapter, lessons are separated by a `---` line with a blank line on
either side. Each lesson's title is its `## ` heading; without one it is
titled `Lesson N`. Lesson indexes in URLs are zero based.

```md
# Interpolation

## Basic interpolation

Output a value with `{{ name }}`.

@file: basic.vuego

---

## Nested access

Reach into maps and arrays with `{{ user.name }}` and `{{ items[0] }}`.

@file: nested.vuego
```

## Attaching files

An `@file:` line attaches a file to the lesson. The line is removed from the
rendered text. Paths resolve against the chapter's slug directory, so
`@file: basic.vuego` in `01-interpolation.md` reads
`interpolation/basic.vuego`. A reference that does not resolve is skipped
without an error, and the lesson is served without that editor.

Two companion files are attached implicitly:

- For `x.vuego`, the first of `x.yaml`, `x.yml`, `x.json` that exists becomes
  the data file for the render.
- For `x.sql`, the first of `x.up.sql`, `x.sqlite3` that exists becomes the
  migration applied before the query runs.

An `@file:` line takes options after the path. Only `hidden` is recognized:
it starts the editor collapsed, which suits schema files that support the
lesson without being its subject.

```md
@file: schema.up.sql hidden
@file: query.sql
```

### File names are flat

Files are keyed by their base name, so `@file: partials/card.vuego` is
offered to the reader, and to the renderer, as `card.vuego`. Includes and
requires inside a lesson must therefore use the base name:

```html
<template include="card.vuego"></template>
```

```php
require "card.php";
```

Referring to `partials/card.vuego` instead fails with `file does not exist`,
because no such path exists in the filesystem assembled for the render. Paths
that are not attached with `@file:` still resolve against the tour content
folder, so shared templates the reader is not meant to edit can be included by
their real path, relative to the tour root rather than to the chapter.

### Runnable code fences

A fenced code block whose info string is `php`, `vuego`, `json`, `yaml` or
`sql` also becomes a lesson file, without an `@file:` line and without a file
on disk. The first block of a kind is named `index` plus the extension, later
ones `snippet-2.php` and so on. The block stays in the lesson text, followed
by a note naming the file it turned into.

## What a lesson can run

| File     | Runs as                                   | Output                             |
|----------|-------------------------------------------|------------------------------------|
| `.vuego` | vuego template, filled from its data file | rendered HTML                      |
| `.php`   | the phpscript VM (`titpetric/phpscript`)  | whatever the script echoes         |
| `.sql`   | in-memory SQLite (`modernc.org/sqlite`)   | the query result, as an HTML table |

A lesson needs at least one of those three; the other files are data,
stylesheets and includes supporting them. `tour.ValidateLesson` states that
rule, but nothing on the serving path calls it, so a lesson with no runnable
file is served and renders nothing.

There is no Go evaluator. A fenced block marked as Go is ordinary markdown: it
is shown, not run, and a request naming `go` as its language is answered with
`unknown language: "go"`.

Shell execution exists in the code block service but is off in the tour.
`tour.DefaultConfig` enables PHP, vuego and SQLite only, so `bash`, `sh`,
`shell` and `exec` answer `evaluation not enabled`. Turning it on runs
arbitrary commands as the server user.

### vuego

The template is rendered with the data file that matches its name. Anything
the template includes is looked up first among the lesson's own files and
then in the tour content folder, which is how a lesson shares components it
does not attach.

Styles have to be inline. A `<style type="text/css+less">` block is compiled
to CSS by the LESS processor, and `.css` files among the lesson's files are
injected as `<style>` tags before `</head>`, falling back to `</body>` and
then to the end of the document. A `.less` file is not injected, and a
`<link rel="stylesheet">` pointing at a lesson file does not load: the preview
is an iframe filled with `srcdoc`, so the browser resolves the href against
the tour server, which serves no route for lesson files.

### PHP

The entry point is `index.php`; if there is no such file, the lexically first
`.php` file in the set is used instead. The script runs with the lesson's
files as its filesystem and the standard library registered. `exit` ends the
run without being reported as an error. Output is whatever the script wrote.

### SQL

Files ending in `.up.sql` or `.sqlite3` are applied as migrations to a fresh
in-memory database. The remaining `.sql` files then run, `index.sql` first and
the rest in name order. Every statement in a file is executed and the result
of its last query is kept; the first file that produces one is rendered as an
HTML table. With no result set at all the output is `ok`.

Statements are split on semicolons outside quotes. Named parameters (`:id`),
`$1` style parameters and bare `?` are all bound from a `*.params.json`,
`*.params.yaml` or `*.params.yml` file if the lesson attaches one; without it
the only value bound is `id` = 1.

## Choosing what runs

The browser sends every editor's contents on each render. The server picks the
evaluator from the file set: any `.sql` file makes it a SQL lesson, otherwise
an empty template makes it PHP, otherwise it renders vuego.

The template and the data are chosen in the browser, and with several
candidates the last one in name order wins: two `.vuego` files in one lesson
means only one of them is the template, and it is not necessarily the one the
lesson is about. Keep one runnable entry per lesson and attach the rest as
includes.

## Server APIs

`POST /render` is what the tour UI calls. It carries the legacy shape and
keeps the template and its data separate from the other files:

```bash
curl -s localhost:8080/render -H 'Content-Type: application/json' -d '{
  "template": "<b>{{ name }}</b>",
  "data": "name: world",
  "files": {}
}'
# {"html":"<b>world</b>\n"}
```

The response is `{"html":...}` or `{"error":...}`; an error is reported with
HTTP 200 and an `error` field, not a status code.

`POST /api/codeblock/eval` is the general form. It names the language and
passes every file in one map:

```bash
curl -s localhost:8080/api/codeblock/eval -H 'Content-Type: application/json' -d '{
  "language": "php",
  "entry": "index.php",
  "files": {"index.php": "<?php echo 1 + 1;"}
}'
# {"contentType":"text/html","content":"2"}
```

`entry` is optional and defaults to `index` plus the language's extension.
The language accepts these spellings:

| Language | Accepted as                      | Default entry | Enabled in the tour |
|----------|----------------------------------|---------------|---------------------|
| vuego    | `vuego`, `html+vuego`            | `index.vuego` | yes                 |
| php      | `php`, `application/x-httpd-php` | `index.php`   | yes                 |
| sql      | `sql`, `sqlite`, `sqlite3`       | `index.sql`   | yes                 |
| exec     | `exec`, `bash`, `sh`, `shell`    | `index.sh`    | no                  |

`GET /lesson/{chapter}/{index}` returns the rendered page, or the lesson as
JSON when the request asks for `application/json`. The JSON carries the lesson
text, its files, the `@file:` options and the navigation links, which is
enough to drive the tour from another frontend.

```bash
curl -s -H 'Accept: application/json' localhost:8080/lesson/interpolation/0
```

The same evaluators back the `docs` server, where a runnable fence in a
markdown page is rewritten into a **Run** button that posts to the same
`/api/codeblock/eval` path.
