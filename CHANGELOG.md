# Changelog

## 0.1.3

- bearty now uses its own dark background while it runs, so options, help and descriptions stay readable on any terminal theme. Your terminal's colors come back when it exits.
- Clearer wording throughout, with US spelling and new step names: Start › Paper › Ink › Type › Layout › Colors › Save › Install.

## 0.1.2

First public release.

- Build custom themes for the Bear notes app in a terminal UI, with a live preview of the note, sidebar and note list.
- Color picker for every color: hue, saturation and lightness bars, a palette on the number keys, and hex entry or paste.
- Separate colors for text, links, list markers, to-do ticks, headings, rule lines, the sidebar and the note list. Anything left alone follows your paper and ink.
- Typefaces: handwriting fonts (Inkina, Google handwriting fonts, macOS fonts), any font on your Mac, or any Google font, plus a separate heading typeface.
- Typography: size, line height, line width, paragraph spacing and paragraph indent.
- Install into a Bear theme slot with a backup of the original. Downloaded fonts go into Bear, and bearty can set Bear's own Typography settings so the theme's font shows up.
- Restore Bear's original theme, your typography settings, and remove fonts bearty added.
- Saved themes in `~/.config/bearty/themes`, editable in the app or by hand.
- Built-in themes: E-Paper, Paper Pro and All these things.
- Animated splash screen and small spring animations (`--no-motion` turns them off).
- Keys: `?` about, `esc` back, `q` or `ctrl+c` quit.
