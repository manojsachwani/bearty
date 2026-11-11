package fonts

import (
	"context"
	"os"
	"testing"
	"time"
)

const sampleMeta = `name: "Kalam"
designer: "Indian Type Foundry"
fonts {
  name: "Kalam"
  style: "normal"
  weight: 300
  filename: "Kalam-Light.ttf"
  post_script_name: "Kalam-Light"
}
fonts {
  name: "Kalam"
  style: "normal"
  weight: 400
  filename: "Kalam-Regular.ttf"
  post_script_name: "Kalam-Regular"
}
fonts {
  name: "Kalam"
  style: "italic"
  weight: 400
  filename: "Kalam-Italic.ttf"
  post_script_name: "Kalam-Italic"
}
`

func TestParseMetadataPicksRegular(t *testing.T) {
	family, faces := parseMetadata(sampleMeta)
	f, ok := regular(faces)
	if family != "Kalam" || !ok || f.file != "Kalam-Regular.ttf" || f.postScript != "Kalam-Regular" {
		t.Errorf("got %q %+v", family, f)
	}
}

func TestGoogleSlug(t *testing.T) {
	if got := GoogleSlug("Shadows Into Light"); got != "shadowsintolight" {
		t.Errorf("GoogleSlug = %s", got)
	}
}

// Set BEARTY_NETWORK_TESTS=1 to check every pick against Google Fonts.
func TestPicksResolveOnGoogle(t *testing.T) {
	if os.Getenv("BEARTY_NETWORK_TESTS") == "" {
		t.Skip("BEARTY_NETWORK_TESTS not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, p := range append(append([]Pick{}, Picks...), CodePicks...) {
		family, ok := GoogleFamily(p.Source)
		if !ok {
			continue
		}
		g, err := LookupGoogle(ctx, family)
		if err != nil || g.PostScript != p.PostScript {
			t.Errorf("%s: got %+v, %v; want PostScript %s", family, g, err, p.PostScript)
		}
	}
	if _, err := LookupGoogle(ctx, "Definitely Not A Real Font"); err == nil {
		t.Error("unknown family should fail")
	}
}

// Set BEARTY_NETWORK_TESTS=1 to download a real file and read its name.
func TestFetchGoogleFile(t *testing.T) {
	if os.Getenv("BEARTY_NETWORK_TESTS") == "" {
		t.Skip("BEARTY_NETWORK_TESTS not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	g, err := LookupGoogle(ctx, "Patrick Hand")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path, err := FetchGoogle(ctx, g, dir)
	if err != nil {
		t.Fatal(err)
	}
	faces := readFaces(path)
	if len(faces) != 1 || faces[0].PostScript != g.PostScript {
		t.Errorf("downloaded file has %+v, want %s", faces, g.PostScript)
	}
}

func TestStyleParsing(t *testing.T) {
	if w, i := ParseStyle("700i"); w != 700 || !i {
		t.Errorf("700i = %d %v", w, i)
	}
	if got := StyleLabel("300i"); got != "Light Italic" {
		t.Errorf("label = %s", got)
	}
	if got := GoogleStyleOf("google:Caveat"); got != RegularStyle {
		t.Errorf("old sources mean Regular, got %s", got)
	}
	if fam, _ := GoogleFamily("google:Caveat@700"); fam != "Caveat" {
		t.Errorf("family = %s", fam)
	}
}

// Set BEARTY_NETWORK_TESTS=1: weights of a variable and a static family, and
// a bold download whose name is read from the file.
func TestGoogleStylesAndFetch(t *testing.T) {
	if os.Getenv("BEARTY_NETWORK_TESTS") == "" {
		t.Skip("BEARTY_NETWORK_TESTS not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	caveat, err := GoogleStyles(ctx, "Caveat") // variable 400–700
	if err != nil || len(caveat) != 4 || caveat[0] != "400" || caveat[3] != "700" {
		t.Errorf("Caveat styles = %v, %v", caveat, err)
	}
	kalam, _ := GoogleStyles(ctx, "Kalam") // static 300, 400, 700
	if len(kalam) != 3 || kalam[0] != "300" {
		t.Errorf("Kalam styles = %v", kalam)
	}
	path, ps, err := FetchGoogleStyle(ctx, "Caveat", "700", t.TempDir())
	if err != nil || ps != "Caveat-Bold" {
		t.Errorf("Caveat Bold: %s %q %v", path, ps, err)
	}
	if _, _, err := FetchGoogleStyle(ctx, "Caveat", "900", t.TempDir()); err == nil {
		t.Error("a weight the family lacks should fail")
	}
}
