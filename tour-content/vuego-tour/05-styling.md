# Styling

## Inline Styles with LESS

Use `<style type="text/css+less">` to write LESS CSS directly in your templates. LESS preprocessor features like variables and nesting are automatically compiled to CSS.

- [LESS Language](http://lesscss.org/)

@file: inline-less.vuego

---

## External Stylesheets

Link to external CSS or LESS files using standard `<link>` tags. External LESS files are compiled to CSS on-the-fly.

@file: external-styles.vuego
@file: styles/forms.less

---

## Styling Form Components

Build styled forms with input validation and responsive design. Adjacent style files are automatically linked when rendering templates.

@file: registration.vuego
@file: registration.less
