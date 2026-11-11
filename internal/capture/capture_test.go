package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/manojsachwani/bearty/internal/theme"
)

const bearThemes = "/Applications/Bear.app/Contents/Frameworks/BearCore.framework/Versions/A/Resources"

func check(t *testing.T, path, dir string) {
	t.Helper()
	m, err := theme.Load(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := theme.Resolve(theme.Flatten(m))
	s, err := File(path, dir, filepath.Base(path))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	got, err := theme.Build(s.Options()).Values()
	if err != nil {
		t.Fatal(err)
	}
	for k, w := range want {
		if !same(got[k], w) {
			t.Errorf("%s: %s = %v, want %v", filepath.Base(path), k, got[k], w)
		}
	}
}

// The two built-ins' original files capture back without needing Extra.
func TestCaptureBeartyThemes(t *testing.T) {
	for _, f := range []string{"paper-pro.theme", "all-these-things.theme"} {
		path := filepath.Join("..", "library", "testdata", f)
		check(t, path, "")
		s, _ := File(path, "", f)
		if len(s.Extra) > 0 {
			t.Errorf("%s: a bearty theme should capture with no extra keys, got %v", f, s.Extra)
		}
	}
}

// Every theme in Bear's own folder must rebuild exactly.
func TestCaptureEveryBearTheme(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join(bearThemes, "*.theme"))
	if len(files) == 0 {
		t.Skip("Bear not installed")
	}
	for _, f := range files {
		check(t, f, bearThemes)
	}
	t.Logf("captured %d themes", len(files))
}

func TestCaptureRejectsNonTheme(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.theme")
	os.WriteFile(p, []byte(`{"hello": 1}`), 0o644)
	if _, err := File(p, "", "x"); err == nil {
		t.Error("a file with no colors should be rejected")
	}
}
