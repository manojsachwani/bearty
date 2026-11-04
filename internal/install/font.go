package install

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// InkinaURL is the free download from shapedfonts.com. Its licence does not
// allow redistribution, so the font is fetched on the user's machine.
const InkinaURL = "https://shapedfonts.com/wp-content/uploads/2017/05/Inkina-Regular.otf_.zip"

// InkinaFile is the font file name inside Bear.app.
const InkinaFile = "Inkina-Regular.otf"

// CacheDir holds downloaded fonts so re-installs after a Bear update work offline.
func CacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bearty"), nil
}

// FetchInkina returns the cached Inkina .otf, downloading it if needed.
func FetchInkina(ctx context.Context) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, InkinaFile)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, InkinaURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download Inkina: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", err
	}
	otf, err := otfFromZip(body)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, otf, 0o644)
}

func otfFromZip(b []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "__MACOSX/") || !strings.EqualFold(filepath.Ext(f.Name), ".otf") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, 10<<20))
	}
	return nil, fmt.Errorf("no .otf in download")
}

// PostScriptName reads the name Bear's "text font" key expects.
func PostScriptName(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	f, err := sfnt.Parse(b)
	if err != nil {
		return "", err
	}
	return f.Name(nil, sfnt.NameIDPostScript)
}
