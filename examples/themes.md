---
title: Theme Gallery
sub_title: One deck, twenty looks
author: tuipresent
theme:
  name: tokyonight-storm
  background: paint
transition: slide
---

How to use this deck
===

Preview any deck in any built-in theme without editing it:

```bash
tuipresent -theme nord examples/themes.md
tuipresent -theme solarized-light -background paint deck.md
tuipresent -themes    # every name, with a color swatch
```

Each slide that follows shows one thing a theme colors:

1. Headings, the gradient title and the rule under it
2. Lists, emphasis, inline code and links
3. Code blocks with syntax highlighting
4. Tables and quotes
5. A text chart, and the footer's progress bar

<!-- speaker_note: Step through with -theme set to each name in turn. -->

<!-- end_slide -->

Headings and text
===

## A second-level heading

### A third-level heading

Body text in the theme's foreground, with **bold**, *italic*,
`inline code` and a [link](https://github.com/mfontanini/presenterm).

* Bullets take the theme's bullet color
* Wrapped items hang under their text, so long bullet points stay readable
  * Nested items indent by two
* [x] Tasks render too

> Quotes use the quote color, set apart by a bar.

<!-- end_slide -->

Code
===

```go
// Each theme picks a matching chroma style.
func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2) // recursion
}
```

```python
def greet(name: str) -> str:
    return f"hi, {name}!"  # f-strings too
```

Code blocks sit on a panel in the theme's code background.

<!-- end_slide -->

Marks and callouts
===

Styles: ==highlight==, ~~strike~~, [faint]{.faint}, ++underline++,
[curly]{.curly}.

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

> [!NOTE]
> Useful background.

> [!TIP]
> A better way.

> [!IMPORTANT]
> Needed to succeed.

<!-- column: 1 -->

> [!WARNING]
> Urgent, mind it.

> [!CAUTION]
> Risk of harm.

<!-- end_slide -->

Tables and quotes
===

| Theme family | Dark variants | Light variants |
|--------------|---------------|----------------|
| tokyonight | storm, night, moon | day |
| catppuccin | mocha, macchiato, frappe | latte |
| gruvbox | dark | light |
| rose-pine | rose-pine | dawn |
| solarized | dark | light |

Table borders use the theme's border color.

> "Light themes for bright rooms; dark themes for dark ones."
>
> When you don't know the room, paint the background.

<!-- end_slide -->

A small chart
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

```text
Projector  ██████████████  14
Laptop     █████████▌       9
Monitor    ████████████▏   12
Phone      ███▍             3
```

Where this deck was last shown.

<!-- column: 1 -->

<!-- pause -->

The fade after a pause blends up from the
theme's background.

<!-- pause -->

The progress bar at the bottom uses the
accent; the footer text uses dim.

<!-- end_slide -->

All twenty themes
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

* dark
* tokyonight-storm
* tokyonight-night
* tokyonight-moon
* catppuccin-mocha
* catppuccin-macchiato
* catppuccin-frappe
* gruvbox-dark
* dracula
* nord

<!-- column: 1 -->

* rose-pine
* solarized-dark
* everforest-dark
* kanagawa
* **light**
* **tokyonight-day**
* **catppuccin-latte**
* **gruvbox-light**
* **rose-pine-dawn**
* **solarized-light**

<!-- reset_layout -->

Light themes are in bold. Old names such as `tokyo-night` still work.

<!-- end_slide -->

Make it yours
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

```yaml
# themes/brand.yaml
extends: nord
accent: "#ff6b35"
accent2: "#f7c548"
code_style: monokai
```

<!-- column: 1 -->

```yaml
---
theme:
  path: themes/brand.yaml
  background: paint
---
```

<!-- reset_layout -->

Or override a color inline: `theme: {name: nord, accent: "#88c0d0"}`.
See `examples/brand.md` for a deck that uses a custom theme file.
