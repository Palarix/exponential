# Walkthrough: Bundle Inter Variable + JetBrains Mono and pin their vertical metrics

## What was built and why

`index.css` has always said `font-family: "Inter", …` and `"JetBrains Mono", …`, but nothing ever loaded those fonts. Each machine used whatever copy it had installed, or a fallback font. Different files have different vertical metrics, so the same CSS produced different line boxes on different machines. That was the first suspect for "text sits ~1px high next to icons on Linux".

This change ships the fonts with the app and pins their metrics, so every platform builds the same layout.

## How the pieces fit

### 1. Font files: `web/src/assets/fonts/`
There are four woff2 files, downloaded from the Google Fonts CSS2 API:
- Inter v20, `wght 100..900`
- JetBrains Mono v24, `wght 100..800`

Each font has a **latin** and a **latin-ext** subset, about 190 KB in total. They live under `src/` (not `public/`) and are referenced by relative `url()`, so Vite fingerprints them (`inter-latin-Dx4kXJAl.woff2`). `make frontend` copies `dist` into `internal/server/static`, and from there they are embedded in the `xpo` binary. There are no CDN requests at runtime, so xpo keeps working offline.

### 2. `@font-face` rules: `web/src/index.css`
There is one rule per subset. Each keeps Google's `unicode-range`, so a page only downloads the subsets it uses, and each pins three descriptors:

| Font | upem | hhea asc / desc / gap | `ascent-override` | `descent-override` | `line-gap-override` |
|---|---|---|---|---|---|
| Inter | 2048 | 1984 / −494 / 0 | 96.875% | 24.1211% | 0% |
| JetBrains Mono | 1000 | 1020 / −300 / 0 | 102% | 30% | 0% |

The values come from the files themselves (fontTools), as hhea values ÷ unitsPerEm. We use hhea because macOS CoreText uses it, so the Mac keeps its look and Linux is forced to match. For both fonts hhea equals OS/2 typo; only Inter's *win* metrics (2269/660) differ. That kind of disagreement lets platforms build different line boxes from the same file. The overrides remove the choice.

The family names stay `"Inter"` / `"JetBrains Mono"`, so `--font-sans` / `--font-mono` (fallback stacks included) work unchanged. A declared `@font-face` takes precedence over a locally installed font with the same name.

### 3. Licences: `web/public/licenses/`
Both fonts are under the SIL Open Font License 1.1, and neither declares a Reserved Font Name, so the subsetted files may keep their names. OFL §2 requires every copy to carry the copyright notice **and the licence text**. The woff2 files only carry the copyright line and a URL, so the upstream `google/fonts` `OFL.txt` files are in `public/licenses/`. Vite copies `public/` into `dist` verbatim, so the licences are embedded in the binary alongside the fonts (served at `/licenses/Inter-OFL.txt`, `/licenses/JetBrainsMono-OFL.txt`). Bundling with xpo's MIT licence is fine: OFL §5 keeps the fonts under OFL without affecting the software around them.

### 4. Test: `web/src/fonts.test.ts`
For each family, the test checks that:
- `@font-face` rules exist;
- each rule has all three overrides;
- every `url()` resolves to a bundled file (via `import.meta.glob`);
- an OFL file with that family's copyright line exists.

It uses Vite's `?raw` and `import.meta.glob` rather than `node:fs`, because the app tsconfig only has `vite/client` types. A mutation check (breaking one `ascent-override`) confirmed it isn't passing vacuously.

### 5. `web/vite.config.ts`
`test.css.include: [/index\.css/]`. Vitest replaces CSS imports with an empty string by default, *including* `?raw`. Without this setting the test reads `""`.

## Verification: what we learned about the Linux offset

The user tests on Chrome/Linux (Hyprland). To avoid one tophat round per hypothesis, I built a measurement rig:

- **Linux renderer:** Playwright's Linux Chromium in Docker, with fontconfig forced to the user's settings (slight hinting, greyscale) and the user's exact `devicePixelRatio`.
- **Mac renderer:** Chrome on the Mac.
- **Measure:** a script finds every `flex` / `items-center` row that pairs an `<svg>` with text (611 pairs on 9 routes). For each pair it measures the icon centre against the text's cap centre, at layout level using a baseline probe and at pixel level from the rendered ink.

Findings:

1. **Layout is now identical on both platforms:** the delta is 0.00px for all 611 pairs.
2. **Layout is already centred in the label pills.** Applying `text-box: trim-both cap alphabetic` to pill text puts the cap box centre within 0.01px of the dot centre, and the rendered pixels don't change. So the planned `IconText` / `text-box-trim` migration (xpo-6b4d21) would not fix the pills. It was canceled.
3. **The remaining Linux offset is rasterisation.** FreeType hinting plus baseline snapping to whole device pixels at fractional DPRs causes it. The user's DPR was 1.1328125 (Hyprland 1.25 × GNOME text scaling 0.9091). The rig reproduced the user's screenshot: label pills +1 to +1.5px, sidebar 0px. The user then confirmed on several devices that the offset tracks display and text scaling.
4. **`text-rendering: geometricPrecision`** (which turns off hinting on Linux Chrome) made it *worse*.

Share of pairs off by ≥1 device px, Linux vs Mac, at each DPR:

| DPR | Pairs ≥1px off |
|---|---|
| 1.0 | 0.4% |
| 1.1328 | 3.9% |
| 1.2 | 5.6% |
| 1.25 | 2.6% |
| 1.5 | 4.8% |
| 2.0 | 0% |

The recommended user-side setup for a 1080p Hyprland laptop is monitor scale 1.25, `text-scaling-factor` 1.0, and native-Wayland Chrome.

**For a future reader:** if an icon+text row looks off on Linux, first measure the icon-vs-cap-centre delta in CSS px. If it's ≈0, the cause is rasterisation and there is nothing to fix in the component. Fix a component only if its *layout* delta is non-zero.

## Acceptance criteria
- [x] The woff2 files are in `web/src/assets/fonts/` and appear in the Vite build output. DevTools shows them loaded from the app origin. *Evidence:* `dist/assets/inter-latin-Dx4kXJAl.woff2` etc.; the user saw "Inter — Network resource" on Linux.
- [x] Each `@font-face` declares the three overrides, with a comment documenting where the values came from. *Evidence:* `index.css` header comment; `fonts.test.ts` "pins ascent, descent and line-gap metrics".
- [x] OFL licence texts for both fonts are in `web/public/licenses/` and in the build output. *Evidence:* `internal/server/static/licenses/*.txt` after `make frontend`; `fonts.test.ts` "ships the SIL Open Font License text".
- [x] Linux and macOS produce identical icon+text layout. *Evidence:* 0.00px delta across 611 pairs on 9 routes (rig above).
- [x] macOS shows no visible regression. *Evidence:* user approval; the user noted the Mac renders "sharper, … no alignment issues".
- [x] `make test` passes. *Evidence:* lint + Go + 443 vitest tests green.
