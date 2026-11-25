# Example decks

Each deck passes `-check` at 100x30 and 80x24 (`go test ./internal/docscheck` enforces it). Run them from the repo root after building (`go build -o bin/tuipresent ./cmd/tuipresent`). Press `e` on a slide with code to run its `+exec` blocks; all of them are offline, read-only and finish in well under a second (they run in this directory). For the syntax behind them, see the [authoring guide](../docs/guide/authoring.md) and the [reference](../docs/guide/reference.md).

Any deck can be shared: `tuipresent -export tape examples/tour.md` prints a VHS tape that steps through every slide and pause, `tuipresent -export freeze -dry-run -out slides examples/tour.md` shows the Freeze commands that turn each slide into a PNG, and `-export vhs` / `-export freeze` run the tools if they are installed. See the [sharing guide](../docs/guide/sharing.md).

**[tour.md](tour.md)**, 16 slides. "tuipresent in 5 minutes": a self-demonstrating feature tour. Each slide shows a feature next to the markdown that produced it: front matter, titles (inline markdown, subtitle, curly underline), text styles and callouts, pauses, columns, syntax highlighting, step-through code line highlights (`{1|2-4|all}`), `+exec` (`date`, `uname -s`, a `python3` calculation), text charts, `jump_to_middle`, live reload on save, speaker notes, images (including a photo), and all five transitions set per slide.

```sh
./bin/tuipresent examples/tour.md
```

**[engineering-review.md](engineering-review.md)**, 12 slides. A blameless incident postmortem for a cache-key change that overloaded a database: impact and timeline tables, a text-art latency chart, before/during columns, pause-revealed root cause, `diff` fences of the change and the fix, and action items. Theme `dracula`, `wipe` transitions.

```sh
./bin/tuipresent examples/engineering-review.md
```

**[product-pitch.md](product-pitch.md)**, 11 slides. A keynote-style pitch for a fictional on-call product: centered statement slides, dramatic pauses, very little text per slide, `dissolve` and `cover` transitions, and speaker notes on key slides (try `-notes` in a split). Theme `tokyonight-storm`.

```sh
./bin/tuipresent examples/product-pitch.md
```

**[themes.md](themes.md)**, 9 slides. A theme gallery: headings and the gradient title, lists, inline code and links, syntax-highlighted code, a table, quotes, text marks and callouts, a text chart and a pause fade, plus a slide listing every built-in theme. Made for previewing themes; it paints its background so it looks the same in any terminal:

```sh
./bin/tuipresent -theme nord examples/themes.md
./bin/tuipresent -theme catppuccin-latte examples/themes.md
./bin/tuipresent -themes    # all names with color swatches
```

**[brand.md](brand.md)**, 4 slides. A short deck in a custom theme, [themes/brand.yaml](themes/brand.yaml): it extends `nord` and overrides the accents, code colors and code style. Copy the file as a starting point for your own theme; edit it while the deck is open and the deck reloads.

```sh
./bin/tuipresent examples/brand.md
```

**[terminal-tricks.md](terminal-tricks.md)**, 13 slides. A technical talk, "Things your terminal can do in 2026": escape codes in bash, Go and Rust, a clickable link and the tab progress bar, the kitty graphics protocol (with an image), box-drawing diagrams, a 3-column layout, and four live `+exec` blocks (an `isatty` check, Unicode cell widths, shell arithmetic, an `awk` sum).

```sh
./bin/tuipresent examples/terminal-tricks.md
```
