# dev-cli release site

Single-file static landing page for the `dev-cli` v0.1 release.

## Preview locally

```bash
# any of these work — no build step required
open web/index.html
xdg-open web/index.html
python3 -m http.server -d web 8080   # then http://localhost:8080
```

## Deploy

It's one HTML file with inline CSS and a tiny clipboard script. Drop it on
anything: GitHub Pages, Netlify, Cloudflare Pages, S3, a `caddy file-server`.

### GitHub Pages

```bash
# from main, with web/ committed
# Settings → Pages → Source: "Deploy from branch", Branch: main /web
# (or use a workflow that uploads ./web as the artifact)
```

## Design

Anthropic-inspired palette and type:

- Cream background (`#F0EEE6`), near-black ink, Claude-coral accent (`#CC785C`).
- Serif headlines (system serif fallback chain), sans-serif body, monospace for
  code/terminal.
- Generous whitespace, 14 px rounded corners, subtle shadows on interactive
  surfaces, no gradients in content (one soft radial in the CTA band).

## Sections

1. Sticky nav
2. Hero with live-style terminal demo of `dev-cli fix`
3. Three pillars (Trust-first / Local-first / Learn-from-success)
4. Six-card command grid (`fix`, `explain`, `ask`, `ui`, `doctor`, `runbook`)
5. Safe-mode panel (dark) with a denied-command terminal demo
6. Stats row (11 tools · 75+ patterns · 50+ globs · 0 tokens out)
7. Architecture ASCII diagram
8. Install + first-run code blocks (with copy buttons)
9. CTA band + footer

## Editing checklist

When you change the CLI surface, update:

- Command cards (id `#features` in `index.html`)
- Stats numbers (`<section>` after the safety panel)
- Architecture diagram (`#architecture` block)
- Footer version + license
