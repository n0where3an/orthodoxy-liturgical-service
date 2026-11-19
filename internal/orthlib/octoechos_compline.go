package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The canon of compline in the Octoechos («канон молебный престей бцде»,
// Типикон, гл. 9: «на нем же поем канон молебный престей бцде, во
// осмогласнике»): each day file opens with the compline of its eve («в
// неделю на повечерии» in the Monday file; «в субботу на повечерии» in the
// Sunday file). Keys under oc.<tone>.<day>.compline (the day the evening
// belongs to):
//
//	canon.ode.<O>.irmos | .troparion.<N> | .glory | .theotokion
//	    (.glory: the troparion at «Слава», .theotokion: at «И ныне»)
//	sedalen, sedalen.tone (after the 6th ode, when given)
//	refrain (Saturday evening: «Пресвятая Богородице, спаси нас»)
var (
	reComplineStart = regexp.MustCompile(`^во? \S+ на повечерии`)
	reComplineEnd   = regexp.MustCompile(`^(во? \S+ (утра|на утрени|вечера|на полунощнице)|таже[,:]? достойно есть)`)
	reCplRefrain    = regexp.MustCompile(`^припев:`)
	reCplSedalen    = regexp.MustCompile(`^седален`)
	reCplGlory      = regexp.MustCompile(`^слава:`)
	reCplNow        = regexp.MustCompile(`^и ныне:`)
)

// ImportOctoechosCompline parses the compline canons of all tones and days.
func ImportOctoechosCompline(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		for dow := 1; dow <= 7; dow++ {
			path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[dow]))
			us, err := parseOctoechosCompline(path, tone, dow-1)
			if err != nil {
				warnings = append(warnings, err.Error())
			}
			units = append(units, us...)
		}
	}
	return units, warnings
}

func parseOctoechosCompline(path string, tone, weekday int) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.%s.compline", tone, weekdayKey[weekday])
	var units []corpus.Unit
	add := func(key, kind, body string, meta map[string]any) {
		units = append(units, corpus.Unit{
			ID: prefix + "." + key, Book: "octoechos", Kind: kind, Tone: tone,
			Lang: "cu", Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: meta,
		})
	}
	started := false
	ode, nTrop, odes := 0, 0, 0
	sedalen := "" // the heading of the sessional hymn whose text is next
	after := func(text string) string { return strings.TrimSpace(text[strings.Index(text, ":")+1:]) }
	for _, p := range d.Paras {
		text := strings.TrimLeft(CleanText(pageRef.ReplaceAllString(p.Text(), "")), ". ")
		plain := hip.PlainString(text)
		if !started {
			started = reComplineStart.MatchString(plain)
			continue
		}
		words := strings.Fields(text)
		switch {
		case odes > 0 && reComplineEnd.MatchString(plain):
			return units, checkCompline(path, odes)
		case (reOde.MatchString(plain) || strings.TrimRight(plain, ".") == "песнь") && len(words) <= 3:
			n := 0
			if len(words) >= 2 {
				n = numeralOfWord(words[1])
			}
			if n == 0 {
				n = nextOde(ode) // «песнь ~.»: a misprinted number
			}
			ode, nTrop, sedalen = n, 0, ""
			odes++
			continue
		case reCplRefrain.MatchString(plain):
			if !hasUnit(units, prefix+".refrain") {
				add("refrain", "refrain", after(text), nil)
			}
			continue
		case reCplSedalen.MatchString(plain):
			if body := afterColon(text, 10); body != "" { // the hymn in the heading's paragraph
				add("sedalen", "sedalen", body, map[string]any{"heading": text})
				units[len(units)-1].Tone = toneOf(text)
				continue
			}
			sedalen = text
			continue
		case ode == 0:
			continue
		case sedalen != "" && !reMatinsSkip.MatchString(plain):
			add("sedalen", "sedalen", text, map[string]any{"heading": sedalen})
			units[len(units)-1].Tone = toneOf(sedalen)
			sedalen = ""
			continue
		case reIrmos.MatchString(plain):
			add(fmt.Sprintf("canon.ode.%d.irmos", ode), "irmos", after(text), nil)
			continue
		case reCplGlory.MatchString(plain):
			if b := after(text); len(strings.Fields(b)) > 4 {
				add(fmt.Sprintf("canon.ode.%d.glory", ode), "canon_troparion", b, nil)
			}
			continue
		case reCplNow.MatchString(plain):
			if b := after(text); len(strings.Fields(b)) > 4 {
				add(fmt.Sprintf("canon.ode.%d.theotokion", ode), "theotokion", b, nil)
			}
			continue
		case reMatinsSkip.MatchString(plain) && len(words) <= 12:
			continue
		case len(words) <= 8:
			continue // «таже, гдси помилуй, трижды. слава: и ныне:» and the like
		}
		nTrop++
		add(fmt.Sprintf("canon.ode.%d.troparion.%d", ode, nTrop), "canon_troparion", text, nil)
	}
	if !started {
		return nil, fmt.Errorf("%s: compline not found", filepath.Base(path))
	}
	return units, checkCompline(path, odes)
}

func hasUnit(units []corpus.Unit, id string) bool {
	for _, u := range units {
		if u.ID == id {
			return true
		}
	}
	return false
}

func checkCompline(path string, odes int) error {
	if odes != 8 {
		return fmt.Errorf("%s: compline: %d odes", filepath.Base(path), odes)
	}
	return nil
}
