# Changelog

## 0.1.4

- One-step install: bearty quits Bear, installs the theme, selects it for light and dark mode, and opens Bear again.
- Switch Bear to a saved theme, fonts included (`bearty use <theme>`). Bear ignores fonts inside themes and uses one set of Typography settings for every theme, so switching in Bear's own Themes pane can't change the font; bearty switches both.
- Dark mode: add an automatic dark version of any theme, or choose its dark paper and ink. It's installed into a second slot and used when your Mac is in dark mode.
- Re-apply after Bear updates: bearty notices when an update removed its theme or fonts and offers to put them back (`bearty reapply`).
- Capture any of Bear's themes, or import a `.theme` file, as an editable bearty theme that matches the original exactly.
- New settings: the six highlighters, code block colors, syntax highlighting and code font, tag pills, and heading sizes.
- Font weights: pick any weight or italic for Google fonts (including variable fonts) and Mac fonts, for text and headings.
- A fuller preview with a highlight, a tag and a line of code, contrast warnings, and `t` to switch between light and dark.
- Commands for scripting: `list`, `slots`, `install`, `export`, `capture`, `import`, `reapply`, `restore`.

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
