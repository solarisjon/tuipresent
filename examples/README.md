# Example decks

Each deck passes `-check` at 100x30 and 80x24. Run them from the repo root after building (`go build -o bin/tuipresent ./cmd/tuipresent`). Press `e` on a slide with code to run its `+exec` blocks; all of them are offline, read-only and finish in well under a second.

**[tour.md](tour.md)**, 15 slides. "tuipresent in 5 minutes": a self-demonstrating feature tour. Each slide shows a feature next to the markdown that produced it: front matter, titles, pauses, columns, syntax highlighting, `+exec` (`date`, `uname -s`, a `python3` calculation), text charts, `jump_to_middle`, live reload on save, speaker notes, images (including a photo), and all five transitions set per slide.

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

**[terminal-tricks.md](terminal-tricks.md)**, 12 slides. A technical talk, "Things your terminal can do in 2026": escape codes in bash, Go and Rust, the kitty graphics protocol (with an image), box-drawing diagrams, a 3-column layout, and four live `+exec` blocks (an `isatty` check, Unicode cell widths, shell arithmetic, an `awk` sum).

```sh
./bin/tuipresent examples/terminal-tricks.md
```
