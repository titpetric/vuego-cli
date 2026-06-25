# Creating a tour lesson

A tour lesson is a Markdown file that explains the concept and references runnable files with `@file:` blocks.

The tour UI loads those files into editors. When the user clicks render:

- `.vuego` + `.json` / `.yaml` files render as a Vuego template.
- `.php` files run through the PHP VM.
- Multiple files can be included, so examples can be split into templates, data, partials, or PHP includes.

## Vuego lesson

Use Vuego when the lesson is about templates, data binding, includes, slots, filters, or layout rendering.

### `content/examples/hello-vuego.md`

```md
# Vuego template rendering

This lesson shows a page template using data from a JSON file and a reusable
partial template.

Edit the template or data and click **Render**.

@file: index.vuego
@file: data.json
@file: partials/card.vuego
```

### `content/examples/index.vuego`

```html
<section>
  <h1>{{ title }}</h1>

  <template include="partials/card.vuego" :item="featured"></template>
</section>
```

### `content/examples/data.json`

```json
{
  "title": "Vuego tour example",
  "featured": {
    "title": "Reusable partial",
    "body": "This card was rendered from partials/card.vuego."
  }
}
```

### `content/examples/partials/card.vuego`

```html
<article class="card">
  <h2>{{ item.title }}</h2>
  <p>{{ item.body }}</p>
</article>
```

### How it works

- `index.vuego` is the main template.
- `data.json` provides values such as `title` and `featured`.
- `partials/card.vuego` is available in the same in-memory filesystem.
- The main template includes it with:

```html
<template include="partials/card.vuego" :item="featured"></template>
```

## PHP lesson

Use PHP when the lesson is about PHP syntax, functions, includes, control flow, string output, or generating HTML from PHP snippets.

### `content/examples/hello-php.md`

```md
# PHP rendering

This lesson runs PHP code inside the tour.

The main file is `index.php`. It can include other PHP files from the same
lesson.

Edit the PHP files and click **Render**.

@file: index.php
@file: lib/card.php
```

### `content/examples/index.php`

```php
<?php

require "lib/card.php";

$title = "PHP tour example";

echo "<section>";
echo "<h1>" . $title . "</h1>";
echo card("Reusable PHP include", "This card came from lib/card.php.");
echo "</section>";
```

### `content/examples/lib/card.php`

```php
<?php

function card($title, $body) {
    return "<article class=\"card\">"
        . "<h2>" . $title . "</h2>"
        . "<p>" . $body . "</p>"
        . "</article>";
}
```

### How it works

- `index.php` is the PHP entrypoint.
- Additional `.php` files are available in the same in-memory filesystem.
- `require "lib/card.php";` loads another file from the lesson.
- Anything printed with `echo` becomes the rendered preview.

## Quick contrast

| Feature | Vuego | PHP |
|---|---|---|
| Main file | `.vuego` template | `index.php` |
| Data file | `.json`, `.yaml`, `.yml` | PHP variables/code |
| Reuse | `<template include="...">` | `require "..."` / `include "..."` |
| Output | Rendered template HTML | Anything echoed by PHP |
| Best for | Template features | PHP snippets and generated HTML |

## Rule of thumb

Use **Vuego** when the example should demonstrate declarative template authoring:

```md
@file: index.vuego
@file: index.json
@file: partials/item.vuego
```

Use **PHP** when the example should demonstrate executable PHP:

```md
@file: index.php
@file: lib/helpers.php
```

Both styles support multiple files, and both are runnable directly from the tour editor.
