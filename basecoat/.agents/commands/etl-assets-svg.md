# ETL: Extract SVG Icons to Assets

Extract inline SVG icons from template files. Lucide icons use `<i data-lucide="...">` tags; brand/custom icons go to `assets/svg/` as separate files.

## Scope

Scan all `.vuego` template files in:
- `index.vuego` (root)
- `layouts/`
- `partials/`
- `components/` (including subdirectories)

## Exclusions

Do NOT extract SVGs from:
- CSS files (`assets/css/`)
- JavaScript files (`assets/js/`)
- Documentation/markdown files (`.md`) unless they are rendered templates
- SVGs that use dynamic `v-html` or template interpolation (e.g., `<svg v-if="icon" v-html="icon">`)

## Icon Classification

### Lucide Icons (use `<i>` tag)

Lucide icons are identified by:
- Stroke-based SVGs with `stroke="currentColor"`
- Common Lucide attributes: `stroke-width="2"`, `stroke-linecap="round"`, `stroke-linejoin="round"`
- No `<title>` element with a brand name
- Geometric/UI icons (arrows, shapes, actions)

**Reference:** https://lucide.dev/icons/ — search here to find the icon name

### Brand/Custom Icons (use `<img>` tag)

Brand icons are identified by:
- `<title>` element with brand name (e.g., `<title>PayPal</title>`)
- `fill="currentColor"` with complex brand paths
- Custom logos specific to the project

**Reference:** See existing patterns in `assets/svg/` for naming conventions

## Replacement Patterns

### Lucide Icons → `<i>` tag

**Before:**
```html
<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <circle cx="12" cy="12" r="10"></circle>
  <path d="..."></path>
</svg>
```

**After:**
```html
<i data-lucide="icon-name"></i>
```

**Reference:** Grep for `data-lucide` in templates to see existing usage patterns

### Brand Icons → `<img>` tag

**Before:**
```html
<svg role="img" fill="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
  <title>BrandName</title>
  <path d="..."/>
</svg>
```

**After:**
```html
<img src="/assets/svg/brandname.svg" alt="BrandName" />
```

## CSS Class Updates

When replacing SVGs, update parent element selectors accordingly:
- `[&>svg]:...` → `[&>i]:...` (for Lucide)
- `[&>svg]:...` → `[&>img]:...` (for brand icons)

## Output Location

Brand/custom icons only go to `assets/svg/`. Do NOT create SVG files for Lucide icons.

## Naming Convention

- All lowercase, hyphen-separated
- Brand icons: match the `<title>` content
- Custom icons: descriptive name

**Reference:** List `assets/svg/` to see existing naming patterns

## Verification

After extraction:
1. Run `vuego-cli render index.vuego` to verify templates still render
2. Grep for `data-lucide` to confirm Lucide icons
3. Grep for `/assets/svg/` to confirm brand icon references
4. List `assets/svg/` — should only contain brand/custom icons, no Lucide icons

## Idempotency

- Skip icons already using `<i data-lucide="...">` pattern
- Skip icons already using `<img src="/assets/svg/...">` pattern
- Delete any Lucide icon SVG files that exist in `assets/svg/`
- Do not create duplicate SVG files
