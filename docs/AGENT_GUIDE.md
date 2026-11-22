# tuipresent Agent Guide

Authoritative reference for an AI agent asked to "make a presentation about X" with tuipresent. Every rule here is derived from the source (`internal/deck/parse.go`, `internal/render/render.go`, `internal/transition/transition.go`, `cmd/tuipresent/main.go`). Where this guide and the README disagree, this guide follows the code.

For complete, verified reference decks (feature tour, postmortem, pitch, technical talk), see `examples/` and its README.

Contents:

1. Workflow
2. Verification (mandatory)
3. Syntax reference
4. Layout budget
5. Gotchas
6. Design guidance
7. Annotated example deck
8. Pre-handoff checklist

---

## 1. Workflow

Follow these steps in order. Do not skip step 4.

1. **Gather requirements.** Ask the user, or infer from context and state your assumptions:
   - Topic and the one message the audience should leave with.
   - Audience (executives, engineers, mixed) and what they already know.
   - Length: talk minutes or slide count. Default: about 1 slide per minute, 8-12 slides.
   - Live demos wanted? (`+exec` blocks.) Theme preference? (Default `dark`.)
   - Output path. Default: `<topic-slug>.md` in the current directory.
2. **Outline.** Write one line per slide: title plus the single idea. Show it to the user if the deck is long or the brief is vague.
3. **Write the deck.** One markdown file. Use the syntax in section 3 and stay within the budget in section 4.
4. **Verify.** Build the binary, then run `-check` and `-dump` at 100x30 and 80x24 (section 2). If the deck has `+exec` blocks, also follow Step C. Read the dump output. This is required, not optional.
5. **Fix and re-verify.** Split overflowing slides, shorten lines, and repeat step 4 until clean.
6. **Hand off.** Give the user the file path, the slide count, the run command and the keys:

   ```sh
   ./bin/tuipresent deck.md          # or: tuipresent deck.md if installed
   ```

   Keys: `→`/`space`/`l` next, `←`/`h` back, `e` runs `+exec` blocks on the current slide, `r` reloads from disk, `q` quits. If the deck has `+exec` blocks, list what each one runs.

---

## 2. Verification (mandatory)

Build once from the repo root:

```sh
go build -o bin/tuipresent ./cmd/tuipresent
```

### Step A: `-check` (fast gate)

```sh
./bin/tuipresent -check -w 100 -h 30 deck.md
./bin/tuipresent -check -w 80 -h 24 deck.md
```

- Success: prints `deck.md: N slides fit WxH`, exit 0.
- Failure: prints one line per bad slide, `deck.md: slide N overflows WxH by R rows, C cols`, exit 1.
- Parse errors (for example invalid front matter YAML) also exit 1 with the error on stderr. So `-check` doubles as a syntax check.
- It measures the fully revealed slide: top margin (1 row unless `jump_to_middle`), title block (4 rows), body, and the 2-row footer.
- The `cols` number is the largest of these overflows:
  - body lines wider than the content width,
  - the slide title (titles never wrap, so one wider than 92 cells at 100 wide or 72 at 80 wide is reported),
  - text clipped inside a column,
  - fenced or `+exec` code lines wider than the no-wrap limit (content or column width - 4), checked per column. Tabs count as 4 spaces.

`-check` blind spot: **exec output is not counted** unless you pass `-exec` (Step C). Without it, output from `+exec` exists only after the presenter presses `e`, so reserve rows by hand (section 4). `-check` also cannot judge readability or reveal order. That is why Step B is still required.

### Step B: `-dump` (visual review)

```sh
./bin/tuipresent -dump -w 100 -h 30 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
./bin/tuipresent -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
```

- Prints every frame: each slide at each pause step, headed `=== slide N step k/n ===`. Every frame is exactly H lines of exactly W cells. No TTY needed.
- The perl filter strips ANSI color codes so the frames are readable as plain text.
- To see where lines end, append `| sed 's/ *$/|/'`.
- To look at one slide, pipe through `awk '/^=== slide 5 /,/^=== slide 6 /'`.

Look for:

- Wrapped text-art rows, or chart rows that lost their 2-space code indent (the line was too long).
- Column text that ends mid-word at the column edge (clipped).
- Titles cut off at the right edge.
- Last body lines missing (truncated). Compare against the source.
- Reveal order across pause steps, especially with columns (section 5).
- Slide count. With a front-matter `title`, the count is sections + 1.

### Step C: `-exec` (decks with `+exec` blocks only)

```sh
./bin/tuipresent -exec -check -w 80 -h 24 deck.md
./bin/tuipresent -exec -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
```

- `-exec` only has an effect with `-check` or `-dump`. It runs every `+exec` block in the deck first (all slides, all steps, one at a time, 15 s timeout each). Then `-check` measures the output and `-dump` shows it under the code.
- **This executes the deck's code on this machine.** Only use it on decks you wrote or fully reviewed, after confirming every block follows the `+exec` safety rules (section 6). If the user has not approved running the code, ask first, or skip Step C and budget output rows by hand.

If you cannot build (no Go toolchain), say so explicitly at handoff and apply the section 4 budget conservatively.

---

## 3. Syntax reference

### File structure

```
---                 <- front matter must start at byte 0 of the file
<yaml>
---
<slide 1 markdown>
<!-- end_slide -->
<slide 2 markdown>
```

- The file is a single markdown document. CRLF line endings are normalized.
- Slides are separated by `<!-- end_slide -->` on its own line.
- A chunk with no content and no title (for example a trailing `<!-- end_slide -->`, or a chunk containing only directives) is dropped and does not count as a slide.
- If no slides remain, loading fails with `deck has no slides`.

### Front matter

Optional. The first line of the file must be exactly `---`. It ends at the next line that is exactly `---`. Invalid YAML is a fatal load error. Unknown keys are ignored.

| Key | Type | Effect |
|-----|------|--------|
| `title` | string | Creates an extra cover slide (slide 1) and shows in the footer. |
| `sub_title` | string | Italic line on the cover. |
| `author` | string | On the cover and in the footer (`title · author`). |
| `date` | string | On the cover after the author (`author · date`). Any string is accepted. |
| `theme` | string, or mapping with `name` | Color theme (mapping below). `theme: dracula` and `theme:` / `  name: dracula` are equivalent. |
| `transition` | string | Default transition for every slide (tuipresent extension). |

Both forms are accepted:

```yaml
theme: dracula
```

```yaml
theme:
  name: dracula
```

**Theme mapping.** The name is lowercased, then matched by substring in this order. The first match wins.

| Name contains | Theme used |
|---------------|------------|
| `light` or `day` | light. Includes `tokyonight-day`. Note that any name containing `day` matches. |
| `tokyo` | tokyo-night (dark) |
| `dracula` | dracula |
| anything else, or unset | dark |

Use exactly one of: `dark`, `tokyonight-storm`, `dracula`, `light`.

**Cover slide.** Present only when `title` is set. It is vertically centered and shows the title in spaced uppercase (`T I T L E`), falling back to the plain title if the spaced form is wider than the content width. Below that come a rule, the `sub_title`, then `author · date`.

**Footer.** Shown when the height is at least 6. Row 1 is a progress bar. Row 2 has ` title · author` on the left and `N / total` on the right. With no front matter, the left side is blank. With `author` but no `title`, it reads ` · author`.

### Slide titles

- A **setext H1** becomes the slide title: a non-blank line followed by a line of only `=` characters.
- It only counts if it comes before any other content on the slide. Directives before it are fine. Text, lists or code fences before it are not.
- The title renders as `blank, gradient title, full-width ━ rule, blank` (4 rows).
- Titles do not wrap. Keep them to 60 characters or fewer.
- A later `===` heading, or any ATX heading (`# x`), renders inside the body: `#` as a bold accent-colored line, `##` with a `▌ ` prefix, `###` to `######` as bold lines. Use `##` for in-body subheads.
- Setext H2 (`---` underline) is never a title. Avoid it, because `---` is easy to confuse with front matter or a horizontal rule.

### Directives

All directives are HTML comments. To be recognized, a directive must:

- be alone on one line (leading and trailing whitespace allowed),
- use a lowercase name matching `[a-z_]+`,
- follow the form `<!-- name -->` or `<!-- name: arg -->`.

Anything else (uppercase names, multi-line comments other than `speaker_note`, a directive sharing a line with text) is not a directive. Glamour hides HTML comments, so it vanishes silently. Unknown directive names are also dropped silently. Directives inside fenced code blocks are ignored and render as code.

| Directive | Semantics |
|-----------|-----------|
| `<!-- end_slide -->` | Ends the current slide. Ignored inside ``` or ~~~ fences. |
| `<!-- pause -->` | Everything after it on the slide appears one key press later. Each pause adds one reveal step. Steps are global to the slide, including across columns. |
| `<!-- column_layout: [a, b, ...] -->` | Starts a column layout. Content between this directive and the first `column:` renders full width above the columns. Arguments are positive integer relative widths. Brackets are optional. Invalid or non-positive entries are skipped. If none are valid, you get one full-width column. |
| `<!-- column: N -->` | Sends the following content to column N (0-based). It must come after `column_layout`. N above the last index is clamped to the last column. A negative N becomes 0. A non-numeric N is ignored, and content keeps going to the previous target. |
| `<!-- reset_layout -->` | Ends the column layout. Content after it is full width, below the columns, separated by one blank row. |
| `<!-- jump_to_middle -->` | Centers the whole slide vertically and horizontally, and removes the top margin. The title is centered and its rule shrinks to title width + 4. If the slide is plain prose and headings (no list, table, blockquote or code lines in the source), each body line is centered on its own. Otherwise the body is centered as one left-aligned block. Vertical and horizontal offsets use the fully revealed slide, so pauses do not shift it. Applies to the whole slide wherever the directive is placed. |
| `<!-- transition: KIND -->` | Transition used when entering this slide (tuipresent extension). Also used when leaving this slide backwards. |
| `<!-- speaker_note: text -->` | A speaker note (presenterm). Never rendered on the slide and consumes no rows. Also accepts the multi-line form: `<!-- speaker_note:` on its own line, note lines (markdown), then `-->` on its own line. Several notes on one slide are shown in order. Ignored inside fences. Shown by `tuipresent -notes deck.md` in a second terminal. |

Presenterm directives that are **not supported** and silently dropped include `incremental_lists`, `new_lines`/`newline`, `alignment`, `font_size`, `include`, `skip_slide`, and `no_footer`. Do not use them. You cannot insert vertical space with a directive.

### Transitions

Kinds: `slide`, `wipe`, `cover`, `dissolve`, `none`. They take 380 ms with ease-out.

| Kind | Effect |
|------|--------|
| `slide` | New frame pushes the old one horizontally (right-to-left going forward). |
| `wipe` | A vertical edge sweeps across, revealing the new frame. |
| `cover` | New frame slides down from the top over the old one. |
| `dissolve` | Rows switch to the new frame in random order. |
| `none` | Instant cut. |

Resolution order when moving to a slide:

1. The destination slide's `<!-- transition: -->`.
2. Otherwise the front matter `transition`.
3. Otherwise `slide`.
4. When moving backwards, the departing slide's directive wins if it has one.

The CLI flag `-transition KIND` replaces the front matter default but not per-slide directives. It still applies after `r` (reload). Values are case-insensitive. An unknown value falls back to `slide` without an error. Reveal steps within a slide (pauses) never animate.

### Code blocks and `+exec`

- Ordinary fences (``` or ~~~, any length of 3 or more) render as glamour code blocks with a 2-cell left margin and syntax highlighting by language. Tabs are expanded to 4 spaces. Fenced code is rendered separately from surrounding prose, so lines inside a fence that look like list items are left untouched.
- Adding `+exec` anywhere in the info string makes the block executable: ```` ```bash +exec ````.
- The language is the first info word, unless that word starts with `+`.

| Language | Runs as |
|----------|---------|
| `python`, `python3`, `py` | `python3 -c <body>` |
| `sh` | `sh -c <body>` |
| `bash`, empty, or **anything else** | `bash -c <body>` |

Unknown languages such as `go`, `js` or `ruby` silently run through bash. Only write `+exec` blocks in bash, sh or python.

Exec behavior:

- Nothing runs at load time, or in `-dump` or `-check` without `-exec`. During a presentation, code runs only when the presenter presses `e`.
- `e` runs every `+exec` block visible at the current reveal step (including inside columns), concurrently. The last screen row shows `running…` until output arrives.
- Commands run in the directory where tuipresent was started, with that user's environment and permissions.
- Timeout is 15 seconds. On timeout the process is killed and the output ends with `[signal: killed]`.
- stdout and stderr are combined. On a non-zero exit, `[exit status N]` is appended.
- The output appears in a ```` ```text ```` block directly below the code. It adds rows to the slide and can push later content off-screen.
- Pressing `e` again re-runs and replaces the output. `r` (reload) discards all outputs.
- Output is wrapped in a ```` ``` ```` fence, so an output line that is itself a ```` ``` ```` fence would close it early and garble the slide. Do not print fences.
- The timeout text and the `running…` status come from reading the code. They are not visible in `-dump`.

---

## 4. Layout budget

All numbers come from `render.go` and match `-check` and `-dump`. "Rows" means terminal lines.

### Formulas

| Quantity | Formula |
|----------|---------|
| Content width `cw` | `min(w-8, 110)`. If that is under 20, `w-2`. |
| Left indent | `(w - cw) / 2` |
| Footer | 2 rows if `h >= 6`, else 0 |
| Top margin | 1 row, or 0 with `jump_to_middle` |
| Title block | 4 rows (blank, title, rule, blank) |
| Body rows, titled slide | `h - 2 - 1 - 4 = h - 7` |
| Body rows, untitled slide | `h - 3` |
| Prose wrap width | `cw - 2` (glamour slack) |
| Code line max (keeps the 2-cell margin) | `cw - 4`. Longer lines lose the margin. Lines over `cw - 2` wrap. |
| Column area | `cw - 4*(n-1)` (4-cell gap) |
| Column i width | `floor(area * w_i / sum(w))` |
| Column prose wrap | `colw - 2` |
| Column code max | `colw - 4` |

### Numbers

| | 100x30 | 80x24 |
|---|---|---|
| Content width | 92 | 72 |
| Left indent | 4 | 4 |
| Body rows, titled slide | **23** | **17** |
| Body rows, untitled slide | 27 | 21 |
| Prose wrap | 90 | 70 |
| Code line max | 88 | 68 |
| `[1, 1]` column widths | 44 + 44 | 34 + 34 |
| `[1, 1]` prose wrap / code max | 42 / 40 | 32 / 30 |
| `[2, 1]` column widths | 58 + 29 | 45 + 22 |
| `[3, 2]` column widths | 52 + 35 | 40 + 27 |
| `[1, 1, 1]` column widths | 28 x 3 | 21 x 3 |

### What consumes rows

- A tight list item: 1 row per wrapped line. Wrapped lines get a hanging indent, aligned under the item text after the `• ` or `1. ` marker.
- A paragraph: its wrapped lines, plus 1 blank row before the next block.
- A heading (`##`): 1 row plus 1 blank row.
- A code block: its lines plus a blank row on each side when there is content around it.
- A table: header, separator and one row per entry.
- A columns block: the height of its tallest column, fully revealed. Markdown that follows it gets 1 blank row before it. Markdown right before it (including content before the first `column:`) gets no blank row.
- `+exec` output: an extra block of output lines plus 1 to 2 rows of spacing, added at runtime.
- `speaker_note` comments: 0 rows. They never appear on the slide, and `-dump` does not show them.

### Rules of thumb

Design for 80x24, the binding constraint: 17 body rows on a titled slide.

- At most **6 bullets** per slide, each **68 characters or fewer** so nothing wraps at 80 wide. The `• ` or `1. ` list prefix uses 2 to 3 cells of the 70-cell wrap.
- At most **12 body rows** of planned content per titled slide. That leaves headroom for wrapping and blank rows.
- Code blocks: at most **10 lines**, each **68 characters or fewer**.
- `+exec` slides: code lines + expected output lines + about 5 (fence spacing) must be at most 17. Confirm with `-exec -check` (Step C) when approved. Keep output to 5 lines or fewer.
- Two equal columns: lines **30 characters or fewer** (code) or **30 or fewer** (bullets, after the prefix), so they fit at 80 wide.
- `[2, 1]` columns: narrow column **20 characters or fewer**, wide column **43 or fewer**.
- Avoid three or more columns, or keep each cell to 19 characters or fewer.
- Text-art charts at full width: **68 characters or fewer** per line.
- Titles: **60 characters or fewer**. They never wrap.
- Tables: glamour stretches them to the full content width. Keep cell text short (total 60 characters or fewer) and at most 8 rows.

---

## 5. Gotchas

These come from the code. Each one fails silently.

1. **Pause steps are global per slide.** Every `<!-- pause -->` increments one counter for the whole slide. If column 0 contains two pauses, everything in column 1 has step 2 or later and appears only after all of column 0 has revealed. To reveal columns side by side, put no pauses in the earlier columns. Place the pause before `column_layout` (whole layout appears at once) or after `reset_layout`.
2. **Column height is fixed to the fully revealed height.** Blank rows are reserved for content that has not appeared yet. A paused column still uses its full row budget at step 1.
3. **Overflow is silently truncated.** Rows past the body budget are cut from the bottom. Over-wide lines are clipped at the screen edge, or at the column edge inside columns. `-check` reports it (except for exec output without `-exec`), and `-dump` shows it.
4. **Only a setext H1 before any content is the title.** `# Title` (ATX) renders as an in-body heading with no rule. A paragraph or list before the `===` line makes it an in-body heading too. Directives before the title are fine.
5. **`end_slide` inside a fence is ignored.** That is intended for showing markdown source. But an unclosed fence swallows every following `end_slide`, merging the rest of the file into one slide.
6. **Unknown or malformed directives are silently dropped.** That covers typos (`<!-- colum: 1 -->`), uppercase names, multi-line comments and unsupported presenterm directives. They render as nothing.
7. **A front-matter `title` adds a cover slide.** Slide count = non-empty sections + 1. `-slide N` and the footer `N / total` count the cover.
8. **Content placed after `column_layout` but before the first `column:`** goes full width above the columns, not into column 0. It sits directly on top of the columns with no blank row.
9. **An out-of-range `column: N` is clamped** to the last column (a negative N becomes 0). A typo such as `column: 5` in a 2-column layout silently merges into column 1. A non-numeric N is ignored.
10. **Consecutive markdown in the same target is concatenated.** A paragraph line directly followed by a directive and then another line can merge into one paragraph. Put blank lines around directives.
11. **Leading and trailing blank lines are trimmed** from every rendered block. Blank lines and `&nbsp;` cannot add vertical space.
12. **Theme names match by substring.** A name containing `day` or `light` selects the light theme, even when it also contains `tokyo` or `dracula`. Stick to `dark`, `tokyonight-storm`, `dracula`, `light`.
13. **Exec language fallback.** Any language other than python, python3, py or sh runs through bash.
14. **`r` (reload) clears exec output.**

---

## 6. Design guidance

### Content

- **One idea per slide.** The title states the claim ("Hit rate drives latency"), not a topic label ("Caching").
- 3 to 6 bullets. Use short phrases, not sentences. No nested bullets deeper than one level.
- Put numbers on slides. Terminals are good at aligned numbers.
- End with a slide of next steps or asks, and optionally "Questions?" behind a pause.
- **Talking points go in speaker notes, not on the slide.** Keep the slide to the claim and its evidence. Put the narrative, numbers to mention and the segue to the next slide in `speaker_note` comments. Notes cost no layout budget, so they never cause overflow.

### Text-art charts

Use a ```` ```text ```` fence so glamour keeps alignment and applies no syntax highlighting. Draw bars with block characters: `█` for whole cells, `▏▎▍▌▋▊▉` for fractions, `░▒▓` for shading. Align labels with spaces, never tabs. Keep each line within the code-line maximum for its width (68 at full width at 80 cols, 30 in a `[1, 1]` column).

```text
Q1  42  ████████
Q2  58  ███████████▌
Q3  71  ██████████████▏
```

Other useful glyphs: `→ ← ↑ ↓`, `✓ ✗`, box drawing `─ │ ┌ ┐ └ ┘ ├ ┤`.

### Statement slides

Use `<!-- jump_to_middle -->` with a setext title for a single sentence that should land hard (thesis, key number, section break). Optionally add one supporting line of plain prose underneath, which keeps per-line centering. A list or code line switches the body to block centering. Use them for at most 1 in 5 slides.

### Choosing transitions

- Set one deck-wide default in front matter. `slide` is the safe choice. `dissolve` is softer.
- Override per slide only to signal meaning:
  - `cover` for a new section,
  - `wipe` for before/after or a reveal,
  - `dissolve` for a statement slide,
  - `none` for consecutive slides that should feel like one evolving slide (for example a diagram that gains a step).
- Do not cycle through all kinds for variety. It reads as noise.

### `+exec` safety

The presenter runs these live, on their machine, in front of an audience.

- Only use **harmless, offline, idempotent, fast** commands: `echo`, `printf`, `date`, `uname -a`, `seq`, `wc`, `ls` of a known directory, `python3` doing pure computation.
- **Never** use destructive commands (`rm`, `mv`, `dd`, `git push`, `kill`, `sudo`, package installs, writes outside `/tmp`).
- **Never** use network commands (`curl`, `wget`, `ssh`, `ping`, API calls). They fail offline and leak data.
- Never print secrets or environment variables (`env`, `printenv`, `cat ~/.ssh/*`).
- Finish well under the 15 s timeout, keep output to 5 lines or fewer so it fits, and never print a line of backticks.
- Prefer `python3` or `bash`. Never rely on the language fallback.
- Tell the user at handoff exactly what each `+exec` block runs.

---

## 7. Annotated example deck

This deck passes `-check` at 100x30 and 80x24 (6 slides) and was reviewed with `-dump`. The comments in the walkthrough below explain each choice. They are not part of the deck.

````markdown
---
title: Caching in Five Minutes
sub_title: Why the second request is fast
author: Platform Team
date: 2026-12-08
theme:
  name: tokyonight-storm
transition: slide
---

The problem
===

* Every request recomputes the same answer
<!-- pause -->
* The database does 90% of the work
<!-- pause -->
* Users wait for it on every page load

<!-- end_slide -->

<!-- jump_to_middle -->
<!-- transition: dissolve -->

Remember the answer.
===

Then reuse it until it goes stale.

<!-- end_slide -->

Hit rate drives latency
===

<!-- transition: wipe -->
<!-- column_layout: [3, 2] -->
<!-- column: 0 -->

```text
hit rate   p50 latency
  0%   ████████████████  120ms
 50%   ████████           62ms
 90%   ██                 16ms
 99%   ▌                   5ms
```

<!-- column: 1 -->

* Hits skip the database
* Misses pay full price
* Aim for 90% or more

<!-- reset_layout -->

<!-- pause -->

**Takeaway:** a small rise in hit rate cuts latency a lot.

<!-- end_slide -->

Try it
===

<!-- transition: cover -->

```python +exec
from functools import lru_cache
@lru_cache
def square(n): return n * n
for n in (2, 3, 2, 2):
    square(n)
print(square.cache_info())
```

Press `e` to run. Output appears below the code.

<!-- end_slide -->

Next steps
===

1. Measure the current hit rate
2. Cache the three hottest queries
3. Set an expiry for each one

<!-- pause -->

Questions?
````

Slide by slide:

1. **Cover (generated).** It comes from front matter `title`, so the total is 5 sections + 1 = 6 slides. `theme` uses the nested form. `theme: tokyonight-storm` would be equivalent. The `slide` transition is the deck default.
2. **The problem.** Setext title, three short bullets with pauses: 3 steps. Bullets are under 45 characters, so nothing wraps at 80 wide.
3. **Statement slide.** `jump_to_middle` plus `dissolve`. Directives may precede the setext title. The title, its shortened rule, and the single supporting line are each centered horizontally, because the body is plain prose.
4. **Columns.** `[3, 2]` gives 52 + 35 at 100 wide and 40 + 27 at 80 wide. The chart's widest line is 30 characters, within the 36-character code maximum of column 0 at 80 wide (40 - 4). The right column's bullet text is 22 characters or fewer. Adding the 2-cell `• ` prefix gives 24 cells, within the 25-cell wrap of column 1 at 80 wide (27 - 2). There are no pauses inside the columns, so both appear together. The pause after `reset_layout` reveals the takeaway in step 2, one blank row below the columns. Budget at 80x24: 5 chart rows + 1 blank + 1 takeaway = 7 of 17 body rows.
5. **Exec.** Pure-computation python3. Offline, idempotent, under 1 s, 1 output line. Budget: 6 code lines + 2 blank + 1 output + 1 caption line = about 11 of 17 rows. `cover` marks the hands-on section.
6. **Close.** Ordered list of asks, then "Questions?" behind a pause.

Verified with:

```sh
./bin/tuipresent -check -w 100 -h 30 deck.md   # deck.md: 6 slides fit 100x30
./bin/tuipresent -check -w 80 -h 24 deck.md    # deck.md: 6 slides fit 80x24
./bin/tuipresent -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
```

---

## 8. Pre-handoff checklist

- [ ] Front matter starts at line 1 with `---`. `theme` (plain or nested `name:`) is one of `dark`, `tokyonight-storm`, `dracula`, `light`.
- [ ] Every transition value is one of `slide`, `wipe`, `cover`, `dissolve`, `none`.
- [ ] Every slide has a setext title (`Title` + `===`) as its first content, 60 characters or fewer.
- [ ] Every directive is lowercase, alone on its line, spelled exactly, and one of: `end_slide`, `pause`, `column_layout`, `column`, `reset_layout`, `jump_to_middle`, `transition`, `speaker_note`.
- [ ] Every code fence is closed.
- [ ] No pauses in earlier columns unless delayed later columns are intended.
- [ ] `+exec` blocks are bash, sh or python only. They are harmless, offline, idempotent and under 15 s, with rows reserved for their output.
- [ ] `./bin/tuipresent -check -w 100 -h 30 deck.md` exits 0.
- [ ] `./bin/tuipresent -check -w 80 -h 24 deck.md` exits 0.
- [ ] Decks with `+exec`: if the user approved running the code, `-exec -check` exits 0 at both sizes. Otherwise, rows for the output were budgeted by hand.
- [ ] Read the `-dump` output at both sizes with ANSI stripped. No wrapped charts, clipped column text, cut-off titles or missing last lines. The reveal order is as intended.
- [ ] Slide count matches the plan (sections + 1 if front matter has `title`).
- [ ] Handoff message includes the path, slide count, run command (`./bin/tuipresent deck.md`), keys, and what each `+exec` block runs.
