package orthlib

import (
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

func TestIrmoi(t *testing.T) {
	irm, warnings := ImportIrmoi(extractedRoot(t))
	if len(warnings) > 0 {
		t.Fatal(warnings)
	}
	perTone := map[int]int{}
	for _, u := range irm {
		perTone[u.Tone]++
		if strings.HasPrefix(u.Body, "[") {
			t.Errorf("%s: the note left in the text: %.40s", u.ID, u.Body)
		}
	}
	for tone := 1; tone <= 8; tone++ {
		if perTone[tone] < 50 {
			t.Errorf("tone %d: %d irmoi", tone, perTone[tone])
		}
	}
	units := append([]corpus.Unit{
		{ID: "oc.6.sat.matins.canon.2.ode.1.irmos", Body: "Ꙗ҆́кѡ по сꙋ́хꙋ:"},
		// Abbreviated in the cue («прпⷣбнымъ»): its first words still pick one.
		{ID: "oc.6.sun.matins.canon.3.ode.8.irmos", Body: "И҆з̾ пла́мене прпⷣбнымъ:"},
		// A full irmos is left as it is.
		{ID: "me.01.01.matins.canon.1.ode.1.irmos", Body: "Гдⷭ҇ь бг҃ъ мо́й, ꙗ҆́кѡ свѧ́тъ є҆сѝ, и҆ просвѣще́нїе мое́, и҆ сп҃се́нїе."},
	}, irm...)
	got, _ := SupplementIrmoi(units)
	for _, u := range got[:2] {
		if u.Meta["completed_from"] == nil || len(strings.Fields(u.Body)) < 8 {
			t.Errorf("%s not completed: %q", u.ID, u.Body)
		}
	}
	if got[2].Meta != nil {
		t.Errorf("a full irmos completed: %v", got[2].Meta)
	}
	if !strings.HasPrefix(got[1].Body, "И҆з̾ пла́мене преподѡ́бнымъ") {
		t.Errorf("abbreviated cue: %q", got[1].Body)
	}
}
