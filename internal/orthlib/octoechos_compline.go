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
//	sedalen.<O>, sedalen.<O>.theotokion (after ode O: the 6th at compline,
//	    the 3rd and 6th at the midnight office, when given)
//	refrain (Saturday evening: «Пресвятая Богородице, спаси нас»)
var (
	reComplineStart = regexp.MustCompile(`^во? \S+ на повечерии`)
	reComplineEnd   = regexp.MustCompile(`^(во? \S+ (утра|на утрени|вечера|на полунощнице)|таже[,:]? достойно есть)`)
	reCplRefrain    = regexp.MustCompile(`^припев:`)
	reCplSedalen    = regexp.MustCompile(`^седален`)
	reCplGlory      = regexp.MustCompile(`^слава:`)
	reCplNow        = regexp.MustCompile(`^и ныне(,? бгородичен)?:`)

	// The Sunday midnight office: «канон ко стей и живоначальней троце»
	// (oc.<tone>.sun.midnight), up to the triadica of St Gregory of Sinai.
	reMidnightStart = regexp.MustCompile(`^в неделю (утра,? )?(начало полунощницы|на полунощнице)`)
	reMidnightEnd   = regexp.MustCompile(`^(посем припевы|таже,? трочны|во? \S+ на утрени|на утрени)`)
)

// ImportOctoechosMidnight parses the Trinity canons of the Sunday midnight
// office of all tones.
func ImportOctoechosMidnight(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[1]))
		us, err := parseOctoechosCanonSection(path, tone, 0, "midnight", reMidnightStart, reMidnightEnd)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		units = append(units, us...)
	}
	return units, warnings
}

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
	return parseOctoechosCanonSection(path, tone, weekday, "compline", reComplineStart, reComplineEnd)
}

// parseOctoechosCanonSection parses the canon of a section of a day file
// (from start to end) under oc.<tone>.<day>.<sub>.
func parseOctoechosCanonSection(path string, tone, weekday int, sub string, start, end *regexp.Regexp) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.%s.%s", tone, weekdayKey[weekday], sub)
	var units []corpus.Unit
	add := func(key, kind, body string, meta map[string]any) {
		units = append(units, corpus.Unit{
			ID: prefix + "." + key, Book: "octoechos", Kind: kind, Tone: tone,
			Lang: "cu", Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: meta,
		})
	}
	started := false
	ode, nTrop, odes := 0, 0, 0
	sedalen := ""         // the heading of the sessional hymn whose text is next
	afterSedalen := false // «слава, и ныне, бгородичен:» of the sessional hymn may follow
	after := func(text string) string { return strings.TrimSpace(text[strings.Index(text, ":")+1:]) }
	for _, p := range d.Paras {
		text := strings.TrimLeft(CleanText(pageRef.ReplaceAllString(p.Text(), "")), ". ")
		plain := hip.PlainString(text)
		if !started {
			started = start.MatchString(plain)
			continue
		}
		words := strings.Fields(text)
		switch {
		case odes > 0 && end.MatchString(plain):
			return units, checkCompline(path, sub, odes)
		case (reOde.MatchString(plain) || strings.TrimRight(plain, ".") == "песнь") && len(words) <= 3:
			n := 0
			if len(words) >= 2 {
				n = numeralOfWord(words[1])
			}
			if n == 0 {
				n = nextOde(ode) // «песнь ~.»: a misprinted number
			}
			ode, nTrop, sedalen, afterSedalen = n, 0, "", false
			odes++
			continue
		case reCplRefrain.MatchString(plain):
			if !hasUnit(units, prefix+".refrain") {
				add("refrain", "refrain", after(text), nil)
			}
			continue
		case reCplSedalen.MatchString(plain):
			if body := afterColon(text, 10); body != "" { // the hymn in the heading's paragraph
				add(fmt.Sprintf("sedalen.%d", ode), "sedalen", body, map[string]any{"heading": text})
				units[len(units)-1].Tone = toneOf(text)
				afterSedalen = true
				continue
			}
			sedalen = text
			continue
		case ode == 0:
			continue
		case sedalen != "" && !reMatinsSkip.MatchString(plain):
			add(fmt.Sprintf("sedalen.%d", ode), "sedalen", text, map[string]any{"heading": sedalen})
			units[len(units)-1].Tone = toneOf(sedalen)
			sedalen, afterSedalen = "", true
			continue
		case afterSedalen && reGloryNow.MatchString(plain):
			if b := afterColon(text, 5); b != "" {
				add(fmt.Sprintf("sedalen.%d.theotokion", ode), "theotokion", b, nil)
			}
			afterSedalen = false
			continue
		case reIrmos.MatchString(plain):
			add(fmt.Sprintf("canon.ode.%d.irmos", ode), "irmos", after(text), nil)
			continue
		case reCplGlory.MatchString(plain):
			if b := after(text); len(strings.Fields(b)) > 4 {
				add(fmt.Sprintf("canon.ode.%d.glory", ode), "canon_troparion", b, nil)
			}
			continue
		case reOdeTheot.MatchString(plain):
			add(fmt.Sprintf("canon.ode.%d.theotokion", ode), "theotokion", after(text), nil)
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
		return nil, fmt.Errorf("%s: %s not found", filepath.Base(path), sub)
	}
	return units, checkCompline(path, sub, odes)
}

func hasUnit(units []corpus.Unit, id string) bool {
	for _, u := range units {
		if u.ID == id {
			return true
		}
	}
	return false
}

func checkCompline(path, sub string, odes int) error {
	if odes != 8 {
		return fmt.Errorf("%s: %s: %d odes", filepath.Base(path), sub, odes)
	}
	return nil
}
