package install

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/manojsachwani/bearty/internal/theme"
)

// BearDefaults is Bear's preferences domain. Its Settings → Typography values
// override a theme's font, size and spacing, so a theme's font only shows up
// once these match. A plist path works too, which tests use.
var BearDefaults = "net.shinyfrog.bear"

// Pref is one typed preference value.
type Pref struct {
	Key     string `json:"key"`
	Type    string `json:"type"` // string, int or float
	Value   string `json:"value"`
	Missing bool   `json:"missing,omitempty"` // not set before bearty changed it
}

// typographyKeys are the Settings → Typography preferences bearty manages.
var typographyKeys = []struct{ key, typ string }{
	{"SFEditorFont", "string"},
	{"SFEditorHeadingsFont", "string"},
	{"SFEditorFontSize", "int"},
	{"SFEditorLineHeightMultiplier", "float"},
	{"SFEditorLineWidthMultiplier", "int"},
	{"SFEditorParagraphSpacingMultiplier", "float"},
	{"SFEditorParagraphIndentMultiplier", "float"},
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// TypographyPrefs are Bear's typography settings for a theme. Settings the
// theme leaves at Bear's default are not changed.
func TypographyPrefs(o theme.Options) []Pref {
	lh := o.Type.LineHeight
	if lh == 0 {
		lh = theme.DefaultLineHeight
	}
	prefs := []Pref{
		{Key: "SFEditorFont", Type: "string", Value: o.Font.PostScript},
		{Key: "SFEditorHeadingsFont", Type: "string", Value: o.Font.Heading},
		{Key: "SFEditorFontSize", Type: "int", Value: strconv.Itoa(o.Size)},
		{Key: "SFEditorLineHeightMultiplier", Type: "float", Value: num(lh)},
	}
	if o.Type.LineWidth != 0 {
		prefs = append(prefs, Pref{Key: "SFEditorLineWidthMultiplier", Type: "int", Value: strconv.Itoa(int(o.Type.LineWidth))})
	}
	if o.Type.ParagraphSpacing != 0 {
		prefs = append(prefs, Pref{Key: "SFEditorParagraphSpacingMultiplier", Type: "float", Value: num(o.Type.ParagraphSpacing)})
	}
	if o.Type.ParagraphIndent != 0 {
		prefs = append(prefs, Pref{Key: "SFEditorParagraphIndentMultiplier", Type: "float", Value: num(o.Type.ParagraphIndent)})
	}
	return prefs
}

// ReadTypography reads Bear's current typography settings.
func ReadTypography() []Pref {
	var prefs []Pref
	for _, k := range typographyKeys {
		out, err := exec.Command("defaults", "read", BearDefaults, k.key).Output()
		p := Pref{Key: k.key, Type: k.typ, Value: strings.TrimSpace(string(out))}
		if err != nil {
			p.Value, p.Missing = "", true
		}
		prefs = append(prefs, p)
	}
	return prefs
}

// BackupTypography saves Bear's current typography to path, once: a later
// install keeps the first backup, so it always holds the user's own settings.
func BackupTypography(path string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	b, err := json.MarshalIndent(ReadTypography(), "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, append(b, '\n'), 0o644)
}

// ApplyTypography writes the settings to Bear.
func ApplyTypography(prefs []Pref) error {
	for _, p := range prefs {
		if err := writePref(p); err != nil {
			return err
		}
	}
	return nil
}

// RestoreTypography puts back the settings saved by BackupTypography and
// removes the backup.
func RestoreTypography(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var prefs []Pref
	if err := json.Unmarshal(b, &prefs); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	for _, p := range prefs {
		if p.Missing {
			exec.Command("defaults", "delete", BearDefaults, p.Key).Run() // absent already is fine
			continue
		}
		if err := writePref(p); err != nil {
			return err
		}
	}
	return os.Remove(path)
}

func writePref(p Pref) error {
	flag := map[string]string{"string": "-string", "int": "-int", "float": "-float"}[p.Type]
	if flag == "" {
		return fmt.Errorf("unknown type %q for %s", p.Type, p.Key)
	}
	if out, err := exec.Command("defaults", "write", BearDefaults, p.Key, flag, p.Value).CombinedOutput(); err != nil {
		return fmt.Errorf("set Bear's %s: %v: %s", p.Key, err, strings.TrimSpace(string(out)))
	}
	return nil
}
