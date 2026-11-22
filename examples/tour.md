---
title: tuipresent in 5 minutes
sub_title: Every slide shows a feature and the markdown behind it
author: tuipresent examples
date: 2026
theme: dark
transition: slide
---

Front matter makes the cover
===

<!-- transition: slide -->

The slide you just left was generated from this YAML block, which
must start on the very first line of the file:

```yaml
---
title: tuipresent in 5 minutes
sub_title: Every slide shows a feature and the markdown behind it
author: tuipresent examples
date: 2026
theme: dark          # or tokyonight-storm, dracula, light
transition: slide    # deck-wide default
---
```

<!-- pause -->

Title and author also feed the footer below. Slides are separated
by an `end_slide` comment on its own line.

<!-- end_slide -->

Titles, text and lists
===

<!-- transition: wipe -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Result

**Bold** for emphasis, *italic* for
asides, `code` for names.

* Short phrases
* One idea per slide
* Numbers beat adjectives

<!-- column: 1 -->

## Source

```markdown
Titles, text and lists
===

**Bold** for emphasis,
*italic* for asides,
`code` for names.

* Short phrases
* One idea per slide
* Numbers beat adjectives
```

<!-- reset_layout -->

A setext `===` heading before any other content becomes the title.

<!-- end_slide -->

Pauses reveal one step at a time
===

<!-- transition: dissolve -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Source

```markdown
1. Measure first
<!-- pause -->
2. Change one thing
<!-- pause -->
3. Measure again
```

<!-- column: 1 -->

## Result

1. Measure first

<!-- pause -->

2. Change one thing

<!-- pause -->

3. Measure again

<!-- reset_layout -->

Press space to step. The source sits in the left column because a
pause delays every column after it.

<!-- end_slide -->

Columns split the screen
===

<!-- transition: cover -->
<!-- column_layout: [3, 2] -->
<!-- column: 0 -->

This slide is a `[3, 2]` layout. Its source, trimmed:

```markdown
<!-- column_layout: [3, 2] -->
<!-- column: 0 -->
This slide is a ...
<!-- column: 1 -->
* Widths are relative
<!-- reset_layout -->
Back to full width.
```

<!-- column: 1 -->

* Widths are relative
* Columns are 0-based
* 4-cell gutter
* Any markdown inside

<!-- reset_layout -->

Back to full width after `reset_layout`.

<!-- end_slide -->

Code gets syntax highlighting
===

<!-- transition: slide -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Result

```go
func fib(n int) int {
    if n < 2 {
        return n
    }
    return fib(n-1) + fib(n-2)
}
```

<!-- column: 1 -->

## Source

````markdown
```go
func fib(n int) int {
    if n < 2 {
        return n
    }
    return fib(n-1) + fib(n-2)
}
```
````

<!-- reset_layout -->

Any language glamour knows: go, rust, python, sql, yaml, diff...

<!-- end_slide -->

Add +exec and press e
===

<!-- transition: none -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Source

````markdown
```bash +exec
date "+%a %d %b %Y %H:%M"
uname -s
```

```python +exec
import math
print(math.comb(52, 5))
```
````

<!-- column: 1 -->

## Live

```bash +exec
date "+%a %d %b %Y %H:%M"
uname -s
```

```python +exec
import math
print(math.comb(52, 5))
```

<!-- reset_layout -->

Press `e`: both blocks run and their output appears below them.

<!-- end_slide -->

Charts are just text
===

<!-- transition: wipe -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Result

```text
Mon  ███████▌        7.5
Tue  ████████████   12.0
Wed  █████████▏      9.2
Thu  ██████████████ 14.1
Fri  ████▍           4.4
```

Commits per day, last week.

<!-- column: 1 -->

## Source

````markdown
```text
Mon  ███████▌        7.5
Tue  ████████████   12.0
Wed  █████████▏      9.2
Thu  ██████████████ 14.1
Fri  ████▍           4.4
```
````

<!-- reset_layout -->

A `text` fence keeps alignment and skips highlighting. Tables work too.

<!-- end_slide -->

<!-- jump_to_middle -->
<!-- transition: dissolve -->

Say one thing. Loudly.
===

A centered statement slide, made with:

```markdown
<!-- jump_to_middle -->
```

<!-- end_slide -->

Five ways to arrive
===

<!-- transition: cover -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

| Kind | Used on |
|------|---------|
| slide | code, front matter |
| wipe | text, charts |
| cover | columns, this one |
| dissolve | pauses, statement |
| none | the exec slide |

<!-- column: 1 -->

```markdown
<!-- transition: cover -->
```

Per slide, it overrides the front
matter default. Going backwards,
the departing slide's kind wins.

<!-- reset_layout -->

Try `-transition dissolve` on the command line to change the default.

<!-- end_slide -->

Check it before you present it
===

<!-- transition: slide -->

```bash
go build -o bin/tuipresent ./cmd/tuipresent
./bin/tuipresent -check -w 80 -h 24 examples/tour.md
./bin/tuipresent -dump -w 80 -h 24 examples/tour.md | less -R
./bin/tuipresent -slide 7 examples/tour.md
```

* `-check` reports any slide that overflows the screen
* `-dump` prints every frame, no terminal needed
* `-slide N` starts on slide N; `r` reloads after edits

<!-- pause -->

That's the tour. Press `q` to quit.
