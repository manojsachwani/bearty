# bearty

A terminal UI for building custom themes for the [Bear](https://bear.app) notes app on macOS. Pick your colors and a typeface (handwriting, any font on your Mac, or any Google font) and watch a live preview. Then install the theme into Bear, export it as a `.theme` file, or keep it in your saved themes to edit later.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Huh](https://github.com/charmbracelet/huh), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles) and [Harmonica](https://github.com/charmbracelet/harmonica).

## Install

```bash
brew tap manojsachwani/taps
brew install --cask bearty
```

Or from source:

```bash
go install github.com/manojsachwani/bearty@latest
```

## Use

```bash
bearty
```

Start by creating a new theme or editing a saved one. Either way you can change every setting:

| Setting | Choices |
|---|---|
| Paper | White, Cream, Blush, or any color |
| Ink | Pumpkin, Navy, Plum, Clay, Teal, Charcoal, or any color |
| Links, list markers, ticks | Same as the text, or a color each |
| Headings | Same as the ink, black, or any color |
| Typeface | A handwriting pick, any font on your Mac, or any Google font (see below) |
| Heading typeface | Same as the text, or any handwriting pick |
| Size | Small 15pt, Medium 17pt, Large 19pt, Extra large 22pt, Huge 24pt |
| Line height, paragraph spacing, paragraph indent, line width | Bear's default or a value of your choice |
| Rule lines | Color for separators and table borders, or automatic |
| Sidebar | Background, text, icons and selected row: automatic or any color |
| Note list | Titles, preview text, dates and selected note: automatic or any color |
| What should bearty do with it? | Install into Bear, export a `.theme` file, both, or just save it |

The ink is the text color. Links, list markers and to-do ticks follow it unless you choose **Pick each color**. The paper color is the note background, and the sidebar uses a slightly darker shade of it. Under **Colors**, choose **Pick each color** to set the sidebar and note list yourself. The preview shows a small sidebar and note list, and color changes ease in so you can see what changed.

### Color picker

Wherever bearty asks for a color, it shows a picker:

| Key | Action |
|---|---|
| ↑ / ↓ | Choose the hue, saturation or lightness bar |
| ← / → | Adjust it (shift for bigger steps) |
| 1–9 | Pick from the palette |
| # | Type a hex color; pasting `#RRGGBB` works too |
| x | Back to automatic (optional colors only) |

An optional color left untouched stays **automatic**: it follows your paper and ink, and the picker shows the color it will use.

Bear's note list has no background setting of its own; it always uses the paper color.

### Typefaces

- **Handwriting fonts:** [Inkina](https://shapedfonts.com/inkina/), Google handwriting fonts (Caveat, Patrick Hand, Kalam, Indie Flower and more), and the handwriting fonts that come with macOS. Fonts marked ↓ are downloaded when you install.
- **Any font on this Mac:** type into **Search fonts** to narrow the list, then pick a family and a style. This includes Bear's own bundled fonts, marked *inside Bear only*.
- **Any Google Font:** type a family name from [fonts.google.com](https://fonts.google.com). ctrl+e completes a suggestion. bearty looks the font up in the [google/fonts](https://github.com/google/fonts) repository, so no API key is needed.

Downloaded fonts are cached in `~/Library/Caches/bearty` and copied into `Bear.app/Contents/Resources` next to Inkina. **Restore** can remove them again.

The typeface covers notes and headings. Bear's theme format has no font setting for the sidebar or note list; they always use the system font.

### Bear's typography settings

Bear's own **Settings → Typography** (font, headings font, size, line height and spacing) overrides a theme's. So when you install, bearty asks **Also set Bear's typography to match?** Say yes and bearty writes those settings for you, so the theme's font actually shows up. Your own settings are saved first, to `~/.config/bearty/bear-typography.original.json`, and **Restore** can put them back. If you say no, set the font yourself in Bear → Settings → Typography.

### Saved themes

Every theme you finish is saved to `~/.config/bearty/themes/<name>.json` (or `$XDG_CONFIG_HOME/bearty/themes`). To change one later, choose **Edit a saved theme**. You can also edit the JSON by hand:

```json
{
  "name": "E-Paper",
  "paper": "#F9F9F9",
  "ink": "#CF5624",
  "font": "Inkina-Regular",
  "size": 19,
  "headings": "ink",
  "lines": "#EAEAEA",
  "sidebar": { "background": "#EFE9DF", "icons": "#8A3B12" },
  "notelist": { "title": "#000000" }
}
```

`headings` takes `ink`, `black` or a hex color. `accents` takes `links`, `markers` and `ticks`. `sidebar` takes `background`, `text`, `icons` and `selected`; `notelist` takes `title`, `preview`, `date` and `selected`. Leave out any you don't want to override. `heading_font` (with `heading_font_source`) sets a separate headings typeface, and `typography` takes `line_height`, `line_width`, `paragraph_spacing` and `paragraph_indent`.

`font` takes any installed font's PostScript name. `font_source` says where to download it from if it's missing: `inkina` or `google:<Family>`.

bearty ships with three built-in themes:

- **E-Paper:** grey-white paper, pumpkin ink, faint rules and large Inkina handwriting.
- **Paper Pro:** warm paper, pumpkin Inkina ink and black headings, with a navy sidebar selection and navy note previews.
- **All these things:** blush paper and navy ink with green headings, its own color for links, list markers and ticks, and Indie Flower text under Patrick Hand headings at 24pt.

If you edit a built-in, your copy is saved under the same name and replaces it. Delete that file to get the original back.

### Keys

| Key | Action |
|---|---|
| ↑ / ↓ | Move through a list |
| enter / tab | Next |
| ← / → | Switch inline choices such as size |
| esc or shift+tab | Back (esc on the first screen quits) |
| ? | About bearty |
| q or ctrl+c | Quit (q types normally in text fields) |

bearty opens with a short animated splash screen; press any key to skip it. Run `bearty --no-motion`, or set `BEARTY_NO_MOTION=1`, to turn off the splash and animations.

### Installing into Bear

Bear can't import custom themes, so bearty replaces one of Bear's built-in themes. It asks which one to replace and asks you to confirm.

1. The original is copied to `~/Downloads/<Theme>.original.theme`. Later installs keep that first backup, so it always holds Bear's original.
2. If you chose Inkina, it's downloaded from shapedfonts.com, cached in `~/Library/Caches/bearty`, and copied into `Bear.app/Contents/Resources`, where Bear loads its own fonts.
3. The theme file is copied into Bear. Both copies use `sudo`, so you'll be asked for your password.

Then quit Bear (⌘Q), reopen it, and choose that theme in **Settings → Themes**.

### Restoring Bear's original theme

Run `bearty` and choose **Restore an original Bear theme**. This option appears once bearty has backed up at least one theme. Pick the theme and confirm, and the backup is copied back into Bear. If Inkina is installed, bearty also asks whether to remove it.

To restore by hand instead:

```bash
R="/Applications/Bear.app/Contents/Frameworks/BearCore.framework/Versions/A/Resources"
sudo cp ~/Downloads/"Notes.original.theme" "$R/Notes.theme"
```

Change `Notes` to the theme you replaced. Reinstalling Bear also restores everything.

## Limits

- **Paper patterns:** Bear themes only support colors, fonts and spacing, so there is no ruled, grid or dot paper.
- **Bear updates:** an update puts back Bear's original themes and removes the copied font. Run `bearty` again afterwards; the font is reused from the cache.
- **Code signature:** changing files inside the App Store version of Bear changes its code signature. Bear normally keeps working. If it won't open, reinstall it.
- **Inkina** is free for commercial use but can't be redistributed, so bearty downloads it on your Mac and never bundles it.

## Credits

The theme format and install approach come from [Piyush-linux/Bear-Notes-Theme](https://github.com/Piyush-linux/Bear-Notes-Theme). Inkina is by [Shaped Fonts](https://shapedfonts.com/inkina/).

## License

MIT
