package orthlib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The canons of compline of the Octoechos: every tone and day has its nine
// odes (eight: no second ode).
func TestOctoechosComplineImport(t *testing.T) {
	root := extractedRoot(t)
	units, warnings := ImportOctoechosCompline(root)
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	have := map[string]string{}
	for _, u := range units {
		have[u.ID] = strings.ToLower(hip.PlainString(u.Body))
	}
	for tone := 1; tone <= 8; tone++ {
		for _, day := range weekdayKey {
			for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
				id := fmt.Sprintf("oc.%d.%s.compline.canon.ode.%d.troparion.1", tone, day, o)
				if have[id] == "" {
					t.Errorf("no %s", id)
				}
			}
		}
	}
	// Tone 1, Sunday evening (the Monday file): «канон молебный престей бцде».
	if got := have["oc.1.mon.compline.canon.ode.1.irmos"]; !strings.HasPrefix(got, "твоя победительная десница") {
		t.Errorf("irmos: %q", got)
	}
	if got := have["oc.1.mon.compline.canon.ode.1.glory"]; !strings.HasPrefix(got, "всея моея жизни предстательнице") {
		t.Errorf("glory: %q", got)
	}
	if got := have["oc.1.mon.compline.sedalen"]; !strings.HasPrefix(got, "млтвенницу ко гдсу") {
		t.Errorf("sedalen: %q", got)
	}
	// The 6th ode's misprinted number (tone 4, Wednesday evening).
	if have["oc.4.fri.compline.canon.ode.6.irmos"] == "" {
		t.Error("tone 4, Friday: no 6th ode")
	}
}
