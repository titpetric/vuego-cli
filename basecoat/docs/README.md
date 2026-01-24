# Vuego + Basecoat

This is a documentation project which is essentially a setup of:

- Basecoat UI: https://basecoatui.com/ (this exact page)
- Tailwind
- Lucide Icons (CDN)
- Template driven JS embeddings (highlightjs, htmx, ...)

It provides an usable setup for documentation based pages.
To run it, use `vuego-cli docs bootstrap`.

The goals are:

- Create vuego templates
- Document vuego syntax like basecoat documents components
- Create components for reuse
- Create other documentation sites for learning content (multi-language)

The outside dependencies for this are minimal, see:

- grab latest tailwindcss binary from the releases page
- run `atkins generate` in root to refresh css (`two compile steps`)
- no runtime dependency on npm/node
- vuego for templating is go native

The general basecoat setup is there, docs need work. The alert
page is the only current doc. The contents of the folder can
be navigated, take a look around.
