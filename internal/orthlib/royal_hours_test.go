package orthlib

import (
	"fmt"
	"path/filepath"
	"testing"
)

// The Royal Hours of both eves: every hour complete, the stichera with
// their verses and repetitions, the canonarch's sticheron at the 9th hour.
func TestRoyalHoursImport(t *testing.T) {
	root := extractedRoot(t)
	for feast, file := range royalHoursFiles {
		d, err := LoadDoc(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		units, err := parseRoyalHours(d, feast)
		if err != nil {
			t.Error(err)
		}
		got := map[string]map[string]any{}
		kinds := map[string]string{}
		for _, u := range units {
			got[u.ID], kinds[u.ID] = u.Meta, u.Kind
		}
		p := "rh." + feast + "."
		for _, h := range []int{1, 3, 6, 9} {
			verses := 0
			for n := 1; n <= 12; n++ {
				if kinds[fmt.Sprintf("%s%d.verse.%d", p, h, n)] == "verse" {
					verses++
				}
			}
			if verses != 2 {
				t.Errorf("%s hour %d: %d verses", feast, h, verses)
			}
			if got[fmt.Sprintf("%s%d.prokeimenon", p, h)]["tone"] == nil {
				t.Errorf("%s hour %d: prokeimenon without its tone", feast, h)
			}
		}
		canonarch := false
		for id, m := range got {
			if m["canonarch"] == true && id[len(p):len(p)+1] == "9" {
				canonarch = true
			}
		}
		if !canonarch {
			t.Errorf("%s: no canonarch's sticheron at the 9th hour", feast)
		}
	}
}
