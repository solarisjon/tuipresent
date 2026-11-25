# tuipresent deck language specification

Normative reference for the markdown dialect tuipresent reads. Written for LLM agents. It describes what the code does today (`internal/deck/parse.go`, `internal/render/render.go`, `cmd/tuipresent/`), including quirks. Where this file and other docs disagree, the code wins; report the drift.

Conventions: "MUST" marks a rule the parser enforces (break it and the construct is not recognized). "line" means one `\n`-terminated source line. `cw` is content width, `w`/`h` the screen size in cells. Integer division truncates. Every claim marked (E*n*) is demonstrated with real output in [EXAMPLES.md](EXAMPLES.md) case *n*.

Contents: 1 File structure · 2 Front matter · 3 Slide splitting · 4 Slide parse algorithm · 5 Directives · 6 Blocks and reveal steps · 7 Titles · 8 Speaker notes · 9 Images · 10 Code and `+exec` · 11 Layout model · 12 Centering · 13 Rendering behaviors · 14 `-check` · 15 Runtime · 16 Invariants · 17 Quick grammar · 18 Themes · 19 Warnings and errors

---

## 1. File structure

```
deck      = [ front-matter ] body
body      = chunk { end-slide-line chunk }
```

1.1. A deck is one UTF-8 file. `\r\n` is replaced by `\n` before anything else. No BOM handling: a BOM before `---` disables front matter.
1.2. The body is split into chunks at `end_slide` directive lines outside fences (§3). Each chunk is parsed into at most one slide (§4).
1.3. A chunk that yields no title, no visible content and no speaker notes is dropped (§3.4). If no slide remains (cover included), loading fails: `deck has no slides`, exit 1.
1.4. Slide numbering (`-slide N`, footer `N / total`, `-dump` headers) is 1-based and counts the cover slide.

## 2. Front matter

2.1. Recognized only if the file starts with exactly `---\n` (byte 0). The block ends at the first `\n---\n` after that, or at a final `\n---` at end of file. The delimiters MUST be exactly `---` (no trailing spaces).
2.2. If no closing delimiter is found, there is no front matter: the whole file, including the `---` and YAML lines, is body (renders as a rule and text).
2.3. An empty block (`---\n---\n`, or `---\n---` as the whole file) is valid front matter with no keys (E26).
2.4. The block is parsed with YAML (gopkg.in/yaml.v3, non-strict). Invalid YAML, or a sequence/mapping where text is expected, is a fatal load error printed as `front matter: yaml: ...`, exit 1. Unknown keys are ignored.

| Key | Type | Default | Effect |
|---|---|---|---|
| `title` | text | empty | Non-empty: insert a cover slide as slide 1 (§2.6); footer left text. |
| `sub_title` | text | empty | Italic line on the cover. |
| `author` | text | empty | Cover byline; footer suffix `  ·  author`. |
| `date` | text | empty | Cover byline after author: `author  ·  date`. Free text. |
| `theme` | text, or mapping | `dark` | Color theme (§2.5, §18). |
| `transition` | text | `slide` | Default transition into every slide (§15.3). An unknown kind acts as `slide` and warns (§19). |
| `fade` | text or bool | on | `false`/`off`/`no`/`none` (case-insensitive, trimmed) disable pause fades deck-wide; anything else leaves them on. |
| `progress` | text or bool | on | `false`/`off`/`no`/`none` (case-insensitive, trimmed) stop reporting deck progress to the terminal tab (§15.8); anything else leaves it on. |
| `title_style` | text | `rule` | Default title style (§7.7): `rule`, `curly` or `frame`, case-insensitive. Anything else warns and acts as `rule` (§19). |
| `title_size` | text or int | `1` | Default title size (§7.9): `1`, `2` or `3`. Anything else warns and acts as `1` (§19). |

"text" = any YAML scalar, used as its string form (`title: 123` gives `123`; `date: 2026-12-08` stays `2026-12-08`).

2.5. Theme. A scalar is a theme name: `theme: NAME` is the same as `theme: {name: NAME}`. A mapping takes these keys (§18 has the names, palette and semantics):

| Key | Value |
|---|---|
| `name` | Built-in theme name or alias (§18.1). |
| `path` | Custom theme file (§18.4), relative to the deck file's directory or absolute. If `name` is also set, `path` wins and a warning is emitted. |
| `background` | `terminal` (default) or `paint` (§18.5). Here it is the painting mode, NOT a color; another value warns and means `terminal`. |
| palette key (§18.3, except `background`) | Quoted color `"#rrggbb"` or `"#rgb"`, applied on top of `name`/`path`. An empty or malformed color is a load error. |
| `code_style` | Chroma style name; unknown names warn and keep the base style. |

Any other key, `extends` included, is ignored with a warning (`extends` only works in theme files; use `name`). A key with a non-scalar value (such as presenterm's `override:` block) is ignored with a warning. A sequence is a load error.

2.6. Cover slide (only when `title` is non-empty): vertically centered; lines are the title as spaced uppercase (`Demo` -> `D E M O`; falls back to the title as written if the spaced form is wider than `cw`), blank, a `━` rule of width `min(cw, titleWidth + 8)`, then if `sub_title`: blank + subtitle, then if `author` or `date`: 2 blanks + `author  ·  date` (only the present ones). Each line is centered in `cw`. The cover never has a title block, body or steps (Steps = 1) (E1).

2.7. Schema: [frontmatter.schema.json](frontmatter.schema.json) (JSON Schema 2020-12). Editors with a YAML language server can use it with a modeline as the first line inside the front matter (a YAML comment, ignored by tuipresent):

```yaml
---
# yaml-language-server: $schema=path/to/docs/agent/frontmatter.schema.json
title: My talk
---
```

Whether front matter inside a `.md` file is validated depends on the editor; the modeline is harmless either way.

## 3. Slide splitting

Applied to the body (file minus front matter), line by line, with state `fence` (empty = outside a fence):

3.1. If outside a fence and the line opens a multi-line speaker note (§8.2), the note through its last line is copied into the current chunk unexamined. An unclosed note stops before the next `end_slide` directive line, which then splits as usual (§8.3).
3.2. If the line matches the fence regex `^\s*(```+|~~~+)\s*(.*)$`: outside a fence it opens one (marker = the backtick/tilde run); inside, it closes the fence iff its marker uses the same character, is at least as long as the opener, and the rest of the line is blank.
3.3. If, after 3.2, the line is outside a fence and is a directive (§5.1) named `end_slide` (any argument), the current chunk ends; the line is discarded.
3.4. Chunks are parsed (§4). A parsed slide is dropped iff it has an empty title, no visible content and no notes. Visible content = any Markdown, Exec or Image block, or a Columns block with at least one block in some column; an empty `column_layout` is not content. Consequences: chunks of only blank lines, only directives, or only empty layouts vanish (E1, E12). A chunk with only a title is kept (E12). A chunk with only speaker notes is kept as a blank slide (E16).
3.5. An unclosed fence swallows every later `end_slide` in the file (E5). An `end_slide` inside a closed fence is code (E4).

## 4. Slide parse algorithm

Per chunk, state: `step = 0`, `content = false`, `title = ""`, active layout `L = none`, current column `c = none`, pending markdown lines `P`. For each line, the FIRST matching rule applies:

1. **Fence open** (`^\s*(```+|~~~+)\s*(.*)$`, any indentation): consume through the closing fence (same rule as §3.2) or to the end of the chunk if unclosed. Sets `content`. If the info string has a whitespace-separated field exactly `+exec`, flush `P` and emit an Exec block (§10); otherwise append all consumed lines (fences included) to `P`. Directives, notes, images and titles inside are never interpreted.
2. **Multi-line speaker note opener** (§8.2): consume through the closing line; record the note. Does not set `content`, does not flush `P`.
3. **Directive** (§5.1): execute it (§5). Does not set `content`. Only `pause`, `column_layout`, `column`, `reset_layout` flush `P`.
4. **Image line** (§9.1): set `content`, flush `P`, emit an Image block.
5. **Setext title**: if `content` is false, `title` is empty, the line is non-blank, and the NEXT line matches `^=+\s*$` (no leading space), then `title = trim(line)`, skip both lines (§7).
6. **Otherwise**: append the line to `P`; a non-blank line sets `content`.

At the end: flush `P`; `Steps = step + 1`.

Flush: if `P` contains any non-blank character, emit a Markdown block with text = the raw lines of `P` joined by `\n` (leading/trailing blank lines kept), then clear `P`.

Emit (block `b`, with `b.step = step` at emission time):
- if `L` is active and `c` is set: append `b` to column `c` of `L`;
- else if `L` is active (no `column` yet): insert `b` in the top-level list immediately BEFORE `L` (so it renders above the layout, in order) (E9);
- else: append `b` to the top-level list.

Steps and fences are per slide. Columns cannot nest: every `column_layout` creates a top-level block.

## 5. Directives

5.1. Syntax. A directive is a whole line matching

```
^\s*<!--\s*([a-z_]+)\s*(?::\s*(.*?))?\s*-->\s*$
```

- Name: lowercase ASCII letters and `_` only. `Pause`, `my-note`, `column2` are not directives.
- Argument: optional, after `:`, trimmed. Arguments on directives that take none are ignored (`<!-- end_slide: x -->` still splits; `<!-- pause: 3 -->` is one pause) (E6).
- The comment MUST be alone on its line. A comment that shares a line with text, spans lines (except `speaker_note`, §8.2), or has an unknown name is not executed. It goes to glamour, which hides HTML comments; surrounding text on the same line may still render (output not specified — do not rely on it).
- Inside fences: never a directive (§3, §4.1).
- Unknown names are silently ignored, including unsupported presenterm directives (`incremental_lists`, `new_lines`, `newline`, `alignment`, `font_size`, `include`, `skip_slide`, `no_footer`). There is no directive that adds vertical space.

5.2. Directive table. "Flush" = §4 flush of pending markdown.

| Directive | Argument | Semantics | Repeats / edge cases |
|---|---|---|---|
| `end_slide` | ignored | Ends the chunk (§3.3). | Inside a fence: code. Ends an unclosed multi-line note and still splits (E17); any other directive ends it too. |
| `pause` | ignored | Flush; `step += 1`. | Each occurrence adds one step. A trailing pause adds an empty final step (E7). A pause before all content leaves step 1 with only the title (E8). Not per column (§6.3). |
| `column_layout` | `[w1, w2, ...]` | Flush; append a Columns block (`step` = current) with the parsed widths; make it the active layout `L`; `c = none`. | Brackets optional. Entries are split on `,`, trimmed, parsed as integers; non-integers and values <= 0 are dropped. No valid entry (or no argument) gives `[1]`, one full-width column. A second `column_layout` replaces `L` with a new layout placed after the first (E11, E12). |
| `column` | integer N (0-based) | Flush; if `L` is active and N parses as an integer: `c = clamp(N, 0, len(widths) - 1)`. | Out of range clamps (7 -> last, -3 -> 0). Non-integer: ignored, `c` unchanged. No active layout: ignored (E11, E12). |
| `reset_layout` | ignored | Flush; `L = none`, `c = none`. | Later content is top-level, rendered below the layout. Without an active layout: just a flush. |
| `jump_to_middle` | ignored | Slide is centered (§12). | Anywhere in the slide, any number of times. Does not count as content. |
| `transition` | kind | Slide's transition (§15.3). | Last occurrence wins. Value trimmed; matched case-insensitively; an unknown kind acts as `slide` and warns with the line number (E27); empty argument means "inherit". |
| `speaker_note` | text | Adds a note (§8). | Empty text is ignored. Several notes accumulate in source order. |
| `fade` | `on`/`off`/... | Slide's fade setting (§15.4). | Last occurrence wins. Empty argument means "inherit from front matter". |
| `title_style` | `rule`/`curly`/`frame` | Slide's title style (§7.7). | Last occurrence wins. Case-insensitive; an unknown value warns with the line number and acts as `rule`. Empty means "inherit". No effect without a title. |
| `title_size` | `1`/`2`/`3` | Slide's title size (§7.9). | Last occurrence wins. An unknown value (not an integer 1-3) warns with the line number and acts as `1`. Empty means "inherit". No effect without a title. |
| `subtitle` | inline markdown | Slide's subtitle line (§7.8). | Last occurrence wins. Trimmed; empty means none. No effect without a title. |

## 6. Blocks and reveal steps

6.1. A slide is `(title, blocks, Steps, centered, transition, fade, notes)`. Block kinds: Markdown (source text), Code (highlighted code, §10.5), Exec (code + language + id), Image (alt + path), Columns (widths + one block list per column). Every block has a step `s >= 0`, the number of `pause` directives before it in the slide plus the extra highlight groups of earlier code blocks (§10.5).
6.2. Reveal: the presenter moves through steps `n = 0 .. Steps-1`. At step `n`, a top-level block is visible iff `s <= n`. A Columns block is drawn iff its own `s <= n`; inside it, a child block is visible iff its `s <= n`.
6.3. The step counter is global to the slide. A pause in column 0 raises the step of everything after it in source order, including all later columns and content after `reset_layout`. To reveal columns side by side, put no pauses inside earlier columns; pause before `column_layout` or after `reset_layout` instead (E10).
6.4. The title is not a block: it is visible at every step (E8).
6.5. Invisible blocks consume no rows at the current step, EXCEPT inside a Columns block, whose height is always its fully revealed height (§11.6), and on centered slides, whose offsets come from the fully revealed slide (§12). Image sizes also come from the fully revealed slide (§9.3). On a non-centered slide, content below a hidden block is top-aligned, so earlier rows never move, but see §6.6.
6.6. Adjacent visible Markdown and Exec blocks are concatenated with a single `\n` and rendered as ONE markdown document (§13.1). So `A` / `<!-- pause -->` / `B` with no blank lines renders `A` at step 1 and the single paragraph `A B` at step 2. Surround `pause` (and any directive between paragraphs, including `speaker_note`) with blank lines to keep paragraphs separate (E7, E16).
6.7. `Steps = 1 + number of pause directives + sum over highlight blocks of (groups - 1)` in the chunk (outside fences and notes).

## 7. Titles

7.1. Only a setext H1 that appears before any content (§4 rule 5) is the slide title: a non-blank line followed by a line matching `^=+\s*$`. One `=` suffices. Directives and speaker notes before it are fine; text, lists, fences or image lines before it are not (E3, E8).
7.2. The title is the trimmed line. Inline markdown is parsed: `**bold**`, `*italic*`, `` `code` ``, `~~strike~~`, `==highlight==`, nested, with `\` escaping one of ``\ ` * ~ = _ [ ] # > - + ! . |``; a delimiter with no closing partner is literal (E23). Width (`titleWidth`) is measured on the text without the markers. It never wraps.
7.3. Title detection ignores column state: a setext title placed after `column_layout`/`column` but before any content still becomes the slide title (E13).
7.4. Not titles: ATX headings (`# x`) at any position (E2); a later setext H1 (glamour renders it as an in-body H1); setext H2 (`---` underline); an underline with leading spaces.
7.5. Title block, style `rule`: 4 rows: blank, title (bold gradient; code and highlight runs use the `code`/`code_background` and `highlight`/`background` colors), rule, blank. Rule width `cw` (non-centered) or `min(max(titleWidth, subtitleWidth) + 4, cw)` (centered). With a subtitle (§7.8) the title block has 5 rows.
7.6. In-body headings (rendered by glamour, styled by the theme): `#` bold accent line; `##` prefixed `▌ `; `###`-`######` bold lines.
7.7. Title styles: `rule` (above); `curly`: the title carries a curly underline (SGR `4:3`, color `58` = `accent`) and the rule row is blank, when `render.Curly` is set (the presenter sets it for kitty, Ghostty, WezTerm and iTerm2 outside tmux/screen; `-check` and `-dump` never do), else exactly `rule`: the block is 4 rows either way; `frame`: blank, `╭─╮` top (accent gradient), `│ title │` (and `│ subtitle │`), `╰─╯` bottom, blank: 5 rows (6 with a subtitle), box width `max(titleWidth, subtitleWidth) + 4`, which `-check` counts as the title width (E28).
7.8. Subtitle: `<!-- subtitle: text -->` adds one row under the title, in `foreground`, italic, with inline markdown as in §7.2 (E28).
7.9. Title size (`title_size`: `1`, `2`, `3`; slide directive beats front matter beats `1`): size 1 is the normal title. Sizes 2 and 3 replace the title row of any style with `TitleRows(size)` rows of block letters: size 2 = 3 rows (half blocks `▀▄█`, 1 cell per pixel), size 3 = 5 rows (`██` per pixel); extra rows over size 1: +2 and +4. The font is 5 pixels tall with 1 (size 2) or 2 (size 3) gap cells between characters, so the block is 6n-1 (size 2) or 12n-2 (size 3) cells wide for n characters. Glyphs: `A-Z` (lowercase is upper-cased), `0-9`, space (3 pixels), `. , : ! ? - _ / + = ' " & ( )`. Each character's cells use one color: the title gradient by character index, or `code`/`highlight` for those runs. Layout: rule style = blank, block rows, [subtitle], rule, blank (`2+R` rows plus 1 with a subtitle, R = rows); frame = blank, `╭`, block rows (and subtitle) inside the box, `╰`, blank, box width `max(blockWidth, subtitleWidth) + 4`; `curly` is the rule (no underline on block letters); the subtitle stays normal size; with `jump_to_middle` the block rows are centered as one block of width `max(blockWidth, subtitleWidth)`. Fallback: if any character has no glyph, or the block (plus 4 in a frame) is wider than `cw`, the slide gets the normal title (§7.5-7.7) and `-check`/`-dump` print `slide N: title_size S does not fit (REASON); using the normal title` (§19). `-check` counts the extra rows and the block width in the usual title-block and cols rules (§11.3, §14.2) (E32). Terminals: where `render.TextSizing` is set (the presenter does it for kitty outside tmux/screen, or `TUIPRESENT_TEXT_SIZING=on`; `off` disables) the presenter overlays the same rows with kitty OSC 66 scaled text (scale = R) once the slide has settled; the frame itself (`-dump`, transitions, fades, overview) is always block letters, so the overlay never changes layout.

## 8. Speaker notes

8.1. Single-line form: the directive `<!-- speaker_note: TEXT -->` (§5.1). TEXT is trimmed; empty is ignored.
8.2. Multi-line form: an opening line that contains no `-->` and matches `^\s*<!--\s*speaker_note\s*(?::(.*))?$` (the colon is optional). The note continues to the first later line containing `-->`; that line's text before `-->` is included, text after it is discarded. Body = the opener's remainder (trimmed) + following lines, with the smallest indentation among non-blank following lines removed (bytes; tabs count as one), trailing whitespace stripped, then the whole note trimmed. Empty notes are dropped.
8.3. Forgetting `-->`: the note stops before the next line that is a complete directive (§5.1 form, e.g. `<!-- pause -->` or `<!-- end_slide -->`; it is not consumed and takes effect normally), or at the end of the chunk, and loading warns `line N: speaker_note has no closing -->; ...` with the opener's line (E17). A line that merely contains `-->` after other text still closes the note.
8.4. Notes are never rendered on the slide, use no rows, do not set `content` and do not flush pending markdown (§6.6). Inside fences they are code.
8.5. A chunk with notes but no visible content or title is kept as a blank slide that carries the notes (§3.4, E16).
8.6. Display: `tuipresent -notes deck.md` in a second terminal (§15.6) shows the current slide's notes, joined by a blank line and rendered as markdown.

## 9. Images

9.1. Image line grammar (whole line):

```
^\s*!\[([^\]]*)\]\(\s*(?:<([^>]+)>|([^)\s]+))(?:\s+"[^"]*")?\s*\)\s*$
```

- `![ALT](PATH)`, `![ALT](<PATH WITH SPACES>)`, optional `"title"` (ignored). ALT cannot contain `]`.
- MUST be alone on its line. An image inside a paragraph, list item or table is left to glamour and prints as link text (E18). Inside a fence it is code.
- An image line counts as content, so a setext underline after it does not make a title.
9.2. Path: relative paths resolve against the deck file's directory; absolute paths are used as is. No URL fetching (a URL is a missing file). Decoders: PNG, JPEG, GIF (first frame).
9.3. Width attribute: if ALT matches exactly `^image:width:(\d+)%$`, max width = `max(avail * min(N, 100) / 100, 1)` cells and the label falls back to the file name. No other presenterm image attributes are supported (they are just alt text).
9.4. Missing/undecodable file, or `-images off`: a dim 3-row box `[image: LABEL]` centered in the available width, where LABEL = ALT, or the path's base name if ALT is empty or a width attribute. LABEL is truncated with `…` to `avail - 4`. If the box does not fit, it degrades to one truncated line. Never an error, never reported by `-check` (E18).
9.5. Row allocation (identical at every step, identical for kitty and half-blocks):
   - `B` = body rows (§11.3) of the slide.
   - `U` = rows of the fully revealed body with every decodable image counted as 0 rows (its separating blank rows and missing-image boxes do count).
   - `N` = image stack count: top-level images + for each Columns block, the maximum stack count of any one column.
   - Rows per image `R = max((B - U) / N, 3)`.
   - Inside a column: `R_col = R + (T - H_col) / N_col`, where `T` is the tallest column's height and `H_col` this column's height (both with images at 0 rows) and `N_col` = this column's stack count.
9.6. Cell box: a cell is 1 pixel wide and 2 tall. With image `iw x ih` pixels, max columns `C` (content/column width or §9.3 cap) and max rows `R`: `cols = C`, `rows = round(C * ih / (2 * iw))`; if `rows > R`: `rows = R`, `cols = round(2 * R * iw / ih)`. Both capped at 297, minimum 1. The image is centered: left pad `(avail - cols) / 2` (E19).
9.7. Spacing: an image gets one blank row before it unless it is the first thing in its body/column; markdown after an image gets one blank row. A Columns block directly after an image gets none.

## 10. Code blocks and `+exec`

10.1. Fence grammar: opener `^\s*(```+|~~~+)\s*INFO$`; closer: same character, length >= opener, nothing else on the line. Any indentation is accepted by the slide parser (unlike CommonMark), so an indented fence inside a list item is still a fence (and an `+exec` one is pulled out as its own block).
10.2. Exec block: INFO contains a whitespace-separated field exactly `+exec` (`+exec_replace`, `+exec:x` do not count). Language = first INFO field unless it starts with `+` (```` ```+exec ```` has language empty). Body = lines between the fences (to end of chunk if unclosed). Ids are assigned 1, 2, ... in source order across the deck (E20).
10.3. Rendering: the body renders as a fenced block ```` ```LANG ````. After a run, output (stdout+stderr combined, trailing newlines trimmed) renders as a ```` ```text ```` block directly below. Output is wrapped in a ``` fence, so output containing a ``` line breaks the slide.
10.4. Execution (never at load, never in `-dump`/`-check` without `-exec`):
    - Interpreter by language: `python`, `python3`, `py` -> `python3 -c BODY`; `sh` -> `sh -c BODY`; `bash` and empty -> `bash -c BODY`; anything else (`go`, `js`, `zsh`) is not run and its output is `[unsupported +exec language "LANG" (supported: bash, sh, python, python3, py)]`.
    - Working directory: the deck file's directory (the same base as image paths). Environment: that of the tuipresent process.
    - Timeout 15 s; the process is killed (`[signal: killed]` appended). Non-zero exit appends `\n[exit status N]`; a missing interpreter appends the exec error. After the process exits or is killed, a background child still holding the output pipe gets at most 0.5 s more (`WaitDelay`), then the pipe is closed and the error is appended.
    - Key `e`: runs every Exec block with `s <= current step` on the current slide (columns included), concurrently. Status row shows `running…` until the first result arrives. Re-pressing re-runs and replaces output. Reload (`r` or file watch) discards all outputs.
    - `-exec` with `-check` or `-dump`: runs every Exec block of every slide (all steps) sequentially before measuring/printing. It executes deck code: only with the user's approval.

10.5. Line highlights: INFO may contain one `{SPEC}` (first `{...}` with no braces inside; the rest of INFO, e.g. `rust {1-3|5} +exec` or `+exec rust{2}`, is read as if it were absent; a second `{...}` is dropped silently). SPEC is `|`-separated groups; a group is a `,`-separated list of items: a 1-based line `N`, a range `N-M` (`M >= N`), or `all` (case-insensitive); spaces around items are ignored. Lines count from the first line inside the fence. Each valid group is one reveal step: group `k` (0-based) shows at step `s + k` with only its lines at full color and every other line of the block dimmed (colors blended 62% toward the `code_background`); the last group stays. The block adds `groups - 1` steps, so everything after it moves that many steps later, like pauses (§6.3). A single group, `{2-4}`, is a static highlight with no extra step. Inside a column the shared step counter applies (§6.3).
    - Kinds: a non-exec fence with at least one valid group becomes a Code block (flushing the Markdown before it, rendered as its own code panel); an `+exec` fence stays an Exec block with the groups attached, so `e` still runs it. A fence with no valid group stays an ordinary Markdown fence.
    - Problems (warnings, §19; each prefixed with the fence's line number, the rest of the spec still applies): an item that is not `N`, `N-M` or `all` (`0`, `-2`, `2-`, `4-2`, `a`, numbers too large); a range running past the end (clipped); an item entirely past the end (skipped); an empty group (`1||3`, `1|`, `{}`) is dropped; a group left with no line is dropped; an unclosed `{` in INFO (the text is then ignored as an unknown attribute). An unclosed fence uses the body to the end of the chunk without the final empty line.
    - If the renderer cannot match highlight lines to rendered code lines (the code was reflowed), the block shows undimmed. Blank first/last lines of the body count as lines for numbering (E28).

## 11. Layout model

11.1. Frame: exactly `h` lines of exactly `w` cells (§16). Rows top to bottom: top margin, title block, body, padding, footer.
11.2. Content width and indent:
    - `cw = max(min(w - 8, 110), min(w - 2, 20), 1)`: `w - 8` capped at 110 for `w >= 28`; on narrower screens `min(w - 2, 20)`. `cw` never decreases as `w` grows.
    - Left indent `(w - cw) / 2`. All slide content is placed at this indent.
11.3. Rows:
    - Footer: 2 rows if `h >= 6`, else none. Available rows `A = h - footer`.
    - Top margin: 1 blank row, or 0 on centered slides.
    - Title block: 4 rows if the slide has a title (5 with a subtitle or `frame`, 6 with both; §7.5, §7.7); `title_size` 2 or 3 adds 2 or 4 rows (§7.9).
    - Body rows `B = A - margin - titleBlock`: titled `h - 7`, untitled `h - 3`, titled centered `h - 6`, untitled centered `h - 2` (for `h >= 6`).
    - Body rows beyond `B` are cut from the bottom (silently; `-check` reports them).
11.4. Wrap widths, for a region of width `W` (`cw` at top level, column width in a column):
    - Glamour word-wrap width `G = max(W - 2, 10)` (2 cells of slack because glamour lets punctuation overhang). Prose, list items, quotes and tables wrap at `G`.
    - Code no-wrap limit `K = G - 2` (glamour's 2-cell code margin). Lines up to `K` keep the margin. Longer lines lose it and the block may gain blank rows; lines over `G` wrap. `-check` reports `len - K` (E21).
    - Tabs are expanded to exactly 4 spaces before rendering and measuring (E25).
11.5. Column widths for `column_layout: [w1..wn]` in region width `W`:
    - Gap 4 cells. `avail = W - 4 * (n - 1)`. `colw_i = max(avail * w_i / sum(w), 1)`.
    - Columns are left-aligned at the region's left edge; flooring can leave the layout narrower than `W`.
11.6. Column heights: the layout's height is the tallest column's FULLY REVEALED height; each column is padded to it, at every step. Each column line is truncated to `colw_i` (clipped width is reported by `-check`).
11.7. Worked numbers (verified with `-check`):

| Quantity | 80x24 | 100x30 | 120x40 |
|---|---|---|---|
| `cw` / left indent | 72 / 4 | 92 / 4 | 110 / 5 |
| Body rows, titled | 17 | 23 | 33 |
| Body rows, untitled | 21 | 27 | 37 |
| Body rows, titled + `jump_to_middle` | 18 | 24 | 34 |
| Prose wrap `G` | 70 | 90 | 108 |
| Code limit `K` | 68 | 88 | 106 |
| Max title width | 72 | 92 | 110 |
| `[1, 1]` widths (code limit) | 34+34 (30) | 44+44 (40) | 53+53 (49) |
| `[2, 1]` widths | 45+22 | 58+29 | 70+35 |
| `[3, 2]` widths | 40+27 | 52+35 | 63+42 |
| `[1, 1, 1]` widths | 21x3 | 28x3 | 34x3 |
| Image row budget `R`, titled slide with only an image | 17 | 23 | 33 |

Per-column prose wrap is `colw - 2`, code limit `colw - 4`.

11.8. Row costs (what a construct consumes, fully revealed):
    - Paragraph / list / quote / table: its wrapped lines; blocks inside one markdown run are separated as glamour decides (observed: 1 blank row between paragraphs, lists and headings; 2 before a table).
    - Fenced code (and Exec): its lines, plus exactly 1 blank row between it and adjacent prose or other code in the same run (§13.2).
    - Columns: tallest column; 0 blank rows before; 1 blank row before following markdown.
    - Image: §9.5 rows; spacing §9.7. Missing image: 3 rows.
    - Exec output: output lines + 1 blank row, added only after a run.
    - Notes, directives: 0.
    - Leading/trailing blank lines of any run are trimmed; blank lines and `&nbsp;` cannot add vertical space at the edges.

## 12. Centering (`jump_to_middle`)

12.1. Vertical: top margin removed; `pad = max((A - (titleBlock + fullBody)) / 2, 0)` blank rows above the title block, where `fullBody` is the body height at the LAST step. Stable across steps.
12.2. Title block lines are each centered: `(cw - width) / 2`; the rule shrinks to `min(titleWidth + 4, cw)`.
12.3. Body horizontal mode is "statement" iff every block is Markdown AND no source line of any block matches `^\s*([-*+] |\d+[.)] |\||```|~~~|>)` (list item, table row, fence, quote). Hidden steps count. Any Exec, Image or Columns block disables it. Indented (4-space) code is not detected and counts as prose.
    - Statement mode: each rendered body line is centered on its own width (E14).
    - Block mode: every body line is shifted by `max((cw - M) / 2, 0)`, where `M` is the widest line of the fully revealed body, so the body stays left-aligned internally and does not move between steps (E15).
12.4. The cover slide is always centered (§2.6).

## 13. Rendering behaviors that affect authoring

13.1. Markdown runs: consecutive visible Markdown/Exec blocks are joined with `\n` and rendered together by glamour (§6.6). Glamour's leading and trailing blank lines are trimmed.
13.2. Code segments: within a run, top-level fences (indent 0-3) are split out and rendered separately from prose; segments are joined with exactly one blank row. Prose list rewrapping never touches code lines.
13.3. Hanging indent: list items that wrap are re-wrapped at `G` so continuation lines align under the item text after the marker (`• `, `1. `, `[ ] `, `[x] `) (E24).
13.4. Truncation: every frame line is cut to `w` cells (no ellipsis); column lines to their column width. Titles and wide unbreakable text spill into the right margin before being cut.
13.5. HTML comments that are not directives or notes are hidden by glamour. Raw HTML is otherwise glamour-defined.
13.6. The footer: row 1 progress bar, `floor(w * slideNumber / total)` cells of `━` then `─`; row 2 ` TITLE  ·  AUTHOR` (parts present) left, `N / TOTAL ` right. The runtime status line (`running…`, `reloaded`, warnings, load errors), in the `accent` color, temporarily replaces the last row.
13.7. Text marks, in prose outside code spans and fences (E29): `==x==` highlight (text in `background` on a `highlight` background); `[x]{.faint}` faint (SGR 2); `++x++` and `[x]{.underline}` underline (SGR 4); `[x]{.curly}`, `[x]{.dotted}`, `[x]{.dashed}`, `[x]{.double}` underline variants (SGR `4:3`/`4:4`/`4:5`/`4:2`, color `58` = `accent`; plain SGR 4 unless `render.Curly`, §7.7). `==` needs a non-space after the opening and before the closing delimiter, so `a == b` is untouched; an unknown `{.class}` is left as written. `~~x~~` strikethrough is standard markdown. Marks add no rows or columns, apply inside list items, tables, quotes and columns, and are closed and reopened across wrapped lines.
13.8. Callouts (E30): a blockquote whose first line is `> [!KIND]`, KIND one of NOTE, TIP, IMPORTANT, WARNING, CAUTION (case-insensitive), nothing else on the line, outside fences. Its lines are the consecutive `>` lines that follow (the `>` and one space stripped; a lazy continuation without `>` ends it). Rendered as markdown at width `W - 4` inside a rounded box in the palette color `note`/`tip`/`important`/`warning`/`caution`: top border `╭─ Kind ───╮` (Kind in bold, title case), `│ ` + line + ` │` rows, bottom `╰───╯`. Box width = widest inside line + 4 (at least label + 6), at most `W`. Height = body lines + 2. Other quotes, and unknown KINDs, are ordinary glamour quotes. A callout is a quote for the statement-slide test (§12.3).

## 14. What `-check` measures

`tuipresent -check -w W -h H deck.md` (defaults 100x30). For each slide, at its LAST step:

14.1. Rows: `margin + titleBlock + fullBody - A`. Images are sized first (§9.5), so an image never overflows unless its 3-row minimum does not fit.
14.2. Cols: the maximum of: title width (or subtitle width, or the frame width, §7.7, or the block-letter width, §7.9) - `cw`; body line width - `cw`; widest clipped column line - its column width; fenced/Exec code line (tabs = 4 spaces) - `K` of its region, per column; Exec output - `K` if outputs exist.
14.3. Output: `deck.md: N slides fit WxH` (exit 0), or one line per bad slide `deck.md: slide N overflows WxH by R rows, C cols` (exit 1). Load errors (bad YAML, no slides) exit 1 on stderr.
14.4. It cannot see: Exec output unless `-exec` was given; intermediate steps (reveal order, merged paragraphs, `A B` reflow); dropped slides, swallowed `end_slide`, ignored or misspelled directives; missing images (a box fits); unmatched markdown delimiters; readability. Always read `-dump` too.
14.5. Code lines are found with the parser's fence rules (§10.1), so the lines of a fence nested in a longer one (```` ````markdown ```` around ```` ``` ````) are measured as code too.
14.6. Warnings (§19) go to stderr before the report and never change the exit status.

## 15. Runtime

15.1. Flags: `-w`, `-h` (size for `-dump`/`-check`, default 100x30); `-check`; `-dump` (prints `=== slide N step k/n ===` then the frame for every step); `-exec` (with `-check`/`-dump`); `-slide N` (start slide, 1-based, clamped); `-transition KIND` (replaces front matter `transition`, survives reload, not per-slide directives; unknown kinds warn); `-theme NAME|FILE.yaml` (replaces the theme name, path and inline overrides, keeps `background`; survives reload; §18.6); `-background terminal|paint` (replaces the painting mode; any other value is a usage error, exit 2); `-themes` (list built-in themes with `light`/`dark` and a swatch, then exit; no deck needed); `-fade=false`; `-progress=false` (turns off §15.8 progress, overriding the deck); `-links auto|on|off` (OSC 8 hyperlinks, §15.9; default `auto`; any other value is a usage error, exit 2); `-watch=false`; `-mouse=false` (default on: report mouse clicks and wheel); `-notes`; `-images auto|kitty|blocks|off` (`-dump`/`-check` always use blocks unless off). Exactly one deck path (except with `-themes`); otherwise usage, exit 2.
15.2. Keys: next `right l space n pgdown enter j down`; back `left h p pgup backspace k up`; first slide `g home`; last slide `G end`; overview `o` (§15.7); jump by number: digits then `enter` (esc cancels, backspace erases, leading 0 ignored, clamped to 1..N, shown on the status line); run exec `e`; reload `r`; quit `q ctrl+c esc`. Next advances a step, then the slide; back retreats a step (instantly), then to the previous slide's LAST step. `g`/`G` land on step 1.
15.3. Transitions (380 ms, ease-out): `slide`, `wipe`, `cover`, `dissolve`, `none`. A slide's effective transition is its `transition` directive, else `-transition`, else front matter `transition`, else `slide`. Moving forward to slide X plays X's effective transition. Moving backward from slide X (by key or `g`) plays X's effective transition reversed: it undoes the transition that entered X; the destination's directive is not used. Values case-insensitive; unknown = `slide` with a warning (§19). Never used between steps.
15.4. Pause fade (280 ms): a forward step fades in the changed cells. Disabled by `-fade=false` (absolute), else by the slide's `fade` directive if non-empty, else by front matter `fade`; disabling values `false off no none`. Backward steps are instant. No effect on `-dump`/`-check`.
15.5. Live reload: the deck file, and its custom theme file if any, are polled every 400 ms; a changed identity, mtime or size reloads, keeping slide and step (clamped). A missing file is not a change. A parse error or invalid theme keeps the old deck and shows the error in the status row. Warnings of the new deck show in the status row for 6 s; otherwise it shows `reloaded`. Reload clears Exec outputs.
15.6. Notes view: `tuipresent -notes deck.md` follows a presenter of the same deck (state file in the temp dir keyed by the deck's absolute path; polled every 200 ms). Shows slide title (`slide N (untitled)` if none), `slide i/n · step k/m · H:MM:SS`, the notes as markdown (`(no notes for this slide)` if none, truncated with `…`), and `Next:` title. It reloads the deck when its mtime changes.
15.7. Overview and mouse: `o` replaces the slide with a grid of cards (slide number plus title; the cover shows the deck title; untitled slides show a dash). Columns = `min(W / 24, slides)` (4 at 100 wide, 3 at 80), cards 5 rows high between a header row and a hint row, one page of rows at a time; a screen under 7 rows gets one-line cards. Keys in the overview: arrows/`hjkl` move, `pgup`/`pgdown` a page, `g`/`G` first/last, `enter`/`space` jump to the selected slide's first step, `o`/`esc` close, `q`/`ctrl+c` quit. Mouse (default on, `-mouse=false` off; left button and wheel only): click the left third (x*3 < W) back, the right third (x*3 >= 2W) forward, the middle nothing; wheel up back, down forward; in the overview a click on a card jumps to it and the wheel moves one row. Mouse reporting is enabled and disabled with the program.
15.8. Progress: when stdout is a terminal whose `TERM` is set and not `dumb`/`linux`, `TERM_PROGRAM` is not `Apple_Terminal`, `-progress` is not false and front matter `progress` is not a disabling value, the presenter writes `ESC ] 9 ; 4 ; 1 ; P ST` with `P = (slide+1)*100/slides` (1-based slide, integer division) whenever the current slide, or the slide count after a reload, changes (and once at start), and `ESC ] 9 ; 4 ; 0 ; 0 ST` when progress is switched off by a reload and on exit (normal quit, error, SIGHUP, SIGTERM). Never with `-dump`/`-check`.
15.9. Links: an inline link `[text](dest)` outside code whose `dest` is an absolute URL (has a scheme) renders as `text` wrapped in OSC 8 (`ESC ] 8 ; id=... ; URL ST ... ESC ] 8 ; ; ST`) when links are on (`-links on`, or `auto` with the §15.8 terminal test); the URL is not printed and takes no width, and link text wrapped over lines is re-wrapped in the sequence per line. A relative `dest` shows its text only. When links are off (always for `-dump`/`-check`) glamour's form applies: text, a space, then the URL in the `dim` color, with no OSC sequences. Autolinks and bare URLs print the URL (hyperlinked when on). Transitions and fades keep sequences balanced per line and do not change geometry.

## 16. Invariants

16.1. Every frame is exactly `h` lines, each exactly `w` cells (padded or truncated), at every step and during transitions.
16.2. Layout of a slide is computed from its fully revealed state: column heights, image sizes, centering offsets. Steps only reveal; they never resize.
16.3. Loading never fails except for front matter YAML errors, an empty deck, and an invalid custom theme (§19). A few recoverable mistakes warn (§19); every other mistake degrades silently.
16.4. Rendering never executes code; only `e` or `-exec` do.

## 17. Quick grammar

```ebnf
deck          = [ front_matter ] chunk { end_slide_line chunk } ;
front_matter  = "---" NL yaml_line { yaml_line } "---" ( NL | EOF ) ;
chunk         = { line } ;                         (* split outside fences, §3 *)
line          = fence_block | note_block | directive_line | image_line
              | title_pair | md_line ;            (* first match wins, §4 *)
directive_line= ws "<!--" ws name [ ws ":" ws arg ] ws "-->" ws NL ;
name          = ( "a".."z" | "_" ) { "a".."z" | "_" } ;
note_block    = ws "<!--" ws "speaker_note" ws [ ":" text ] NL   (* no "-->" on this line *)
                { line_without_close } text "-->" text NL ;
fence_block   = ws marker ws info NL      (* info may hold {1-3|5}, §10.5 *) { any_line } [ ws marker' ws NL ] ;
                (* marker = 3+ ` or 3+ ~ ; marker' same char, len >= marker *)
image_line    = ws "![" alt "](" ws ( "<" path_sp ">" | path ) [ ws1 '"' title '"' ] ws ")" ws NL ;
title_pair    = nonblank_line NL "=" { "=" } trailing_ws NL ;   (* only before content *)
widths        = [ "[" ] int { "," int } [ "]" ] ;  (* ints <= 0 or invalid dropped *)
```

## 18. Themes

Source: `internal/render/theme.go`. A theme is a palette of 13 colors plus a chroma syntax style and a painting mode. Layout never depends on the theme, so `-check` results are identical in every theme.

18.1. Built-in names. Matching lowercases and trims the name, then requires an exact match against a canonical name or an alias; there is no substring or prefix matching.

| Name | Kind | Chroma style |
|---|---|---|
| `dark` (default) | dark | onedark |
| `light` | light | github |
| `tokyonight-storm` | dark | tokyonight-storm |
| `tokyonight-night` | dark | tokyonight-night |
| `tokyonight-moon` | dark | tokyonight-moon |
| `tokyonight-day` | light | tokyonight-day |
| `catppuccin-mocha` | dark | catppuccin-mocha |
| `catppuccin-macchiato` | dark | catppuccin-macchiato |
| `catppuccin-frappe` | dark | catppuccin-frappe |
| `catppuccin-latte` | light | catppuccin-latte |
| `gruvbox-dark` | dark | gruvbox |
| `gruvbox-light` | light | gruvbox-light |
| `dracula` | dark | dracula |
| `nord` | dark | nord |
| `rose-pine` | dark | rose-pine |
| `rose-pine-dawn` | light | rose-pine-dawn |
| `solarized-dark` | dark | solarized-dark |
| `solarized-light` | light | solarized-light |
| `everforest-dark` | dark | evergarden |
| `kanagawa` | dark | kanagawa (registered by tuipresent) |

"Kind" is light iff the background's CIE L* exceeds 0.6. `tuipresent -themes` prints the same list with swatches.

18.2. Aliases: `tokyo-night`, `tokyonight`, `tokyo-night-storm` -> `tokyonight-storm`; `tokyo-night-night` -> `tokyonight-night`; `tokyo-night-moon` -> `tokyonight-moon`; `tokyo-night-day`, `tokyo-night-light` -> `tokyonight-day`; `catppuccin` -> `catppuccin-mocha`; `gruvbox` -> `gruvbox-dark`; `rosepine`, `rose-pine-main` -> `rose-pine`; `rosepine-dawn` -> `rose-pine-dawn`; `solarized` -> `solarized-dark`; `everforest` -> `everforest-dark`; `kanagawa-wave` -> `kanagawa`; `terminal-dark`, `default` -> `dark`; `terminal-light` -> `light`. Unknown name: render in `dark` and warn `unknown theme "X", using dark (did you mean Y?); tuipresent -themes lists them` (the suggestion is a built-in with the typed name as a 3+ character prefix, else the closest within edit distance 3, else omitted).

18.3. Palette keys. Values are `#rgb` or `#rrggbb` (normalized to lowercase `#rrggbb`); in YAML they MUST be quoted, since a bare `#` starts a comment and leaves the value empty, which is a load error. A derived key that is unset follows its source after all overrides are applied, so overriding `accent` also recolors `heading`, `link` and `bullet` unless they are set.

| Key | Colors | Unset means |
|---|---|---|
| `background` | Painted background (§18.5); fade base when painting or when the terminal background is unknown | (core) |
| `foreground` | Body text; painted foreground | (core) |
| `dim` | Footer, cover byline, missing-image boxes, progress bar remainder | (core) |
| `accent` | Title/rule gradient start, progress bar, status line | (core) |
| `accent2` | Title/rule gradient end | (core) |
| `heading` | `#` headings | `accent` |
| `subheading` | `##`-`######` headings (`##` has a `▌ ` prefix) | `accent2` |
| `link` | Links and image link text | `accent` |
| `bullet` | List bullets, numbers and task boxes | `accent` |
| `code` | Inline code text | `accent2` |
| `code_background` | Inline code background (padded one space each side) and the fenced-code panel (longest line + 2 cells) | `background` blended 8% toward `foreground` |
| `quote` | Blockquote text and `│` bar (italic) | `dim` |
| `border` | Table borders, horizontal rules | `dim` |
| `highlight` | Background of `==x==` text (§13.7) | `accent` |
| `note` | `[!NOTE]` callout box and label (§13.8) | `#7aa2f7`; `#2e7de9` when the `background` is light |
| `tip` | `[!TIP]` callouts | `#9ece6a`; `#3f7d20` light |
| `important` | `[!IMPORTANT]` callouts | `#bb9af7`; `#8839ef` light |
| `warning` | `[!WARNING]` callouts | `#e0af68`; `#a66a00` light |
| `caution` | `[!CAUTION]` callouts | `#f7768e`; `#c4283c` light |

`code_style` (not a color): chroma style for fenced code; unknown names warn and keep the base theme's.

18.4. Theme files (`theme: {path: FILE}` or `-theme FILE.yaml`): a YAML mapping. `extends: NAME` picks the base built-in (default `dark`; an unknown name is a load error). Other keys: the palette keys of §18.3 (here `background` IS a color) and `code_style`. Unknown keys and non-scalar values warn and are ignored. Errors (load fails, exit 1; during reload the old deck stays): unreadable file, invalid YAML, a file that is not a mapping, an empty or malformed color, unknown `extends`. The resolved theme's name is the file's base name without extension. Front-matter palette keys and `code_style` apply on top of the file. The file is watched like the deck.

18.5. Painting (`background: paint` or `-background paint`): every cell of every frame, the status line and transition frames are drawn with the palette's `foreground` and `background` as the default colors, re-applied after every SGR reset (0, 39, 49); explicit colors in the content are kept. Geometry is unchanged (only escape sequences are added). The pause fade then starts from the theme `background`. With `terminal` (default), unstyled cells use the terminal's own colors and the fade starts from the terminal's background (queried with OSC 11 at startup; the theme `background` until it answers).

18.6. Precedence: `-theme` replaces the front-matter `name`, `path` and inline overrides (a `.yaml`/`.yml` argument is a file path relative to the current directory, anything else a name) and keeps the front-matter `background`; `-background` replaces the painting mode. Both reapply after every reload. The `-notes` view renders notes in the same theme.

## 19. Warnings and errors

19.1. Warnings: the deck loads with a fallback. Printed to stderr as `PATH: warning: TEXT` before `-check`/`-dump` output (exit status unchanged); while presenting, joined with `; ` on the status row for 6 s, at start and after each reload. The `-notes` view lists them (`warning: TEXT`, above its footer, kept up while they apply, recomputed on reload; extra ones collapse to `… N more warning(s)`).

| Source | TEXT (prefix) | Fallback |
|---|---|---|
| Front matter `transition` | `front matter: unknown transition "X", using slide (valid: ...)` | `slide` |
| `transition` directive | `line N: unknown transition "X", ...` | `slide` |
| `-transition` flag | `-transition: unknown transition "X", ...` | `slide` |
| Code fence line highlights (§10.5) | `line N: bad highlight "X": ...`, `highlight "X" is past the end of the K-line code block`, `highlight "X" runs past the end ...; clipped ...`, `empty highlight group G is ignored`, `code fence has an unclosed { ...` | Item skipped / clipped / group dropped / ignored |
| Front matter `title_style` | `front matter: unknown title_style "X" (valid: rule, curly, frame); using rule` | `rule` |
| `title_style` directive | `line N: unknown title_style "X", ...` | `rule` |
| Front matter `title_size` | `front matter: unknown title_size "X" (valid: 1 to 3); using 1` | `1` |
| `title_size` directive | `line N: unknown title_size "X", ...` | `1` |
| `title_size` 2 or 3 that cannot be drawn (§7.9) | `slide N: title_size S does not fit (no block letter for "X" \| it needs C columns but the slide is W wide); using the normal title`; printed by `-check`/`-dump` for their `-w`, not while presenting | Normal title |
| Multi-line `speaker_note` without `-->` | `line N: speaker_note has no closing -->; ...` | Note ends before the next `end_slide` or at end of file |
| Theme name | `unknown theme "X", using dark ...` | `dark` |
| Theme mapping / file keys | unknown key, `extends` in front matter, nested value, unknown `code_style`, bad `background`, both `name` and `path` | Ignored / base style / `terminal` / path |

`N` is the 1-based line number in the deck file (front matter lines included) of the offending directive, note opener or code fence.

19.2. Errors (exit 1 at startup with the message on stderr; during a reload the old deck stays and the message shows on the status row): invalid front-matter YAML (`front matter: yaml: ...`), no slides (`deck has no slides`), invalid custom theme (§18.4, also an empty or malformed inline color). Usage errors exit 2.

19.3. Silent: everything else, including misspelled or unknown directives, an out-of-range `column`, a missing image, an unclosed fence, a title after content. Read `-dump`.
