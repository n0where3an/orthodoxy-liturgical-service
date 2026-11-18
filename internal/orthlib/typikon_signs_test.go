package orthlib

import (
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestImportTypikonSigns(t *testing.T) {
	us, err := ImportTypikonSigns(extractedRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	days := map[string]bool{}
	by := map[string]string{}
	for _, u := range us {
		if _, dup := by[u.ID]; dup {
			t.Errorf("duplicate %s", u.ID)
		}
		by[u.ID] = u.Meta["sign"].(string) + " " + strings.ToLower(hip.PlainString(u.Body))
		count[u.Meta["sign"].(string)]++
		days[u.ID[:8]] = true
	}
	if len(days) != 366 {
		t.Errorf("%d days, want 366", len(days))
	}
	// As many as the signs in the twelve files (<(:.> 80; red: (:. 60, + 51, \+/ 13, (+) 15).
	for sign, want := range map[string]int{"six_fold": 80, "doxology": 60, "polyeleos": 51, "saint_vigil": 13, "vigil": 15} {
		if count[sign] != want {
			t.Errorf("%s: %d, want %d", sign, count[sign], want)
		}
	}
	for id, want := range map[string]string{
		"ty.09.08.sign.1": "vigil ржство престыя",
		"ty.09.05.sign.1": "six_fold стаго пророка захарии",
		"ty.09.07.sign.2": "doxology и память", // the same sign in red
		"ty.09.25.sign.2": "polyeleos и прпдбнаго отца нашего сергиа",
		"ty.09.26.sign.1": "saint_vigil преставление стаго апсла",
		"ty.09.02.sign.1": "none стаго мчнка маманта",
		// «г~. Слава…» inside the Meeting's service is not 3 February.
		"ty.02.03.sign.1": "six_fold стаго и првднаго симеона",
	} {
		if !strings.HasPrefix(by[id], want) {
			t.Errorf("%s = %.60q, want %q…", id, by[id], want)
		}
	}
}
