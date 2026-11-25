# Sharing and export

Turn a deck into something you can send: an animated recording (GIF, MP4 or WebM) or one image per slide (PNG or SVG), and from the images a PDF. tuipresent adds no dependencies for this. It writes the input and shells out to two Charm tools, and `-dry-run` shows what would happen without either installed.

| You want | Mode | Tool |
|----------|------|------|
| A recording that steps through every slide and pause | `-export tape` (write the script) or `-export vhs` (write it and record) | [VHS](https://github.com/charmbracelet/vhs) |
| One PNG or SVG per slide | `-export freeze` | [Freeze](https://github.com/charmbracelet/freeze) |
| A PDF | `-export freeze`, then combine the PNGs (below) | ImageMagick, img2pdf or similar |

## Record a GIF or video with VHS

Install VHS (it also needs `ttyd` and `ffmpeg`; the Homebrew formula pulls them in):

```sh
brew install vhs
# or: go install github.com/charmbracelet/vhs@latest
```

Generate a tape and look at it first:

```sh
tuipresent -export tape deck.md                 # print the tape to stdout
tuipresent -export tape -out deck.tape deck.md  # or write it (the name must end in .tape)
vhs deck.tape                                   # records deck.gif
```

Or do it all in one step. This writes `deck-recording/deck.tape` and runs `vhs` inside that directory, so `deck.gif` lands next to it:

```sh
tuipresent -export vhs deck.md
tuipresent -export vhs -out recordings -format mp4 -w 100 -h 30 deck.md
```

The tape:

- sets the font, padding, a pixel size computed from `-w` x `-h` (default 100x30) and a VHS theme made from the deck's theme colors;
- starts `tuipresent -watch=false -progress=false -mouse=false -images=blocks deck.md` out of sight (`Hide` and `Show`), so the recording opens on the first slide;
- rests on each slide, presses `Right` for every pause step and slide, and finally `q`. The rest per step is 2 s plus 100 ms per word on the fully revealed slide (at most 10 s, shared between the slide's steps, never under 1 s so a transition can finish). `-dwell 3s` replaces that with a fixed time for every step;
- passes your `-theme`, `-background` and `-transition` on to the presenter. Open the tape in an editor to tweak anything else: timings, `Set FontSize`, `Set Theme`, extra `Sleep`s.

The recording is the real presenter, so transitions and pause fades are in it. `-images=blocks` keeps pictures as half-blocks, which VHS's terminal can draw (it cannot show kitty graphics).

`+exec` blocks do not run by default. With `-exec`, the tape types `e` on the last step of each slide that has one and waits 2 s for the output. That executes the deck's code when you record, so only do it for a deck you trust.

Pick the format with `-format gif` (default), `mp4` or `webm`. The tape's first lines (`Output`, `Set`) are plain VHS settings, so see the VHS README for everything else, such as `Set Framerate`.

## Export slide images with Freeze

```sh
brew install charmbracelet/tap/freeze
# or: go install github.com/charmbracelet/freeze@latest

tuipresent -export freeze -out slides deck.md
tuipresent -export freeze -out slides -format svg -w 100 -h 30 deck.md
```

For each slide this writes `slide-NN-title.ansi` (the frame with its colors, exactly what `-dump` prints, in the fully revealed state) and runs `freeze --language ansi ... --output slide-NN-title.png slide-NN-title.ansi` inside the out directory. Everything is rendered with the theme painted into every cell (as `-background paint`), so the images look the same whatever your terminal is. Use `-theme` to export in another theme and `-w`/`-h` for the size; `-check` the deck at that size first. `-out` defaults to `<deck>-slides`. Keep the `.ansi` files to re-run Freeze with your own options (`freeze --help`), or delete them.

`+exec` output is empty unless you pass `-exec`, which runs the deck's code first (only for a deck you trust). Pause steps are not exported separately; each slide is shown with all its content revealed.

### Combine the images into a PDF

Freeze writes PNG, SVG and WebP, not PDF, so combine the slide images with another tool. The zero-padded names sort in slide order:

```sh
magick slides/slide-*.png deck.pdf        # ImageMagick
img2pdf slides/slide-*.png -o deck.pdf    # img2pdf (pip install img2pdf)
```

For vector PDFs, export with `-format svg` and convert each page with `rsvg-convert -f pdf`, then join them with `pdfunite` or `qpdf`.

## Dry runs and safety

- `-dry-run` prints the tape, or the `freeze` commands and file names, and writes and runs nothing. It works without VHS or Freeze installed (a missing Freeze is mentioned on stderr).
- Without `-dry-run`, a missing `vhs` or `freeze` stops before anything is written, with an install hint and exit status 1.
- Generated file names are `slide-NN-` plus the slide title reduced to `a-z0-9-` (at most 40 characters), so a title can never contain a path. Files are only written inside the `-out` directory. tuipresent refuses to write through a symlink or over a directory, and refuses an `-out` that is an existing file. For `-export tape`, `-out` must be a file ending in `.tape`.
- Existing regular files in the out directory with the same names are replaced; nothing else is touched.
- `-export` cannot be combined with `-check`, `-dump` or `-notes`.

## Verify live

The tape's pixel size assumes a cell of 9.6 x 19.2 px at font size 16, and the Freeze options (`--font.size`, `--line-height`, `--padding`, `--background`) assume current Freeze. If a recording is cropped or has extra margin, edit `Set Width`, `Set Height` or `Set FontSize` in the tape. See the [reference](reference.md#sharing-and-export) for every flag.
