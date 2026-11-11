// Package theme builds Bear .theme files from a paper colour, an ink colour
// and a font. The key set mirrors Bear's built-in themes.
package theme

import (
	"encoding/json"
	"os"
)

// Rule is the near-black used for separators and table borders.
const Rule = "#1A1A1A"

// Options are the user's choices.
type Options struct {
	Name  string
	Paper string // #RRGGBB
	Ink   string // #RRGGBB
	Font  Font
	Size  int

	// Optional; derived from paper and ink when empty.
	Headings string // HeadingsInk, HeadingsBlack or #RRGGBB
	Lines    string // #RRGGBB for note-list separators and table borders
	Sidebar  SidebarColors
	NoteList NoteListColors
	Accents  Accents
	Type     Typography

	Highlights   Highlights
	Code         CodeColors
	Tag          TagColors
	HeadingScale float64

	// Extra sets any other theme key, by dotted path such as
	// "editor.toolbar.text color". Applied last, so it wins.
	Extra map[string]any
}

// HeadingColor resolves the Headings choice to a colour.
func (o Options) HeadingColor() string {
	switch {
	case o.Headings == HeadingsBlack:
		return o.contrastText()
	case ValidateHex(o.Headings) == nil:
		return o.Headings
	}
	return o.Ink
}

// LineColor is the soft stroke used for note-list separators and table borders.
func (o Options) LineColor() string {
	if ValidateHex(o.Lines) == nil {
		return o.Lines
	}
	return o.depth(o.Paper, 0.06)
}

type Theme struct {
	Base        Base        `json:"base"`
	Notes       Notes       `json:"notes"`
	Sidebar     Sidebar     `json:"sidebar"`
	Placeholder Placeholder `json:"placeholder"`
	Editor      Editor      `json:"editor"`

	extra map[string]any
}

type Base struct {
	TextColor           string `json:"text color"`
	TextSecondaryColor  string `json:"text secondary color"`
	TextTertiaryColor   string `json:"text tertiary color"`
	BackgroundColor     string `json:"background color"`
	BackgroundSecondary string `json:"background secondary color"`
	BackgroundTertiary  string `json:"background tertiary color"`
	StrokeColor         string `json:"stroke color"`
	AccentColor         string `json:"accent color"`
	SearchPrimary       string `json:"search primary color"`
	SearchSecondary     string `json:"search secondary color"`
	SelectionColor      string `json:"selection color"`
	PlaceholderColor    string `json:"placeholder color"`
}

type Notes struct {
	TitleColor          string       `json:"title color"`
	SubtitleColor       string       `json:"subtitle color"`
	PlaceholderColor    string       `json:"placeholder color"`
	DateColor           string       `json:"date color"`
	PinColor            string       `json:"pin color"`
	SeparatorColor      string       `json:"separator color"`
	SearchBackground    string       `json:"search background color"`
	AttachmentBg        string       `json:"attachment background color"`
	EncryptedTokenColor string       `json:"encrypted token color"`
	SelectionBackground string       `json:"selection background color"`
	RibbonColor         string       `json:"ribbon color"`
	Toolbar             NotesToolbar `json:"toolbar"`
}

type NotesToolbar struct {
	BackgroundColor string `json:"background color"`
	TextColor       string `json:"text color"`
	IconColor       string `json:"icon color"`
	HoverColor      string `json:"hover color"`
}

type Sidebar struct {
	TextColor           string         `json:"text color"`
	TextSecondaryColor  string         `json:"text secondary color"`
	BackgroundColor     string         `json:"background color"`
	BackgroundSecondary string         `json:"background secondary color"`
	StrokeColor         string         `json:"stroke color"`
	AccentColor         string         `json:"accent color"`
	IconColor           string         `json:"icon color"`
	SelectedIconColor   string         `json:"selected icon color"`
	Toolbar             SidebarToolbar `json:"toolbar"`
}

type SidebarToolbar struct {
	BackgroundColor string `json:"background color"`
	IconColor       string `json:"icon color"`
	HoverColor      string `json:"hover color"`
}

type Placeholder struct {
	BackgroundColor       string `json:"background color"`
	BackgroundStrokeColor string `json:"background stroke color"`
	ShadowColor           string `json:"shadow color"`
	StrokeColor           string `json:"stroke color"`
}

type Editor struct {
	BackgroundColor        string        `json:"background color"`
	TextColor              string        `json:"text color"`
	TextLightColor         string        `json:"text light color"`
	AccentColor            string        `json:"accent color"`
	CursorColor            string        `json:"cursor color"`
	LinkColor              string        `json:"link color"`
	ListMarkerColor        string        `json:"list marker color"`
	MarkerColor            string        `json:"marker color"`
	SelectionColor         string        `json:"selection color"`
	SelectionInactiveColor string        `json:"selection inactive color"`
	TextFont               string        `json:"text font"`
	TextSize               int           `json:"text size"`
	LineHeightMultiplier   float64       `json:"line height multiplier"`
	LineWidthMultiplier    float64       `json:"line width multiplier,omitempty"`
	ParagraphSpacing       float64       `json:"paragraph spacing multiplier,omitempty"`
	ParagraphIndent        float64       `json:"paragraph indent multiplier,omitempty"`
	Headers                Headers       `json:"headers"`
	Code                   Code          `json:"code"`
	Task                   Task          `json:"task"`
	Tag                    Tag           `json:"tag"`
	Highlighter            Highlighter   `json:"highlighter"`
	Separator              Border        `json:"separator"`
	Table                  Table         `json:"table"`
	Toolbar                EditorToolbar `json:"toolbar"`
}

type Headers struct {
	TextColor            string  `json:"text color"`
	Font                 string  `json:"font"`
	ModularScale         float64 `json:"modular scale"`
	LineHeightMultiplier float64 `json:"line height multiplier"`
	AddTopBottomPadding  int     `json:"add top bottom padding"`
	PaddingTop           float64 `json:"padding top multiplier"`
	PaddingBottom        float64 `json:"padding bottom multiplier"`
}

type Code struct {
	TextColor       string `json:"text color"`
	BorderColor     string `json:"border color"`
	BackgroundColor string `json:"background color"`
	Font            string `json:"font"`
	Syntax          Syntax `json:"syntax highlight"`
}

type Syntax struct {
	Comment  string `json:"comment"`
	Constant string `json:"constant"`
	Number   string `json:"number"`
	String   string `json:"string"`
	Entity   string `json:"entity"`
	Keyword  string `json:"keyword"`
	Function string `json:"function"`
	Variable string `json:"variable"`
}

type Task struct {
	BackgroundColor string `json:"background color"`
	BorderColor     string `json:"border color"`
	CheckColor      string `json:"check color"`
}

type Tag struct {
	BackgroundColor string `json:"background color"`
	TextColor       string `json:"text color"`
	MarkerColor     string `json:"marker color"`
}

type Highlight struct {
	BackgroundColor string `json:"background color"`
	TextColor       string `json:"text color"`
}

type Highlighter struct {
	Default Highlight `json:"default"`
	Red     Highlight `json:"red"`
	Blue    Highlight `json:"blue"`
	Green   Highlight `json:"green"`
	Yellow  Highlight `json:"yellow"`
	Purple  Highlight `json:"purple"`
}

type Border struct {
	BorderColor string `json:"border color"`
}

type Table struct {
	BorderColor        string `json:"border color"`
	CellBackground     string `json:"cell background color"`
	CellAlternateColor string `json:"cell alternate background color"`
}

type EditorToolbar struct {
	IconColor  string `json:"icon color"`
	HoverColor string `json:"hover color"`
}

// Build maps Options onto a full Bear theme.
func Build(o Options) Theme {
	paper, ink, lines := o.Paper, o.Ink, o.LineColor()
	side, list := o.SidebarResolved(), o.NoteListResolved()
	lineHeight := o.Type.LineHeight
	if lineHeight == 0 {
		lineHeight = DefaultLineHeight
	}
	// Unset accents keep Bear's reference to the accent colour.
	accent := func(v string) string { return or(v, "$base.accent color") }
	hl, code, tag := o.HighlightsResolved(), o.CodeResolved(), o.TagResolved()
	mark := func(bg string) Highlight { return Highlight{bg, readableOn(bg)} }
	text := o.contrastText()

	return Theme{
		extra: o.Extra,
		Base: Base{
			TextColor:           ink,
			TextSecondaryColor:  Mix(text, paper, 0.45),
			TextTertiaryColor:   Mix(text, paper, 0.65),
			BackgroundColor:     paper,
			BackgroundSecondary: o.depth(paper, 0.04),
			BackgroundTertiary:  o.depth(paper, 0.10),
			StrokeColor:         lines,
			AccentColor:         ink,
			SearchPrimary:       "#FFFA5E",
			SearchSecondary:     "$base.selection color",
			SelectionColor:      Mix(ink, paper, 0.8),
			PlaceholderColor:    "$base.text secondary color",
		},
		Notes: Notes{
			TitleColor:          list.Title,
			SubtitleColor:       list.Preview,
			PlaceholderColor:    "$base.text secondary color",
			DateColor:           list.Date,
			PinColor:            "$base.accent color",
			SeparatorColor:      "$base.stroke color",
			SearchBackground:    "$base.search primary color",
			AttachmentBg:        "$base.background secondary color",
			EncryptedTokenColor: "$base.background tertiary color",
			SelectionBackground: list.Selected,
			RibbonColor:         "$base.accent color",
			Toolbar: NotesToolbar{
				BackgroundColor: "$base.background color",
				TextColor:       "$base.text color",
				IconColor:       "$base.text secondary color",
				HoverColor:      "$base.background secondary color",
			},
		},
		Sidebar: Sidebar{
			TextColor:           side.Text,
			TextSecondaryColor:  paper,
			BackgroundColor:     side.Background,
			BackgroundSecondary: side.Selected,
			StrokeColor:         "$sidebar.background color",
			AccentColor:         side.Icons,
			IconColor:           side.Icons,
			SelectedIconColor:   paper,
			Toolbar: SidebarToolbar{
				BackgroundColor: "$sidebar.background color",
				IconColor:       "$sidebar.icon color",
				HoverColor:      "$sidebar.background secondary color",
			},
		},
		Placeholder: Placeholder{
			BackgroundColor:       "$base.background color",
			BackgroundStrokeColor: "$base.background tertiary color",
			ShadowColor:           "$base.background tertiary color",
			StrokeColor:           "$base.text tertiary color",
		},
		Editor: Editor{
			BackgroundColor:        "$base.background color",
			TextColor:              "$base.text color",
			TextLightColor:         Mix(ink, paper, 0.5),
			AccentColor:            "$base.accent color",
			CursorColor:            "$base.accent color",
			LinkColor:              accent(o.Accents.Links),
			ListMarkerColor:        accent(o.Accents.Markers),
			MarkerColor:            "$base.text tertiary color",
			SelectionColor:         "$base.selection color",
			SelectionInactiveColor: "$base.background tertiary color",
			TextFont:               o.Font.PostScript,
			TextSize:               o.Size,
			LineHeightMultiplier:   lineHeight,
			LineWidthMultiplier:    o.Type.LineWidth,
			ParagraphSpacing:       o.Type.ParagraphSpacing,
			ParagraphIndent:        o.Type.ParagraphIndent,
			Headers: Headers{
				TextColor:            o.HeadingColor(),
				Font:                 o.Font.Heading,
				ModularScale:         o.headingScale(),
				LineHeightMultiplier: 1.3,
				AddTopBottomPadding:  1,
				PaddingTop:           0.5,
				PaddingBottom:        0.3,
			},
			Code: Code{
				TextColor:       code.Text,
				BorderColor:     "$base.text tertiary color",
				BackgroundColor: or(o.Code.Background, "$base.background secondary color"),
				Font:            code.Font,
				Syntax: Syntax{
					Comment:  code.Comment,
					Constant: code.Constant,
					Number:   code.Number,
					String:   code.String,
					Entity:   code.Entity,
					Keyword:  code.Keyword,
					Function: code.Function,
					Variable: code.Variable,
				},
			},
			Task: Task{
				BackgroundColor: "$base.background color",
				BorderColor:     "$base.text secondary color",
				CheckColor:      accent(o.Accents.Ticks),
			},
			Tag: Tag{
				BackgroundColor: tag.Background,
				TextColor:       tag.Text,
				MarkerColor:     "$base.text secondary color",
			},
			Highlighter: Highlighter{
				Default: mark(hl.Default),
				Red:     mark(hl.Red),
				Blue:    mark(hl.Blue),
				Green:   mark(hl.Green),
				Yellow:  mark(hl.Yellow),
				Purple:  mark(hl.Purple),
			},
			Separator: Border{text},
			Table: Table{
				BorderColor:        lines,
				CellBackground:     "$base.background color",
				CellAlternateColor: "$base.background secondary color",
			},
			Toolbar: EditorToolbar{
				IconColor:  "$base.text secondary color",
				HoverColor: "$base.background secondary color",
			},
		},
	}
}

// JSON renders the theme the way Bear's own files are laid out. Extra keys,
// when there are any, are merged in last.
func (t Theme) JSON() ([]byte, error) {
	var v any = t
	if len(t.extra) > 0 {
		b, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		for path, val := range t.extra {
			SetPath(m, path, val)
		}
		v = m
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Write saves the theme to path.
func Write(o Options, path string) error {
	b, err := Build(o).JSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
