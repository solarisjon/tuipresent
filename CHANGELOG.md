# Changelog

All notable changes to tuipresent are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0-rc1]

First release candidate: a markdown slide presenter for the terminal, compatible with most presenterm decks.

### Added

Presenting and navigation
- Decks are plain markdown split on `<!-- end_slide -->`, with `<!-- pause -->` reveal steps, `<!-- jump_to_middle -->` centering, columns, and a generated cover from front matter.
- Five slide transitions (`slide`, `wipe`, `cover`, `dissolve`, `none`) that also play in reverse when going back, and a per-cell fade for content revealed by a pause.
- Slide overview grid (`o`), jump to a slide by typing its number and `enter`, `g`/`G` first and last, and mouse navigation (click the left or right third, wheel to step, click a card in the overview).
- Live reload: saving the deck or its theme file updates the open presentation in place, keeping the slide and step.
- Step-through code line highlighting with `{1-3|5}` fence groups.
- Live demos: `+exec` code blocks (bash, sh, Python) run on `e` and show their output under the code.
- `-slide`, `-transition`, `-fade`, `-watch` and `-mouse` flags.
- `-check` (reports slides that overflow a `-w` x `-h` screen, no TTY needed) and `-dump` (prints every frame as text), so decks can be verified by agents and in CI.
- `-version` flag; release builds inject the version with `-ldflags "-X main.version=..."`.

Text and titles
- Setext slide titles drawn with a gradient, inline markdown in titles, framed and curly-underlined title styles, subtitles.
- Big titles (`title_size` 2 and 3): block letters everywhere, kitty text sizing where available.
- Text marks: `==highlight==`, `~~strikethrough~~`, faint and underline variants including `[text]{.curly}`.
- Themed GitHub-style callouts (`> [!NOTE]`, `> [!WARNING]` and the rest).

Terminal integration
- Clickable links with OSC 8 (`-links auto|on|off`), with the URL shown dimmed where they are unsupported.
- Deck progress in the terminal tab with OSC 9;4 (`-progress`, front matter `progress`).
- Terminal and capability detection for kitty, Ghostty, WezTerm, iTerm2 and foot, never enabling image protocols or text sizing inside tmux or screen.

Images
- PNG, JPEG and GIF (first frame) with width control (`![image:width:50%](...)`).
- Full-resolution drawing through the kitty graphics protocol (Unicode placeholders), the iTerm2 inline-image protocol and sixel, with a truecolor half-block fallback everywhere else (`-images auto|kitty|sixel|iterm2|blocks|off`).

Speaker notes
- presenterm-compatible speaker notes.
- `-notes` view in a second terminal that follows the presenter (slide, step, timer, notes, next title, deck and theme warnings) and can drive it with the same navigation keys.

Export and sharing
- `-export tape|vhs|freeze`: write a VHS tape, record a GIF/MP4/WebM with VHS, or render one PNG/SVG per slide with Freeze; `-out`, `-format`, `-dry-run`, `-dwell`. Freeze has no PDF output; combine the PNGs.

Themes
- Twenty built-in palette-based themes (tokyonight, catppuccin, gruvbox, dracula, nord, rose-pine, solarized, kanagawa and more), inline color overrides, custom theme files (`.yaml`), `-theme`, `-themes`, and `-background paint` to draw the theme background into every cell.

Documentation and tooling
- User guide (`docs/guide/`), agent documentation (language spec, verified examples, front-matter JSON Schema, `llms.txt`, agent guide and a Claude Code skill) and ready-to-run example decks.
- `internal/docscheck` fails `go test` when docs or examples drift from the code; git pre-commit and Claude Code hooks run it.
- `Makefile` with `build`, `test`, `check-examples` and `release` (cross-compiled static binaries and `SHA256SUMS`).

### Fixed

Robustness
- Backward transitions undo the transition that entered the slide.
- `+exec` runs in the deck's directory, with a timeout and a bounded wait on output pipes; unsupported `+exec` languages report an error instead of running through bash.
- An unterminated speaker note stops at the next directive line.
- Fence edge cases, notes parsing, and theme error handling (theme errors are checked before the notes view; inline `extends` warns).
- Deck and theme warnings show in the `-notes` view.

### Known limitations

See the Known limitations section of the [README](README.md).
