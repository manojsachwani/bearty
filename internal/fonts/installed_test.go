package fonts

import (
	"os"
	"testing"
	"time"
)

// Runs against the real Mac fonts; skipped where there are none.
func TestInstalledFindsSystemFonts(t *testing.T) {
	if _, err := os.Stat("/System/Library/Fonts"); err != nil {
		t.Skip("no macOS fonts")
	}
	start := time.Now()
	faces := Installed(Dirs(), "/Applications/Bear.app/Contents/Resources")
	t.Logf("%d faces, %d families in %v", len(faces), len(Families(faces)), time.Since(start))
	for _, ps := range []string{"Noteworthy-Light", "Helvetica"} {
		if _, ok := ByPostScript(faces, ps); !ok {
			t.Errorf("%s not found", ps)
		}
	}
	if f, ok := ByPostScript(faces, "BearSansUI-Regular"); ok && !f.BearOnly {
		t.Error("Bear's bundled font should be marked BearOnly")
	}
	if s := Styles(faces, "Noteworthy"); len(s) < 2 {
		t.Errorf("Noteworthy styles = %v", s)
	}
}
