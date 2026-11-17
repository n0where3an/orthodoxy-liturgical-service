package orthlib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestImportOctoechosSundayVespers(t *testing.T) {
	us, warnings := ImportOctoechosSundayVespers(extractedRoot(t))
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	by := map[string]string{}
	for _, u := range us {
		if _, dup := by[u.ID]; dup {
			t.Errorf("duplicate %s", u.ID)
		}
		by[u.ID] = strings.ToLower(hip.PlainString(u.Body))
	}
	if len(us) != 8*20 {
		t.Errorf("%d units, want %d", len(us), 8*20)
	}
	for id, start := range map[string]string{
		"oc.1.sun.gvespers.lic.sticheron.1":       "вечерния нашя млтвы",
		"oc.1.sun.gvespers.lic.sticheron.4":       "веселитеся нбса",
		"oc.1.sun.gvespers.lic.theotokos.1":       "стейшая стых всех сил",
		"oc.1.sun.gvespers.lic.dogmatikon":        "всемирную славу",
		"oc.1.sun.gvespers.aposticha.sticheron.1": "стрстию твоею хрсте",
		"oc.1.sun.gvespers.aposticha.verse.2":     "гдсь воцрися",
		"oc.1.sun.gvespers.aposticha.sticheron.2": "да радуется тварь",
		"oc.1.sun.gvespers.aposticha.theotokion":  "се исполнися исаиино",
		"oc.1.sun.troparion":                      "камени запечатану",
		"oc.2.sun.gvespers.lic.dogmatikon":        "прейде сень законная",
		"oc.8.sun.gvespers.aposticha.theotokion":  "безневестная дво",
		"oc.8.sun.troparion":                      "с высоты снизшел еси",
	} {
		if !strings.HasPrefix(by[id], start) {
			t.Errorf("%s = %.50q, want %q…", id, by[id], start)
		}
	}
	for tone := 1; tone <= 8; tone++ {
		if v := by[fmt.Sprintf("oc.%d.sun.gvespers.aposticha.verse.4", tone)]; !strings.HasPrefix(v, "дому твоему") {
			t.Errorf("tone %d: verse 4 %q", tone, v)
		}
	}
}
