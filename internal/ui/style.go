package ui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/theme"
)

// UI colours: pumpkin ink on the terminal's own background.
var (
	pumpkin = lipgloss.Color(theme.Inks[0].Hex)
	ember   = lipgloss.Color("#E8875A") // lighter pumpkin, readable on dark backgrounds
	paperFg = lipgloss.Color("#F9F9F9")
	dim     = lipgloss.Color("244")
	faintFg = lipgloss.Color("240")
	errFg   = lipgloss.Color("#E5484D")
)

// formTheme restyles Huh's Charm theme with bearty's paper-and-pumpkin palette.
func formTheme(isDark bool) *huh.Styles {
	t := huh.ThemeCharm(isDark)
	accent := pumpkin
	text := lipgloss.Color("235")
	if isDark {
		accent, text = ember, lipgloss.Color("252")
	}

	f := &t.Focused
	f.Base = f.Base.BorderForeground(accent)
	f.Card = f.Base
	f.Title = f.Title.Foreground(accent).Bold(true)
	f.NoteTitle = f.NoteTitle.Foreground(accent).Bold(true)
	f.Description = f.Description.Foreground(dim)
	f.ErrorIndicator = f.ErrorIndicator.Foreground(errFg)
	f.ErrorMessage = f.ErrorMessage.Foreground(errFg)
	f.SelectSelector = f.SelectSelector.Foreground(accent).SetString("▸ ")
	f.NextIndicator = f.NextIndicator.Foreground(accent)
	f.PrevIndicator = f.PrevIndicator.Foreground(accent)
	f.Option = f.Option.Foreground(text)
	f.SelectedOption = f.SelectedOption.Foreground(accent)
	f.UnselectedOption = f.UnselectedOption.Foreground(text)
	f.FocusedButton = f.FocusedButton.Foreground(paperFg).Background(pumpkin).Bold(true)
	f.Next = f.FocusedButton
	f.BlurredButton = f.BlurredButton.Foreground(text).Background(lipgloss.Color("237"))
	f.TextInput.Cursor = f.TextInput.Cursor.Foreground(accent)
	f.TextInput.Prompt = f.TextInput.Prompt.Foreground(accent)
	f.TextInput.Placeholder = f.TextInput.Placeholder.Foreground(faintFg)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.Title = t.Blurred.Title.Foreground(dim).Bold(false)
	t.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}
