#!/usr/bin/env python3
"""Regenerate docs/agent/EXAMPLES.md from the cases below.

Run from the repo root after `go build -o bin/tuipresent ./cmd/tuipresent`:

    python3 docs/agent/gen_examples.py

Every result block in EXAMPLES.md is real output of `-dump` and `-check` at
80x24. This script never passes -exec, so no deck code is run.
"""

import os
import re
import shutil
import subprocess
import sys
import tempfile

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
BIN = os.path.join(ROOT, "bin", "tuipresent")
OUT = os.path.join(ROOT, "docs", "agent", "EXAMPLES.md")
PHOTO = os.path.join(ROOT, "examples", "assets", "mt-evans-goat.jpg")
W, H = 80, 24

ANSI = re.compile(r"\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b\[[0-9;:]*[A-Za-z]")
HALF = re.compile(r"^( *)([▀▄ ]*[▀▄][▀▄ ]*)$")

# Each case: (id, title, deck source, notes). Notes state what the output shows.
CASES = [
    ("slides", "Slide splitting and the cover slide",
     """---
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
""",
     ["3 slides: the cover (from `title`), `One`, `Two`.",
      "The empty chunk between the two consecutive `end_slide` lines and the empty chunk after the last one are dropped.",
      "A slide with only a title (`Two`) is kept.",
      "Footer: ` Demo  ·  Ann` left, `N / 3` right."]),

    ("titles", "Setext title vs ATX heading",
     """Setext title
===

Body.
<!-- end_slide -->
# ATX heading

Body.
""",
     ["Slide 1: `Setext title` is the slide title: blank row, title, full-width rule, blank row.",
      "Slide 2: `# ATX heading` is an in-body heading. No rule, and only the 1-row top margin above it."]),

    ("late-title", "Setext underline after other content is not a title",
     """Intro line.

Late
===

Body.
""",
     ["`Intro line.` is content, so `Late`/`===` is rendered by glamour as an in-body H1. The slide has no title and no rule."]),

    ("fence-end-slide", "`end_slide` inside a fence is code",
     """Showing source
===

```markdown
<!-- end_slide -->
```
<!-- end_slide -->
Next
===
""",
     ["2 slides. The `end_slide` inside the fence is printed as code; the one after the closing fence splits."]),

    ("unclosed-fence", "An unclosed fence swallows later `end_slide`s",
     """One
===

```text
never closed
<!-- end_slide -->
Two
===
""",
     ["1 slide. Everything after the opening fence, including `end_slide` and the would-be title `Two`, is code."]),

    ("unknown-directives", "Unknown and malformed directives vanish",
     """Directives
===

<!-- colum: 1 -->
<!-- Pause -->
<!-- incremental_lists: true -->
<!-- my-note -->
visible
<!-- end_slide: anything -->
second slide
""",
     ["Typo `colum`, uppercase `Pause`, unsupported `incremental_lists` and the hyphenated `my-note` are not directives (or are unknown) and render as nothing. The slide has 1 step.",
      "`end_slide` ignores its argument: `<!-- end_slide: anything -->` still splits, giving 2 slides."]),

    ("pause-merge", "Pauses without blank lines merge paragraphs; a trailing pause adds an empty step",
     """Pauses
===

A
<!-- pause -->
B

<!-- pause -->

C
<!-- pause -->
""",
     ["4 steps (3 pauses).",
      "Step 2 shows `A B` on one line: the blocks on either side of a pause are rejoined with a single newline before rendering, so `A` and `B` form one paragraph.",
      "`C` is a separate paragraph because blank lines surround its pause.",
      "Step 4 is identical to step 3: the trailing pause reveals nothing."]),

    ("pause-first", "A pause before all content; the title is never paused",
     """<!-- pause -->
Title
===

Body.
""",
     ["2 steps. The title is part of the slide, not a block, so it shows at step 1; `Body.` appears at step 2.",
      "Directives before the setext title do not stop it being the title."]),

    ("columns-above-below", "Content before the first `column`, and after `reset_layout`",
     """Columns
===

<!-- column_layout: [1, 1] -->
Above the columns.
<!-- column: 0 -->
Left.
<!-- column: 1 -->
Right.
<!-- reset_layout -->
Below the columns.
""",
     ["`Above the columns.` sits full width directly on top of the columns (no blank row).",
      "Column widths at 80x24: (72 - 4) / 2 = 34 each, 4-cell gap: `Right.` starts 38 cells right of `Left.`.",
      "`Below the columns.` is full width after one blank row."]),

    ("columns-pause", "A pause inside a column delays every later column",
     """Columns and pauses
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
L1

<!-- pause -->

L2
<!-- column: 1 -->
R1
""",
     ["2 steps. Step counters are per slide, not per column: `R1` comes after the pause in column 0, so it has step 1 and appears only at step 2, together with `L2`.",
      "At step 1 the layout already reserves `L2`'s rows (column height is the fully revealed height)."]),

    ("columns-clamp", "Width parsing, out-of-range and non-numeric `column` indices",
     """Clamping
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
""",
     ["`[2, x, 0, 1]` keeps only positive integers: widths `[2, 1]` = 45 and 22 cells at 80x24.",
      "`column: 7` clamps to the last column (1). `column: two` is ignored, so `still seven` keeps going to column 1. `column: -3` clamps to 0.",
      "`zero` and `negative` are consecutive blocks in column 0 with no blank line between them, so they render as one paragraph `zero negative`."]),

    ("columns-adjacent", "Adjacent layouts, `column` after `reset_layout`, empty layout",
     """Adjacent
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
""",
     ["A second `column_layout` starts a new layout directly under the first, with no blank row.",
      "`column: 1` after `reset_layout` is ignored (no active layout); `after reset` is full width.",
      "An empty `column_layout` still counts as content: slide 2 is a blank slide. A chunk of only `jump_to_middle`/`transition` is dropped: 2 slides total."]),

    ("title-in-column", "A setext title inside a column is still the slide title",
     """<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
Title in column
===
text
""",
     ["Title detection ignores column state: before any content, `Title in column`/`===` becomes the slide title and renders full width above the body. `text` goes to column 0."]),

    ("centered-prose", "`jump_to_middle`: prose is centered line by line",
     """Statement
===

<!-- jump_to_middle -->

A short line.

A somewhat longer second line.
""",
     ["No top margin. The title is centered and its rule is title width + 4 (13 cells).",
      "Body is prose only, so each line is centered on its own.",
      "Vertical: (22 available - 4 head - 3 body) / 2 = 7 pad rows, then the head's own leading blank row, so the title is on row 9."]),

    ("centered-list", "`jump_to_middle`: a list is centered as one block",
     """Listy
===

<!-- jump_to_middle -->

* one
* a longer item two
""",
     ["A list line makes the body structured, so the body is shifted as one left-aligned block: offset (72 - 19) / 2 = 26, where 19 is the widest line (`• a longer item two`)."]),

    ("notes", "Speaker notes: single-line, multi-line, notes-only slide",
     """Notes
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
""",
     ["Notes never render. Parsed notes for slide 1: `single line note` and `First note line\\n  indented more\\nlast line` (common indent removed).",
      "Notes do not separate paragraphs: `Visible text More visible` is one paragraph.",
      "The second chunk has notes but no content or title, so it is dropped with its note: 1 slide."]),

    ("unclosed-note", "A multi-line note without `-->` runs to the next line containing `-->`",
     """One
===

<!-- speaker_note: forgot to close
<!-- end_slide -->
Two
===
""",
     ["The note ends at the `-->` of the `end_slide` line, so that `end_slide` is consumed as note text.",
      "1 slide. `Two`/`===` lands in slide `One`'s body as an in-body H1 (the slide already has a title)."]),

    ("image-missing", "Image lines: missing file, width percentage, inline image",
     """Images
===

![Missing diagram](nope.png)

![image:width:50%](also-missing.png)

Inline ![x](nope.png) in prose.
""",
     ["A missing or undecodable file renders as a 3-row dim box `[image: ALT]`, centered. It never fails the load or `-check`.",
      "With `image:width:N%` as alt, the label is the file name.",
      "An image that is not alone on its line is left to glamour, which prints it as link text."]),

    ("image-sizing", "Image sizing from free rows",
     """Image
===

Caption.

![Photo](photo.jpg)

After.
""",
     ["`photo.jpg` is 1200x800. Rows for the image: 22 available - 5 (top margin + title block) - 4 other body rows (`Caption.`, blank, blank, `After.`) = 13.",
      "Width: a cell is 1x2 pixels, so 13 rows fit 39 columns of a 3:2 image; it is centered in the 72-cell content width.",
      "Half-block rows are abbreviated here as `<N half-block cells>`."]),

    ("exec", "`+exec` blocks render as code until run",
     """Exec
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
""",
     ["The first two fences are exec blocks (ids 1 and 2; languages `python` and empty, which runs as bash). `+exec_replace` is not `+exec`, so the third is an ordinary code block.",
      "Without `-exec`, `-dump` and `-check` show no output; the code renders like any fence."]),

    ("code-limit", "The code no-wrap limit (cw - 4)",
     """Code
===

```text
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```
""",
     ["At 80x24, cw = 72 and the code limit is 68. The 68-cell line keeps its 2-cell margin.",
      "The 69- and 71-cell lines lose the margin (they start at the content edge) and the block gains blank rows between lines; the 71-cell line also wraps at 70. Treat the limit as hard.",
      "`-check` reports the worst excess: 71 - 68 = 3 cols."]),

    ("title-too-long", "Titles never wrap",
     """A title that is far too long to fit on one line of an eighty column screen
===

Body.
""",
     ["The title is 74 cells; cw is 72. It is not wrapped; `-check` reports 74 - 72 = 2 cols. It stays visible here only because it spills into the right margin; at 2 more cells it would be cut at the screen edge."]),

    ("title-literal", "Title text is literal, not markdown",
     """**Bold** and `code`
===

**Bold** and `code` in the body.
""",
     ["The slide title is printed verbatim (asterisks and backticks included) in the title gradient. Inline markdown only renders in the body."]),

    ("hanging-indent", "List items wrap with a hanging indent",
     """Lists
===

1. An ordered item long enough that it has to wrap onto a second line at eighty
2. Short
* A bullet item that is also long enough to need a second line at eighty columns
""",
     ["Prose and list items wrap at cw - 2 = 70. Wrapped lines align under the item text, after `1. ` or `• `.",
      "Switching from `1.` to `*` starts a new list, which glamour separates with a blank row."]),

    ("tabs", "Tabs expand to 4 spaces",
     "Tabs\n===\n\n```text\n\tTAB\n```\n\na\tb\n",
     ["Each tab becomes exactly 4 spaces (not a tab stop), in code and in prose."]),

    ("empty-front-matter", "An empty front matter block is not recognized",
     """---
---
Body
===
""",
     ["`---` directly followed by `---` is not parsed as front matter (the closing `---` must follow at least one line). Both lines render as horizontal rules, which counts as content, so `Body` is not a title either."]),
]


def strip(s):
    return ANSI.sub("", s)


def frames(dump):
    out, cur = [], None
    for line in dump.split("\n"):
        if line.startswith("=== slide "):
            cur = [line]
            out.append(cur)
        elif cur is not None:
            cur.append(line)
    return out


def excerpt(frame):
    head, body = frame[0], frame[1:H + 1]
    body = body[:H - 2]  # drop the 2-row footer
    body = [l.rstrip() for l in body]
    while body and body[-1] == "":
        body.pop()
    lines = [head]
    for l in body:
        m = HALF.match(l)
        if m:
            l = "%s<%d half-block cells>" % (m.group(1), len(m.group(2)))
        lines.append(l)
    return lines


def run(dirpath, src):
    with open(os.path.join(dirpath, "deck.md"), "w") as f:
        f.write(src)
    if "photo.jpg" in src:
        shutil.copy(PHOTO, os.path.join(dirpath, "photo.jpg"))
    d = subprocess.run([BIN, "-dump", "-w", str(W), "-h", str(H), "deck.md"],
                       cwd=dirpath, capture_output=True, text=True)
    c = subprocess.run([BIN, "-check", "-w", str(W), "-h", str(H), "deck.md"],
                       cwd=dirpath, capture_output=True, text=True)
    check = (c.stdout + c.stderr).strip() + "   (exit %d)" % c.returncode
    return strip(d.stdout), check


HEADER = """# tuipresent language: verified examples

Minimal pairs for [LANGUAGE.md](LANGUAGE.md). Every result below is real output,
generated by `docs/agent/gen_examples.py` (do not edit by hand; edit the script
and rerun it after `go build -o bin/tuipresent ./cmd/tuipresent`).

Conventions for each result:

- `-check` line: `tuipresent -check -w 80 -h 24 deck.md` and its exit status.
- Frames: `tuipresent -dump -w 80 -h 24 deck.md` with ANSI escapes stripped,
  trailing spaces removed, the 2-row footer omitted (except in case 1), and
  trailing blank rows dropped. Leading blank rows are kept: they are the top
  margin or vertical centering. Each frame starts with its
  `=== slide N step k/n ===` header.
- At 80x24: content width cw = 72, left indent 4, 22 rows above the footer.
- No case passes `-exec`; no deck code runs.

"""


def main():
    if not os.path.exists(BIN):
        sys.exit("build first: go build -o bin/tuipresent ./cmd/tuipresent")
    parts = [HEADER, "## Index\n\n"]
    for i, (cid, title, _, _) in enumerate(CASES, 1):
        parts.append("%d. [%s](#%s)\n" % (i, title, "case-%d" % i))
    parts.append("\n")
    with tempfile.TemporaryDirectory() as tmp:
        for i, (cid, title, src, notes) in enumerate(CASES, 1):
            d = os.path.join(tmp, cid)
            os.mkdir(d)
            dump, check = run(d, src)
            fr = frames(dump)
            parts.append('<a id="case-%d"></a>\n\n## %d. %s\n\n' % (i, i, title))
            parts.append("````markdown\n%s````\n\n" % (src if src.endswith("\n") else src + "\n"))
            parts.append("```text\n$ tuipresent -check -w 80 -h 24 deck.md\n%s\n```\n\n" % check)
            body = []
            for f in fr:
                if i == 1:
                    lines = [f[0]] + [l.rstrip() for l in f[1:H + 1]]
                else:
                    lines = excerpt(f)
                body.append("\n".join(lines))
            parts.append("```text\n%s\n```\n\n" % "\n".join(body))
            for n in notes:
                parts.append("- %s\n" % n)
            parts.append("\n")
    with open(OUT, "w") as f:
        f.write("".join(parts).rstrip("\n") + "\n")
    print("wrote", OUT)


if __name__ == "__main__":
    main()
