package orthlib

import (
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestOctoechosVespers(t *testing.T) {
	root := extractedRoot(t)
	units, warnings := ImportOctoechosVespers(root)
	for _, w := range warnings {
		t.Error(w)
	}
	byID := map[string]string{}
	for _, u := range units {
		byID[u.ID] = hip.PlainString(u.Body)
	}
	// Tone 1, Monday (Sunday evening).
	checks := map[string]string{
		"oc.1.mon.vespers.lic.sticheron.1":       "вся привел еси",
		"oc.1.mon.vespers.lic.theotokion":        "мрие непорочная",
		"oc.1.mon.vespers.aposticha.sticheron.1": "яко пучина многа",
		"oc.1.mon.vespers.aposticha.verse.2":     "к тебе возведох очи мои",
		"oc.1.mon.vespers.aposticha.sticheron.3": "млтвами гдси",
		"oc.1.mon.vespers.aposticha.theotokion":  "нбсных чинов радование",
		// Tone 5, Wednesday (Tuesday evening): cross-theotokion in the next paragraph.
		"oc.5.wed.vespers.aposticha.theotokion": "весь восприим мой зрак",
		// Tone 3, Saturday (Friday evening): labelled stichera, no verses.
		"oc.3.sat.vespers.aposticha.sticheron.2": "все суетие человеческое",
	}
	for id, prefix := range checks {
		if got, ok := byID[id]; !ok || !strings.HasPrefix(got, prefix) {
			t.Errorf("%s = %q, want prefix %q", id, got, prefix)
		}
	}
	if len(units) < 6*8*10 {
		t.Errorf("only %d units", len(units))
	}
}
