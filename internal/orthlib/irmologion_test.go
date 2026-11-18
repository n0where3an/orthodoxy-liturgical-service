package orthlib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestImportIrmologionOdes(t *testing.T) {
	us, warnings := ImportIrmologionOdes(extractedRoot(t))
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	by := map[string]string{}
	on := map[string]any{}
	count := map[string]int{}
	for _, u := range us {
		if _, dup := by[u.ID]; dup {
			t.Errorf("duplicate %s", u.ID)
		}
		by[u.ID] = strings.ToLower(hip.PlainString(u.Body))
		if v, ok := u.Meta["on"]; ok {
			on[u.ID] = v
		}
		p := strings.Split(u.ID, ".")
		if len(p) == 5 && strings.Trim(p[4], "0123456789") == "" {
			count[p[2]+"."+p[3]]++
		}
	}
	// Verses per ode: the weekday set from the beginning of the ode, the
	// Sunday set the first four and the last six.
	for k, want := range map[string]int{"1.weekday": 16, "3.weekday": 16, "6.weekday": 10, "8.weekday": 16, "9.weekday": 10,
		"1.sunday": 10, "5.sunday": 10, "8.sunday": 10, "9.sunday": 10, "1.lent": 19} {
		if count[k] != want {
			t.Errorf("ode %s: %d verses, want %d", k, count[k], want)
		}
	}
	if count["2.lent"] < 40 {
		t.Errorf("the 2nd ode (Lent only): %d verses", count["2.lent"])
	}
	for id, start := range map[string]string{
		"ir.ode.1.weekday.1":       "гдсеви поем, славно бо прославися",
		"ir.ode.1.weekday.5":       "огустеша яко стена воды",
		"ir.ode.1.weekday.11":      "тогда потщашася", // «На и:» cut
		"ir.ode.8.weekday.glory":   "блгословим оца",
		"ir.ode.8.weekday.refrain": "хвалим, блгословим",
		"ir.ode.9.weekday.1":       "блгословен гдсь бг иилев", // the Song of Zacharias, without the Magnificat
		"ir.ode.1.sunday.5":        "тогда потщашася",
		"ir.sunday.refrain":        "слава гдси, стому воскрснию твоему",
		"ir.ode.1.lent.4":          "избранныя всадники тристаты",
		"ir.ode.8.lent.1":          "блгословите вся дела гдсня",
	} {
		if !strings.HasPrefix(by[id], start) {
			t.Errorf("%s = %.50q, want %q…", id, by[id], start)
		}
	}
	for id, want := range map[string]int{"ir.ode.1.weekday.5": 14, "ir.ode.1.weekday.11": 8, "ir.ode.1.weekday.13": 6,
		"ir.ode.1.weekday.15": 4, "ir.ode.1.sunday.5": 8, "ir.ode.1.lent.8": 14} {
		if fmt.Sprint(on[id]) != fmt.Sprint(want) {
			t.Errorf("%s: on %v, want %d", id, on[id], want)
		}
	}
	// «На N» counts «Слава» and «И ныне»: from the mark to the end N−2 verses.
	for _, o := range []int{1, 3, 4, 5, 8} {
		for id, v := range on {
			if !strings.HasPrefix(id, fmt.Sprintf("ir.ode.%d.weekday.", o)) {
				continue
			}
			var n int
			fmt.Sscanf(id[strings.LastIndex(id, ".")+1:], "%d", &n)
			if left := count[fmt.Sprintf("%d.weekday", o)] - n + 1; left != v.(int)-2 {
				t.Errorf("%s: on %v but %d verses to the end", id, v, left)
			}
		}
	}
}

func TestImportOdeNineRefrains(t *testing.T) {
	us, err := ImportOdeNineRefrains(extractedRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	for _, u := range us {
		if _, dup := by[u.ID]; dup {
			t.Errorf("duplicate %s", u.ID)
		}
		by[u.ID] = strings.ToLower(hip.PlainString(u.Body))
	}
	for id, start := range map[string]string{
		"ir.pripev.nativity_of_theotokos.1.irmos": "величай, душе моя, преславное ржство",
		"ir.pripev.nativity_of_theotokos.2.irmos": "величай, душе моя, от неплодове",
		"ir.pripev.exaltation.1.irmos":            "величай, душе моя, пречстный крст",
		"ir.pripev.entry_of_theotokos.1.irmos":    "аггли, вхождение пречстыя",
		"ir.pripev.entry_of_theotokos.1.6":        "аггли и человецы дву песньми",
		"ir.pripev.entry_of_theotokos.2.glory":    "величай, душе моя, триипостаснаго", // «вместо Слава»
		"ir.pripev.nativity.2.irmos2":             "днесь влдка раждается",             // «вторый же лик поет припев»
		"ir.pripev.meeting.1.irmos2":              "бцде дво, миру блгая помощнице",    // «два припевы ко ирмосом»
		"ir.pripev.meeting.1.1":                   "бгоносе симеоне",
		"ir.pripev.circumcision.2.irmos":          "величай, душе моя, во иерарсех василиа",
		"ir.pripev.circumcision.2.now":            "величай, душе моя, чтснейшую",  // inline after «поем припев бцде:»
		"ir.pripev.transfiguration.2.irmos":       "величай душе моя, на фаворе",   // «и ко второму канону тойже припев»
		"ir.pripev.dormition.1.irmos":             "роди вси блжим тя едину бцду.", // «или:» cut, the alternative skipped
		"ir.pripev.annunciation.1.irmos":          "блговествуй земле радость велию",
	} {
		if !strings.HasPrefix(by[id], start) {
			t.Errorf("%s = %.50q, want %q…", id, by[id], start)
		}
	}
	for id := range by {
		if strings.HasSuffix(by[id], ":") {
			t.Errorf("%s is a cue: %q", id, by[id])
		}
	}
	if _, ok := by["ir.pripev.dormition.1.1"]; ok {
		t.Error("dormition: the alternative refrain taken as a troparion's")
	}
}
