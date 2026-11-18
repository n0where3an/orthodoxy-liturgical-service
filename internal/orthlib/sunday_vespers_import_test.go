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

func TestImportOctoechosSundayLiturgy(t *testing.T) {
	us, warnings := ImportOctoechosSundayLiturgy(extractedRoot(t))
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	by := map[string]string{}
	tones := map[string]int{}
	for _, u := range us {
		by[u.ID] = strings.ToLower(hip.PlainString(u.Body))
		tones[u.ID] = u.Tone
	}
	// kontakion, 6 beatitudes, glory, now, prokeimenon with verse, 2 alleluia verses
	if len(us) != 8*13 {
		t.Errorf("%d units, want %d", len(us), 8*13)
	}
	for id, start := range map[string]string{
		"oc.1.sun.kontakion":             "воскрсл еси яко бг из гроба",
		"oc.4.sun.kontakion":             "спс и избавитель мой",
		"oc.7.sun.kontakion":             "не ктому держава смертная",
		"oc.8.sun.kontakion":             "воскрс из гроба, умершыя", // not «Взбранной» of the compline canon
		"oc.1.sun.lit.beatitudes.1":      "снедию изведе из рая",
		"oc.1.sun.lit.beatitudes.6":      "апсли твои гдси на гору",
		"oc.2.sun.lit.beatitudes.glory":  "оца прославим",
		"oc.1.sun.lit.beatitudes.now":    "мтрь твою приводят",
		"oc.1.sun.lit.prokeimenon":       "буди, гдси, млсть твоя",
		"oc.3.sun.lit.prokeimenon.verse": "вси языцы восплещите", // «стих:» in the same paragraph
		"oc.5.sun.lit.alleluia.verse.1":  "млсти твоя гдси во век",
		"oc.5.sun.lit.alleluia.verse.2":  "зане рекл еси",
		"oc.8.sun.lit.alleluia.verse.2":  "предварим лице его",
	} {
		if !strings.HasPrefix(by[id], start) {
			t.Errorf("%s = %.50q, want %q…", id, by[id], start)
		}
	}
	if p := by["oc.7.sun.lit.prokeimenon"]; !strings.HasSuffix(p, "миром.") {
		t.Errorf("tone 7 prokeimenon %q", p)
	}
	for tone := 1; tone <= 8; tone++ {
		if got := tones[fmt.Sprintf("oc.%d.sun.lit.prokeimenon", tone)]; got != tone {
			t.Errorf("tone %d prokeimenon has tone %d", tone, got)
		}
	}
}
