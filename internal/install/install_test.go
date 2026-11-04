package install

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/manojsachwani/bearty/internal/theme"
)

func fakeBear(t *testing.T) {
	t.Helper()
	old := BearApp
	BearApp = t.TempDir()
	t.Cleanup(func() { BearApp = old })
	if err := os.MkdirAll(ThemesDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"Notes", "Dark Notes"} {
		os.WriteFile(filepath.Join(ThemesDir(), n+".theme"), []byte(`{"orig":"`+n+`"}`), 0o644)
	}
}

func TestSlots(t *testing.T) {
	fakeBear(t)
	got, err := Slots()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "Dark Notes" || got[1] != "Notes" {
		t.Errorf("Slots = %v", got)
	}
}

func TestBackupOnlyOnce(t *testing.T) {
	fakeBear(t)
	t.Setenv("HOME", t.TempDir())
	os.MkdirAll(filepath.Join(os.Getenv("HOME"), "Downloads"), 0o755)

	path, created, err := Backup("Notes")
	if err != nil || !created {
		t.Fatalf("first backup: created=%v err=%v", created, err)
	}
	// Simulate an installed custom theme; a second backup must not copy it.
	os.WriteFile(filepath.Join(ThemesDir(), "Notes.theme"), []byte(`{"custom":1}`), 0o644)
	if _, created, _ := Backup("Notes"); created {
		t.Error("second backup overwrote the original")
	}
	if b, _ := os.ReadFile(path); string(b) != `{"orig":"Notes"}` {
		t.Errorf("backup = %s", b)
	}
}

func TestOtfFromZipSkipsMacMetadata(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("__MACOSX/._Inkina-Regular.otf")
	w.Write([]byte("junk"))
	w, _ = zw.Create("Inkina-Regular.otf")
	w.Write([]byte("font"))
	zw.Close()

	got, err := otfFromZip(buf.Bytes())
	if err != nil || string(got) != "font" {
		t.Errorf("otfFromZip = %q, %v", got, err)
	}
}

// Set BEARTY_INKINA_OTF to a real Inkina-Regular.otf to check its name.
func TestInkinaPostScriptName(t *testing.T) {
	path := os.Getenv("BEARTY_INKINA_OTF")
	if path == "" {
		t.Skip("BEARTY_INKINA_OTF not set")
	}
	name, err := PostScriptName(path)
	if err != nil || name != "Inkina-Regular" {
		t.Errorf("PostScriptName = %q, %v", name, err)
	}
}

func TestBackupsListsOnlySavedSlots(t *testing.T) {
	fakeBear(t)
	t.Setenv("HOME", t.TempDir())
	os.MkdirAll(filepath.Join(os.Getenv("HOME"), "Downloads"), 0o755)

	if got := Backups([]string{"Dark Notes", "Notes"}); len(got) != 0 {
		t.Errorf("Backups before any backup = %v", got)
	}
	Backup("Notes")
	if got := Backups([]string{"Dark Notes", "Notes"}); len(got) != 1 || got[0] != "Notes" {
		t.Errorf("Backups = %v", got)
	}
}

func TestAddedFontsOnlyListsBeartyFonts(t *testing.T) {
	fakeBear(t)
	t.Setenv("HOME", t.TempDir()) // CacheDir is under ~/Library/Caches on macOS
	cache, _ := CacheDir()
	os.MkdirAll(cache, 0o755)
	os.MkdirAll(FontsDir(), 0o755)
	for _, f := range []string{"Inkina-Regular.otf", "Caveat[wght].ttf"} {
		os.WriteFile(filepath.Join(cache, f), []byte("x"), 0o644)
	}
	os.WriteFile(filepath.Join(FontsDir(), "Inkina-Regular.otf"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(FontsDir(), "BearSansUI-Regular.otf"), []byte("x"), 0o644)

	got := AddedFonts()
	if len(got) != 1 || got[0] != "Inkina-Regular.otf" {
		t.Errorf("AddedFonts = %v", got)
	}
}

// Uses a plist file instead of Bear's real preferences.
func TestTypographyBackupApplyRestore(t *testing.T) {
	dir := t.TempDir()
	old := BearDefaults
	BearDefaults = filepath.Join(dir, "bear.plist")
	t.Cleanup(func() { BearDefaults = old })

	// The user's own settings before bearty: a font and size, no indent.
	ApplyTypography([]Pref{{Key: "SFEditorFont", Type: "string", Value: "Avenir-Book"}, {Key: "SFEditorFontSize", Type: "int", Value: "17"}})

	backup := filepath.Join(dir, "typography.json")
	if created, err := BackupTypography(backup); err != nil || !created {
		t.Fatalf("backup: %v %v", created, err)
	}
	o := theme.Options{Size: 24, Font: theme.Font{PostScript: "IndieFlower-Regular", Heading: "PatrickHand-Regular"},
		Type: theme.Typography{LineHeight: 1.4, ParagraphIndent: 0.2}}
	if err := ApplyTypography(TypographyPrefs(o)); err != nil {
		t.Fatal(err)
	}
	got := map[string]Pref{}
	for _, p := range ReadTypography() {
		got[p.Key] = p
	}
	if got["SFEditorFont"].Value != "IndieFlower-Regular" || got["SFEditorFontSize"].Value != "24" ||
		got["SFEditorHeadingsFont"].Value != "PatrickHand-Regular" || got["SFEditorParagraphIndentMultiplier"].Value != "0.2" {
		t.Errorf("applied: %+v", got)
	}
	if created, _ := BackupTypography(backup); created {
		t.Error("a second backup must not replace the first")
	}

	if err := RestoreTypography(backup); err != nil {
		t.Fatal(err)
	}
	got = map[string]Pref{}
	for _, p := range ReadTypography() {
		got[p.Key] = p
	}
	if got["SFEditorFont"].Value != "Avenir-Book" || got["SFEditorFontSize"].Value != "17" || !got["SFEditorParagraphIndentMultiplier"].Missing {
		t.Errorf("restored: %+v", got)
	}
}
