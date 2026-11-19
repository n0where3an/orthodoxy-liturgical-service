package orthlib

import (
	"testing"
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
