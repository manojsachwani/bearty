package ui

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/theme"
)

// UI colours. bearty sets the terminal's background to uiBg while it runs
// (see paint), so every colour here is chosen for that one surface and
// nothing depends on the user's terminal theme.
var (
	uiBg    = lipgloss.Color("#1E1F27")
	uiText  = lipgloss.Color("#ECEAE6")
	uiMuted = lipgloss.Color("#B4B1BA") // descriptions, help
	uiFaint = lipgloss.Color("#8A8794") // separators, hints; ≥ 4.5:1 on uiBg

	pumpkin = lipgloss.Color(theme.Inks[0].Hex)
	ember   = lipgloss.Color("#E8875A") // lighter pumpkin for text on uiBg
	paperFg = lipgloss.Color("#F9F9F9")
	errFg   = lipgloss.Color("#FF6B6B")

	// Older names, kept so call sites read naturally.
	dim     = uiMuted
	faintFg = uiFaint
)

// paint gives a view bearty's own background and text colour. Bubble Tea
// sets them on the terminal (OSC 11/10) and restores the user's on exit.
func paint(v tea.View) tea.View {
	v.BackgroundColor = color.Color(uiBg)
	v.ForegroundColor = color.Color(uiText)
	v.WindowTitle = "bearty"
	return v
}

// helpStyles colours key help for uiBg.
func helpStyles() help.Styles {
	s := help.New().Styles
	s.ShortKey = s.ShortKey.Foreground(uiMuted)
	s.ShortDesc = s.ShortDesc.Foreground(uiFaint)
	s.ShortSeparator = s.ShortSeparator.Foreground(uiFaint)
	s.FullKey = s.FullKey.Foreground(uiMuted)
	s.FullDesc = s.FullDesc.Foreground(uiFaint)
	s.FullSeparator = s.FullSeparator.Foreground(uiFaint)
	s.Ellipsis = s.Ellipsis.Foreground(uiFaint)
	return s
}

// formTheme styles Huh forms with bearty's palette. It ignores isDark on
// purpose: the background is always uiBg, so a light/dark guess that goes
// wrong (as it did on terminals that don't report their colours) can't
// make options unreadable.
func formTheme(bool) *huh.Styles {
	t := huh.ThemeCharm(true)

	f := &t.Focused
	f.Base = f.Base.BorderForeground(ember)
	f.Card = f.Base
	f.Title = f.Title.Foreground(ember).Bold(true)
	f.NoteTitle = f.NoteTitle.Foreground(ember).Bold(true)
	f.Description = f.Description.Foreground(uiMuted)
	f.ErrorIndicator = f.ErrorIndicator.Foreground(errFg)
	f.ErrorMessage = f.ErrorMessage.Foreground(errFg)
	f.SelectSelector = f.SelectSelector.Foreground(ember).SetString("▸ ")
	f.NextIndicator = f.NextIndicator.Foreground(ember)
	f.PrevIndicator = f.PrevIndicator.Foreground(ember)
	f.Option = f.Option.Foreground(uiText)
	f.SelectedOption = f.SelectedOption.Foreground(ember)
	f.UnselectedOption = f.UnselectedOption.Foreground(uiText)
	f.SelectedPrefix = f.SelectedPrefix.Foreground(ember)
	f.UnselectedPrefix = f.UnselectedPrefix.Foreground(uiMuted)
	f.FocusedButton = f.FocusedButton.Foreground(paperFg).Background(pumpkin).Bold(true)
	f.Next = f.FocusedButton
	f.BlurredButton = f.BlurredButton.Foreground(uiText).Background(lipgloss.Color("#3A3B46"))
	f.TextInput.Cursor = f.TextInput.Cursor.Foreground(ember)
	f.TextInput.Prompt = f.TextInput.Prompt.Foreground(ember)
	f.TextInput.Text = f.TextInput.Text.Foreground(uiText)
	f.TextInput.Placeholder = f.TextInput.Placeholder.Foreground(uiFaint)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.Title = t.Blurred.Title.Foreground(uiMuted).Bold(false)
	t.Blurred.Option = t.Blurred.Option.Foreground(uiMuted)
	t.Blurred.UnselectedOption = t.Blurred.UnselectedOption.Foreground(uiMuted)
	t.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	t.Help = helpStyles()
	return t
}
