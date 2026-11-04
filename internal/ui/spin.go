package ui

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/theme"
)

type doneMsg struct{ err error }

type spinModel struct {
	spinner spinner.Model
	title   string
	task    func() error
	err     error
	done    bool
}

func (m spinModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return doneMsg{m.task()} })
}

func (m spinModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.err, m.done = msg.err, true
		return m, tea.Quit
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.err = ErrAborted
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinModel) View() tea.View {
	if m.done || m.err != nil {
		return tea.NewView("")
	}
	return tea.NewView(m.spinner.View() + " " + m.title + "\n")
}

// Spin runs task while showing a spinner and returns its error.
func Spin(title string, task func() error) error {
	m := spinModel{
		spinner: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Inks[0].Hex))),
		),
		title: title,
		task:  task,
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	return final.(spinModel).err
}
