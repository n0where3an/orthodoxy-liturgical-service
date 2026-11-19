package orthlib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Saturday matins of the Octoechos: every tone has the sessional hymns
// after both kathismata, the martyrs' canon with its eight odes and the
// martyrs' stichera of the praises.
func TestOctoechosSaturdayMatinsImport(t *testing.T) {
	units, warnings := ImportOctoechosSaturdayMatins(extractedRoot(t))
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	have := map[string]string{}
	for _, u := range units {
		if have[u.ID] != "" {
			t.Errorf("duplicate %s", u.ID)
		}
		have[u.ID] = strings.ToLower(hip.PlainString(u.Body))
	}
	for tone := 1; tone <= 8; tone++ {
		p := fmt.Sprintf("oc.%d.sat.matins.", tone)
		need := []string{"sedalen.1.1", "sedalen.1.2", "sedalen.1.theotokion", "sedalen.2.3", "sedalen.2.theotokion",
			"praises.sticheron.1", "praises.sticheron.2", "praises.sticheron.3", "praises.theotokion",
			"aposticha.sticheron.3", "aposticha.theotokion"}
		for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
			b := fmt.Sprintf("canon.1.ode.%d.", o)
			need = append(need, b+"irmos", b+"troparion.3", b+"glory", b+"theotokion", fmt.Sprintf("canon.2.ode.%d.troparion.1", o))
		}
		for _, k := range need {
			if have[p+k] == "" {
				t.Errorf("no %s%s", p, k)
			}
		}
	}
	// Tone 1: «в субботу на утрени»; the label «мчнком:» is not sung.
	for id, prefix := range map[string]string{
		"oc.1.sat.matins.sedalen.1.1":               "страдания похвалою",
		"oc.1.sat.matins.canon.1.ode.1.irmos":       "столпом огненным",
		"oc.1.sat.matins.canon.1.ode.1.troparion.1": "щитом блгочестия",
		"oc.1.sat.matins.canon.1.ode.1.glory":       "мертв быв",
		"oc.1.sat.matins.canon.2.ode.1.irmos":       "твоя победительная десница",
		"oc.1.sat.matins.praises.departed":          "делом спсе мой",
		"oc.4.sat.matins.praises.theotokion":        "едина чстая и пречстая",
		"oc.8.sat.matins.canon.1.ode.8.glory":       "правосуде",
		"oc.8.sat.matins.praises.departed":          "безмерна есть",
		"oc.8.sat.matins.aposticha.theotokion":      "яко заченши",
		"oc.5.sat.matins.canon.1.ode.5.theotokion":  "поем тя дво",
	} {
		if got := have[id]; !strings.HasPrefix(got, prefix) {
			t.Errorf("%s: %q, want %q…", id, got[:min(len(got), 60)], prefix)
		}
	}
}
