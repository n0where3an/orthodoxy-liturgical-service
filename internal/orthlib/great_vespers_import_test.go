package orthlib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMenaionGreatVespers(t *testing.T) {
	d, err := LoadDoc(filepath.Join(extractedRoot(t), "mineya", "dec", "dec", "Декабрь", "06.hip"))
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{}
	for _, u := range parseMenaionGreatVespers(d, "me.12.06", "06.hip") {
		keys[strings.TrimPrefix(u.ID, "me.12.06.gvespers.")] = u.Body
	}
	// The great vespers, not the small ones printed first.
	if !strings.HasPrefix(keys["lic.sticheron.1"], "Въ мѵ́рѣхъ") {
		t.Errorf("first sticheron: %q", keys["lic.sticheron.1"])
	}
	for _, k := range []string{"lic.on.8", "lic.sticheron.8", "lic.glory", "lic.now", "paremia.1.title", "paremia.3",
		"litiya.sticheron.6", "litiya.glory", "litiya.now", "aposticha.sticheron.3", "aposticha.verse.2",
		"aposticha.glory", "aposticha.now", "loaves"} {
		if keys[k] == "" {
			t.Errorf("missing %s", k)
		}
	}
	// «Дважды»: the sticheron takes two places.
	if keys["lic.sticheron.5"] != keys["lic.sticheron.6"] {
		t.Error("«Киими похвальными» (дважды) not repeated")
	}
	// «И ныне, глас тойже: Византиево:» is a heading, not a cue.
	if !strings.HasPrefix(keys["aposticha.now"], "Безневѣ́стнаѧ") {
		t.Errorf("aposticha now: %q", keys["aposticha.now"])
	}
	if strings.Contains(keys["paremia.1"], "[") {
		t.Error("chapter mark left in the reading")
	}
}

func TestGreatVespersReferences(t *testing.T) {
	units, _, err := ImportMenaion(extractedRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, u := range units {
		ids[u.ID] = true
	}
	// 10.09: «чтения три, и лития: ищи сего месяца в ѕ҃ день».
	for _, id := range []string{"me.10.09.gvespers.see", "me.10.09.gvespers.paremia.3"} {
		if !ids[id] {
			t.Errorf("missing %s", id)
		}
	}
	th, err := ImportResurrectionalTheotokia(extractedRoot(t))
	if err != nil || len(th) != 16 || !strings.HasPrefix(th[10].Body, "Кто̀ тебє̀ не ѹ҆блажи́тъ") {
		t.Errorf("resurrectional theotokia: %d, %v", len(th), err)
	}
}
