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
	for _, p := range Picks {
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
