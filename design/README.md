# Brand assets

`brand-sheet.png` is the source brand sheet (logo lockup, app icon, dark
variant, color palette) the "Luma" name and visual identity are based on.

The actual files the app serves — cropped out of that sheet with their
backgrounds keyed to transparency — live in
[`frontend/public/`](../frontend/public/):

| File | Used for |
|------|----------|
| `favicon.png` | Browser tab icon (referenced from `frontend/index.html`) |
| `logo-light.png` / `logo-dark.png` | Full lockup (icon + wordmark + tagline), Login page |
| `logo-light-compact.png` / `logo-dark-compact.png` | Icon + wordmark only (no tagline), dashboard header |

The `-light`/`-dark` pair of each asset is swapped by a CSS
`@media (prefers-color-scheme: dark)` rule (see `.brand-logo*` in
`frontend/src/styles/global.css`), the same way every other color in that
file follows the OS theme — there is no in-app light/dark toggle.

The interface's color system (background, surface, text, primary, and the
lavender/yellow accents, plus success/warning/error) is defined as design
tokens — CSS custom properties in `frontend/src/styles/global.css`,
documented alongside the rest of the visual system in
[`../luma-design-agent.md`](../luma-design-agent.md) — rather than by
referencing this sheet's raw values directly. The primary blue
(`--color-primary: #5278d9`) and the dark theme's page background
(`--color-bg` in its dark variant) both stay close in hue to this sheet's
logo blue and its "versão escura" card, so the logo blends into the page
in both themes without a visible edge, while every other interactive color
(button fills, borders, semantic states) is chosen for accessible contrast
first.
