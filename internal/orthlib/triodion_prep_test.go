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
	for _, day := range triodionPrepDays {
		for _, k := range []string{"kontakion", "lit.rubric", "canon.ode.3.troparion.1", "canon.ode.6.troparion.1", "canon.ode.6.theotokion"} {
			if _, ok := got["tp."+day.id+"."+k]; !ok {
				t.Errorf("tp.%s.%s missing", day.id, k)
			}
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
