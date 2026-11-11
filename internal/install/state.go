package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Installed records what bearty last put into Bear, so it can tell when a
// Bear update has undone it and put it back.
type Installed struct {
	Theme      string    `json:"theme"` // saved theme name
	Light      SlotFile  `json:"light"`
	Dark       *SlotFile `json:"dark,omitempty"`
	Fonts      []string  `json:"fonts,omitempty"` // font files copied into Bear.app
	Typography bool      `json:"typography"`      // Bear's typography was set to match
	At         time.Time `json:"at"`
}

// SlotFile is one theme file bearty wrote over a Bear slot.
type SlotFile struct {
	Slot   string `json:"slot"`
	SHA256 string `json:"sha256"`
	Copy   string `json:"copy"` // bearty's own copy, used to put it back
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// NewSlotFile records data installed over slot, keeping a copy in dir.
func NewSlotFile(slot string, data []byte, dir string) (SlotFile, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SlotFile{}, err
	}
	cp := filepath.Join(dir, slot+".theme")
	if err := os.WriteFile(cp, data, 0o644); err != nil {
		return SlotFile{}, err
	}
	return SlotFile{Slot: slot, SHA256: sum(data), Copy: cp}, nil
}

// SaveInstalled writes the record.
func SaveInstalled(path string, in Installed) error {
	b, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// LoadInstalled reads the record; ok is false when nothing was installed.
func LoadInstalled(path string) (Installed, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Installed{}, false
	}
	var in Installed
	if json.Unmarshal(b, &in) != nil || in.Light.Slot == "" {
		return Installed{}, false
	}
	return in, true
}

// Undone reports whether Bear no longer has what bearty installed, which
// happens when a Bear update restores its own themes and fonts.
func (in Installed) Undone() bool {
	for _, f := range in.slots() {
		b, err := os.ReadFile(filepath.Join(ThemesDir(), f.Slot+".theme"))
		if err != nil || sum(b) != f.SHA256 {
			return true
		}
	}
	for _, font := range in.Fonts {
		if _, err := os.Stat(filepath.Join(FontsDir(), filepath.Base(font))); err != nil {
			return true
		}
	}
	return false
}

func (in Installed) slots() []SlotFile {
	s := []SlotFile{in.Light}
	if in.Dark != nil {
		s = append(s, *in.Dark)
	}
	return s
}

// SlotHas reports whether Bear's slot holds exactly data.
func SlotHas(slot string, data []byte) bool {
	b, err := os.ReadFile(filepath.Join(ThemesDir(), slot+".theme"))
	return err == nil && sum(b) == sum(data)
}
