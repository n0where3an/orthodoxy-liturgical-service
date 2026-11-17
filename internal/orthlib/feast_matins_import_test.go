package orthlib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMenaionFeastMatins(t *testing.T) {
	d, err := LoadDoc(filepath.Join(extractedRoot(t), "mineya", "dec", "dec", "Декабрь", "06.hip"))
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]map[string]any{}
	for _, u := range parseMenaionFeastMatins(d, "me.12.06", "06.hip") {
		keys[strings.TrimPrefix(u.ID, "me.12.06.matins.")] = u.Meta
	}
	for _, k := range []string{"k_sedalen.1.1", "k_sedalen.1.glory", "k_sedalen.2.theotokion", "k_sedalen.p.1",
		"ps50.sticheron", "praises.on.6", "praises.sticheron.6", "praises.glory", "praises.now", "rubric"} {
		if keys[k] == nil {
			t.Errorf("missing %s", k)
		}
	}
	if r := keys["rubric"]; r == nil || r["gospel"] != 36 || r["gospel_book"] != "jn" || r["prok_tone"] != 4 {
		t.Errorf("rubric: %v", r)
	}
}

func TestPsalmsAndTheotokia(t *testing.T) {
	ps, err := ImportPsalms(extractedRoot(t), 134, 135)
	if err != nil || len(ps) != 2 || !strings.HasPrefix(ps[0].Body, "Хвали́те и҆́мѧ гдⷭ҇не") {
		t.Errorf("psalms: %v", err)
	}
	th, err := ImportResurrectionalDismissalTheotokia(extractedRoot(t))
	if err != nil || len(th) != 8 || !strings.HasPrefix(th[3].Body, "Є҆́же ѿ вѣ́ка") {
		t.Errorf("theotokia: %v", err)
	}
}
