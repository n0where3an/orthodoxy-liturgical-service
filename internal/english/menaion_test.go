package english

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/orthlib"
)

func TestAlignOctober30(t *testing.T) {
	en, orth := os.Getenv("ENGLISH_MD_DIR"), os.Getenv("ORTHLIB_EXTRACTED")
	if en == "" || orth == "" {
		t.Skip("ENGLISH_MD_DIR / ORTHLIB_EXTRACTED not set")
	}
	services, err := LoadServices(en)
	if err != nil {
		t.Fatal(err)
	}
	cu, err := orthlib.ParseMenaionDay(filepath.Join(orth, orthlib.MonthDir(10), "30.hip"), 10, "30")
	if err != nil {
		t.Fatal(err)
	}
	units, stats := Align(services, cu)
	t.Logf("stats %v", stats)
	by := map[string]string{}
	for _, u := range units {
		by[u.ID] = u.Body
	}
	for id, want := range map[string]string{
		"me.10.30.troparion.1":             "In their sufferings, O Lord, Thy martyrs",
		"me.10.30.vespers.lic.sticheron.1": "Having dyed thy vesture",
		"me.10.30.vespers.lic.sticheron.3": "Being of one mind with thee",
		"me.10.30.vespers.lic.glory":       "Let us form a choir",
		"me.10.30.vespers.lic.theotokion":  "The most immaculate Theotokos",
	} {
		if !strings.HasPrefix(by[id], want) {
			t.Errorf("%s = %.60q, want %q", id, by[id], want)
		}
	}
}
