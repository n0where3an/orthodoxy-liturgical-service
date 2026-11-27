package orthlib

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

func TestImportOctoechosMatins(t *testing.T) {
	units, warnings := ImportOctoechosMatins(extractedRoot(t))
	for _, w := range warnings {
		t.Error(w)
	}
	ids := map[string]bool{}
	for _, u := range units {
		if ids[u.ID] {
			t.Errorf("duplicate %s", u.ID)
		}
		ids[u.ID] = true
	}
	p := "oc.3.thu.matins."
	for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
		for c := 1; c <= 2; c++ {
			for _, k := range []string{"irmos", "troparion.1", "theotokion"} {
				if id := fmt.Sprintf("%scanon.%d.ode.%d.%s", p, c, o, k); !ids[id] {
					t.Errorf("missing %s", id)
				}
			}
		}
	}
	for _, id := range []string{p + "sedalen.1.1", p + "sedalen.1.verse.2", p + "sedalen.1.theotokion",
		p + "sedalen.2.3", p + "sedalen.3.1", p + "aposticha.sticheron.3", p + "aposticha.verse.2", p + "aposticha.theotokion",
		"oc.1.mon.matins.canon.2.ode.6.irmos",  // «Ин, ирмос тойже»
		"oc.7.tue.matins.sedalen.1.theotokion", // ". Слава, и ныне" misprint
	} {
		if !ids[id] {
			t.Errorf("missing %s", id)
		}
	}
}

func menaionMatins(t *testing.T, month string, day string) map[string]corpus.Unit {
	t.Helper()
	path := filepath.Join(extractedRoot(t), "mineya", month, month, map[string]string{"nov": "Ноябрь", "dec": "Декабрь"}[month], day+".hip")
	us, err := ParseMenaionDay(path, map[string]int{"nov": 11, "dec": 12}[month], day)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]corpus.Unit{}
	for _, u := range us {
		if strings.Contains(u.ID, ".matins.") {
			out[u.ID[strings.Index(u.ID, ".matins.")+8:]] = u
		}
	}
	return out
}

func TestMenaionMatins(t *testing.T) {
	nov4 := menaionMatins(t, "nov", "04")
	for _, k := range []string{"canon.1.ode.1.irmos", "canon.2.ode.9.theotokion", "canon.1.ode.8.troparion.4",
		"sedalen.1", "sedalen.glory", "sedalen.theotokion", "ikos.1", "exapostilarion.1", "exapostilarion.theotokion",
		"aposticha.glory", "aposticha.theotokion"} {
		if _, ok := nov4[k]; !ok {
			t.Errorf("11.04: missing %s", k)
		}
	}
	if _, ok := nov4["canon.3.ode.1.troparion.1"]; ok {
		t.Error("11.04: two canons only")
	}
	// Dec 4: the Typikon's note «Канон Октоиха един… и святых два канона»
	// is not a canon; the second saint's troparia under «Преподобнаго:».
	dec4 := menaionMatins(t, "dec", "04")
	for _, k := range []string{"canon.2.ode.1.irmos", "canon.2.ode.4.troparion.1", "sedalen.cross_theotokion"} {
		if _, ok := dec4[k]; !ok {
			t.Errorf("12.04: missing %s", k)
		}
	}
	if _, ok := dec4["canon.3.ode.1.troparion.1"]; ok {
		t.Error("12.04: two canons only")
	}
	// Dec 15: a second service printed in the same file is not merged.
	dec15 := menaionMatins(t, "dec", "15")
	if _, ok := dec15["canon.3.ode.1.troparion.1"]; ok {
		t.Error("12.15: the second service's canon merged into the first")
	}
	if u := dec15["canon.1.ode.1.troparion.3"]; u.ID == "" {
		t.Error("12.15: first canon missing")
	} else if _, ok := dec15["canon.1.ode.1.troparion.4"]; ok {
		t.Error("12.15: troparia of another service added to the first canon")
	}
	// Dec 6: «канон бцды со ирмосом на 6» (a rubric), then the canon itself.
	dec6 := menaionMatins(t, "dec", "06")
	if _, ok := dec6["canon.1.theotokos"]; !ok {
		t.Error("12.06: the first canon not marked as the Theotokos's")
	}
	for k := range dec6 {
		if strings.HasSuffix(k, ".theotokos") && k != "canon.1.theotokos" {
			t.Errorf("12.06: %s", k)
		}
	}
	if _, ok := nov4["canon.1.theotokos"]; ok {
		t.Error("11.04: no canon of the Theotokos")
	}
}

// «Ирмос: Воду прошед…»: the irmos in the heading's paragraph (its first
// letter red with the heading); «ин канон… Ирмос: …» opens the next canon.
func TestMenaionInlineIrmos(t *testing.T) {
	keys := func(file string) map[string]string {
		d, err := LoadDoc(filepath.Join(extractedRoot(t), "mineya", file))
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, u := range parseMenaionMatins(d, "me.x", file) {
			out[strings.TrimPrefix(u.ID, "me.x.matins.")] = u.Body
		}
		return out
	}
	k := keys(filepath.Join("feb", "feb", "Февраль", "20p.hip"))
	if !strings.HasPrefix(k["canon.1.ode.1.irmos"], "Во́дꙋ проше́дъ") || !strings.HasPrefix(k["canon.1.ode.3.irmos"], "Нбⷭ҇нагѡ кр") {
		t.Errorf("02.20p: %q, %q", k["canon.1.ode.1.irmos"], k["canon.1.ode.3.irmos"])
	}
	k = keys(filepath.Join("jan", "jan", "Январь", "02p.hip"))
	if k["canon.1.ode.1.irmos"] == "" || k["canon.2.ode.1.irmos"] == "" || k["canon.2.ode.1.troparion.1"] == "" {
		t.Errorf("01.02p: the second canon's 1st ode: %v", k["canon.2.ode.1.irmos"])
	}
}
