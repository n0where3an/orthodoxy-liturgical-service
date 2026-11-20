package orthlib

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

func TestImportSpecialDays(t *testing.T) {
	units, warnings := ImportSpecialDays(extractedRoot(t))
	for _, w := range warnings {
		t.Error(w)
	}
	got := map[string]map[string]any{}
	for _, u := range units {
		got[u.ID] = u.Meta
	}
	for id, want := range map[string]map[string]any{
		"ms.fathers.lit.rubric":           {"apostle": 328, "gospel": 1, "gospel_book": "mt"},
		"ms.after_nativity.lit.rubric":    {"apostle": 200, "gospel": 4, "gospel_book": "mt"},
		"ms.before_exaltation.lit.rubric": {"apostle": 215, "gospel": 9, "gospel_book": "jn"},
		"ms.before_theophany.lit.rubric":  {"apostle": 298, "gospel": 1, "gospel_book": "mk"},
		"ms.after_theophany.lit.rubric":   {"apostle": 224, "gospel": 8, "gospel_book": "mt"},
		"ms.after_exaltation.lit.rubric":  {"apostle": 203, "gospel": 37, "gospel_book": "mk"},
	} {
		m := got[id]
		if m == nil {
			t.Errorf("missing %s", id)
			continue
		}
		for k, v := range want {
			if m[k] != v {
				t.Errorf("%s: %s = %v, want %v", id, k, m[k], v)
			}
		}
	}
	for _, id := range []string{"ms.fathers.lit.rubric", "ms.after_nativity.lit.rubric", "ms.before_exaltation.lit.rubric",
		"ms.before_theophany.lit.rubric", "ms.after_theophany.lit.rubric"} {
		if m := got[id]; m != nil && (m["prok"] == nil || m["alleluia"] == nil) {
			t.Errorf("%s: no prokeimenon or alleluia: %v", id, m)
		}
	}
}

// The liturgy of the Nativity and the Theophany: «на литургии: антифон а»,
// «апстол», «вместо тристаго, елицы во хрста»; the troparion of the 3rd
// antiphon given in full and repeated by its first words.
func TestNativityTheophanyLiturgyImport(t *testing.T) {
	for _, c := range []struct{ file, month, day string }{
		{"mineya/dec/dec/Декабрь/25.hip", "12", "25"}, {"mineya/jan/jan/Январь/06.hip", "01", "06"},
	} {
		m, _ := strconv.Atoi(c.month)
		units, err := ParseMenaionDay(filepath.Join(extractedRoot(t), c.file), m, c.day)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]corpus.Unit{}
		for _, u := range units {
			got[u.ID] = u
		}
		p := "me." + c.month + "." + c.day + ".lit."
		for _, k := range []string{"rubric", "antiphon.1.1", "antiphon.3.2", "entrance", "trisagion"} {
			if _, ok := got[p+k]; !ok {
				t.Errorf("missing %s%s", p, k)
			}
		}
		if got[p+"trisagion"].Meta["copy"] != "lit.trisagion.baptized" {
			t.Errorf("%strisagion: %v", p, got[p+"trisagion"].Meta)
		}
		r := got[p+"rubric"].Meta
		if r["apostle"] == nil || r["gospel"] == nil || r["prok"] == nil || r["kinonik"] == nil {
			t.Errorf("%srubric: %v", p, r)
		}
		// The 3rd antiphon: the troparion in full, then repeated.
		if u := got[p+"antiphon.3.2"]; u.Meta["copy"] != nil || u.Kind != "troparion" {
			t.Errorf("%santiphon.3.2: %s %v", p, u.Kind, u.Meta)
		}
		if u := got[p+"antiphon.3.8"]; u.Meta["repeat_of"] != p+"antiphon.3.2" {
			t.Errorf("%santiphon.3.8: %v %s", p, u.Meta, u.Body)
		}
	}
}
