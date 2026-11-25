# tuipresent language: verified examples

Minimal pairs for [LANGUAGE.md](LANGUAGE.md). Every result below is real output.
`go test ./internal/docscheck` renders each case's deck and fails if a `-check`
or `-dump` block here differs; `go test ./internal/docscheck -run Examples -update`
rewrites those blocks and the index. The deck sources and the notes under each
case are hand-written: to add a case, append a section with a deck block and two
empty ```` ```text ```` blocks, run `-update`, and write the notes from the output.

Conventions for each result:

- `-check` block: `tuipresent -check -w 80 -h 24 deck.md` with its warnings
  (stderr) first, then its report and exit status.
- Frames: `tuipresent -dump -w 80 -h 24 deck.md` with ANSI escapes stripped,
  trailing spaces removed, the 2-row footer omitted (except in cases marked
  `<!-- frames: full -->`, such as case 1), and trailing blank rows dropped.
  Leading blank rows are kept: they are the top margin or vertical centering.
  Each frame starts with its `=== slide N step k/n ===` header. Half-block
  image rows are abbreviated as `<N half-block cells>`.
- At 80x24: content width cw = 72, left indent 4, 22 rows above the footer.
- No case passes `-exec`; no deck code runs.

## Index

1. [Slide splitting and the cover slide](#case-1)
2. [Setext title vs ATX heading](#case-2)
3. [Setext underline after other content is not a title](#case-3)
4. [`end_slide` inside a fence is code](#case-4)
5. [An unclosed fence swallows later `end_slide`s](#case-5)
6. [Unknown and malformed directives vanish](#case-6)
7. [Pauses without blank lines merge paragraphs; a trailing pause adds an empty step](#case-7)
8. [A pause before all content; the title is never paused](#case-8)
9. [Content before the first `column`, and after `reset_layout`](#case-9)
10. [A pause inside a column delays every later column](#case-10)
11. [Width parsing, out-of-range and non-numeric `column` indices](#case-11)
12. [Adjacent layouts, `column` after `reset_layout`, empty layout](#case-12)
13. [A setext title inside a column is still the slide title](#case-13)
14. [`jump_to_middle`: prose is centered line by line](#case-14)
15. [`jump_to_middle`: a list is centered as one block](#case-15)
16. [Speaker notes: single-line, multi-line, notes-only slide](#case-16)
17. [A multi-line note without `-->` stops before the next directive and warns](#case-17)
18. [Image lines: missing file, width percentage, inline image](#case-18)
19. [Image sizing from free rows](#case-19)
20. [`+exec` blocks render as code until run](#case-20)
21. [The code no-wrap limit (cw - 4)](#case-21)
22. [Titles never wrap](#case-22)
23. [Titles render inline markdown](#case-23)
24. [List items wrap with a hanging indent](#case-24)
25. [Tabs expand to 4 spaces](#case-25)
26. [An empty front matter block](#case-26)
27. [Unknown transition names fall back to `slide` and warn](#case-27)
28. [Code line highlights are reveal steps, and bad specs warn](#case-28)
29. [Subtitle and the frame title style](#case-29)
30. [Highlight, strikethrough, faint and underline marks](#case-30)
31. [Callout boxes](#case-31)
32. [Big titles: block letters, their row cost and the width fallback](#case-32)

<a id="case-1"></a>

## 1. Slide splitting and the cover slide

<!-- frames: full -->

````markdown
---
title: Demo
author: Ann
---

One
===

First slide.

<!-- end_slide -->

<!-- end_slide -->

Two
===
<!-- end_slide -->
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 3 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===








                                    D E M O

                                ━━━━━━━━━━━━━━━


                                      Ann








━━━━━━━━━━━━━━━━━━━━━━━━━━──────────────────────────────────────────────────────
 Demo  ·  Ann                                                             1 / 3
=== slide 2 step 1/1 ===


    One
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    First slide.
















━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━───────────────────────────
 Demo  ·  Ann                                                             2 / 3
=== slide 3 step 1/1 ===


    Two
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


















━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Demo  ·  Ann                                                             3 / 3
```

- 3 slides: the cover (from `title`), `One`, `Two`.
- The empty chunk between the two consecutive `end_slide` lines and the empty chunk after the last one are dropped.
- A slide with only a title (`Two`) is kept.
- Footer: ` Demo  ·  Ann` left, `N / 3` right.

<a id="case-2"></a>

## 2. Setext title vs ATX heading

````markdown
Setext title
===

Body.
<!-- end_slide -->
# ATX heading

Body.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 2 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Setext title
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Body.
=== slide 2 step 1/1 ===

    ATX heading

    Body.
```

- Slide 1: `Setext title` is the slide title: blank row, title, full-width rule, blank row.
- Slide 2: `# ATX heading` is an in-body heading. No rule, and only the 1-row top margin above it.

<a id="case-3"></a>

## 3. Setext underline after other content is not a title

````markdown
Intro line.

Late
===

Body.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===

    Intro line.

    Late

    Body.
```

- `Intro line.` is content, so `Late`/`===` is rendered by glamour as an in-body H1. The slide has no title and no rule.

<a id="case-4"></a>

## 4. `end_slide` inside a fence is code

````markdown
Showing source
===

```markdown
<!-- end_slide -->
```
<!-- end_slide -->
Next
===
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 2 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Showing source
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      <!-- end_slide -->
=== slide 2 step 1/1 ===


    Next
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- 2 slides. The `end_slide` inside the fence is printed as code; the one after the closing fence splits.

<a id="case-5"></a>

## 5. An unclosed fence swallows later `end_slide`s

````markdown
One
===

```text
never closed
<!-- end_slide -->
Two
===
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    One
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      never closed
      <!-- end_slide -->
      Two
      ===
```

- 1 slide. Everything after the opening fence, including `end_slide` and the would-be title `Two`, is code.

<a id="case-6"></a>

## 6. Unknown and malformed directives vanish

````markdown
Directives
===

<!-- colum: 1 -->
<!-- Pause -->
<!-- incremental_lists: true -->
<!-- my-note -->
visible
<!-- end_slide: anything -->
second slide
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 2 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Directives
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    visible
=== slide 2 step 1/1 ===

    second slide
```

- Typo `colum`, uppercase `Pause`, unsupported `incremental_lists` and the hyphenated `my-note` are not directives (or are unknown) and render as nothing. The slide has 1 step.
- `end_slide` ignores its argument: `<!-- end_slide: anything -->` still splits, giving 2 slides.

<a id="case-7"></a>

## 7. Pauses without blank lines merge paragraphs; a trailing pause adds an empty step

````markdown
Pauses
===

A
<!-- pause -->
B

<!-- pause -->

C
<!-- pause -->
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/4 ===


    Pauses
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    A
=== slide 1 step 2/4 ===


    Pauses
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    A B
=== slide 1 step 3/4 ===


    Pauses
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    A B

    C
=== slide 1 step 4/4 ===


    Pauses
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    A B

    C
```

- 4 steps (3 pauses).
- Step 2 shows `A B` on one line: the blocks on either side of a pause are rejoined with a single newline before rendering, so `A` and `B` form one paragraph.
- `C` is a separate paragraph because blank lines surround its pause.
- Step 4 is identical to step 3: the trailing pause reveals nothing.

<a id="case-8"></a>

## 8. A pause before all content; the title is never paused

````markdown
<!-- pause -->
Title
===

Body.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/2 ===


    Title
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
=== slide 1 step 2/2 ===


    Title
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Body.
```

- 2 steps. The title is part of the slide, not a block, so it shows at step 1; `Body.` appears at step 2.
- Directives before the setext title do not stop it being the title.

<a id="case-9"></a>

## 9. Content before the first `column`, and after `reset_layout`

````markdown
Columns
===

<!-- column_layout: [1, 1] -->
Above the columns.
<!-- column: 0 -->
Left.
<!-- column: 1 -->
Right.
<!-- reset_layout -->
Below the columns.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Columns
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Above the columns.
    Left.                                 Right.

    Below the columns.
```

- `Above the columns.` sits full width directly on top of the columns (no blank row).
- Column widths at 80x24: (72 - 4) / 2 = 34 each, 4-cell gap: `Right.` starts 38 cells right of `Left.`.
- `Below the columns.` is full width after one blank row.

<a id="case-10"></a>

## 10. A pause inside a column delays every later column

````markdown
Columns and pauses
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
L1

<!-- pause -->

L2
<!-- column: 1 -->
R1
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/2 ===


    Columns and pauses
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    L1
=== slide 1 step 2/2 ===


    Columns and pauses
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    L1                                    R1

    L2
```

- 2 steps. Step counters are per slide, not per column: `R1` comes after the pause in column 0, so it has step 1 and appears only at step 2, together with `L2`.
- At step 1 the layout already reserves `L2`'s rows (column height is the fully revealed height).

<a id="case-11"></a>

## 11. Width parsing, out-of-range and non-numeric `column` indices

````markdown
Clamping
===

<!-- column_layout: [2, x, 0, 1] -->
<!-- column: 0 -->
zero
<!-- column: 7 -->
seven
<!-- column: two -->
still seven
<!-- column: -3 -->
negative
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Clamping
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    zero negative                                    seven still seven
```

- `[2, x, 0, 1]` keeps only positive integers: widths `[2, 1]` = 45 and 22 cells at 80x24.
- `column: 7` clamps to the last column (1). `column: two` is ignored, so `still seven` keeps going to column 1. `column: -3` clamps to 0.
- `zero` and `negative` are consecutive blocks in column 0 with no blank line between them, so they render as one paragraph `zero negative`.

<a id="case-12"></a>

## 12. Adjacent layouts, `column` after `reset_layout`, empty layout

````markdown
Adjacent
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
a
<!-- column: 1 -->
b
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
c
<!-- column: 1 -->
d
<!-- reset_layout -->
<!-- column: 1 -->
after reset
<!-- end_slide -->
<!-- column_layout: [1, 1] -->
<!-- end_slide -->
<!-- jump_to_middle -->
<!-- transition: none -->
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Adjacent
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    a                                     b
    c                                     d

    after reset
```

- A second `column_layout` starts a new layout directly under the first, with no blank row.
- `column: 1` after `reset_layout` is ignored (no active layout); `after reset` is full width.
- An empty `column_layout` is not content: the second chunk (only an empty layout) is dropped, and so is the third (only `jump_to_middle`/`transition`): 1 slide total.

<a id="case-13"></a>

## 13. A setext title inside a column is still the slide title

````markdown
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
Title in column
===
text
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Title in column
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    text
```

- Title detection ignores column state: before any content, `Title in column`/`===` becomes the slide title and renders full width above the body. `text` goes to column 0.

<a id="case-14"></a>

## 14. `jump_to_middle`: prose is centered line by line

````markdown
Statement
===

<!-- jump_to_middle -->

A short line.

A somewhat longer second line.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===








                                   Statement
                                 ━━━━━━━━━━━━━

                                 A short line.

                         A somewhat longer second line.
```

- No top margin. The title is centered and its rule is title width + 4 (13 cells).
- Body is prose only, so each line is centered on its own.
- Vertical: (22 available - 4 head - 3 body) / 2 = 7 pad rows, then the head's own leading blank row, so the title is on row 9.

<a id="case-15"></a>

## 15. `jump_to_middle`: a list is centered as one block

````markdown
Listy
===

<!-- jump_to_middle -->

* one
* a longer item two
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===









                                     Listy
                                   ━━━━━━━━━

                              • one
                              • a longer item two
```

- A list line makes the body structured, so the body is shifted as one left-aligned block: offset (72 - 19) / 2 = 26, where 19 is the widest line (`• a longer item two`).

<a id="case-16"></a>

## 16. Speaker notes: single-line, multi-line, notes-only slide

````markdown
Notes
===

<!-- speaker_note: single line note -->
Visible text
<!-- speaker_note:
    First note line
      indented more
    last line -->
More visible
<!-- end_slide -->
<!-- speaker_note: a slide with only notes -->
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 2 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Notes
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Visible text More visible
=== slide 2 step 1/1 ===
```

- Notes never render. Parsed notes for slide 1: `single line note` and `First note line\n  indented more\nlast line` (common indent removed).
- Notes do not separate paragraphs: `Visible text More visible` is one paragraph.
- The second chunk has a note but no content or title. A slide with notes is kept, as a blank slide (a "pause and talk" slide): 2 slides.

<a id="case-17"></a>

## 17. A multi-line note without `-->` stops before the next directive and warns

````markdown
One
===

<!-- speaker_note: forgot to close
<!-- end_slide -->
Two
===
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: warning: line 4: speaker_note has no closing -->; the note ends before the next directive or at the end of the slide
deck.md: 2 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    One
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
=== slide 2 step 1/1 ===


    Two
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- The unclosed note ends on the line before `<!-- end_slide -->`, which still splits: 2 slides, and the note `forgot to close` belongs to slide 1.
- Loading warns with the line of the opening comment (stderr under `-check`/`-dump`, status line while presenting).
- Any line that is a complete directive (`<!-- pause -->`, `<!-- column: 1 -->`, ...) ends an unclosed note the same way and is still applied. Without any later directive the note would run to the end of the slide.

<a id="case-18"></a>

## 18. Image lines: missing file, width percentage, inline image

````markdown
Images
===

![Missing diagram](nope.png)

![image:width:50%](also-missing.png)

Inline ![x](nope.png) in prose.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Images
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

                          ╭──────────────────────────╮
                          │ [image: Missing diagram] │
                          ╰──────────────────────────╯

                         ╭───────────────────────────╮
                         │ [image: also-missing.png] │
                         ╰───────────────────────────╯

    Inline Image: x → /nope.png in prose.
```

- A missing or undecodable file renders as a 3-row dim box `[image: ALT]`, centered. It never fails the load or `-check`.
- With `image:width:N%` as alt, the label is the file name.
- An image that is not alone on its line is left to glamour, which prints it as link text.

<a id="case-19"></a>

## 19. Image sizing from free rows

````markdown
Image
===

Caption.

![Photo](photo.jpg)

After.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Image
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Caption.

                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>
                    <39 half-block cells>

    After.
```

- `photo.jpg` is 1200x800. Rows for the image: 22 available - 5 (top margin + title block) - 4 other body rows (`Caption.`, blank, blank, `After.`) = 13.
- Width: a cell is 1x2 pixels, so 13 rows fit 39 columns of a 3:2 image; it is centered in the 72-cell content width.
- Half-block rows are abbreviated here as `<N half-block cells>`.

<a id="case-20"></a>

## 20. `+exec` blocks render as code until run

````markdown
Exec
===

```python +exec
print("hi")
```

```+exec
echo no language
```

~~~bash +exec_replace
echo not executable
~~~
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Exec
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      print("hi")

      echo no language

      echo not executable
```

- The first two fences are exec blocks (ids 1 and 2; languages `python` and empty, which runs as bash). `+exec_replace` is not `+exec`, so the third is an ordinary code block.
- Without `-exec`, `-dump` and `-check` show no output; the code renders like any fence.

<a id="case-21"></a>

## 21. The code no-wrap limit (cw - 4)

````markdown
Code
===

```text
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: slide 1 overflows 80x24 by 0 rows, 3 cols   (exit 1)
```

```text
=== slide 1 step 1/1 ===


    Code
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

    xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

    xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    x
```

- At 80x24, cw = 72 and the code limit is 68. The 68-cell line keeps its 2-cell margin.
- The 69- and 71-cell lines lose the margin (they start at the content edge) and the block gains blank rows between lines; the 71-cell line also wraps at 70. Treat the limit as hard.
- `-check` reports the worst excess: 71 - 68 = 3 cols.

<a id="case-22"></a>

## 22. Titles never wrap

````markdown
A title that is far too long to fit on one line of an eighty column screen
===

Body.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: slide 1 overflows 80x24 by 0 rows, 2 cols   (exit 1)
```

```text
=== slide 1 step 1/1 ===


    A title that is far too long to fit on one line of an eighty column screen
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Body.
```

- The title is 74 cells; cw is 72. It is not wrapped; `-check` reports 74 - 72 = 2 cols. It stays visible here only because it spills into the right margin: the title starts at the 4-cell indent, so a title of 77 cells or more (3 more) would be cut at the screen edge.

<a id="case-23"></a>

## 23. Titles render inline markdown

````markdown
**Bold** and `code`
===

**Bold** and `code` in the body.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Bold and code
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Bold and  code  in the body.
```

- The title's `**bold**` and `` `code` `` are rendered, not printed: the markers vanish and the title is 13 cells wide, not 19. `-dump` strips colors, so the bold and the code background are not visible here. Unmatched markers stay literal.

<a id="case-24"></a>

## 24. List items wrap with a hanging indent

````markdown
Lists
===

1. An ordered item long enough that it has to wrap onto a second line at eighty
2. Short
* A bullet item that is also long enough to need a second line at eighty columns
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Lists
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    1. An ordered item long enough that it has to wrap onto a second line
       at eighty
    2. Short

    • A bullet item that is also long enough to need a second line at
      eighty columns
```

- Prose and list items wrap at cw - 2 = 70. Wrapped lines align under the item text, after `1. ` or `• `.
- Switching from `1.` to `*` starts a new list, which glamour separates with a blank row.

<a id="case-25"></a>

## 25. Tabs expand to 4 spaces

````markdown
Tabs
===

```text
	TAB
```

a	b
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Tabs
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

          TAB

    a    b
```

- Each tab becomes exactly 4 spaces (not a tab stop), in code and in prose.

<a id="case-26"></a>

## 26. An empty front matter block

````markdown
---
---
Body
===
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Body
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- `---` directly followed by `---` is an empty front matter block: no keys, no cover slide, and `Body` is the slide title.

<a id="case-27"></a>

## 27. Unknown transition names fall back to `slide` and warn

````markdown
---
transition: swipe
---
One
===
<!-- end_slide -->
<!-- transition: Dissolve -->
Two
===
<!-- end_slide -->
<!-- transition: fade -->
Three
===
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: warning: front matter: unknown transition "swipe", using slide (valid: slide, wipe, cover, dissolve, none)
deck.md: warning: line 11: unknown transition "fade", using slide (valid: slide, wipe, cover, dissolve, none)
deck.md: 3 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    One
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
=== slide 2 step 1/1 ===


    Two
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
=== slide 3 step 1/1 ===


    Three
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- `swipe` (front matter) and `fade` (directive, line 11) are not transition kinds: both act as `slide`, and each produces a warning. Directive warnings give the line; the `-transition` flag gets its own warning.
- Kinds are matched case-insensitively, so `Dissolve` is valid.
- Warnings never change the exit status: `-check` still exits 0.

<a id="case-28"></a>

## 28. Code line highlights are reveal steps, and bad specs warn

````markdown
Highlights
===

```rust {1-2|4,9|all}
fn main() {
    let a = 1;

    let b = 2;
}
```

After

```bash {2} +exec
echo one
echo two
```
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: warning: line 4: highlight "9" is past the end of the 5-line code block
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/3 ===


    Highlights
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      fn main() {
          let a = 1;

          let b = 2;
      }
=== slide 1 step 2/3 ===


    Highlights
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      fn main() {
          let a = 1;

          let b = 2;
      }
=== slide 1 step 3/3 ===


    Highlights
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

      fn main() {
          let a = 1;

          let b = 2;
      }

    After

      echo one
      echo two
```

- `{1-2|4,9|all}` makes the fence a highlight block with three groups, so the slide has 3 steps (`Highlights` has no pauses). `After` appears at the last step, when the code is fully revealed.
- Line 9 is past the end of the 5-line block: it is skipped with a warning and the group still highlights line 4. A warning's line number is the fence's.
- `{2}` on the `+exec` fence is a static highlight (one group, no extra step) and the block stays runnable. Highlight dimming is a color change, so it does not show in these text frames.

<a id="case-29"></a>

## 29. Subtitle and the frame title style

````markdown
Why caching **wins**
===
<!-- subtitle: measured over 30 days -->
<!-- title_style: frame -->

Body text.

<!-- end_slide -->

Curly falls back to the rule here
===
<!-- title_style: curly -->

Body text.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 2 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    ╭───────────────────────╮
    │ Why caching wins      │
    │ measured over 30 days │
    ╰───────────────────────╯

    Body text.
=== slide 2 step 1/1 ===


    Curly falls back to the rule here
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Body text.
```

- `frame` boxes the title and subtitle: 5 rows plus 1 for the subtitle, against 4 for the default. `-check` counts the box's width (text + 4) against the content width.
- `curly` draws a curly underline instead of the rule only where the presenter detected support (kitty, Ghostty, WezTerm, iTerm2). `-dump` and `-check` never do, so slide 2 shows the plain rule and is 4 rows either way.

<a id="case-30"></a>

## 30. Highlight, strikethrough, faint and underline marks

````markdown
Marks
===

Ship ==this week==, not ~~never~~.

Some [quiet]{.faint} and ++underlined++ and [wavy]{.curly} words, and a literal a == b and `==code==`.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Marks
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Ship this week, not never.

    Some quiet and underlined and wavy words, and a literal a == b
    and  ==code== .
```

- The markers vanish and cost no columns: the colors and underlines are escape sequences that `-dump` strips.
- `a == b` (spaces around `==`) and text inside a code span are left alone.

<a id="case-31"></a>

## 31. Callout boxes

````markdown
Callouts
===

> [!NOTE]
> A one-line note costs 3 rows.

> [!WARNING]
> Rotate the key **before** the deploy, or the new service cannot read the old data, and a long line wraps inside the box.

> A plain quote stays a quote.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: 1 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    Callouts
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    ╭─ Note ────────────────────────╮
    │ A one-line note costs 3 rows. │
    ╰───────────────────────────────╯

    ╭─ Warning ────────────────────────────────────────────────────────╮
    │ Rotate the key before the deploy, or the new service cannot read │
    │ the old data, and a long line wraps inside the box.              │
    ╰──────────────────────────────────────────────────────────────────╯

    │ A plain quote stays a quote.
```

- Each callout is its lines plus 2 border rows, with the type in the top border. The box is as wide as its widest line plus 4, up to the content width.
- Its text wraps 4 cells narrower than normal prose. An unknown type such as `[!BOGUS]`, or a quote without `[!TYPE]`, is an ordinary blockquote.

<a id="case-32"></a>

## 32. Big titles: block letters, their row cost and the width fallback

<!-- frames: full -->

````markdown
Big idea
===
<!-- title_size: 2 -->
<!-- subtitle: block letters, 3 rows -->

One short title, drawn large.

<!-- end_slide -->

This title is far too long for size three
===
<!-- title_size: 3 -->

Falls back to the normal title.

<!-- end_slide -->

Bad
===
<!-- title_size: huge -->

Unknown size.
````

```text
$ tuipresent -check -w 80 -h 24 deck.md
deck.md: warning: line 20: unknown title_size "huge" (valid: 1 to 3); using 1
deck.md: warning: slide 2: title_size 3 does not fit (it needs 458 columns but the slide is 72 wide); using the normal title
deck.md: 3 slides fit 80x24   (exit 0)
```

```text
=== slide 1 step 1/1 ===


    █▀▀▀▄ ▀▀█▀▀ ▄▀▀▀▀     ▀▀█▀▀ █▀▀▀▄ █▀▀▀▀ ▄▀▀▀▄
    █▀▀▀▄   █   █  ▀█       █   █   █ █▀▀   █▀▀▀█
    ▀▀▀▀  ▀▀▀▀▀  ▀▀▀      ▀▀▀▀▀ ▀▀▀▀  ▀▀▀▀▀ ▀   ▀
    block letters, 3 rows
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    One short title, drawn large.













━━━━━━━━━━━━━━━━━━━━━━━━━━──────────────────────────────────────────────────────
                                                                          1 / 3
=== slide 2 step 1/1 ===


    This title is far too long for size three
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Falls back to the normal title.
















━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━───────────────────────────
                                                                          2 / 3
=== slide 3 step 1/1 ===


    Bad
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

    Unknown size.
















━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                                                                          3 / 3
```

- `title_size: 2` replaces the 1-row title with 3 rows of block letters (+2 rows); size 3 uses 5 rows (+4). `-check` counts them, so a slide's body gets 2 or 4 fewer rows. A subtitle stays normal size and costs its usual row.
- A title that does not fit the content width (here 72 columns), or contains a character with no block letter (anything outside `A-Z`, `0-9`, space and basic punctuation, so CJK too), is drawn normally and `-check` and `-dump` warn with the slide number. The warning depends on `-w`, so it is not part of the deck's own warnings.
- `huge` is not a size: it warns with its line number and acts as `1`.
- Kitty draws the same rows as scaled text; `-check` and `-dump` always show these block letters.
