package orthlib

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// The vespers of the eves: the stichera at «Господи, воззвах» (four, each
// twice), 8 / 13 readings, two troparia with their verses and refrains, the
// liturgy's rubric.
func TestEveVespersImport(t *testing.T) {
	root := extractedRoot(t)
	for feast, want := range map[string]int{"nativity": 8, "theophany": 13} {
		d, err := LoadDoc(filepath.Join(root, eveVespersFiles[feast]))
		if err != nil {
			t.Fatal(err)
		}
		units, err := parseEveVespers(d, feast)
		if err != nil {
			t.Error(err)
		}
		counts := map[string]int{}
		var rubric map[string]any
		for _, u := range units {
			k := strings.TrimPrefix(u.ID, "ev."+feast+".")
			kind := k[:strings.Index(k+".", ".")]
			if u.Meta["copy"] == nil {
				counts[kind]++
			}
			if k == "lit.rubric" {
				rubric = u.Meta
			}
			if kind == "reading" && strings.HasPrefix(strings.ToLower(u.Body), "а҆́ще") {
				t.Errorf("%s: a rubric as a reading: %.60s", u.ID, u.Body)
			}
		}
		if counts["reading"] != want || counts["title"] != want || counts["troparion"] != 2 || counts["refrain"] != 2 ||
			counts["verse"] != 6 || counts["lic"] != 5 {
			t.Errorf("%s: %v", feast, counts)
		}
		if rubric["apostle"] == nil || rubric["gospel"] == nil || rubric["prok"] == nil || rubric["alleluia"] == nil {
			t.Errorf("%s: liturgy rubric %v", feast, rubric)
		}
		if feast == "nativity" {
			var wv map[string]any
			for _, u := range units {
				if u.ID == "ev.nativity.wv.lit.rubric" {
					wv = u.Meta
				}
			}
			if wv["apostle"] != 207 || wv["gospel"] != 53 || wv["prok"] == nil {
				t.Errorf("the vespers apart: %v", wv)
			}
		}
	}
}

// The great compline of the feasts: the stichera at the litiya and the
// aposticha with their verses.
func TestFeastComplineImport(t *testing.T) {
	root := extractedRoot(t)
	for feast, want := range map[string][2]int{"nativity": {4, 3}, "theophany": {5, 3}} {
		d, err := LoadDoc(filepath.Join(root, eveVespersFiles[feast]))
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, u := range parseFeastCompline(d, feast) {
			got[strings.TrimPrefix(u.ID, "gc."+feast+".")] = true
			if testing.Verbose() {
				t.Logf("%s %.50s", u.ID, u.Body)
			}
		}
		for _, k := range []string{fmt.Sprintf("litiya.%d", want[0]), fmt.Sprintf("aposticha.%d", want[1]), "aposticha.verse.2"} {
			if !got[k] {
				t.Errorf("%s: no %s", feast, k)
			}
		}
		if !(got["litiya.glory"] && got["litiya.now"]) && !got["litiya.glory_now"] {
			t.Errorf("%s: no «Слава» at the litiya: %v", feast, got)
		}
	}
}
