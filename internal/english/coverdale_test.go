package english

import (
	"os"
	"strings"
	"testing"
)

func coverdalePath(t *testing.T) string {
	p := os.Getenv("COVERDALE_TXT")
	if p == "" {
		t.Skip("COVERDALE_TXT not set")
	}
	return p
}

func TestCoverdale(t *testing.T) {
	ps, err := ParseCoverdale(coverdalePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.Psalms[1][0]; got != "Blessed is the man that hath not walked in the counsel of the ungodly, nor stood in the way of sinners: and hath not sat in the seat of the scornful." {
		t.Errorf("Ps 1:1 = %q", got)
	}
	if got := ps.Psalms[83][0]; !strings.HasPrefix(got, "O how amiable are thy dwellings") {
		t.Errorf("Ps 83:1 = %q", got)
	}
	if got := ps.Psalms[103][0]; !strings.HasPrefix(got, "Praise the Lord, O my soul") {
		t.Errorf("Ps 103:1 = %q", got)
	}
	for k := 1; k <= 20; k++ {
		for s := 1; s <= 3; s++ {
			if len(ps.Parts[[2]int{k, s}]) == 0 {
				t.Errorf("kathisma %d stasis %d empty", k, s)
			}
		}
	}
	for _, v := range ps.Psalms[3] {
		if strings.Contains(v, "A psalm of David") {
			t.Errorf("superscription kept: %q", v)
		}
	}
	units, warnings := coverdaleUnits(ps)
	for _, w := range warnings {
		t.Error(w)
	}
	if len(units) != 60+14 {
		t.Errorf("%d units", len(units))
	}
}
