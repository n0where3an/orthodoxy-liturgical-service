package orthlib

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// These tests need the extracted orthlib tree:
//
//	ORTHLIB_EXTRACTED=var/orthlib go test ./internal/orthlib
func extractedRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("ORTHLIB_EXTRACTED")
	if root == "" {
		t.Skip("ORTHLIB_EXTRACTED not set")
	}
	return root
}

func repoFile(rel string) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", rel)
}

func TestManifestExtracts(t *testing.T) {
	root := extractedRoot(t)
	m, err := LoadManifest(repoFile("data/orthlib/units.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	units, err := m.Extract(root)
	if err != nil {
		t.Fatal(err)
	}
	srcs := map[string]bool{}
	for _, s := range m.Sources {
		srcs[s.ID] = true
	}
	for _, u := range units {
		if !srcs[u.SourceID] {
			t.Errorf("%s: unknown source %q", u.ID, u.SourceID)
		}
		if strings.TrimSpace(u.Body) == "" {
			t.Errorf("%s: empty body", u.ID)
		}
		if len([]rune(u.Body)) > 4000 {
			t.Errorf("%s: suspiciously long body (%d runes)", u.ID, len([]rune(u.Body)))
		}
	}
	// Spot checks of exact boundaries.
	want := map[string][2]string{
		"common.trisagion":  {"стый бже", "помилуй нас."},
		"ch.none.psalm.83":  {"коль возлюбленна", "уповаяй на тя."},
		"ch.none.psalm.85":  {"приклони гдси", "утешил мя еси."},
		"common.our_father": {"оче наш", "от лукаваго."},
	}
	for _, u := range units {
		if w, ok := want[u.ID]; ok {
			p := hip.PlainString(u.Body)
			if !strings.HasPrefix(p, w[0]) || !strings.HasSuffix(p, w[1]) {
				t.Errorf("%s: %q", u.ID, p)
			}
		}
	}
}

func TestMenaionSeptember25(t *testing.T) {
	root := extractedRoot(t)
	units, err := ParseMenaionDay(filepath.Join(root, MonthDir(9), "25.hip"), 9, "25")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]corpus.Unit{}
	var kont int
	for _, u := range units {
		byID[u.ID] = u
		if u.Kind == "kontakion" {
			kont++
		}
	}
	// 25 September: St Euphrosyne (tone 8) and St Sergius of Radonezh (tone 4).
	checks := []struct {
		id     string
		tone   int
		prefix string
	}{
		{"me.09.25.troparion.1", 8, "в тебе мати, известно спасеся"},
		{"me.09.25.troparion.2", 4, "иже добродетелей подвижник"},
	}
	for _, c := range checks {
		u, ok := byID[c.id]
		if !ok {
			t.Errorf("%s missing", c.id)
			continue
		}
		if u.Tone != c.tone {
			t.Errorf("%s tone %d, want %d", c.id, u.Tone, c.tone)
		}
		if p := hip.PlainString(u.Body); !strings.HasPrefix(p, c.prefix) {
			t.Errorf("%s: %q", c.id, p)
		}
	}
	if kont < 1 {
		t.Errorf("no kontakion")
	}
}
