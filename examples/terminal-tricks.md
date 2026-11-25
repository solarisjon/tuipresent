---
title: Things your terminal can do in 2026
sub_title: Escape codes, protocols and a few party tricks
author: tuipresent examples
date: 2026
theme: dark
transition: slide
---

The terminal grew up
===

<!-- column_layout: [1, 1, 1] -->
<!-- column: 0 -->

## Draw

* 24-bit color
* Undercurl
* Inline images
* No-flicker frames

<!-- column: 1 -->

## Talk

* Hyperlinks
* Clipboard access
* Desktop notices
* Cwd reporting

<!-- column: 2 -->

## Listen

* Every key, exact
* Key releases
* Mouse pixels
* Focus in and out

<!-- reset_layout -->

All of it is plain bytes on stdout. Your program prints a sequence;
the emulator turns it into a feature.

<!-- end_slide -->

Escape codes are an API
===

```bash
# OSC 8: a clickable hyperlink
printf '\e]8;;https://example.com\e\\docs\e]8;;\e\\\n'

# OSC 52: put text on the system clipboard, even over ssh
printf '\e]52;c;%s\a' "$(printf 'copied!' | base64)"

# OSC 9: a desktop notification
printf '\e]9;build finished\a'

# 24-bit color: ESC [ 38 ; 2 ; R ; G ; B m
printf '\e[38;2;255;121;198mpink\e[0m\n'
```

`ESC ]` starts an operating-system command; `ESC [` starts a CSI.

<!-- end_slide -->

Two of those, live in this deck
===

* Links are [OSC 8 hyperlinks](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda): click one, or read its URL after the text where your terminal can't
* The tab shows how far through the talk you are, using `OSC 9;4`

```bash
# OSC 9;4: set the tab's progress bar to 40%, then clear it
printf '\e]9;4;1;40\e\\'
printf '\e]9;4;0;0\e\\'
```

<!-- pause -->

Turn the bar off with `-progress=false`, or `progress: false` in the front matter.

<!-- end_slide -->

Same idea in two more languages
===

```go
// Go: an OSC 8 hyperlink
func link(url, text string) string {
    return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
```

```rust
// Rust: a 24-bit foreground color
fn fg(r: u8, g: u8, b: u8) -> String {
    format!("\x1b[38;2;{r};{g};{b}m")
}
```

No library needed: a hyperlink or a color is a string concatenation.

<!-- end_slide -->

Is anybody listening?
===

Tools turn off color when stdout is not a terminal. Here, it isn't:

```python +exec
import os, sys
print("stdout is a tty:", os.isatty(1))
print("encoding:       ", sys.stdout.encoding)
```

<!-- pause -->

That is why `ls` loses its colors in a pipe, and why `--color=always`
and the `NO_COLOR` convention exist.

<!-- end_slide -->

One character, two cells
===

```python +exec
import unicodedata as u
for ch in "aé漢Ａ→":
    w = u.east_asian_width(ch)
    print(f"U+{ord(ch):04X}  {ch}  {w:<2}  {u.name(ch).title()}")
```

<!-- pause -->

`W` and `F` take two cells. Get this wrong and every column after
it is off by one: the classic broken table in a TUI.

<!-- end_slide -->

Pictures are escape codes too
===

<!-- column_layout: [2, 3] -->
<!-- column: 0 -->

![a terminal window](assets/terminal.png)

<!-- column: 1 -->

The kitty graphics protocol sends a PNG once, base64 in chunks:

```text
ESC _G a=T,U=1,f=100,i=42,
       c=24,r=6,q=2;iVBORw0K… ESC \
```

<!-- pause -->

Then every cell of the picture is the
character `U+10EEEE`, with the image id
in its foreground color. Text moves, so
the picture moves with it.

<!-- reset_layout -->

This slide uses it in kitty and ghostty, and half-blocks elsewhere.

<!-- end_slide -->

What happens when you press a key
===

```text
 ┌──────────┐  bytes  ┌──────────┐  bytes  ┌──────────┐
 │ keyboard ├────────>│ emulator ├────────>│   pty    │
 └──────────┘         └────┬─────┘         └────┬─────┘
                           ^                    │ read()
                           │ escape codes       v
                      ┌────┴─────┐         ┌──────────┐
                      │ renderer │<────────┤ your app │
                      └──────────┘  write  └──────────┘
```

* The app never sees your keyboard, only bytes from the pty
* Everything on screen is the app writing bytes back

<!-- end_slide -->

Legacy keys are ambiguous
===

| You press | App receives | Indistinguishable from |
|-----------|--------------|------------------------|
| Tab | `0x09` | Ctrl+I |
| Enter | `0x0d` | Ctrl+M |
| Backspace | `0x7f` | Ctrl+? |
| Esc | `0x1b` | the start of Alt+key |

<!-- pause -->

The kitty keyboard protocol fixes this. Opt in with `CSI > 1 u` and
every key arrives as `CSI code ; modifiers u`, releases included.

<!-- end_slide -->

Frames without flicker
===

```text
 without sync                     with mode 2026
 ┌───────────────┐                ┌───────────────┐
 │ half old      │  <- you see    │ old frame     │  <- you see
 │ half new  ▌   │     tearing    │               │     this...
 └───────────────┘                └───────────────┘
                                  ┌───────────────┐
                                  │ new frame     │  <- ...then
                                  │               │     this
                                  └───────────────┘
```

```bash
printf '\e[?2026h'   # begin synchronized update
draw_everything      # emulator buffers, does not paint
printf '\e[?2026l'   # end: paint the whole frame at once
```

<!-- end_slide -->

The shell is still a calculator
===

```bash +exec
echo "2^40         = $((2**40))"
printf 'hex 0xBEEF   = %d\n' 0xBEEF
echo "epoch now    = $(date -u +%s)"
```

<!-- pause -->

```sh +exec
seq 1 100 | awk '{ s += $1 } END { print "sum 1..100   =", s }'
```

<!-- end_slide -->

Try these tonight
===

* Print a hyperlink with the OSC 8 `printf` from slide 3
* Look up kitty keys and mode 2026 in your emulator's docs
* Run `python3 -c 'print("漢")'` and count the cells
* Pipe a colorful tool through `cat` and watch the color vanish

<!-- pause -->

The terminal is a protocol. Learn the protocol.
