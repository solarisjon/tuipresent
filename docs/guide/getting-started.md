# Getting started

This page takes you from a fresh checkout to presenting your own three-slide deck in about five minutes. It assumes you know Go and markdown but have never used tuipresent.

## 1. Build it

You need Go 1.27 or newer. From the root of a checkout of the repository:

```sh
go build -o bin/tuipresent ./cmd/tuipresent
```

That leaves the binary at `bin/tuipresent`. If you'd rather have it on your `PATH`, run `go install ./cmd/tuipresent` instead.

Check it works by opening the feature tour that ships with the repo:

```sh
./bin/tuipresent examples/tour.md
```

Press `space` to move forward and `q` to quit.

### Which terminal

Any truecolor terminal works. kitty and ghostty are the best fit: images are drawn at full resolution there. Other terminals, and anything running inside tmux, draw images with colored half-block characters instead. Everything else (transitions, fades, colors) works the same everywhere.

## 2. Write a first deck

A deck is one markdown file. Create `hello.md` and build it up in three steps.

### Step 1: front matter and a cover

```markdown
---
title: Hello, terminal
sub_title: My first tuipresent deck
author: Your Name
---
```

The YAML block must start on the very first line. Because it has a `title`, tuipresent generates a cover slide from it: the title in large spaced capitals, the subtitle under it, and the author at the bottom. You don't write the cover yourself.

### Step 2: a slide with a title and a reveal

Add this below the front matter:

```markdown
Why slides in a terminal?
===

* You already live here
<!-- pause -->
* Code runs where you show it
<!-- pause -->
* Markdown diffs nicely in review

<!-- end_slide -->
```

Three things are going on:

- A line underlined with `===` is the **slide title**. It must be the first thing on the slide.
- `<!-- pause -->` splits the slide into **steps**. Each press of `space` reveals the next part, which fades in.
- `<!-- end_slide -->` ends the slide. Slides are separated only by this comment, not by `---`.

### Step 3: a centered closing slide

Add a final slide:

```markdown
Questions?
===

<!-- jump_to_middle -->

Thanks for watching.
```

`<!-- jump_to_middle -->` centers the slide on the screen, which suits one-line statements and endings.

### The whole file

```markdown
---
title: Hello, terminal
sub_title: My first tuipresent deck
author: Your Name
---

Why slides in a terminal?
===

* You already live here
<!-- pause -->
* Code runs where you show it
<!-- pause -->
* Markdown diffs nicely in review

<!-- end_slide -->

Questions?
===

<!-- jump_to_middle -->

Thanks for watching.
```

That's three slides: the generated cover, the bullet slide and the closing slide.

## 3. Check it fits

Before presenting, ask tuipresent whether every slide fits the screen:

```sh
./bin/tuipresent -check hello.md
# hello.md: 3 slides fit 100x30
```

Check the smallest screen you might present on, too. 80x24 is a good worst case:

```sh
./bin/tuipresent -check -w 80 -h 24 hello.md
```

If a slide is too tall or too wide, `-check` names it and exits with status 1. To see exactly what each slide looks like without opening a full-screen session, use `-dump`, which prints every slide at every pause step:

```sh
./bin/tuipresent -dump hello.md | less -R
```

## 4. Present it

```sh
./bin/tuipresent hello.md
```

The essential keys:

| Key | Action |
|-----|--------|
| `space`, `→`, `enter` | Next step or slide |
| `←`, `backspace` | Previous step or slide |
| `g` / `G` | First / last slide |
| `q`, `esc` | Quit |

The full list is in the [reference](reference.md#key-bindings).

Leave the deck open and edit `hello.md` in another window: the slides reload as soon as you save, and you stay on the same slide.

## Where to go next

- [Authoring](authoring.md): columns, images, runnable code, speaker notes, transitions, and advice on designing slides that read well in a terminal.
- [Presenting](presenting.md): a speaker-notes window in a split pane, live demos, checking the projector size, troubleshooting.
- [Reference](reference.md): every flag, key, directive and layout number in tables.
- [`examples/`](../../examples/README.md): four complete decks to copy from.
