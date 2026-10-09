# Spec: Bundle Inter Variable + JetBrains Mono and pin vertical metrics

## What
Self-host Inter Variable and JetBrains Mono Variable (woff2 files from Google Fonts). Declare them with `@font-face` rules that pin `ascent-override`, `descent-override` and `line-gap-override`, so Chrome on Linux and Chrome/Safari on macOS build identical line boxes. Ship the fonts' SIL OFL 1.1 licence texts with them.

## Why
Neither font is loaded today. `--font-sans` / `--font-mono` name them, but each machine falls back to whatever it has installed, and those files have different vertical metrics. As a result, text sits ~1px high next to icons on Linux (the user tests on **Chrome/Linux**). Fixing the font source in one place may make the ~120-site `IconText` migration (xpo-6b4d21) unnecessary.

## How
1. **Fetch** the woff2 files from the Google Fonts CSS2 API using a modern Chrome UA:
   - `Inter:wght@100..900`
   - `JetBrains+Mono:wght@100..800`

   Keep the **latin** and **latin-ext** subsets for each font, together with their `unicode-range` values, so the browser only downloads what a page needs.
2. **Location:** put them in `web/src/assets/fonts/` and reference them by relative `url()` from `index.css`. Vite fingerprints them and emits them into the built assets, which `make build` embeds in the binary. No runtime CDN requests.
3. **Metrics:** read `unitsPerEm` and the hhea ascender/descender/lineGap from each file (fontTools, run from the scratchpad). Express them as percentages. Use **hhea** because macOS CoreText uses hhea today: macOS keeps its current look and Linux matches it. Leave a CSS comment recording the source values and the formula.
4. **@font-face** for each subset:
   - `font-family`: `"Inter"` or `"JetBrains Mono"` (the names `--font-*` already use)
   - `font-style: normal`
   - `font-weight` range: Inter `100 900`, JetBrains Mono `100 800`
   - `font-display: swap`
   - `src: url(...) format("woff2")`
   - `unicode-range`
   - the three overrides
5. Keep the `--font-sans` / `--font-mono` fallback stacks unchanged.
6. **Licensing (OFL 1.1 §2):** copy the upstream licence files from `google/fonts` (`ofl/inter/OFL.txt`, `ofl/jetbrainsmono/OFL.txt`) to `web/public/licenses/Inter-OFL.txt` and `web/public/licenses/JetBrainsMono-OFL.txt`. Vite copies `public/` into `dist`, so the licences are embedded in the binary next to the fonts and served at `/licenses/…`. Reference them from the font comment in `index.css`.
7. **Test (written first):** add a vitest test `web/src/fonts.test.ts` that reads `index.css?raw`, lists the fonts with `import.meta.glob`, and loads the licence files. For each family it asserts:
   - there are `@font-face` rules with all three overrides;
   - every `url()` resolves to a bundled file;
   - an OFL licence file with the family's copyright line exists.

   Vitest blanks CSS imports by default, so `vite.config.ts` gets `test.css.include: [/index\.css/]`.

## Acceptance criteria
- [x] The woff2 files are in `web/src/assets/fonts/` and appear in the Vite build output. DevTools on the running app shows Inter and JetBrains Mono loaded from the app origin. *Confirmed by the user.*
- [x] Each `@font-face` declares `ascent-override`, `descent-override` and `line-gap-override`, with a comment documenting where the values came from.
- [x] OFL licence texts for both fonts are in `web/public/licenses/` and are embedded in the build output.
- [x] Linux and macOS produce identical icon+text **layout**: measured delta 0.00px across 611 pairs on 9 routes.
- [ ] macOS shows no visible regression. *Checked by the user.*
- [x] `make test` passes (lint + Go + vitest).

## Findings (from verification)
The residual Linux offset is **rasterisation, not layout**: FreeType hinting plus baseline snapping to whole device pixels at fractional DPRs. It depends on display and text scaling, which the user confirmed on several devices. CSS can't fix it:
- `text-box-trim` doesn't move the label-pill glyphs, whose layout is already centred;
- `text-rendering: geometricPrecision` makes it worse.

Measured share of pairs off by ≥1 device px:

| DPR | Pairs ≥1px off |
|---|---|
| 1.0 | 0.4% |
| 1.1328 | 3.9% |
| 1.2 | 5.6% |
| 1.25 | 2.6% |
| 1.5 | 4.8% |
| 2.0 | 0% |

## Decisions
- **Self-host, not the Google CSS link:** the overrides need our own `@font-face`, and xpo must work offline.
- **hhea metrics as the reference:** this preserves the current macOS rendering, which the user considers correct. For both fonts hhea equals OS/2 typo; only Inter's win metrics (2269/660) differ.
- **latin + latin-ext subsets only:** these cover English and European UI text. Other scripts fall back to system fonts, as they do today.
- **Licences in `public/licenses/`:** this puts one copy in the repo, the build and the binary, which satisfies OFL §2. Neither font declares a Reserved Font Name, so the subsetted files keep their names.

## Out of scope
- The `IconText` / `text-box-trim` migration (xpo-6b4d21).
- Pixel-snapping drift that depends on the user's system at fractional DPRs.
