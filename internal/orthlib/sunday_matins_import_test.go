package orthlib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestImportOctoechosSundayMatins(t *testing.T) {
	us, warnings := ImportOctoechosSundayMatins(extractedRoot(t))
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
	for id, start := range map[string]string{
		"oc.1.sun.matins.sedalen.1.1":               "гроб твой спсе",
		"oc.1.sun.matins.sedalen.1.verse.2":         "воскрсни гдси бже мой",
		"oc.2.sun.matins.sedalen.1.verse.2":         "воскрсни гдси бже мой", // without «Стих:»
		"oc.1.sun.matins.sedalen.2.theotokion":      "мрие, чстное влдки",
		"oc.1.sun.matins.hypakoe":                   "разбойничо покаяние",
		"oc.3.sun.matins.hypakoe":                   "удивляя видением",
		"oc.1.sun.matins.stepenna.1.1":              "внегда скорбети ми",
		"oc.1.sun.matins.stepenna.3.glory":          "стому дху, честь",
		"oc.1.sun.matins.prokeimenon":               "ныне воскрсну, глаголет гдсь",
		"oc.1.sun.matins.prokeimenon.verse":         "словеса гдсня",
		"oc.6.sun.matins.prokeimenon":               "гдси, воздвигни силу твою", // after «И ныне, тойже:»
		"oc.3.sun.matins.prokeimenon.verse":         "воспойте гдсеви песнь нову",
		"oc.1.sun.matins.resurrection_beheld":       "воскресение хрстово видевше",
		"oc.1.sun.matins.ps50.sticheron":            "воскрес иис от гроба",
		"oc.1.sun.matins.canon.1.ode.1.irmos":       "твоя победительная десница",
		"oc.1.sun.matins.canon.1.ode.1.troparion.1": "иже рукама пречистыма", // without «Тропарь:»
		"oc.1.sun.matins.canon.2.ode.1.troparion.1": "хрстос обожает мя",
		"oc.1.sun.matins.canon.3.ode.1.troparion.1": "кую ти достойную песнь",
		"oc.6.sun.matins.canon.3.ode.1.troparion.1": "причащшися еиа саду", // the 3rd canon begins with its irmos
		"oc.6.sun.matins.canon.1.ode.3.irmos":       "несть ст, якоже ты",
		"oc.1.sun.matins.ikos":                      "воскрсшаго тридневно",
		"oc.1.sun.matins.praises.sticheron.1":       "поем твою хрсте",
		"oc.1.sun.matins.praises.sticheron.8":       "любомятежный роде",
		"oc.1.sun.matins.end_troparion":             "днесь спсение миру",
		"oc.4.sun.matins.end_troparion":             "воскрс из гроба",
	} {
		got, ok := by[id]
		if !ok || !strings.HasPrefix(got, start) {
			t.Errorf("%s = %.50q (present %v), want %q…", id, got, ok, start)
		}
	}
	for tone := 1; tone <= 8; tone++ {
		for _, k := range []string{"canon.1.ode.9.troparion.1", "canon.3.ode.9.troparion.1", "sedalen.2.theotokion", "praises.sticheron.8"} {
			if by[fmt.Sprintf("oc.%d.sun.matins.%s", tone, k)] == "" {
				t.Errorf("tone %d: no %s", tone, k)
			}
		}
	}
}

func TestImportOctoechosSundayAppendix(t *testing.T) {
	us, err := ImportOctoechosSundayAppendix(extractedRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	tones := map[string]int{}
	for _, u := range us {
		by[u.ID] = strings.ToLower(hip.PlainString(u.Body))
		tones[u.ID] = u.Tone
	}
	for id, start := range map[string]string{
		"oc.sun.evlogitaria.verse":     "блгословен еси гдси",
		"oc.sun.evlogitaria.1":         "аггльский собор",
		"oc.sun.evlogitaria.4":         "мироносицы жены",
		"oc.sun.evlogitaria.now":       "жизнодавца рождши",
		"oc.sun.praises.now":           "преблгословенна еси",
		"oc.eothinon.1.exapostilarion": "со учнки взыдем",
		"oc.eothinon.1.sticheron":      "на гору учнком",
		"oc.eothinon.7.exapostilarion": "яко взяша гдса",
		"oc.eothinon.11.theotokion":    "о страшное таинство",
		"oc.eothinon.11.sticheron":     "являя себе учником",
	} {
		if !strings.HasPrefix(by[id], start) {
			t.Errorf("%s = %.50q, want %q…", id, by[id], start)
		}
	}
	// The Gospel stichera are in tones 1..8, then 5, 6, 8.
	for n, tone := range map[int]int{1: 1, 8: 8, 9: 5, 10: 6, 11: 8} {
		if got := tones[fmt.Sprintf("oc.eothinon.%d.sticheron", n)]; got != tone {
			t.Errorf("eothinon %d: tone %d, want %d", n, got, tone)
		}
	}
}
