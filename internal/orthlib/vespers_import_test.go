package orthlib

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestPsalter(t *testing.T) {
	units, warnings := ImportPsalter(extractedRoot(t))
	for _, w := range warnings {
		t.Error(w)
	}
	if len(units) != 60 {
		t.Errorf("%d stases, want 60", len(units))
	}
	by := map[string]string{}
	for _, u := range units {
		by[u.ID] = hip.PlainString(u.Body)
	}
	if !strings.HasPrefix(by["ps.kathisma.1.1"], "блжен муж, иже не иде") {
		t.Errorf("kathisma 1: %.60q", by["ps.kathisma.1.1"])
	}
	if !strings.HasPrefix(by["ps.kathisma.18.1"], "ко гдсу, внегда скорбети ми") {
		t.Errorf("kathisma 18: %.60q", by["ps.kathisma.18.1"])
	}
}

func TestDismissalTheotokia(t *testing.T) {
	units, warnings := ImportDismissalTheotokia(extractedRoot(t))
	for _, w := range warnings {
		t.Error(w)
	}
	by := map[string]string{}
	for _, u := range units {
		by[u.ID] = hip.PlainString(u.Body)
	}
	for id, want := range map[string]string{
		"ch.theotokion.dismissal.1.mon": "чудо чудес блгодатная",
		"ch.theotokion.dismissal.1.thu": "заченшая неопально огнь", // cue resolved
		"ch.theotokion.dismissal.1.wed": "твое предстательство имуще",
		"ch.theotokion.dismissal.2.mon": "блгоутробия сущи источник",
	} {
		if !strings.HasPrefix(by[id], want) {
			t.Errorf("%s = %.50q, want %q", id, by[id], want)
		}
	}
}

func TestMenaionVespers(t *testing.T) {
	root := extractedRoot(t)
	units, err := ParseMenaionDay(filepath.Join(root, MonthDir(10), "30.hip"), 10, "30")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	for _, u := range units {
		by[u.ID] = hip.PlainString(u.Body)
	}
	for id, want := range map[string]string{
		"me.10.30.vespers.lic.sticheron.3": "спострадати тебе",
		"me.10.30.vespers.lic.glory":       "песненный лик",
		"me.10.30.vespers.lic.theotokion":  "на древе жизнь нашу",
	} {
		if !strings.HasPrefix(by[id], want) {
			t.Errorf("%s = %.40q, want %q", id, by[id], want)
		}
	}
	if _, ok := by["me.10.30.vespers.great"]; ok {
		t.Error("a weekday service marked as great vespers")
	}
}
