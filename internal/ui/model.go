package ui

import (
	"errors"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/fonts"
	"github.com/manojsachwani/bearty/internal/library"
)

// ErrAborted is returned when the user quits before finishing the form.
var ErrAborted = errors.New("aborted")

type stage int

const (
	stageStart stage = iota
	stageEdit
)

var steps = []string{"Start", "Paper", "Ink", "Type", "Layout", "Colors", "Save", "Install"}

// stepOf maps a focused field's key to its breadcrumb step.
var stepOf = map[string]string{
	"mode": "Start", "saved": "Start",
	"name": "Paper", "paper": "Paper", "paperhex": "Paper",
	"ink": "Ink", "inkhex": "Ink", "accentmode": "Ink", "acc-links": "Ink", "acc-markers": "Ink",
	"acc-ticks": "Ink", "headings": "Ink", "headinghex": "Ink",
	"fontsrc": "Type", "pick": "Type", "fontquery": "Type", "family": "Type", "headingfont": "Type", "face": "Type", "google": "Type",
	"size": "Layout", "lines": "Layout", "lineheight": "Layout", "pspacing": "Layout",
	"pindent": "Layout", "linewidth": "Layout",
	"regions": "Colors", "side-bg": "Colors", "side-text": "Colors", "side-icons": "Colors",
	"side-sel": "Colors", "list-title": "Colors", "list-preview": "Colors", "list-date": "Colors",
	"list-sel": "Colors",
	"output":   "Output", "path": "Save",
	"slot": "Install", "confirm": "Install", "typography": "Install",
}

// keys are bearty's own shortcuts; each field adds its own help below the form.
type keys struct {
	About, Back, Quit key.Binding
}

func (k keys) ShortHelp() []key.Binding  { return []key.Binding{k.About, k.Back, k.Quit} }
func (k keys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

var globalKeys = keys{
	About: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "about")),
	Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type model struct {
	stage   stage
	form    *huh.Form
	choices *Choices
	env     Env
	help    help.Model
	size    tea.WindowSizeMsg
	about   bool
	aborted bool

	splash *splash // nil once the splash screen is over
	motion *motion
}

// motion holds the springs behind the small animations: the preview's paper
// and ink easing into new colours, and the breadcrumb underline sliding.
type motion struct {
	paper, ink     colour
	crumbX, crumbW value
	ticking        bool
	off            bool // --no-motion: jump straight to targets
}

func (mo *motion) step() {
	mo.paper.step()
	mo.ink.step()
	mo.crumbX.step(glide)
	mo.crumbW.step(glide)
}

func (mo *motion) settled() bool {
	return mo.paper.settled() && mo.ink.settled() && mo.crumbX.settled() && mo.crumbW.settled()
}

func (mo *motion) snap() {
	mo.paper.snap()
	mo.ink.snap()
	mo.crumbX.snap()
	mo.crumbW.snap()
}

type facesMsg []fonts.Face

func (m model) Init() tea.Cmd {
	m.retarget()
	m.motion.snap() // start in place; only later changes animate
	cmds := []tea.Cmd{m.form.Init()}
	if load := m.env.LoadFaces; load != nil {
		cmds = append(cmds, func() tea.Msg { return facesMsg(load()) }) // runs behind the splash
	}
	if m.splash != nil {
		m.motion.ticking = true
		cmds = append(cmds, frame())
	}
	return tea.Batch(cmds...)
}

// retarget points the springs at the current state.
func (m model) retarget() {
	pc := m.previewChoices()
	m.motion.paper.set(pc.Paper())
	m.motion.ink.set(pc.Ink())
	x, w := m.crumbSpan()
	m.motion.crumbX.target, m.motion.crumbW.target = float64(x), float64(w)
	if m.motion.off {
		m.motion.snap()
	}
}

// animate keeps frames coming while anything is still moving.
func (m model) animate(cmd tea.Cmd) tea.Cmd {
	m.retarget()
	if m.motion.ticking || m.motion.settled() {
		return cmd
	}
	m.motion.ticking = true
	return tea.Batch(cmd, frame())
}

// filtering reports whether a list is being searched, where esc clears the search.
func (m model) filtering() bool {
	f, ok := m.form.GetFocusedField().(interface{ GetFiltering() bool })
	return ok && f.GetFiltering()
}

// typing reports whether printable keys belong to the focused field as text.
func (m model) typing() bool {
	_, input := m.form.GetFocusedField().(*huh.Input)
	return input || m.filtering()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case facesMsg:
		m.env.Faces, m.env.LoadFaces = msg, nil
		return m, nil
	case frameMsg:
		if m.splash != nil && m.splash.step(time.Time(msg)) {
			m.splash = nil
		}
		m.motion.step()
		if m.splash == nil && m.motion.settled() {
			m.motion.ticking = false
			return m, nil
		}
		return m, frame()
	case tea.WindowSizeMsg:
		m.size = msg
	case tea.KeyPressMsg:
		if m.splash != nil { // any key skips the splash
			if key.Matches(msg, globalKeys.Quit) {
				m.aborted = true
				return m, tea.Quit
			}
			m.splash = nil
			return m, nil
		}
		if m.about { // any key closes the about screen
			if key.Matches(msg, globalKeys.Quit) {
				m.aborted = true
				return m, tea.Quit
			}
			m.about = false
			return m, nil
		}
		switch {
		// q types a letter in text fields; ctrl+c always quits.
		case key.Matches(msg, globalKeys.Quit) && (msg.String() == "ctrl+c" || !m.typing()):
			m.aborted = true
			return m, tea.Quit
		case key.Matches(msg, globalKeys.About) && !m.typing():
			m.about = true
			return m, nil
		case key.Matches(msg, globalKeys.Back) && !m.filtering():
			return m.back()
		}
	}

	f, cmd := m.form.Update(msg)
	if f, ok := f.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateAborted:
		m.aborted = true
		return m, tea.Quit
	case huh.StateCompleted:
		if m.stage == stageEdit || m.choices.Restoring() {
			return m, tea.Quit
		}
		mm, cmd := m.startEditing()
		return mm, mm.(model).animate(cmd)
	}
	return m, m.animate(cmd)
}

// back steps to the previous field. From the first field of the editor it
// returns to the start screen; on the start screen it quits.
func (m model) back() (tea.Model, tea.Cmd) {
	first := m.form.GetFocusedField().GetKey()
	switch {
	case m.stage == stageStart && first == "mode":
		m.aborted = true
		return m, tea.Quit
	case m.stage == stageEdit && first == "name":
		m.stage = stageStart
		m.form = newStartForm(m.choices, m.env)
		return m, m.animate(m.resized(m.form.Init()))
	}
	return m, m.animate(m.form.PrevField())
}

// startEditing opens the editor, filled in with the picked theme.
func (m model) startEditing() (tea.Model, tea.Cmd) {
	if load := m.env.LoadFaces; load != nil { // still scanning: finish now
		m.env.Faces, m.env.LoadFaces = load(), nil
	}
	if m.choices.Mode == ModeEdit {
		if s, ok := m.selected(); ok {
			m.choices = fromSaved(s, m.env.Faces)
		}
	}
	m.stage = stageEdit
	m.form = newEditForm(m.choices, m.env)
	return m, m.resized(m.form.Init())
}

// resized replays the window size so a new form lays itself out.
func (m model) resized(cmd tea.Cmd) tea.Cmd {
	size := m.size
	return tea.Batch(cmd, func() tea.Msg { return size })
}

func (m model) selected() (library.Saved, bool) {
	for _, s := range m.env.Saved {
		if library.Slug(s.Name) == m.choices.SavedSel {
			return s, true
		}
	}
	return library.Saved{}, false
}

// previewChoices is what the preview shows: the highlighted saved theme on
// the start screen, otherwise the theme being edited.
func (m model) previewChoices() *Choices {
	if m.stage == stageStart {
		if m.choices.Mode == ModeEdit {
			if s, ok := m.selected(); ok {
				return fromSaved(s, m.env.Faces)
			}
		}
		return defaults()
	}
	return m.choices
}

func badge() string {
	return lipgloss.NewStyle().Foreground(paperFg).Background(pumpkin).Bold(true).Padding(0, 1).Render("bearty")
}

// currentStep is the breadcrumb step of the focused field.
func (m model) currentStep() string {
	if f := m.form.GetFocusedField(); f != nil {
		if s, ok := stepOf[f.GetKey()]; ok {
			return s
		}
	}
	return "Start"
}

const crumbSep = " › "

// crumbSpan is the column and width of the current step in the breadcrumbs.
func (m model) crumbSpan() (x, w int) {
	cur := m.currentStep()
	for _, s := range steps {
		if s == cur {
			return x, len(s)
		}
		x += len(s) + lipgloss.Width(crumbSep)
	}
	return 0, 0
}

func (m model) header() string {
	tagline := lipgloss.NewStyle().Foreground(dim).Render("  custom themes for Bear")
	if m.choices.Restoring() {
		return badge() + tagline
	}

	current := m.currentStep()
	var crumbs []string
	reached := true
	for _, s := range steps {
		switch {
		case s == current:
			crumbs = append(crumbs, lipgloss.NewStyle().Foreground(ember).Bold(true).Render(s))
			reached = false
		case reached:
			crumbs = append(crumbs, lipgloss.NewStyle().Foreground(dim).Render(s))
		default:
			crumbs = append(crumbs, lipgloss.NewStyle().Foreground(faintFg).Render(s))
		}
	}
	sep := lipgloss.NewStyle().Foreground(faintFg).Render(crumbSep)

	// The underline slides to the current step.
	x := int(math.Round(m.motion.crumbX.pos))
	w := int(math.Round(m.motion.crumbW.pos))
	underline := strings.Repeat(" ", max(x, 0)) + lipgloss.NewStyle().Foreground(ember).Render(strings.Repeat("▔", max(w, 1)))
	return badge() + tagline + "\n\n" + strings.Join(crumbs, sep) + "\n" + underline
}

func (m model) version() string {
	if m.env.Version == "" {
		return "dev"
	}
	return m.env.Version
}

func (m model) aboutView() string {
	h := lipgloss.NewStyle().Foreground(ember).Bold(true)
	t := lipgloss.NewStyle().Foreground(uiText)
	d := lipgloss.NewStyle().Foreground(dim)
	rows := []string{
		badge() + d.Render("  "+m.version()),
		"",
		t.Render("Custom themes for the Bear notes app on macOS."),
		t.Render("Pick colors and a typeface, watch the preview,"),
		t.Render("then install the theme into Bear or export it."),
		"",
		h.Render("Made by"),
		t.Render("@manojsachwani · github.com/manojsachwani/bearty"),
		d.Render("MIT license"),
		"",
		h.Render("Thanks to"),
		t.Render("Charm for Bubble Tea, Huh, Lip Gloss, Bubbles, Harmonica"),
		t.Render("Shaped Fonts for Inkina · Google Fonts"),
		t.Render("Piyush-linux/Bear-Notes-Theme for the install approach"),
		"",
		h.Render("How-to"),
		t.Render("Font or size not changing? Bear's own Typography"),
		t.Render("settings override themes. Say yes to setting them when"),
		t.Render("you install, or pick the font in Bear → Settings."),
		t.Render("After installing, quit Bear (⌘Q), reopen it, and choose"),
		t.Render("the replaced theme in Settings → Themes."),
		t.Render("Bear can't draw ruled or grid pages, and it always uses"),
		t.Render("the system font in the sidebar and note list."),
		"",
		h.Render("Keys"),
		t.Render("↑/↓ move · enter next · ←/→ switch inline choices"),
		t.Render("shift+tab or esc back · q or ctrl+c quit"),
		"",
		d.Render("Press any key to close."),
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(pumpkin).
		Padding(1, 3).Render(strings.Join(rows, "\n"))
}

func (m model) View() tea.View {
	if m.form.State != huh.StateNormal {
		return tea.NewView("")
	}
	if m.splash != nil {
		v := paint(tea.NewView(m.splash.view(m.version(), m.size.Width, m.size.Height)))
		v.AltScreen = true
		return v
	}
	var content string
	if m.about {
		content = m.aboutView()
	} else {
		form := lipgloss.NewStyle().Width(formWidth).Render(m.form.View())
		var body string
		pc := m.previewChoices()
		o := pc.Options()
		o.Paper, o.Ink = m.motion.paper.hex(), m.motion.ink.hex()
		switch {
		case m.choices.Restoring():
			body = form
		case m.size.Width > 0 && m.size.Width < formWidth+previewWidth+8:
			body = lipgloss.JoinVertical(lipgloss.Left, renderPreview(pc, o), "", form)
		default:
			body = lipgloss.JoinHorizontal(lipgloss.Top, form, "    ", renderPreview(pc, o))
		}
		content = lipgloss.JoinVertical(lipgloss.Left, m.header(), "", body, "", m.help.View(globalKeys))
	}
	v := paint(tea.NewView(lipgloss.NewStyle().Padding(1, 3).Render(content)))
	v.AltScreen = true
	return v
}

// Run shows the form and returns the user's choices.
func Run(env Env) (*Choices, error) {
	c := defaults()
	if len(env.Saved) > 0 {
		c.SavedSel = library.Slug(env.Saved[0].Name)
	}
	m := model{
		form:    newStartForm(c, env),
		choices: c,
		env:     env,
		help:    newHelp(),
		motion:  &motion{off: env.NoMotion},
	}
	if !env.NoMotion {
		m.splash = newSplash()
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, err
	}
	fm := final.(model)
	if fm.aborted {
		return nil, ErrAborted
	}
	c = fm.choices
	if c.SavePath == "" {
		c.SavePath = c.DefaultSavePath()
	}
	return c, nil
}

func newHelp() help.Model {
	h := help.New()
	h.Styles = helpStyles()
	return h
}
