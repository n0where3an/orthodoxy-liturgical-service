package orthlib

import (
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

func TestTriodionPrep(t *testing.T) {
	units, warnings := ImportTriodionPrep(extractedRoot(t))
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	got := map[string]corpus.Unit{}
	for _, u := range units {
		got[u.ID] = u
	}
	// What the liturgy takes from each day (triodionProfiles in package liturgy).
	need := map[string][]string{
		"publican": {"kontakion", "canon.ode.6.troparion.1"}, "prodigal": {"kontakion", "canon.ode.6.troparion.1"},
		"sat_meatfare":   {"kontakion", "canon.ode.3.troparion.1", "canon.ode.6.troparion.1"},
		"meatfare":       {"kontakion", "canon.ode.6.troparion.1", "lit.rubric"},
		"sat_cheesefare": {"kontakion", "canon.ode.3.troparion.1", "canon.ode.6.troparion.1"},
		"cheesefare":     {"kontakion", "canon.ode.6.troparion.1", "lit.rubric"},
		"sat_lent1":      {"kontakion", "canon.ode.3.troparion.1", "canon.ode.6.troparion.1", "lit.gospel.2"},
		"lent1":          {"kontakion", "canon.ode.6.troparion.1", "lit.rubric"},
		"sat_lent2":      {"lit.apostle.2", "lit.gospel.2"},
		"lent2":          {"kontakion"},
		"lent3":          {"troparion", "kontakion", "canon.ode.3.troparion.1"},
		"sat_lent4":      {"lit.apostle.2", "lit.gospel.2"},
		"lent4":          {"kontakion", "lit.apostle.2", "lit.gospel.2"},
		"sat_akathist":   {"canon.ode.3.troparion.1", "canon.ode.6.troparion.1", "lit.apostle.2", "lit.gospel.2"},
		"lent5":          {"kontakion", "lit.apostle.2", "lit.gospel.2"},
	}
	for _, day := range triodionPrepDays {
		for _, k := range need[day.id] {
			if _, ok := got["tp."+day.id+"."+k]; !ok {
				t.Errorf("tp.%s.%s missing", day.id, k)
			}
		}
	}
	// The second readings: copies of the lectionary's by the rubric's pericope.
	for id, src := range map[string]string{"tp.sat_cheesefare.lit.gospel.2": "lect.zach.mt.43", "tp.sat_lent4.lit.apostle.2": "lect.zach.ap.163",
		"tp.lent5.lit.gospel.2": "lect.zach.lk.33", "tp.sat_akathist.lit.apostle.2": "lect.zach.ap.320"} {
		if got[id].Meta["copy"] != src {
			t.Errorf("%s: copy %v, want %s", id, got[id].Meta["copy"], src)
		}
	}
	for _, id := range []string{"tp.sat_meatfare.troparion", "tp.sat_meatfare.theotokion", "tp.sat_cheesefare.troparion"} {
		if _, ok := got[id]; !ok {
			t.Errorf("%s missing", id)
		}
	}
	// The own prokeimena: meatfare «Велий Господь наш» (3), cheesefare
	// «Помолитеся» (8), the departed «Души их» (6, not 7: ѕ in the source).
	for id, tone := range map[string]int{"tp.meatfare.lit.rubric": 3, "tp.cheesefare.lit.rubric": 8, "tp.sat_meatfare.lit.rubric": 6, "tp.sat_cheesefare.lit.rubric": 4} {
		if got[id].Meta["prok_tone"] != tone {
			t.Errorf("%s: prokeimenon tone %v, want %d", id, got[id].Meta["prok_tone"], tone)
		}
	}
	// «Слава, троичен:» is the glory, not a troparion.
	if b := got["tp.sat_meatfare.canon.ode.3.glory"].Body; b == "" {
		t.Error("sat_meatfare: no glory of the 3rd ode")
	}
}
