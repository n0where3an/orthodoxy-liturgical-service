package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

var (
	reTheotHead = regexp.MustCompile(`^в[о]? (неделю|понедельник|вторник|среду|четверток|пяток) вечера`)
	reMatinsEnd = regexp.MustCompile(`^в конец утрени`)
	reToneTail  = regexp.MustCompile(`глас ([а-из])\s*:?\s*$`)
)

// the day whose vespers a heading "в X вечера" belongs to
var eveToDay = map[string]string{
	"неделю": "mon", "понедельник": "tue", "вторник": "wed", "среду": "thu", "четверток": "fri", "пяток": "sat",
}

// ImportDismissalTheotokia parses «Богородичны отпустительные» of the
// Chasoslov: the theotokion after the troparion at vespers ("на Бог Господь"
// at matins) by tone and day: ch.theotokion.dismissal.<tone>.<day>.
// Cues ("Заченшая неопально:") are resolved to the full text given earlier
// in the same tone.
func ImportDismissalTheotokia(root string) ([]corpus.Unit, []string) {
	path := filepath.Join(root, "chasoslov", "ch", "15otpbogor.hip")
	d, err := LoadDoc(path)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var units []corpus.Unit
	var warnings []string
	// Some theotokia are given as cues; their texts stand elsewhere in the
	// Chasoslov (Saturday ones among the Sunday dismissal troparia, a
	// cross-theotokion at the sixth hour).
	elsewhere := map[string]string{}
	files, _ := filepath.Glob(filepath.Join(root, "chasoslov", "ch", "*.hip"))
	for _, f := range files {
		sd, err := LoadDoc(f)
		if err != nil {
			continue
		}
		for _, p := range sd.Paras {
			t := CleanText(p.Text())
			if len(strings.Fields(t)) < 8 {
				continue
			}
			for _, cand := range []string{t, afterColon(t, 8)} {
				if cand != "" {
					if _, ok := elsewhere[incipitKey(cand)]; !ok {
						elsewhere[incipitKey(cand)] = cand
					}
				}
			}
		}
	}
	tone, day := 0, ""
	full := map[string]string{} // incipit key -> text, within the tone
	skipNext := false
	for _, p := range d.Paras {
		text := CleanText(p.Text())
		plain := hip.PlainString(text)
		if m := reTheotHead.FindStringSubmatch(plain); m != nil {
			skipNext = false
			day = eveToDay[m[1]]
			if t := toneOf(text); t > 0 && t != tone {
				tone = t
				full = map[string]string{}
			}
			continue
		}
		if reMatinsEnd.MatchString(plain) {
			// "В конец утрени:" — the next paragraph is the matins one,
			// unless it is given inline ("В конец утрени: Дево пречистая:").
			skipNext = len(strings.Fields(plain)) <= 3
			continue
		}
		if skipNext {
			skipNext = false
			continue
		}
		if day == "" || tone == 0 {
			continue
		}
		body := text
		key := incipitKey(text)
		if len(strings.Fields(plain)) <= 5 && strings.HasSuffix(strings.TrimSpace(plain), ":") {
			// a cue: find the full text with this beginning
			found := ""
			for _, m := range []map[string]string{full, elsewhere} {
				for k, v := range m {
					if found == "" && strings.HasPrefix(squeeze(k), squeeze(key)) {
						found = v
					}
				}
			}
			if found == "" {
				warnings = append(warnings, fmt.Sprintf("theotokia: tone %d %s: cue %q not resolved", tone, day, plain))
				day = ""
				continue
			}
			body = found
		} else {
			full[key] = text
		}
		units = append(units, corpus.Unit{
			ID: fmt.Sprintf("ch.theotokion.dismissal.%d.%s", tone, day), Book: "chasoslov", Kind: "theotokion",
			Tone: tone, Lang: "cu", Body: body, SourceID: "orthlib.chasoslov",
		})
		day = ""
	}
	if len(units) != 48 {
		have := map[string]bool{}
		for _, u := range units {
			have[u.ID] = true
		}
		var missing []string
		for t := 1; t <= 8; t++ {
			for _, d := range []string{"mon", "tue", "wed", "thu", "fri", "sat"} {
				if id := fmt.Sprintf("ch.theotokion.dismissal.%d.%s", t, d); !have[id] {
					missing = append(missing, id)
				}
			}
		}
		warnings = append(warnings, fmt.Sprintf("theotokia: %d found, want 48; missing %v", len(units), missing))
	}
	return units, warnings
}

// squeeze collapses doubled letters (spelling variants like «препрославлен(н)а»).
func squeeze(s string) string {
	var b []rune
	for _, r := range s {
		if n := len(b); n > 0 && b[n-1] == r {
			continue
		}
		b = append(b, r)
	}
	return string(b)
}
