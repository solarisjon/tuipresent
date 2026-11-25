# Presenting

Everything for the day of the talk: driving the deck, a private speaker-notes window, editing live, running demos, making sure the deck fits the projector, and fixing the usual problems. For writing decks, see [Authoring](authoring.md).

## Contents

- [Driving the deck](#driving-the-deck)
- [Speaker notes in a second pane](#speaker-notes-in-a-second-pane)
- [Editing while it runs](#editing-while-it-runs)
- [Running code live](#running-code-live)
- [Fitting the projector](#fitting-the-projector)
- [Before-the-talk checklist](#before-the-talk-checklist)
- [Troubleshooting](#troubleshooting)

## Driving the deck

```sh
tuipresent deck.md            # start at the beginning
tuipresent -slide 7 deck.md   # resume at slide 7 (the cover counts as slide 1)
```

| Key | Action |
|-----|--------|
| `space` `→` `l` `n` `j` `↓` `enter` `pgdown` | Next step, or next slide when the slide is fully revealed |
| `←` `h` `p` `k` `↑` `backspace` `pgup` | Previous step, or previous slide (arriving fully revealed) |
| `g` `home` | First slide |
| `G` `end` | Last slide |
| `o` | Open or close the slide overview grid |
| `0`-`9` then `enter` | Jump to that slide number (see below) |
| `e` | Run the visible `+exec` blocks on this slide |
| `r` | Reload the deck from disk |
| `q` `esc` `ctrl+c` | Quit |

### Jump to a slide by number

Type the slide number and press `enter`: `12` then `enter` goes to slide 12 (the cover counts as slide 1, like `-slide`). The number you are typing shows on the status line. `backspace` erases a digit, `esc` cancels without quitting, and any other key cancels the number and does its normal job. Numbers past the end go to the last slide. Jumping lands on the slide's first step.

### Slide overview

`o` replaces the slide with a grid of cards, one per slide, each showing the slide number and title (the cover shows the deck title). The current slide is selected and highlighted.

| Key | Action |
|-----|--------|
| `←` `h` / `→` `l` | Move the selection |
| `↑` `k` / `↓` `j` | Move up / down a row |
| `pgup` / `pgdown` | Move a page of rows |
| `g` `home` / `G` `end` | First / last slide |
| `enter` `space` | Jump to the selected slide and close the overview |
| `o` `esc` | Close the overview, staying on the current slide |
| `q` `ctrl+c` | Quit |

The grid fits as many columns as the width allows (4 at 100 columns, 3 at 80) and shows a page of rows at a time, scrolling as the selection moves. On a screen too short for cards it shows one line per slide.

### Mouse

The mouse is on by default (turn it off with `-mouse=false`, which also gives the terminal back its text selection; most terminals let you hold `shift` to select anyway). Only the left button and the wheel do anything.

| Mouse | Action |
|-------|--------|
| Click the left third of the screen | Back (like `←`) |
| Click the right third | Forward (like `→`) |
| Click the middle third | Nothing, so a stray click on the speaker's screen is harmless |
| Wheel up / down | Back / forward |
| In the overview: click a card | Jump to that slide |
| In the overview: wheel | Move the selection a row up or down |

Mouse reporting is switched on at start and off again on exit, including after `ctrl+c`.

Presentation remotes ("clickers") usually send `pgdown`/`pgup` or the arrow keys, so they work without setup.

The footer shows a progress bar, the deck title and author, and `N / total`. It disappears in terminals under 6 rows tall.

## Speaker notes in a second pane

Run the deck as usual, and run the notes view for the **same file** anywhere else:

```sh
tuipresent deck.md           # the slides, where the audience can see them
tuipresent -notes deck.md    # the notes, where only you can see them
```

The notes view follows the presenter automatically. It shows:

- the current slide's title, `slide N/M`, the reveal step and the time since the presenter started,
- that slide's notes (from `speaker_note` comments; see [Authoring](authoring.md#speaker-notes)),
- the title of the next slide.
- any warnings about the deck or its theme (the same ones the status line and `-check` give), in yellow above the bottom, for as long as they apply.

Either one can be started first; the notes view says "Waiting for the presenter" until the slides are running, and goes back to waiting when they quit. Press `q` to close it. The notes view can't drive the slides; use the presenter window for that.

### Where to put the notes

There are two setups, depending on what the audience sees:

- **Separate OS windows** (the usual case for a real talk). The audience sees a projector or a shared window. Put the slides window there and the notes window on your laptop screen. Open a second window with `ctrl+shift+n` in kitty, `cmd+n` (macOS) or `ctrl+shift+n` (Linux) in ghostty.
- **Split panes** (rehearsing, or authoring with notes beside the slides). Both panes are in one window, so don't share that window with an audience.

### kitty splits

kitty arranges its windows with layouts. Enable the splits layout and add keys for splitting in `~/.config/kitty/kitty.conf`:

```conf
enabled_layouts splits,stack
map f5 launch --location=hsplit --cwd=current
map f6 launch --location=vsplit --cwd=current
```

Reload the config (or restart kitty), press `f6` to split side by side, and run `tuipresent -notes deck.md` in the new pane. `ctrl+shift+]` and `ctrl+shift+[` move between panes; `ctrl+shift+l` cycles layouts, which with the config above toggles to `stack` (one pane fills the window) and back.

### ghostty splits

ghostty has native splits with default keys:

| Action | macOS | Linux |
|--------|-------|-------|
| Split to the right | `cmd+d` | `ctrl+shift+o` |
| Split downward | `cmd+shift+d` | `ctrl+shift+e` |

Split, then run `tuipresent -notes deck.md` in the new pane. If you've rebound these, the actions are named `new_split:right` and `new_split:down`.

## Editing while it runs

The presenter reloads the deck whenever the file changes (it checks every 0.4 s, so editors that save by writing a new file and renaming it work too). You stay on the same slide and step.

- If your edit breaks the deck (for example invalid front matter), the old version stays on screen and the error appears in the status line at the bottom. Fix it and save again.
- If you delete slides you were on, you land on the last one that still exists.
- `r` forces a reload. `-watch=false` turns automatic reloading off, which is worth doing during the real talk if you might save by accident.
- The notes view reloads on its own when the file changes, so edited notes appear immediately.
- Reloading clears any `+exec` output on screen.

A comfortable authoring setup: editor in one pane, `tuipresent deck.md` in another, and `tuipresent -notes deck.md` in a third if you're writing notes.

## Running code live

Slides with `+exec` code blocks (see [Authoring](authoring.md#runnable-code-exec)) do nothing until you press `e`. Then every runnable block visible on the slide runs, "running…" shows at the bottom, and the output appears under the code.

- Each run has a 15-second limit.
- Code runs in the deck file's directory, wherever you started `tuipresent` from. Keep demo files next to the deck.
- Read every block before the talk. You're running it on your machine, in front of people.
- Do a dry run in the terminal you'll present from, on the network you'll have (or without one).

To see how the output fits before the talk, you can run the blocks during a check. This executes the deck's code, so only do it for decks you trust:

```sh
tuipresent -exec -check -w 80 -h 24 deck.md
```

## Fitting the projector

Text in a terminal is measured in cells, and a projector usually means a bigger font, which means fewer cells. Find out the size you'll really have, then check the deck against it.

1. Open the terminal on the projector (or at the font size you'll use) and run:

   ```sh
   echo "$(tput cols)x$(tput lines)"
   ```

2. Check the deck at that size:

   ```sh
   tuipresent -check -w 96 -h 27 deck.md
   ```

   It prints `deck.md: 12 slides fit 96x27`, or names each slide that overflows and by how many rows and columns, and exits with status 1.

3. To see a problem slide, dump it at that size and look:

   ```sh
   tuipresent -dump -w 96 -h 27 deck.md | less -R
   ```

If you don't know the projector in advance, make the deck pass at 80x24. That's the common worst case, and `-check -w 80 -h 24` covers it.

The slides window resizes live, so you can also just resize and step through the deck.

## Before-the-talk checklist

- `tuipresent -check deck.md` and `tuipresent -check -w 80 -h 24 deck.md` both pass.
- Step through the whole deck once in the terminal and font size you'll use.
- Run every `+exec` block once.
- Images show in full resolution (or you're happy with the block fallback).
- Notes window is on your screen, not the shared one.
- Consider `-watch=false` so a stray save can't change the deck mid-talk.

## Troubleshooting

**Images look blocky or pixelated.** You're seeing the half-block fallback. Full-resolution images need kitty, ghostty, iTerm2, WezTerm or foot, outside tmux and screen. tuipresent detects these from `TERM`, `TERM_PROGRAM`, `LC_TERMINAL`, `ITERM_SESSION_ID`, `KITTY_WINDOW_ID` and `GHOSTTY_RESOURCES_DIR`; if detection misses (for example over SSH with an unusual `TERM`), force it with `-images kitty`, `-images iterm2` or `-images sixel`. Over SSH it works as long as the local terminal supports the protocol. (`LC_TERMINAL` is how iTerm2 announces itself over SSH, if your SSH config passes it.)

**An image shows as `[image: ...]`.** The file couldn't be found or decoded. Paths are relative to the deck file, not the current directory. Only PNG, JPEG and GIF are supported. Also check you didn't pass `-images off`.

**Images are blank or garbled.** You may have forced `-images kitty`, `iterm2` or `sixel` in a terminal that doesn't support that protocol, or inside tmux. Use `-images auto` (the default) or `-images blocks`.

**Colors look wrong or washed out.** tuipresent needs a truecolor terminal. If your terminal supports it but colors are still off, it's probably not being advertised: try `COLORTERM=truecolor tuipresent deck.md`. In tmux, also enable truecolor, for example `set -as terminal-features ',*:RGB'` in `~/.tmux.conf` (tmux 3.2 or newer).

**The pause fade flashes or looks muddy.** The fade blends from your terminal's background color, which tuipresent asks the terminal for. Terminals that don't answer get the theme's background instead, which may not match. Turn the fade off with `-fade=false`.

**Links print their URL, or the tab shows no progress bar.** Links are clickable only where the terminal supports OSC 8, and the tab progress bar needs OSC 9;4; otherwise the text is followed by its dimmed URL and no bar is drawn. Force links with `-links on`; stop the bar with `-progress=false` or `progress: false`.

**Animations stutter.** Common over slow SSH links or screen shares. Use `-transition none` and `-fade=false`.

**Inside tmux.** Everything works, but images fall back to half-blocks, and colors need the truecolor setting above. For the best result, present outside tmux.

**Content is cut off.** The terminal is smaller than the deck was designed for. Slides are clipped at the bottom and right without a warning. Make the font smaller or the window bigger, or fix the deck after running `-check` at that size ([Fitting the projector](#fitting-the-projector)).

**Very small terminals.** Under 6 rows the footer is hidden. Under 28 columns, the content is at most 20 cells wide. The slides still render but won't be readable; this is a sign to resize, not to redesign.

**The notes view stays on "Waiting for the presenter".** Both must open the same deck file (relative paths and symlinks are fine; a copy isn't). They sync through a small file in the system temp directory, so both must also run as the same user with the same `TMPDIR`; running one over SSH on another machine won't work. If the presenter crashed, restart it.

**A message appears at the bottom of the screen.** That's the status line. At start and after a reload it shows warnings about the deck for a few seconds: an unknown theme or transition name, an unknown theme key, or a speaker note missing its `-->`. Each one names the problem (with a line number where it has one); fix the deck and save. `tuipresent -check deck.md` prints the same warnings. The notes view lists them too, without a timeout.

**A light theme is unreadable, or colors clash with the terminal.** Without painting, slides use the terminal's own background. Add `background: paint` to the theme in front matter, or run with `-background paint`.

**A directive did nothing.** Unknown or misspelled directives are silently ignored. Compare against the [directive list](reference.md#directives) and check the slide with `-dump`.
