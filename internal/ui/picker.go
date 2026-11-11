package ui

import (
	"fmt"
	"io"
	"math"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/manojsachwani/bearty/internal/theme"
)

// picker is a Huh field for choosing a colour: hue, saturation and lightness
// bars, a palette on the number keys, and hex entry with # or a paste.
//
// An optional picker may be left empty, meaning "automatic"; it then shows
// the automatic colour and x returns to it.
type picker struct {
	key, title string
	desc       func() string
	value      *string
	optional   bool
	auto       func() string // the colour in effect while empty
	palette    []theme.Swatch

	h, s, l float64
	row     int // 0 hue, 1 saturation, 2 lightness
	typing  bool
	buf     string

	focused bool
	width   int
	theme   huh.Theme
	dark    bool

	prev, next, submit key.Binding
}

var pickerKeys = struct {
	rows, adjust, hex, palette, clear key.Binding
}{
	rows:    key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "bar")),
	adjust:  key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→", "adjust")),
	hex:     key.NewBinding(key.WithKeys("#"), key.WithHelp("#", "type hex")),
	palette: key.NewBinding(key.WithKeys("1"), key.WithHelp("1-9", "palette")),
	clear:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "automatic")),
}

// newPicker binds a required colour; seed gives the starting colour when the
// value is empty.
func newPicker(key, title string, v *string, seed func() string) *picker {
	return &picker{key: key, title: title, value: v, auto: seed, palette: append(append([]theme.Swatch{}, theme.Inks...), theme.Papers...)}
}

// newOptionalPicker binds a colour that may stay automatic.
func newOptionalPicker(key, title string, v *string, auto func() string) *picker {
	p := newPicker(key, title, v, auto)
	p.optional = true
	return p
}

func (p *picker) Description(f func() string) *picker { p.desc = f; return p }

// current is the colour shown: the value, or the automatic/seed colour.
func (p *picker) current() string {
	if theme.ValidateHex(*p.value) == nil {
		return strings.ToUpper(*p.value)
	}
	return p.auto()
}

func (p *picker) sync() { p.h, p.s, p.l = toHSL(p.current()) }

func (p *picker) set(hex string) {
	*p.value = strings.ToUpper(hex)
	p.sync()
}

func (p *picker) adjust(delta float64) {
	switch p.row {
	case 0:
		p.h = math.Mod(p.h+delta*3+360, 360)
	case 1:
		p.s = clamp01(p.s + delta/100)
	case 2:
		p.l = clamp01(p.l + delta/100)
	}
	*p.value = fromHSL(p.h, p.s, p.l)
}

// GetFiltering makes bearty treat hex entry as typing, so esc and ? reach it.
func (p *picker) GetFiltering() bool { return p.typing }

func (p *picker) Init() tea.Cmd { return nil }

func (p *picker) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		p.dark = msg.IsDark()
	case tea.PasteMsg:
		if hex := parseHex(msg.Content); hex != "" {
			p.set(hex)
		}
	case tea.KeyPressMsg:
		if p.typing {
			p.typeHex(msg)
			return p, nil
		}
		k := msg.String()
		switch {
		case key.Matches(msg, p.prev):
			return p, huh.PrevField
		case key.Matches(msg, p.next, p.submit):
			if !p.optional && theme.ValidateHex(*p.value) != nil {
				p.set(p.current()) // commit the colour on screen
			}
			return p, huh.NextField
		case k == "up" || k == "k":
			p.row = max(p.row-1, 0)
		case k == "down" || k == "j":
			p.row = min(p.row+1, 2)
		case k == "left" || k == "h":
			p.adjust(-1)
		case k == "right" || k == "l":
			p.adjust(1)
		case k == "shift+left" || k == "H":
			p.adjust(-10)
		case k == "shift+right" || k == "L":
			p.adjust(10)
		case k == "#":
			p.typing, p.buf = true, ""
		case len(k) == 1 && k >= "1" && k <= "9":
			if i := int(k[0] - '1'); i < len(p.palette) {
				p.set(p.palette[i].Hex)
			}
		case p.optional && (k == "x" || k == "delete" || k == "backspace"):
			*p.value = ""
			p.sync()
		}
	}
	return p, nil
}

func (p *picker) typeHex(msg tea.KeyPressMsg) {
	k := msg.String()
	switch {
	case k == "esc":
		p.typing = false
	case k == "backspace":
		if p.buf == "" {
			p.typing = false
		} else {
			p.buf = p.buf[:len(p.buf)-1]
		}
	case len(k) == 1 && strings.Contains("0123456789abcdefABCDEF", k):
		p.buf += strings.ToUpper(k)
		if len(p.buf) == 6 {
			p.set("#" + p.buf)
			p.typing = false
		}
	}
}

func parseHex(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		s = "#" + s
	}
	if theme.ValidateHex(s) != nil {
		return ""
	}
	return strings.ToUpper(s)
}

func (p *picker) styles() *huh.FieldStyles {
	t := p.theme
	if t == nil {
		t = huh.ThemeFunc(formTheme)
	}
	if p.focused {
		return &t.Theme(p.dark).Focused
	}
	return &t.Theme(p.dark).Blurred
}

func (p *picker) View() string {
	st := p.styles()
	cur := p.current()
	inner := max(p.width-st.Base.GetHorizontalFrameSize(), 20)

	value := lipgloss.NewStyle().Foreground(lipgloss.Color(cur)).Render("███████") + " " + cur
	switch {
	case p.typing:
		value += lipgloss.NewStyle().Foreground(ember).Render("   #" + p.buf + "▏")
	case p.optional && *p.value == "":
		value += lipgloss.NewStyle().Foreground(dim).Render("  automatic")
	}

	lines := []string{st.Title.Render(p.title)}
	if p.focused && p.desc != nil {
		if d := p.desc(); d != "" {
			lines = append(lines, st.Description.Render(d))
		}
	}
	lines = append(lines, value)
	if p.focused {
		barW := max(inner-10, 10)
		lines = append(lines,
			p.bar(0, "H", barW, p.h/360, func(t float64) string { return fromHSL(t*360, 0.75, 0.55) }, fmt.Sprintf("%3.0f°", p.h)),
			p.bar(1, "S", barW, p.s, func(t float64) string { return fromHSL(p.h, t, p.l) }, fmt.Sprintf("%3.0f%%", p.s*100)),
			p.bar(2, "L", barW, p.l, func(t float64) string { return fromHSL(p.h, p.s, t) }, fmt.Sprintf("%3.0f%%", p.l*100)),
			p.paletteRow(),
		)
	}
	return st.Base.Width(p.width).Render(strings.Join(lines, "\n"))
}

// bar draws one gradient with a marker at the current position.
func (p *picker) bar(row int, label string, w int, at float64, colourAt func(float64) string, val string) string {
	mark := int(math.Round(clamp01(at) * float64(w-1)))
	var b strings.Builder
	for i := range w {
		c := colourAt(float64(i) / float64(w-1))
		cell := lipgloss.NewStyle().Background(lipgloss.Color(c))
		if i == mark {
			b.WriteString(cell.Foreground(lipgloss.Color(contrast(c))).Bold(true).Render("┃"))
		} else {
			b.WriteString(cell.Render(" "))
		}
	}
	lead := lipgloss.NewStyle().Foreground(dim).Render("  " + label + " ")
	if row == p.row {
		lead = lipgloss.NewStyle().Foreground(ember).Bold(true).Render("▸ " + label + " ")
	}
	return lead + b.String() + lipgloss.NewStyle().Foreground(dim).Render(" "+val)
}

func (p *picker) paletteRow() string {
	parts := []string{lipgloss.NewStyle().Foreground(dim).Render("   ")}
	for i, s := range p.palette {
		if i == 9 {
			break
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(faintFg).Render(fmt.Sprint(i+1))+
			lipgloss.NewStyle().Foreground(lipgloss.Color(s.Hex)).Render("██"))
	}
	return strings.Join(parts, " ")
}

func (p *picker) Focus() tea.Cmd { p.focused = true; p.sync(); return nil }
func (p *picker) Blur() tea.Cmd  { p.focused, p.typing = false, false; return nil }
func (p *picker) Error() error   { return nil }
func (p *picker) Run() error     { return huh.Run(p) }
func (p *picker) Skip() bool     { return false }
func (p *picker) Zoom() bool     { return false }
func (p *picker) GetKey() string { return p.key }
func (p *picker) GetValue() any  { return *p.value }

func (p *picker) RunAccessible(w io.Writer, _ io.Reader) error {
	_, err := fmt.Fprintln(w, p.title+": "+p.current())
	return err
}

func (p *picker) KeyBinds() []key.Binding {
	b := []key.Binding{pickerKeys.rows, pickerKeys.adjust, pickerKeys.hex, pickerKeys.palette}
	if p.optional {
		b = append(b, pickerKeys.clear)
	}
	return append(b, p.prev, p.next, p.submit)
}

func (p *picker) WithTheme(t huh.Theme) huh.Field {
	if p.theme == nil {
		p.theme = t
	}
	return p
}

func (p *picker) WithKeyMap(k *huh.KeyMap) huh.Field {
	p.prev, p.next, p.submit = k.Input.Prev, k.Input.Next, k.Input.Submit
	return p
}

func (p *picker) WithWidth(w int) huh.Field { p.width = w; return p }
func (p *picker) WithHeight(int) huh.Field  { return p }
func (p *picker) WithPosition(pos huh.FieldPosition) huh.Field {
	p.prev.SetEnabled(!pos.IsFirst())
	p.next.SetEnabled(!pos.IsLast())
	p.submit.SetEnabled(pos.IsLast())
	return p
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// contrast picks black or white text for a background.
func contrast(hex string) string {
	r, g, b := rgb(hex)
	if 0.299*r+0.587*g+0.114*b > 150 {
		return "#1A1A1A"
	}
	return "#FFFFFF"
}

func toHSL(hex string) (h, s, l float64) { return theme.ToHSL(hex) }
func fromHSL(h, s, l float64) string     { return theme.FromHSL(h, s, l) }
