# bearty

A terminal UI for building custom themes for the [Bear](https://bear.app) notes app on macOS. Pick your colors and a typeface (handwriting, any font on your Mac, or any Google font, in any weight) and watch a live preview. Then install the theme into Bear in one step, with a dark version if you like, or export it as a `.theme` file. Every theme is kept in your library so you can edit it later, and you can capture any of Bear's own themes to start from.

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

Start by creating a new theme, editing a saved one, capturing one of Bear's themes or importing a `.theme` file. Either way you can change every setting:

| Step | Settings |
|---|---|
| Paper | White, Cream, Blush, or any color |
| Ink | Pumpkin, Navy, Plum, Clay, Teal, Charcoal, or any color; links, list markers and to-do ticks follow it or get a color each; headings follow it, are black, or any color |
| Type | Typeface and weight, heading typeface and weight, code font |
| Layout | Size (15–24pt), heading sizes, line height, paragraph spacing and indent, line width, rule lines |
| Colors | Sidebar (background, text, icons, selected row) and note list (titles, previews, dates, selected note): automatic or any color |
| Details | The six highlighters, code block colors and syntax highlighting, and tag pills: automatic or any color |
| Dark | Use the theme in dark mode too, add an automatic dark version, or choose the dark paper and ink |
| Save | Install into Bear, export a `.theme` file, both, or just save it |

Anything left automatic is worked out from your paper and ink. The preview shows a note (with a highlight, a tag and a line of code), a small sidebar and note list, and warns when a color pair is hard to read. Press `t` to switch the preview between the light and dark versions.

Bear's note list has no background setting of its own; it always uses the paper color.

### Color picker

Wherever bearty asks for a color, it shows a picker:

| Key | Action |
|---|---|
| ↑ / ↓ | Choose the hue, saturation or lightness bar |
| ← / → | Adjust it (shift for bigger steps) |
| 1–9 | Pick from the palette |
| # | Type a hex color; pasting `#RRGGBB` works too |
| x | Back to automatic (optional colors only) |

### Typefaces

- **Handwriting fonts:** [Inkina](https://shapedfonts.com/inkina/), Google handwriting fonts (Caveat, Patrick Hand, Kalam, Indie Flower and more), and the handwriting fonts that come with macOS. Fonts marked ↓ are downloaded when you install.
- **Any font on this Mac:** type into **Search fonts** to narrow the list, then pick a family and a style. This includes Bear's own bundled fonts, marked *inside Bear only*.
- **Any Google Font:** type a family name from [fonts.google.com](https://fonts.google.com). ctrl+e completes a suggestion. No API key is needed.

Every typeface has a **Weight** choice. For Google fonts, the weights (and italics) come from the family's listing, including the full range of variable fonts, and bearty downloads a file cut to the exact weight you chose. Mac fonts offer the styles installed on your Mac.

The code font can be Bear's Roboto Mono, a Mac monospace font (Menlo, Monaco, Courier New, Andale Mono, PT Mono), or a Google one (JetBrains Mono, IBM Plex Mono, Space Mono, Inconsolata).

Downloaded fonts are cached in `~/Library/Caches/bearty` and copied into `Bear.app/Contents/Resources` next to Bear's own fonts. **Restore** can remove them again.

The typeface covers notes and headings. Bear's theme format has no font setting for the sidebar or note list; they always use the system font.

### Installing into Bear

Bear can't import custom themes, so bearty replaces one of Bear's built-in themes (and a second one for the dark version), then:

1. Quits Bear, if you agree, so the new files take effect.
2. Backs up the originals to `~/Downloads/<Theme>.original.theme`. Later installs keep that first backup.
3. Downloads any fonts the theme needs and copies them, and the theme, into Bear. This uses `sudo`, so you'll be asked for your password.
4. Selects the theme in Bear for light mode, and the dark version for dark mode, so you don't have to pick it in Settings → Themes.
5. If you say yes, sets Bear's own **Settings → Typography** to match (see below).
6. Opens Bear again.

macOS may ask once whether your terminal may control Bear; bearty uses that to quit it the way ⌘Q does.

### Fonts and switching themes

Bear ignores the fonts inside theme files. It draws every theme with one set of **Settings → Typography** (text, headings and code fonts, size, line height and spacing), so picking another theme in Bear's own Themes pane changes the colors but never the font.

bearty works around this. When you install, it asks **Use this theme's fonts in Bear?** and sets Typography to the theme's fonts and sizes. To switch between your themes later, fonts included, use **Switch Bear to a saved theme** (the first choice once you've installed a theme) or:

```bash
bearty use "Paper Pro"
```

This selects the theme in Bear, sets Typography to match, and quits and reopens Bear. If Bear no longer has the theme's current files, for example after an update, it installs them again first.

Your own Typography settings, and which theme Bear had selected, are saved before bearty first changes them, to `~/.config/bearty/bear-typography.original.json`. **Restore** can put them back.

### After a Bear update

A Bear update puts back Bear's own themes and removes fonts bearty added. bearty keeps a copy of what it installed, notices when it's gone, and offers **Re-apply** as the first choice when you next run it. You can also run `bearty reapply`.

### Capture and import

**Capture one of Bear's themes** turns any theme in Bear's list into a bearty theme you can edit, including one you installed earlier. **Import a .theme file** does the same for a file someone shared. Everything bearty can edit becomes a normal setting; anything else is kept exactly as it was, so the captured theme matches the original. (bearty's tests capture all of Bear's built-in themes and check they come out identical.)

### Saved themes

Every theme you finish is saved to `~/.config/bearty/themes/<name>.json` (or `$XDG_CONFIG_HOME/bearty/themes`). To change one later, choose **Edit a saved theme**. You can also edit the JSON by hand:

```json
{
  "name": "E-Paper",
  "paper": "#F9F9F9",
  "ink": "#CF5624",
  "font": "Inkina-Regular",
  "font_source": "inkina",
  "size": 19,
  "headings": "ink",
  "lines": "#EAEAEA",
  "sidebar": { "background": "#EFE9DF", "icons": "#8A3B12" },
  "notelist": { "title": "#000000" },
  "dark": { "mode": "auto" }
}
```

| Key | Takes |
|---|---|
| `font`, `heading_font` | A PostScript name |
| `font_source`, `heading_font_source`, `code_font_source` | Where to download a missing font: `inkina`, `google:<Family>` or `google:<Family>@<weight>` (add `i` for italic, e.g. `google:Caveat@700`) |
| `headings` | `ink`, `black` or a hex color |
| `accents` | `links`, `markers`, `ticks` |
| `sidebar` | `background`, `text`, `icons`, `selected` |
| `notelist` | `title`, `preview`, `date`, `selected` |
| `highlights` | `default`, `red`, `blue`, `green`, `yellow`, `purple` |
| `code` | `font`, `background`, `text`, `comment`, `constant`, `number`, `string`, `entity`, `keyword`, `function`, `variable` |
| `tag` | `background`, `text` |
| `typography` | `line_height`, `line_width`, `paragraph_spacing`, `paragraph_indent` |
| `heading_scale` | How much bigger each heading level is (default 1.15) |
| `dark` | `mode` (`auto` or `custom`), and for `custom` the `paper` and `ink` |
| `extra` | Any other Bear theme key by dotted path, e.g. `"editor.toolbar.text color"` |

Leave out anything you don't want to override.

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
| t | Switch the preview between light and dark (when there's a dark version) |
| ? | About bearty |
| q or ctrl+c | Quit (q types normally in text fields) |

bearty opens with a short animated splash screen; press any key to skip it. Run `bearty --no-motion`, or set `BEARTY_NO_MOTION=1`, to turn off the splash and animations.

## Commands

Everything the editor does can also be scripted:

```bash
bearty list                                   # your themes and what's installed
bearty slots                                  # Bear's themes you can replace
bearty install "Paper Pro" --slot Notes       # install a saved theme
bearty install "My Theme" --slot Notes --dark-slot "Dark Notes"
bearty use "Paper Pro"                        # switch Bear to an installed theme, fonts included
bearty export "E-Paper" -o e-paper.theme      # write a .theme file (and e-paper-dark.theme)
bearty capture "Solarized Light" --name Sol   # one of Bear's themes → a bearty theme
bearty import ~/Downloads/shared.theme        # a .theme file → a bearty theme
bearty reapply                                # put the last install back after a Bear update
bearty restore Notes --remove-fonts           # put Bear's original back
```

`install`, `use` and `reapply` take `--no-restart` to leave Bear running, and `install` takes `--no-typography` to leave Bear's typography alone. Run `bearty help` for everything.

## Restoring Bear's original theme

Run `bearty` and choose **Restore an original Bear theme**, or run `bearty restore <theme>`. The backup is copied back into Bear, and bearty offers to remove the fonts it added and to put your own Bear settings back.

To restore by hand instead:

```bash
R="/Applications/Bear.app/Contents/Frameworks/BearCore.framework/Versions/A/Resources"
sudo cp ~/Downloads/"Notes.original.theme" "$R/Notes.theme"
```

Change `Notes` to the theme you replaced. Reinstalling Bear also restores everything.

## Limits

- **Paper patterns:** Bear themes only support colors, fonts and spacing, so there is no ruled, grid or dot paper.
- **Sidebar and note list fonts:** always the system font; Bear's theme format can't change them.
- **Bear for iPhone and iPad:** doesn't support custom themes.
- **Code signature:** changing files inside the App Store version of Bear changes its code signature. Bear normally keeps working. If it won't open, reinstall it.
- **Inkina** is free for commercial use but can't be redistributed, so bearty downloads it on your Mac and never bundles it.

## Credits

The theme format and install approach come from [Piyush-linux/Bear-Notes-Theme](https://github.com/Piyush-linux/Bear-Notes-Theme). Inkina is by [Shaped Fonts](https://shapedfonts.com/inkina/).

## License

MIT
