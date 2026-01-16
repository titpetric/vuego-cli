# Filters

## Basic Filters

Apply filters using the pipe `|` operator: `{{ value | filterName }}`.

Vuego implements several built-in functions:

- **`upper`** - Converts string to uppercase
- **`lower`** - Converts string to lowercase
- **`title`** - Title-cases string
- **`trim`** - Removes leading and trailing whitespace
- **`default(fallback)`** - Returns fallback value if input is nil or empty
- **`len`** - Returns length of string, array, or map
- **`json`** - Converts value to JSON string

You can provide your own functions via the `Funcs` API in `vuego.Template`.

- [FuncMap - Template Functions](https://github.com/titpetric/vuego/blob/main/docs/funcmap.md)
- [Syntax - Filters and Pipes](https://github.com/titpetric/vuego/blob/main/docs/syntax.md#filters-and-pipes)

@file: basic.vuego

---

## Chained Filters

Chain multiple filters together: `{{ name | trim | lower | title }}`.

Each filter transforms the result of the previous one.

- [Syntax - Filters and Pipes](https://github.com/titpetric/vuego/blob/main/docs/syntax.md#filters-and-pipes)

@file: chained.vuego

---

## Utility Filters

Use `default` to provide fallback values: `{{ name | default("Anonymous") }}`.

Use `json` to encode values as JSON for debugging or JavaScript integration.

Use `len` to get the length of strings, arrays, or maps.

- [Built-in Functions](https://github.com/titpetric/vuego/blob/main/docs/funcmap.md#built-in-functions)

@file: utility.vuego

---

## File access

It's possible to provide file access with the `file` filter. This allows
you to read a file based on the document root, either an embed.FS or a
folder opened with `os.DirFS`.

- [Built-in Functions](https://github.com/titpetric/vuego/blob/main/docs/funcmap.md#built-in-functions)

@file: composition.less
@file: composition.js
@file: composition.vuego
