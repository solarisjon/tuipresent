# tuipresent Agent Guide

How an AI agent goes from "make a presentation about X" to a verified tuipresent deck: the workflow, the verification commands, the layout budget, design guidance and a pre-handoff checklist.

This guide does not restate the syntax. The exact rules (front matter, directives, titles, notes, images, `+exec`, themes, warnings, layout formulas) are in [docs/agent/LANGUAGE.md](agent/LANGUAGE.md), with real output for each edge case in [docs/agent/EXAMPLES.md](agent/EXAMPLES.md). Read LANGUAGE.md for anything you are unsure about. Complete reference decks are in [examples/](../examples/README.md). Where any doc and the code disagree, the code wins.

Contents:

1. Workflow
2. Verification (mandatory)
3. Syntax at a glance
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
   - Live demos wanted? (`+exec` blocks.) Theme preference? (Default `dark`; see "Choosing a theme" in section 6.) Where will it be shown: a laptop, a projector in a bright room, someone else's terminal?
   - Output path. Default: `<topic-slug>.md` in the current directory.
2. **Outline.** Write one line per slide: title plus the single idea. Show it to the user if the deck is long or the brief is vague.
3. **Write the deck.** One markdown file. Use section 3 (and LANGUAGE.md) for syntax and stay within the budget in section 4.
4. **Verify.** Build the binary, then run `-check` and `-dump` at 100x30 and 80x24 (section 2). If the deck has `+exec` blocks, also follow Step C. Read the dump output and the warnings. This is required, not optional.
5. **Fix and re-verify.** Split overflowing slides, shorten lines, fix every warning, and repeat step 4 until clean.
6. **Hand off.** Give the user the file path, the slide count, the run command and the keys:

   ```sh
   ./bin/tuipresent deck.md          # or: tuipresent deck.md if installed
   ```

   Keys: `→`/`space`/`l` next, `←`/`h` back, `g`/`G` first/last slide, `o` slide overview grid, a number then `enter` jumps to that slide, mouse click on the left/right third or the wheel goes back/forward (`-mouse=false` disables), `e` runs `+exec` blocks on the current slide, `r` reloads from disk, `q` quits. If the deck has `+exec` blocks, list what each one runs. Mention `tuipresent -notes deck.md` in a second terminal if the deck has speaker notes, and `-theme NAME` to preview other themes.

   Live reload: while the deck is open, tuipresent watches the file (and its custom theme file) and reloads on every save, also when editors or agents replace the file via rename, keeping the current slide and step. So the user can keep the deck presenting in one terminal while you edit it. If a save fails to load, the old deck stays up and the error shows in the bottom status line; warnings show there too. `-watch=false` turns this off. Live reload is not a substitute for step 4.

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
- Load errors (invalid front matter YAML, no slides, an invalid custom theme) exit 1 with the error on stderr. So `-check` doubles as a syntax check.
- **Warnings** go to stderr as `deck.md: warning: ...` and do not change the exit status: an unknown theme or transition name, an unknown theme key, a speaker note missing its `-->` (with line numbers where they apply). Treat every warning as a bug to fix.
- It measures the fully revealed slide: top margin, title block, body and footer. The `cols` number is the largest of: body lines wider than the content width, a title wider than the content width (titles never wrap), text clipped inside a column, and fenced or `+exec` code lines over the no-wrap limit, checked per column.

`-check` blind spots: **exec output is not counted** unless you pass `-exec` (Step C), and it cannot judge readability, reveal order, misspelled directives or missing images. That is why Step B is still required.

### Step B: `-dump` (visual review)

```sh
./bin/tuipresent -dump -w 100 -h 30 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
./bin/tuipresent -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
```

- Prints every frame: each slide at each pause step, headed `=== slide N step k/n ===`. Every frame is exactly H lines of exactly W cells. No TTY needed.
- The perl filter strips ANSI color codes so the frames are readable as plain text.
- To see where lines end, append `| sed 's/ *$/|/'`. To look at one slide, pipe through `awk '/^=== slide 5 /,/^=== slide 6 /'`.

Look for:

- Wrapped text-art rows, or chart rows that lost their 2-space code indent (the line was too long).
- Column text that ends mid-word at the column edge (clipped).
- Titles cut off at the right edge.
- Last body lines missing (truncated). Compare against the source.
- Reveal order across pause steps, especially with columns (section 5).
- Paragraphs merged across a directive (`A B` on one line).
- Images: they show as rows of `▀`/`▄` (half-blocks). A `[image: alt]` box means the file was not found or not decodable. An image of only 3 rows means the slide has no room for it.
- Slide count. With a front-matter `title`, the count is sections + 1. A blank frame is a slide with only speaker notes.

### Step C: `-exec` (decks with `+exec` blocks only)

```sh
./bin/tuipresent -exec -check -w 80 -h 24 deck.md
./bin/tuipresent -exec -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
```

- `-exec` only has an effect with `-check` or `-dump`. It runs every `+exec` block in the deck first (all slides, all steps, one at a time, in the deck's directory, 15 s timeout each). Then `-check` measures the output and `-dump` shows it under the code.
- **This executes the deck's code on this machine.** Only use it on decks you wrote or fully reviewed, after confirming every block follows the `+exec` safety rules (section 6). If the user has not approved running the code, ask first, or skip Step C and budget output rows by hand.

If you cannot build (no Go toolchain), say so explicitly at handoff and apply the section 4 budget conservatively.

---

## 3. Syntax at a glance

A reminder, not the rules: see LANGUAGE.md (section numbers in parentheses) before relying on any detail.

```markdown
---
title: Deck title            # adds a cover slide (§2)
author: Name
theme:                       # or just `theme: nord` (§2.5, §18)
  name: catppuccin-latte
  background: paint
transition: slide            # deck default (§15.3)
---

Slide title
===

Body markdown. <!-- comments like this one, sharing a line, are NOT directives -->

<!-- pause -->

More body.

<!-- speaker_note: talking point -->

<!-- end_slide -->
```

- **Directives** (§5), each alone on its line: `end_slide`, `pause`, `column_layout: [2, 1]`, `column: N` (0-based), `reset_layout`, `jump_to_middle`, `transition: KIND`, `speaker_note: text` (or the multi-line form, §8), `fade: off`/`on`, `title_style: rule|curly|frame`, `subtitle: text`.
- **Transitions** (§15.3): `slide`, `wipe`, `cover`, `dissolve`, `none`. Going back from a slide plays its transition in reverse.
- **Links**: `[text](https://url)` is an OSC 8 hyperlink showing only `text` where supported, else `text` plus its dimmed URL (always in `-dump`/`-check`). Relative and `#anchor` links show text only.
- **Titles** (§7): only a setext H1 (`Title` + `===`) before any other content. `# Title` is a body heading. Titles take inline markdown (`**bold**`, `` `code` ``). `<!-- title_style: curly -->` (curly underline where supported, else the rule) or `frame` (boxed, +1 row); `<!-- subtitle: text -->` adds a line (+1 row). `title_style` also works in front matter.
- **Text styles and callouts** (§13.7, §13.8): `~~strike~~`, `==highlight==`, `++underline++`, `[faint]{.faint}`, `[x]{.curly}`/`.dotted`/`.dashed`/`.double`; a blockquote starting `> [!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]` or `[!CAUTION]` becomes a themed box costing its lines + 2 rows.
- **Images** (§9): `![alt](path)` alone on a line, path relative to the deck; `![image:width:50%](path)` caps the width.
- **Code** (§10): `+exec` in the info string makes a block runnable (`bash`, `sh`, `python`/`python3`/`py`; anything else shows an error as the output), in the deck's directory. `{1-3|5|all}` in the info string (`rust {1-3|5}`, combinable with `+exec`) steps through line groups: each group is one reveal step with the other lines dimmed, and the groups add steps like pauses. `{2-4}` alone is a static highlight (§10.5).
- **Themes** (§18): twenty built-ins (table in section 6), palette overrides inline (`accent: "#ff6b35"`, quoted), or `path: themes/x.yaml` for a theme file with `extends:`.
- **Flags** (§15.1): `-check`, `-dump`, `-w`, `-h`, `-exec` (verification); `-slide N`, `-transition KIND`, `-theme NAME`, `-background terminal|paint`, `-themes`, `-notes`, `-fade=false`, `-progress=false`, `-links auto|on|off`, `-watch=false`, `-mouse=false`, `-images auto|kitty|sixel|iterm2|blocks|off` (presenting).
- **Front matter keys** (§2.4): `title`, `sub_title`, `author`, `date`, `theme`, `transition`, `fade`, `progress`, `title_style`. Schema: [frontmatter.schema.json](agent/frontmatter.schema.json).

---

## 4. Layout budget

All numbers come from the renderer (`go test ./internal/docscheck` checks this table). "Rows" means terminal lines. Formulas are in LANGUAGE.md §11.

| | 100x30 | 80x24 |
|---|---|---|
| Content width | 92 | 72 |
| Left indent | 4 | 4 |
| Body rows, titled slide | **23** | **17** |
| Body rows, untitled slide | 27 | 21 |
| Body rows, titled `jump_to_middle` slide | 24 | 18 |
| Prose wrap | 90 | 70 |
| Code line max | 88 | 68 |
| `[1, 1]` column widths | 44 + 44 | 34 + 34 |
| `[1, 1]` prose wrap / code max | 42 / 40 | 32 / 30 |
| `[2, 1]` column widths | 58 + 29 | 45 + 22 |
| `[3, 2]` column widths | 52 + 35 | 40 + 27 |
| `[1, 1, 1]` column widths | 28 x 3 | 21 x 3 |

### What consumes rows

- A tight list item: 1 row per wrapped line. Wrapped lines get a hanging indent under the item text.
- A paragraph: its wrapped lines, plus 1 blank row before the next block.
- A callout (`> [!NOTE]`): its wrapped lines (4 cells narrower than usual) plus 2 border rows. A `subtitle` or the `frame` title style costs 1 row each from the body budget.
- A heading (`##`): 1 row plus 1 blank row.
- A code block: its lines plus a blank row on each side when there is content around it.
- A table: header, separator and one row per entry.
- A columns block: the height of its tallest column, fully revealed. Markdown that follows it gets 1 blank row before it. Markdown right before it (including content before the first `column:`) gets no blank row.
- `+exec` output: an extra block of output lines plus 1 to 2 rows of spacing, added at runtime.
- `speaker_note` comments and directives: 0 rows.
- An image: whatever rows are left, plus 1 blank row before it and before the markdown that follows. A missing image: 3 rows.

### Rules of thumb

Design for 80x24, the binding constraint: 17 body rows on a titled slide.

- At most **6 bullets** per slide, each **68 characters or fewer** so nothing wraps at 80 wide.
- At most **12 body rows** of planned content per titled slide. That leaves headroom for wrapping and blank rows.
- Code blocks: at most **10 lines**, each **68 characters or fewer** (tabs count as 4 spaces).
- `+exec` slides: code lines + expected output lines + about 5 (fence spacing) must be at most 17. Keep output to 5 lines or fewer.
- Two equal columns: lines **30 characters or fewer**, code or bullets (after the `• ` prefix).
- `[2, 1]` columns: narrow column **20 characters or fewer**, wide column **43 or fewer**.
- Avoid three or more columns, or keep each cell to 19 characters or fewer.
- Text-art charts at full width: **68 characters or fewer** per line.
- Titles: **60 characters or fewer**. They never wrap.
- Tables: glamour stretches them to the full content width. Keep cell text short (total 60 characters or fewer) and at most 8 rows.
- Images: one per slide, with at most 4 rows of other body content, or in one column of a `[1, 1]` layout beside 4 to 6 short bullets. Wide images (2:1 or wider) fit best. Keep files under about 1000 px wide.

---

## 5. Gotchas

Each of these fails silently or only warns. LANGUAGE.md §19 lists what warns and what is silent.

1. **Pause steps are global per slide.** If column 0 contains two pauses, everything in column 1 appears only after all of column 0 has revealed. To reveal columns side by side, put no pauses in the earlier columns. Place the pause before `column_layout` or after `reset_layout`.
2. **Column height is fixed to the fully revealed height.** A paused column still uses its full row budget at step 1.
3. **Overflow is silently truncated.** Rows past the budget are cut from the bottom; wide lines are clipped at the screen or column edge. `-check` reports it (except exec output without `-exec`).
4. **Only a setext H1 before any content is the title.** `# Title`, or a `===` heading after a paragraph, renders as an in-body heading with no rule.
5. **An unclosed fence swallows every later `end_slide`,** merging the rest of the file into one slide. (`end_slide` inside a closed fence is code, which is intended.)
6. **Unknown or malformed directives vanish**: typos (`<!-- colum: 1 -->`), uppercase names, comments sharing a line with text, unsupported presenterm directives.
7. **A front-matter `title` adds a cover slide.** Slide count = sections + 1. `-slide N` and the footer `N / total` count the cover. The footer reads ` title  ·  author` (two spaces either side of `·`).
8. **Content after `column_layout` but before the first `column:`** goes full width above the columns, with no blank row.
9. **An out-of-range `column: N` is clamped** to the last column (negative becomes 0); a non-numeric N is ignored.
10. **Directives do not separate paragraphs.** A line, then any directive (`pause`, `speaker_note`, `column`, ...), then another line can merge into one paragraph. Put blank lines around every directive.
11. **Blank lines and `&nbsp;` cannot add vertical space;** leading and trailing blank lines are trimmed.
12. **Unknown theme or transition names fall back** (to `dark` and `slide`) with only a stderr warning; `-check` still exits 0. Copy names from the tables. In YAML an unquoted `#88c0d0` is a comment: always quote colors.
13. **A speaker note missing `-->`** ends before the next directive line (warned), and that directive still applies.
14. **Unsupported exec language.** Any language other than bash, sh, python, python3, py or none is not run; the output is an `unsupported +exec language` error.
15. **Reload clears exec output,** including every automatic reload on save.

---

## 6. Design guidance

### Content

- **One idea per slide.** The title states the claim ("Hit rate drives latency"), not a topic label ("Caching").
- 3 to 6 bullets. Use short phrases, not sentences. No nesting deeper than one level.
- Put numbers on slides. Terminals are good at aligned numbers.
- End with a slide of next steps or asks, and optionally "Questions?" behind a pause.
- **Talking points go in speaker notes, not on the slide.** Keep the slide to the claim and its evidence; put the narrative, numbers to mention and the segue in `speaker_note` comments. Notes cost no rows. A slide with only notes is a blank slide, useful for "pause and talk".

### Text-art charts

Use a ```` ```text ```` fence so glamour keeps alignment and applies no syntax highlighting. Draw bars with block characters: `█` for whole cells, `▏▎▍▌▋▊▉` for fractions, `░▒▓` for shading. Align labels with spaces, never tabs. Keep each line within the code-line maximum (68 at full width at 80 cols, 30 in a `[1, 1]` column).

```text
Q1  42  ████████
Q2  58  ███████████▌
Q3  71  ██████████████▏
```

Other useful glyphs: `→ ← ↑ ↓`, `✓ ✗`, box drawing `─ │ ┌ ┐ └ ┘ ├ ┤`.

### Statement slides

Use `<!-- jump_to_middle -->` with a setext title for a single sentence that should land hard (thesis, key number, section break). Optionally add one supporting line of plain prose, which keeps per-line centering. A list or code line switches the body to block centering. Use them for at most 1 in 5 slides.

### Choosing a theme

| Theme | Kind | Character |
|---|---|---|
| `dark` | dark | Default; high contrast, neutral |
| `light` | light | Neutral light |
| `tokyonight-storm` | dark | Cool blue-violet |
| `tokyonight-night` | dark | Cool, darkest tokyonight, high contrast |
| `tokyonight-moon` | dark | Cool, softer |
| `tokyonight-day` | light | Cool light |
| `catppuccin-mocha` | dark | Pastel, high contrast |
| `catppuccin-macchiato` | dark | Pastel |
| `catppuccin-frappe` | dark | Pastel, lighter background |
| `catppuccin-latte` | light | Pastel light |
| `gruvbox-dark` | dark | Warm, retro |
| `gruvbox-light` | light | Warm light, cream |
| `dracula` | dark | Vivid purple and pink |
| `nord` | dark | Calm arctic blue, low contrast |
| `rose-pine` | dark | Muted, soft |
| `rose-pine-dawn` | light | Muted light |
| `solarized-dark` | dark | Low contrast |
| `solarized-light` | light | Low-contrast light |
| `everforest-dark` | dark | Warm green, soft |
| `kanagawa` | dark | Warm ink-and-paper |

- **Match the room.** Bright room or projector: a light theme; projectors wash out dark backgrounds and thin colored text. Dark room, laptop or large monitor: any dark theme.
- **Paint the background when the terminal is unknown.** If the deck will be shown from someone else's terminal, shared, or uses a light theme, add `background: paint`. A light theme in a dark terminal without painting is unreadable.
- **Pick for contrast, then mood.** Low-contrast themes (`solarized-dark`, `nord`) suit small rooms and big screens better than projectors.
- **Brand colors:** extend the closest built-in in a theme file and override only `accent`/`accent2` (and maybe `code_background`); keep the base's foreground and background. See `examples/themes/brand.yaml`.
- You cannot see colors in `-dump`. Tell the user which theme you chose and why, and suggest they preview with `./bin/tuipresent -theme NAME deck.md` or `./bin/tuipresent -themes`.

### Choosing transitions

- Set one deck-wide default in front matter. `slide` is the safe choice. `dissolve` is softer.
- Override per slide only to signal meaning: `cover` for a new section, `wipe` for before/after or a reveal, `dissolve` for a statement slide, `none` for consecutive slides that should feel like one evolving slide.
- Do not cycle through all kinds for variety. It reads as noise.

### Designing pauses

- Code walk-throughs: instead of several slides of the same code, use one fence with `{1-3|5,7|all}`. Each group is one step, so it counts in the slide's steps (and `-dump` shows the step frames, though dimming is color only and not visible in the text dump). Keep the group count small and line numbers within the block; `-check` warns about numbers past the end.

- Every reveal fades in, so a pause draws the eye to what it adds. Use pauses where the order of ideas matters (problem before fix, question before answer), not on every slide.
- Reveal whole units (a bullet, a column, a takeaway line). A pause that adds one word fades in too little to notice.
- Leave the fade on. Add `<!-- fade: off -->` for slides stepped through rapidly; use front matter `fade: false` for recordings or slow remote terminals.

### `+exec` safety

The presenter runs these live, on their machine, in front of an audience.

- Only use **harmless, offline, idempotent, fast** commands: `echo`, `printf`, `date`, `uname -a`, `seq`, `wc`, `ls` of a known directory, `python3` doing pure computation.
- **Never** use destructive commands (`rm`, `mv`, `dd`, `git push`, `kill`, `sudo`, package installs, writes outside `/tmp`).
- **Never** use network commands (`curl`, `wget`, `ssh`, `ping`, API calls).
- Never print secrets or environment variables (`env`, `printenv`, `cat ~/.ssh/*`).
- Code runs in the deck's directory: refer to demo files by paths relative to the deck.
- Finish well under the 15 s timeout, keep output to 5 lines or fewer, and never print a line of backticks (output is shown in a fence).
- Tell the user at handoff exactly what each `+exec` block runs.

---

## 7. Annotated example deck

This deck passes `-check` at 100x30 and 80x24 (6 slides) with no warnings. The walkthrough below explains each choice.

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

<!-- speaker_note: Ask who has seen a slow dashboard this week. -->

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

1. **Cover (generated).** From front matter `title`, so the total is 5 sections + 1 = 6 slides. `theme: tokyonight-storm` would be equivalent to the nested form.
2. **The problem.** Setext title, three short bullets separated by pauses with blank lines around each directive: 3 steps. The speaker note costs no rows.
3. **Statement slide.** `jump_to_middle` plus `dissolve`. Directives may precede the setext title. The body is plain prose, so each line is centered.
4. **Columns.** `[3, 2]` gives 52 + 35 at 100 wide and 40 + 27 at 80 wide. The chart's widest line is 30 characters, within column 0's 36-character code maximum at 80 wide. No pauses inside the columns, so both appear together; the pause after `reset_layout` reveals the takeaway in step 2. Budget at 80x24: 5 chart rows + 1 blank + 1 takeaway = 7 of 17 body rows.
5. **Exec.** Pure-computation python3: offline, idempotent, under 1 s, 1 output line. Budget: 6 code lines + 2 blank + 1 output + 1 caption = about 11 of 17 rows. Going back from slide 6 to this one plays slide 6's `slide` transition in reverse.
6. **Close.** Ordered list of asks, then "Questions?" behind a pause.

---

## 8. Pre-handoff checklist

- [ ] Front matter starts at line 1 with `---`. `theme` is a built-in name from the table (or a `path:` to a theme file that exists), colors are quoted, and light theme or unknown terminal means `background: paint`.
- [ ] Every transition value is one of `slide`, `wipe`, `cover`, `dissolve`, `none`.
- [ ] Every slide has a setext title (`Title` + `===`) as its first content, 60 characters or fewer.
- [ ] Every directive is lowercase, alone on its line, surrounded by blank lines, spelled exactly, and one of: `end_slide`, `pause`, `column_layout`, `column`, `reset_layout`, `jump_to_middle`, `transition`, `speaker_note`, `fade`. Every multi-line `speaker_note` ends with `-->`.
- [ ] Every code fence is closed.
- [ ] No pauses in earlier columns unless delayed later columns are intended.
- [ ] `+exec` blocks are bash, sh or python only: harmless, offline, idempotent and under 15 s, with rows reserved for their output.
- [ ] `./bin/tuipresent -check -w 100 -h 30 deck.md` and `-check -w 80 -h 24` exit 0 **and print no warnings**.
- [ ] Decks with `+exec`: if the user approved running the code, `-exec -check` exits 0 at both sizes. Otherwise, rows for the output were budgeted by hand.
- [ ] Read the `-dump` output at both sizes with ANSI stripped: no wrapped charts, clipped column text, cut-off titles, merged paragraphs or missing last lines; the reveal order is as intended.
- [ ] Slide count matches the plan (sections + 1 if front matter has `title`).
- [ ] Handoff message includes the path, slide count, run command (`./bin/tuipresent deck.md`), keys, and what each `+exec` block runs.
