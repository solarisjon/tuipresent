# tuipresent deck language specification

Normative reference for the markdown dialect tuipresent reads. Written for LLM agents. It describes what the code does today (`internal/deck/parse.go`, `internal/render/render.go`, `cmd/tuipresent/`), including quirks. Where this file and other docs disagree, the code wins; report the drift.

Conventions: "MUST" marks a rule the parser enforces (break it and the construct is not recognized). "line" means one `\n`-terminated source line. `cw` is content width, `w`/`h` the screen size in cells. Integer division truncates. Every claim marked (E*n*) is demonstrated with real output in [EXAMPLES.md](EXAMPLES.md) case *n*.

Contents: 1 File structure · 2 Front matter · 3 Slide splitting · 4 Slide parse algorithm · 5 Directives · 6 Blocks and reveal steps · 7 Titles · 8 Speaker notes · 9 Images · 10 Code and `+exec` · 11 Layout model · 12 Centering · 13 Rendering behaviors · 14 `-check` · 15 Runtime · 16 Invariants · 17 Quick grammar

---

## 1. File structure

```
deck      = [ front-matter ] body
body      = chunk { end-slide-line chunk }
```

1.1. A deck is one UTF-8 file. `\r\n` is replaced by `\n` before anything else. No BOM handling: a BOM before `---` disables front matter.
1.2. The body is split into chunks at `end_slide` directive lines outside fences (§3). Each chunk is parsed into at most one slide (§4).
1.3. A chunk that yields no blocks and no title is dropped (§3.4). If no slide remains (cover included), loading fails: `deck has no slides`, exit 1.
1.4. Slide numbering (`-slide N`, footer `N / total`, `-dump` headers) is 1-based and counts the cover slide.

## 2. Front matter

2.1. Recognized only if the file starts with exactly `---\n` (byte 0). The block ends at the first `\n---\n` after that, or at a final `\n---` at end of file. The delimiters MUST be exactly `---` (no trailing spaces).
2.2. If no closing delimiter is found, there is no front matter: the whole file, including the `---` and YAML lines, is body (renders as a rule and text).
2.3. Quirk: an empty block `---\n---\n` is not recognized (the closing search starts after the opening newline, so at least one line must sit between the delimiters). Both lines become body (E26). Put at least one key in front matter or omit it.
2.4. The block is parsed with YAML (gopkg.in/yaml.v3, non-strict). Invalid YAML, or a sequence/mapping where text is expected, is a fatal load error printed as `front matter: yaml: ...`, exit 1. Unknown keys are ignored.

| Key | Type | Default | Effect |
|---|---|---|---|
| `title` | text | empty | Non-empty: insert a cover slide as slide 1 (§2.6); footer left text. |
| `sub_title` | text | empty | Italic line on the cover. |
| `author` | text | empty | Cover byline; footer suffix `  ·  author`. |
| `date` | text | empty | Cover byline after author: `author  ·  date`. Free text. |
| `theme` | text, or mapping with `name` | empty | Color theme (§2.5). |
| `transition` | text | `slide` | Default transition into every slide (§15.3). |
| `fade` | text or bool | on | `false`/`off`/`no`/`none` (case-insensitive, trimmed) disable pause fades deck-wide; anything else leaves them on. |

"text" = any YAML scalar, used as its string form (`title: 123` gives `123`; `date: 2026-12-08` stays `2026-12-08`).

2.5. Theme. <!-- TODO(themes): fill from theme system -->
Accepted shapes, both equivalent:

```yaml
theme: NAME
```

```yaml
theme:
  name: NAME
```

A scalar sets the name; a mapping is decoded for `name` and other keys are currently ignored. A sequence is a load error. Theme names and how they map to colors are defined by the theme system, which is being replaced; this section will be completed from it.

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

3.1. If outside a fence and the line opens a multi-line speaker note (§8.2), the note through its closing line is copied into the current chunk unexamined. An unclosed note runs to the end of the body.
3.2. If the line matches the fence regex `^\s*(```+|~~~+)\s*(.*)$`: outside a fence it opens one (marker = the backtick/tilde run); inside, it closes the fence iff its marker uses the same character, is at least as long as the opener, and the rest of the line is blank.
3.3. If, after 3.2, the line is outside a fence and is a directive (§5.1) named `end_slide` (any argument), the current chunk ends; the line is discarded.
3.4. Chunks are parsed (§4). A parsed slide with zero blocks and an empty title is dropped. Consequences: chunks of only blank lines, only directives, or only speaker notes vanish (their notes are lost) (E1, E12, E16). A chunk with only a title, or with an empty `column_layout`, is kept (E12).
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
| `end_slide` | ignored | Ends the chunk (§3.3). | Inside a fence: code. Inside an unclosed multi-line note: consumed as the note's closing line (E17). |
| `pause` | ignored | Flush; `step += 1`. | Each occurrence adds one step. A trailing pause adds an empty final step (E7). A pause before all content leaves step 1 with only the title (E8). Not per column (§6.3). |
| `column_layout` | `[w1, w2, ...]` | Flush; append a Columns block (`step` = current) with the parsed widths; make it the active layout `L`; `c = none`. | Brackets optional. Entries are split on `,`, trimmed, parsed as integers; non-integers and values <= 0 are dropped. No valid entry (or no argument) gives `[1]`, one full-width column. A second `column_layout` replaces `L` with a new layout placed after the first (E11, E12). |
| `column` | integer N (0-based) | Flush; if `L` is active and N parses as an integer: `c = clamp(N, 0, len(widths) - 1)`. | Out of range clamps (7 -> last, -3 -> 0). Non-integer: ignored, `c` unchanged. No active layout: ignored (E11, E12). |
| `reset_layout` | ignored | Flush; `L = none`, `c = none`. | Later content is top-level, rendered below the layout. Without an active layout: just a flush. |
| `jump_to_middle` | ignored | Slide is centered (§12). | Anywhere in the slide, any number of times. Does not count as content. |
| `transition` | kind | Slide's transition (§15.3). | Last occurrence wins. Value trimmed; matched case-insensitively at runtime; unknown kinds act as `slide`; empty argument means "inherit". |
| `speaker_note` | text | Adds a note (§8). | Empty text is ignored. Several notes accumulate in source order. |
| `fade` | `on`/`off`/... | Slide's fade setting (§15.4). | Last occurrence wins. Empty argument means "inherit from front matter". |

## 6. Blocks and reveal steps

6.1. A slide is `(title, blocks, Steps, centered, transition, fade, notes)`. Block kinds: Markdown (source text), Exec (code + language + id), Image (alt + path), Columns (widths + one block list per column). Every block has a step `s >= 0`, the number of `pause` directives before it in the slide.
6.2. Reveal: the presenter moves through steps `n = 0 .. Steps-1`. At step `n`, a top-level block is visible iff `s <= n`. A Columns block is drawn iff its own `s <= n`; inside it, a child block is visible iff its `s <= n`.
6.3. The step counter is global to the slide. A pause in column 0 raises the step of everything after it in source order, including all later columns and content after `reset_layout`. To reveal columns side by side, put no pauses inside earlier columns; pause before `column_layout` or after `reset_layout` instead (E10).
6.4. The title is not a block: it is visible at every step (E8).
6.5. Invisible blocks consume no rows at the current step, EXCEPT inside a Columns block, whose height is always its fully revealed height (§11.6), and on centered slides, whose offsets come from the fully revealed slide (§12). Image sizes also come from the fully revealed slide (§9.3). On a non-centered slide, content below a hidden block is top-aligned, so earlier rows never move, but see §6.6.
6.6. Adjacent visible Markdown and Exec blocks are concatenated with a single `\n` and rendered as ONE markdown document (§13.1). So `A` / `<!-- pause -->` / `B` with no blank lines renders `A` at step 1 and the single paragraph `A B` at step 2. Surround `pause` (and any directive between paragraphs, including `speaker_note`) with blank lines to keep paragraphs separate (E7, E16).
6.7. `Steps = 1 + number of pause directives` in the chunk (outside fences and notes).

## 7. Titles

7.1. Only a setext H1 that appears before any content (§4 rule 5) is the slide title: a non-blank line followed by a line matching `^=+\s*$`. One `=` suffices. Directives and speaker notes before it are fine; text, lists, fences or image lines before it are not (E3, E8).
7.2. The title is the trimmed line, used literally: no inline markdown (`**x**` shows the asterisks) (E23). It never wraps.
7.3. Title detection ignores column state: a setext title placed after `column_layout`/`column` but before any content still becomes the slide title (E13).
7.4. Not titles: ATX headings (`# x`) at any position (E2); a later setext H1 (glamour renders it as an in-body H1); setext H2 (`---` underline); an underline with leading spaces.
7.5. Title block: 4 rows: blank, title (bold gradient), rule, blank. Rule width `cw` (non-centered) or `min(titleWidth + 4, cw)` (centered).
7.6. In-body headings (rendered by glamour, styled by the theme): `#` bold accent line; `##` prefixed `▌ `; `###`-`######` bold lines.

## 8. Speaker notes

8.1. Single-line form: the directive `<!-- speaker_note: TEXT -->` (§5.1). TEXT is trimmed; empty is ignored.
8.2. Multi-line form: an opening line that contains no `-->` and matches `^\s*<!--\s*speaker_note\s*(?::(.*))?$` (the colon is optional). The note continues to the first later line containing `-->`; that line's text before `-->` is included, text after it is discarded. Body = the opener's remainder (trimmed) + following lines, with the smallest indentation among non-blank following lines removed (bytes; tabs count as one), trailing whitespace stripped, then the whole note trimmed. Empty notes are dropped.
8.3. Forgetting `-->`: the note runs to the next line containing `-->` anywhere, typically the next directive, which is swallowed (an `end_slide` there merges two slides) (E17). With no later `-->`, it runs to the end of the file.
8.4. Notes are never rendered on the slide, use no rows, do not set `content` and do not flush pending markdown (§6.6). Inside fences they are code.
8.5. A chunk with notes but no blocks or title is dropped with its notes (§3.4).
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
    - Interpreter by language: `python`, `python3`, `py` -> `python3 -c BODY`; `sh` -> `sh -c BODY`; anything else, including empty and unknown (`go`, `js`) -> `bash -c BODY`.
    - Working directory and environment: those of the tuipresent process.
    - Timeout 15 s; the process is killed (`[signal: killed]` appended). Non-zero exit appends `\n[exit status N]`; a missing interpreter appends the exec error. (From code reading, not tested: no `WaitDelay` is set, so a background child that keeps the output pipe open can delay completion past the timeout.)
    - Key `e`: runs every Exec block with `s <= current step` on the current slide (columns included), concurrently. Status row shows `running…` until the first result arrives. Re-pressing re-runs and replaces output. Reload (`r` or file watch) discards all outputs.
    - `-exec` with `-check` or `-dump`: runs every Exec block of every slide (all steps) sequentially before measuring/printing. It executes deck code: only with the user's approval.

## 11. Layout model

11.1. Frame: exactly `h` lines of exactly `w` cells (§16). Rows top to bottom: top margin, title block, body, padding, footer.
11.2. Content width and indent:
    - `cw = min(w - 8, 110)` if that is >= 20, else `max(w - 2, 1)`. (Quirk: `w = 27` gives `cw = 25`, `w = 28` gives `cw = 20`.)
    - Left indent `(w - cw) / 2`. All slide content is placed at this indent.
11.3. Rows:
    - Footer: 2 rows if `h >= 6`, else none. Available rows `A = h - footer`.
    - Top margin: 1 blank row, or 0 on centered slides.
    - Title block: 4 rows if the slide has a title.
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
13.6. The footer: row 1 progress bar, `floor(w * slideNumber / total)` cells of `━` then `─`; row 2 ` TITLE  ·  AUTHOR` (parts present) left, `N / TOTAL ` right. The runtime status line (`running…`, `reloaded`, parse errors) temporarily replaces the last row.

## 14. What `-check` measures

`tuipresent -check -w W -h H deck.md` (defaults 100x30). For each slide, at its LAST step:

14.1. Rows: `margin + titleBlock + fullBody - A`. Images are sized first (§9.5), so an image never overflows unless its 3-row minimum does not fit.
14.2. Cols: the maximum of: title width - `cw`; body line width - `cw`; widest clipped column line - its column width; fenced/Exec code line (tabs = 4 spaces) - `K` of its region, per column; Exec output - `K` if outputs exist.
14.3. Output: `deck.md: N slides fit WxH` (exit 0), or one line per bad slide `deck.md: slide N overflows WxH by R rows, C cols` (exit 1). Load errors (bad YAML, no slides) exit 1 on stderr.
14.4. It cannot see: Exec output unless `-exec` was given; intermediate steps (reveal order, merged paragraphs, `A B` reflow); dropped slides, swallowed `end_slide`, ignored or misspelled directives; missing images (a box fits); literal markdown in titles; readability. Always read `-dump` too.
14.5. Known measuring gap: the code-line measurement uses a simplified fence scanner (any line starting with ``` closes, including ```` ```lang ````, and 4-backtick outer fences are closed by an inner ```), so code inside nested fences may be mis-measured (from code reading, `codeOverflow`/`fencedCode` in render.go; not demonstrated in EXAMPLES.md).

## 15. Runtime

15.1. Flags: `-w`, `-h` (size for `-dump`/`-check`, default 100x30); `-check`; `-dump` (prints `=== slide N step k/n ===` then the frame for every step); `-exec` (with `-check`/`-dump`); `-slide N` (start slide, 1-based, clamped); `-transition KIND` (replaces front matter `transition`, survives reload, not per-slide directives); `-fade=false`; `-watch=false`; `-notes`; `-images auto|kitty|blocks|off` (`-dump`/`-check` always use blocks unless off). Exactly one deck path; otherwise usage, exit 2.
15.2. Keys: next `right l space n pgdown enter j down`; back `left h p pgup backspace k up`; first slide `g home`; last slide `G end`; run exec `e`; reload `r`; quit `q ctrl+c esc`. Next advances a step, then the slide; back retreats a step (instantly), then to the previous slide's LAST step. `g`/`G` land on step 1.
15.3. Transitions (380 ms, ease-out): `slide`, `wipe`, `cover`, `dissolve`, `none`. Kind used when moving to slide X: X's `transition` directive, else front matter/`-transition`, else `slide`; moving backwards, the departing slide's directive wins if set. Values case-insensitive; unknown = `slide`. Never used between steps.
15.4. Pause fade (280 ms): a forward step fades in the changed cells. Disabled by `-fade=false` (absolute), else by the slide's `fade` directive if non-empty, else by front matter `fade`; disabling values `false off no none`. Backward steps are instant. No effect on `-dump`/`-check`.
15.5. Live reload: the file is polled every 400 ms; a changed identity, mtime or size reloads it, keeping slide and step (clamped). A missing file is not a change. A parse error keeps the old deck and shows the error in the status row. Reload clears Exec outputs.
15.6. Notes view: `tuipresent -notes deck.md` follows a presenter of the same deck (state file in the temp dir keyed by the deck's absolute path; polled every 200 ms). Shows slide title (`slide N (untitled)` if none), `slide i/n · step k/m · H:MM:SS`, the notes as markdown (`(no notes for this slide)` if none, truncated with `…`), and `Next:` title. It reloads the deck when its mtime changes.

## 16. Invariants

16.1. Every frame is exactly `h` lines, each exactly `w` cells (padded or truncated), at every step and during transitions.
16.2. Layout of a slide is computed from its fully revealed state: column heights, image sizes, centering offsets. Steps only reveal; they never resize.
16.3. Parsing never fails except for front matter YAML errors and an empty deck. Every other mistake degrades silently.
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
fence_block   = ws marker ws info NL { any_line } [ ws marker' ws NL ] ;
                (* marker = 3+ ` or 3+ ~ ; marker' same char, len >= marker *)
image_line    = ws "![" alt "](" ws ( "<" path_sp ">" | path ) [ ws1 '"' title '"' ] ws ")" ws NL ;
title_pair    = nonblank_line NL "=" { "=" } trailing_ws NL ;   (* only before content *)
widths        = [ "[" ] int { "," int } [ "]" ] ;  (* ints <= 0 or invalid dropped *)
```
