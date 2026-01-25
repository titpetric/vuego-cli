---
title: Alert
subtitle: Displays a callout for user attention
layout: page
---

@tabs
@render "Preview" alert-demo.vuego
@file "Code" alert-demo.vuego
@file "Data" alert-demo.yaml

## Usage

Use the `alert` or `alert-destructive` class name.

```html
<div class="alert">
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10" />
    <path d="m9 12 2 2 4-4" />
  </svg>
  <h2>Success! Your changes have been saved</h2>
  <section>This is an alert with icon, title and description.</section>
</div>
```

The component has the following HTML structure:

Use `alert` for default styling or `alert-destructive` for error states.

- `<div class="alert">` - Main container.
  - `<svg>` - optional, the icon
  - `<h2>` - the title
  - `<section>` - optional, the description

## Examples

### Destructive

@example examples/alert-destructive.vuego

### No description

@example examples/alert-no-description.vuego

### No icon

@example examples/alert-no-icon.vuego
