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

func TestMenaionNativityTheophanyMatins(t *testing.T) {
	units := func(month, file string) map[string]bool {
		d, err := LoadDoc(filepath.Join(extractedRoot(t), "mineya", month, month, file))
		if err != nil {
			t.Fatal(err)
		}
		keys := map[string]bool{}
		for _, u := range parseMenaionMatins(d, "me.x", file) {
			keys[strings.TrimPrefix(u.ID, "me.x.matins.")] = true
		}
		for _, u := range parseMenaionFeastMatins(d, "me.x", file) {
			keys[strings.TrimPrefix(u.ID, "me.x.matins.")] = true
		}
		return keys
	}
	check := func(name string, keys map[string]bool, have, lack []string) {
		for _, k := range have {
			if !keys[k] {
				t.Errorf("%s: missing %s", name, k)
			}
		}
		for _, k := range lack {
			if keys[k] {
				t.Errorf("%s: unexpected %s", name, k)
			}
		}
	}
	// The Nativity: the 7th ode with no heading after the ikos, the ipakoi
	// after the 3rd ode, the 9th ode's refrains not taken for troparia.
	check("12.25", units("dec", filepath.Join("Декабрь", "25.hip")),
		[]string{"canon.1.ode.7.irmos", "canon.2.ode.9.troparion.2", "sedalen.1"},
		[]string{"canon.3.ode.9.irmos", "canon.1.ode.9.troparion.4"})
	// The Theophany: begins at the sessional hymn after the hexapsalmos.
	check("01.06", units("jan", filepath.Join("Январь", "06.hip")),
		[]string{"k_sedalen.1.1", "ps50.glory", "ps50.now", "ps50.sticheron", "rubric", "praises.glory"}, nil)
	// The Circumcision: «таже стаго припевы» begins the saint's canon.
	check("01.01", units("jan", filepath.Join("Январь", "01.hip")),
		[]string{"canon.2.ode.9.troparion.5"}, []string{"canon.1.ode.9.troparion.3"})
}
