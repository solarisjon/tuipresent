# Authoring decks

How to write a tuipresent deck that reads well on a terminal: every construct the parser understands, a minimal example of each, and the traps worth knowing. If you haven't built a deck yet, start with [Getting started](getting-started.md); for compact tables of everything here, see the [Reference](reference.md).

## Contents

- [Deck structure](#deck-structure)
- [Front matter](#front-matter)
- [Themes](#themes)
- [Slide titles](#slide-titles)
- [Text, lists, tables, quotes and code](#text-lists-tables-quotes-and-code)
- [Pauses](#pauses)
- [Code line highlights](#code-line-highlights)
- [Columns](#columns)
- [Centered statement slides](#centered-statement-slides)
- [Images](#images)
- [Runnable code (`+exec`)](#runnable-code-exec)
- [Transitions and fade](#transitions-and-fade)
- [Speaker notes](#speaker-notes)
- [Designing for the terminal](#designing-for-the-terminal)

## Deck structure

A deck is a single markdown file: optional front matter, then slides separated by an `end_slide` comment on its own line.

```markdown
First slide
===

Some text.

<!-- end_slide -->

Second slide
===

More text.
```

Gotchas:

- A `---` line is **not** a slide separator (it's a horizontal rule, or the front matter fence at the top of the file). Only `<!-- end_slide -->` splits slides.
- Directives are HTML comments on a line of their own, with a lowercase name: `<!-- pause -->`, `<!-- column: 1 -->`. A misspelled or unknown directive is silently ignored, and glamour hides HTML comments, so a typo just vanishes. Run `-dump` if something didn't happen.
- Directives inside fenced code blocks are ignored, so you can show markdown source on a slide. The flip side: an unclosed fence swallows every `end_slide` after it and merges the rest of the file into one slide.
- Empty sections (for example after a trailing `end_slide`) are dropped, not shown as blank slides. A section with only speaker notes is kept as a blank slide, which is handy for a "pause and talk" moment.
- Put a blank line before and after each directive. Otherwise lines either side can join into one paragraph.

## Front matter

Optional YAML that must start on the very first line of the file.

```markdown
---
title: Caching at the edge
sub_title: What we learned in Q3
author: Platform team
date: 2026-11-30
theme: dark
transition: slide
fade: true
---
```

| Key | Effect |
|-----|--------|
| `title` | Generates a cover slide, and appears in the footer of every slide |
| `sub_title` | Italic line under the title on the cover |
| `author` | On the cover and in the footer |
| `date` | On the cover, after the author |
| `theme` | Color theme; see [Themes](reference.md#themes) |
| `transition` | Default transition between slides (see [Transitions and fade](#transitions-and-fade)) |
| `fade` | `false` makes pause reveals instant for the whole deck |

Gotchas:

- Without a `title` there is no cover slide, and the footer shows only the slide number.
- The cover counts as slide 1. `-slide 3` and the footer's `3 / 10` both count it.
- Invalid YAML stops the deck from loading, with a `front matter:` error.

## Themes

Pick one of twenty built-in color themes by name, and add `background: paint` when you don't control the terminal:

```markdown
---
theme:
  name: catppuccin-latte
  background: paint
---
```

- `tuipresent -themes` lists them with color swatches; `tuipresent -theme nord deck.md` previews your deck in another theme without editing it. `examples/themes.md` is a gallery deck made for this.
- By default slides are drawn on the terminal's own background, so a light theme in a dark terminal looks wrong. `background: paint` draws every cell in the theme's colors, so the deck looks the same anywhere.
- Override a few colors inline (`accent: "#ff6b35"`, always quoted), or put a whole brand theme in a file and use `theme: {path: themes/brand.yaml}`. See `examples/brand.md`.
- A misspelled theme name renders in `dark` and warns.

The theme list, every color key and the theme file format are in the [Reference](reference.md#themes).

## Slide titles

A line underlined with `===` (a "setext" heading), placed before any other content on the slide, becomes the slide title. It's drawn in a color gradient with a rule underneath and takes 4 rows (blank, title, rule, blank).

```markdown
Hit rate drives latency
===

Body text starts here.
```

Gotchas:

- `# Title` is **not** a slide title. It renders as an ordinary heading inside the body, with no rule.
- If any text comes before the `===` line, it's not a title either. Directives before it are fine.
- Titles never wrap. Keep them under about 60 characters so they fit at 80 columns.
- Use the title for the claim ("Hit rate drives latency"), not a topic label ("Caching").

## Text, lists, tables, quotes and code

Everything between directives is ordinary markdown, rendered by glamour and wrapped to the slide width.

````markdown
Markdown on a slide
===

**Bold** for emphasis, *italic* for asides, `code` for names.

## A subheading

* Short phrases
* One idea per bullet
  * One level of nesting at most

| Region | p99 |
|--------|-----|
| eu     | 9 ms |
| us     | 12 ms |

> Quotes work too.

```go
func main() { fmt.Println("hi") }
```
````

Gotchas:

- Wrapped list items get a hanging indent, so long bullets stay readable, but they still cost a row per wrapped line.
- Tables stretch to the full content width. Keep cells short.
- Use spaces, not tabs, to align things. Tabs are expanded to 4 spaces.
- Blank lines and `&nbsp;` can't add vertical space: leading and trailing blank lines of every block are trimmed.
- Code lines longer than the content width minus 4 lose their margin; longer still and they wrap. `-check` reports code that's too wide.

## Pauses

`<!-- pause -->` splits a slide into steps. Each press of `space` reveals the next step, which fades in while what's already shown stays put.

```markdown
Three reasons
===

1. It's fast
<!-- pause -->
2. It's cheap
<!-- pause -->
3. It's already deployed
```

Gotchas:

- A pause reveals **everything** after it on the slide, up to the next pause, including content after the columns end.
- Going backwards is instant: it hides the last step without animating. Moving back onto a previous slide lands on its last step, fully revealed.
- Reveal whole units (a bullet, a column, a conclusion). A one-word reveal is barely noticeable.

### Pauses and columns

There is one step counter per slide, not one per column. A pause inside column 0 delays everything after it, including all of column 1. That makes "left side first, then right side" easy:

```markdown
Before and after
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Before

* 40 ms p99
* Manual deploys

<!-- pause -->
<!-- column: 1 -->

## After

* 9 ms p99
* One-click deploys

<!-- reset_layout -->

Full-width takeaway under both columns.
```

Here step 1 shows only "Before". Step 2 shows "After" and the takeaway together, because the takeaway also comes after the pause.

The flip side: if column 0 has two pauses, column 1 doesn't appear until both have been revealed. You can't reveal two columns in parallel. Columns always reserve their fully revealed height, so the layout never shifts as steps appear.

## Code line highlights

Add a selection in braces to a code fence's info string to walk through the code a few lines at a time: `{1-3|5,7|all}`. The groups are separated by `|`; each is one step, like a pause, and in each step only the selected lines are drawn at full color while the rest of the block is dimmed.

````markdown
Walking through main
===

```rust {1-2|4-5|all}
fn main() {
    let name = "tuipresent";
    greet(name);
    println!("done");
}
```
````

Step 1 lights lines 1-2, step 2 lines 4-5, step 3 all of them. The keys that advance a pause advance the groups, `←` goes back through them, and the fade and the slide's step total work as for pauses. `-dump` prints one frame per step, but dimming is color only, so its text frames look alike.

- A group is a comma-separated list of 1-based lines inside the fence (`5`), ranges (`2-4`) and `all` (every line): `{1,3-4|all}`.
- A single group, `{2-4}`, is a static highlight: those lines stand out and there is no extra step.
- Content after the code block appears once the last group is showing, as if a pause came after it. Pauses before it, and several highlight blocks on one slide, add up in order.
- It works with the other info-string words: ```` ```bash {1|2} +exec ```` highlights the code and still runs it on `e`. The braces can go anywhere after the opening fence.
- Mistakes warn and never stop the deck: a bad item (`0`, `2-`, `4-2`, `a`), a line past the end of the block (skipped, or clipped for a range), an empty group (`{1||3}`) and an unclosed brace. Warnings give the fence's line number. A fence with no usable group is an ordinary code block.
- Dimming blends the code's colors toward the code background, so it reads in every theme. Code that wraps is shown without dimming.

## Columns

```markdown
Two columns
===

<!-- column_layout: [2, 1] -->
<!-- column: 0 -->

The wide column gets two thirds of the width.

<!-- column: 1 -->

The narrow one gets a third.

<!-- reset_layout -->

Back to full width.
```

- `column_layout: [2, 1]` starts a layout. The numbers are relative widths, so `[1, 1]` is two equal halves and `[1, 1, 1]` is three thirds. There's a 4-cell gap between columns.
- `column: N` sends what follows to column N, counting from 0.
- `reset_layout` ends the layout.

Gotchas:

- **Forgetting `reset_layout`** puts everything after it into the last column you selected.
- Content between `column_layout` and the first `column:` goes full width above the columns, not into column 0.
- `column: 5` in a two-column layout is clamped to the last column. `column:` before any `column_layout` does nothing.
- Columns are narrow: two equal columns at 80 columns wide leave about 32 characters per line. Three columns are rarely worth it.

## Centered statement slides

`<!-- jump_to_middle -->` centers the slide vertically and horizontally. Use it for a thesis, a key number, a section break or a closing slide.

```markdown
Cache everything at the edge
===

<!-- jump_to_middle -->

p99 dropped from 40 ms to 9 ms.
```

Gotchas:

- A body of plain prose and headings is centered line by line. As soon as the body has a list, table, quote or code block, it's centered as one block instead, so the lines stay aligned with each other.
- The directive can go anywhere on the slide.
- Use it sparingly, say one slide in five. Its power comes from contrast.

## Images

Put an image on a line of its own:

```markdown
Architecture
===

![Request path](diagram.png)
```

- Paths are relative to the deck file. Wrap paths containing spaces in angle brackets: `![alt](<my diagram.png>)`.
- PNG, JPEG and GIF are supported (GIFs show the first frame).
- The image is scaled down, keeping its aspect ratio, to fit the width and whatever rows the rest of the slide leaves free (at least 3). Images work inside columns too.
- To cap the width, use `image:width:N%` as the alt text: `![image:width:50%](diagram.png)` uses at most half the available width.

How it's drawn:

| Where | Result |
|-------|--------|
| kitty, ghostty (not in tmux) | Full resolution, through the kitty graphics protocol |
| Other terminals, tmux, screen, `-dump`, `-check` | Truecolor half-block characters (about 2 pixels per cell) |
| File missing or unreadable, or `-images off` | A dim box: `[image: alt]` |

Gotchas:

- An image inside a paragraph (text on the same line) isn't treated as an image.
- With the width option, the alt text is gone, so a missing file shows its filename in the box instead.
- The more text on the slide, the smaller the image. One image beside or under four or five short lines works best. Wide images (2:1 or wider) fill a slide better than tall ones.
- Images are shrunk to a few hundred pixels at most, so there's no point in huge files.

## Runnable code (`+exec`)

Add `+exec` to a code fence's info string. The code shows like any other block; press `e` during the talk and its output appears underneath.

````markdown
Live demo
===

```bash +exec
echo "Hello from $(uname -s)"
date +%Y
```
````

| Fence language | Runs as |
|----------------|---------|
| `python`, `python3`, `py` | `python3 -c <code>` |
| `sh` | `sh -c <code>` |
| `bash`, or none | `bash -c <code>` |

- `e` runs every `+exec` block visible on the current slide (blocks behind a pause you haven't reached yet are skipped).
- stdout and stderr are both shown. A non-zero exit appends a line like `[exit status 1]`.
- Each run is killed after 15 seconds.
- Code runs in the deck file's directory, so relative paths in a demo mean the same as image paths.
- Output is cleared when the deck reloads (on save, or `r`).

**Safety.** Nothing runs unless you press `e`, or pass `-exec` to `-check` / `-dump`. But you are still running code in front of an audience:

- Stick to fast, offline, harmless commands: `echo`, `date`, `seq`, `wc`, pure computation in Python.
- Never put anything destructive, networked or secret-revealing (`rm`, `curl`, `env`) in a block.
- Don't run `-exec` on a deck you didn't write or haven't read.

Gotchas:

- `go +exec`, `js +exec` and similar are **not run**: the output shows `[unsupported +exec language "go" (supported: bash, sh, python, python3, py)]`. Only bash, sh and Python are supported.
- Output takes space on the slide that `-check` can't see unless you pass `-exec`. Leave room: keep output to about 5 lines.

## Transitions and fade

Moving between slides animates (about 0.4 s). Set a deck-wide default in front matter and override it per slide:

```markdown
New section
===

<!-- transition: cover -->

This slide drops in from the top.
```

| Kind | Effect |
|------|--------|
| `slide` | The new slide pushes the old one sideways (the default) |
| `wipe` | The new slide is revealed left to right |
| `cover` | The new slide drops in from the top |
| `dissolve` | Rows switch over in random order |
| `none` | Instant |

- The slide's own `transition:` applies when you arrive at it. Going back from a slide undoes the transition that brought you there: its transition (its own, or the deck default) plays in reverse.
- `-transition KIND` on the command line overrides the front matter default (but not per-slide directives).
- An unrecognized kind becomes `slide` and warns: on the status line while presenting, on stderr from `-check`.

Pick one default and override only to mean something: `cover` for a new section, `wipe` for a before/after, `none` for a slide that continues the previous one.

**Fade.** Content revealed by a pause fades in (about 0.3 s). Turn it off for one slide with `<!-- fade: off -->`, for the whole deck with `fade: false` in front matter, or for one session with `-fade=false`. A slide's `<!-- fade: on -->` re-enables it when the front matter turned it off.

## Speaker notes

Notes are never drawn on the slide and take no space. They show up in the notes view (`tuipresent -notes deck.md`, see [Presenting](presenting.md#speaker-notes-in-a-second-pane)).

```markdown
Roadmap
===

* Q1: edge cache
* Q2: regional failover

<!-- speaker_note: Mention the 40% cost saving first. -->

<!-- speaker_note:
- Pause for questions
- Then demo the dashboard
-->
```

- A slide can have several notes; they're shown in order.
- Notes are markdown, so lists and emphasis work.
- The multi-line form is the only multi-line comment the parser understands. If you forget its closing `-->`, the note ends before the next directive line (a `<!-- pause -->`, `<!-- end_slide -->` and so on, which still takes effect) and tuipresent warns; with no later directive it runs to the end of the slide.
- A note doesn't separate paragraphs. Put blank lines around it.
- A slide with notes but nothing else is kept as a blank slide.

Put the narrative, the numbers to mention and the segue to the next slide in notes, and keep the slide itself to the claim and its evidence.

## Designing for the terminal

A terminal slide holds much less than a browser slide. At 80x24, a titled slide has **17 rows** and **72 columns** for its body; at 100x30, 23 rows and 92 columns ([full layout numbers](reference.md#layout)). Some habits that help:

- **One idea per slide.** The title states it; the body backs it up.
- **3 to 6 bullets**, as short phrases. If a bullet wraps, cut words.
- **Numbers beat adjectives.** Terminals are good at aligned numbers.
- **Move the talking into speaker notes.** They cost no space.
- **Draw charts with text.** A `text` fence keeps alignment and skips syntax highlighting:

  ````markdown
  Latency by quarter
  ===

  ```text
  Q1  42 ms  ████████████████████▉
  Q2  31 ms  ███████████████▌
  Q3   9 ms  ████▌
  ```
  ````

  Useful glyphs: `█ ▉ ▊ ▋ ▌ ▍ ▎ ▏` for bars, `░ ▒ ▓` for shading, `→ ✓ ✗`, and box drawing `─ │ ┌ ┐ └ ┘`.
- **Design for the smallest screen you might get.** Projectors and screen shares are often small. Check both sizes before every talk:

  ```sh
  tuipresent -check deck.md               # 100x30
  tuipresent -check -w 80 -h 24 deck.md   # 80x24
  ```

  A slide that doesn't fit is cut off at the bottom or right edge with no warning on screen. `-check` names it; `-dump` shows you what got cut.
- **Write while presenting.** Keep the deck open in one pane and your editor in another; every save reloads the slides in place.
