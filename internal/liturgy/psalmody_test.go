package liturgy

import (
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

// expandUnits expands a template with the given slots (the others empty)
// and returns the performer of each unit it shows.
func expandUnits(t *testing.T, name string, psalmody string, slots map[string][]SlotItem) map[string]Performer {
	t.Helper()
	tpl, err := LoadTemplate(dataDir(), name)
	if err != nil {
		t.Fatal(err)
	}
	lang, err := LoadLang(dataDir(), "cu")
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Service: tpl.Service, Params: Params{Priest: true, Psalmody: psalmody}, Slots: map[string][]SlotItem{}}
	var walk func([]Node)
	walk = func(ns []Node) {
		for _, n := range ns {
			if n.Slot != "" {
				plan.Slots[n.Slot] = nil
			}
			walk(n.Children)
		}
	}
	walk(tpl.Nodes)
	for k, v := range slots {
		plan.Slots[k] = v
	}
	texts := map[string]corpus.Text{}
	for _, ref := range tpl.Refs() {
		texts[ref] = corpus.Text{UnitID: ref, Body: ref}
	}
	for _, items := range plan.Slots {
		for _, it := range items {
			texts[it.Ref] = corpus.Text{UnitID: it.Ref, Body: it.Ref}
		}
	}
	nodes, err := Expand(tpl, plan, texts, lang)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Performer{}
	for _, n := range nodes {
		if n.UnitID != "" {
			out[n.UnitID] = n.Performer
		}
	}
	return out
}

// «Господи, воззвах»: the beginning sung; from «Положи, Господи» to the verse
// of the first sticheron as the practice is — left out, sung or read.
func TestPsalmodyLordICall(t *testing.T) {
	stichera := []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8"} // on 8: verses 21, 22 without stichera
	for _, c := range []struct {
		psalmody string
		has      bool
		perf     Performer
	}{{"short", false, ""}, {"sung", true, Choir}, {"read", true, Reader}, {"", false, ""}} {
		f := Facts{Psalmody: Params{Psalmody: c.psalmody}.PsalmodyOrDefault()}
		units := expandUnits(t, "great_vespers", c.psalmody, map[string][]SlotItem{"lord_i_call": licWithVerses(f, stichera)})
		if units["ch.vespers.lic.verse.1"] != Choir || units["ch.vespers.lic.verse.3"] != Choir {
			t.Errorf("%q: the beginning not sung", c.psalmody)
		}
		for _, v := range []string{"ch.vespers.lic.verse.4", "ch.vespers.lic.verse.20", "ch.vespers.lic.verse.21", "ch.vespers.lic.verse.22"} {
			p, ok := units[v]
			if ok != c.has || ok && p != c.perf {
				t.Errorf("%q: %s present %v as %q", c.psalmody, v, ok, p)
			}
		}
		if _, ok := units["ch.vespers.lic.verse.23"]; !ok {
			t.Errorf("%q: no verse of the first sticheron", c.psalmody)
		}
	}
	// Daily vespers: the same up to the 25th verse.
	if units := expandUnits(t, "vespers", "short", nil); units["ch.vespers.lic.verse.24"] != "" || units["ch.vespers.lic.verse.25"] == "" {
		t.Errorf("daily vespers, short: 24 %q, 25 %q", units["ch.vespers.lic.verse.24"], units["ch.vespers.lic.verse.25"])
	}
}

// The praises: read with the doxology read; with it sung the beginning sung
// and the rest as the practice is.
func TestPsalmodyPraises(t *testing.T) {
	gd := map[string][]SlotItem{"great_doxology": {{Ref: "ch.matins.great_doxology.1"}}}
	for _, c := range []struct {
		tpl, psalmody string
		slots         map[string][]SlotItem
		begin, ps148  Performer
	}{
		{"matins", "short", nil, Reader, Reader}, // the doxology read: all read
		{"matins", "sung", nil, Reader, Reader},
		{"matins", "short", gd, Choir, ""},
		{"matins", "sung", gd, Choir, Choir},
		{"matins", "read", gd, Choir, Reader},
		{"great_matins", "short", nil, Choir, ""},
		{"great_matins", "read", nil, Choir, Reader},
	} {
		units := expandUnits(t, c.tpl, c.psalmody, c.slots)
		if units["ch.matins.praises.1"] != c.begin || units["ch.matins.praises.148"] != c.ps148 {
			t.Errorf("%s %q great doxology %v: «Хвалите Господа с небес» %q, Ps 148 %q", c.tpl, c.psalmody, c.slots != nil,
				units["ch.matins.praises.1"], units["ch.matins.praises.148"])
		}
	}
}

func TestPraisesVersesBeforeStichera(t *testing.T) {
	f := Facts{Menaion: map[string][]string{"matins.praises.sticheron": {"me.10.14.matins.praises.sticheron.1"}},
		Psalmody: "short"}
	f.Day.Menaion.Month, f.Day.Menaion.Day = 10, 14
	f.Day.Weekday = time.Tuesday
	// Black sign: read, with «Сотворити в них суд».
	items, _ := gmPraises(f)
	if refs(items[:1]) != "ch.matins.praises.149.verse" {
		t.Errorf("black: %s", refs(items))
	}
	// Red sign, short: straight to the verse of the first sticheron.
	f.Menaion["matins.great_doxology"] = []string{"me.10.14.matins.great_doxology"}
	items, _ = gmPraises(f)
	if refs(items[:1]) != "ch.matins.praises.150.2" {
		t.Errorf("red, short: %s", refs(items))
	}
}
