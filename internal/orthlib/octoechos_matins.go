package orthlib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Weekday matins of the Octoechos (Monday..Friday): the sessional hymns
// after each kathisma, the two canons and the aposticha. Keys under
// oc.<tone>.<day>.matins:
//
//	sedalen.<K>.<N>, sedalen.<K>.verse.<N>, sedalen.<K>.theotokion
//	    (K: 1, 2 — after the 1st and 2nd kathisma; 3 — «иные седальны,
//	    егда стихословится третия кафисма»)
//	canon.<C>.ode.<O>.irmos | .troparion.<N> | .theotokion
//	aposticha.sticheron.<N>, aposticha.verse.<N>, aposticha.theotokion
var (
	reMatinsStart  = regexp.MustCompile(`утра(,|$| по )`)
	reSedalenSec   = regexp.MustCompile(`^по (\S+?)-?[ймя]?,? (стихословии|кафисме)`)
	reSedalenThird = regexp.MustCompile(`^ины седальны,? егда стихословится`)
	reSedalenAgain = regexp.MustCompile(`^и паки глаголем тойже седален`)
	reCanonHead    = regexp.MustCompile(`^(другий |ин |иный )?канон`)
	reOde          = regexp.MustCompile(`^песнь `)
	reOtherCanon   = regexp.MustCompile(`^(ин|иный)\.?$`)
	reIrmosSame    = regexp.MustCompile(`^((ин|иный)[.,]? )?ирмос тойже`)
	reIrmos        = regexp.MustCompile(`^ирмос:`)
	reOdeTheot     = regexp.MustCompile(`^(крсто?)?бгородичен:`)
	reTrinity      = regexp.MustCompile(`^тр(о|ои)?и?чен:`)
	reMatinsAposti = regexp.MustCompile(`^на стиховне`)
	reOcMatinsEnd  = regexp.MustCompile(`^(таже[,:]? бла?го есть|в \S+ бла?женны|в \S+ вечера|в \S+ на литургии|тропарь,? ектени)`)
	reTwice        = regexp.MustCompile(`\s*\[?(дважды|трижды)\.?\]?\.?\s*$`)
	reMatinsSkip   = regexp.MustCompile(`^(подобен|таже|глас |творение|его же краегранесие)`)

	// Saturday matins (Октоих, «в субботу утра»): the hymns of the martyrs,
	// saints and departed.
	reSatMatinsStart = regexp.MustCompile(`^в субботу,? (утра|на утрени)`)
	reMatinsPraises  = regexp.MustCompile(`^на хвалитех`)
	reCanonOtherIrm  = regexp.MustCompile(`^(ин|иный)[.,]? ирмос:`)
	reCanonGlory     = regexp.MustCompile(`^слава(,? (мертвен|покоин))?:`)
	reCanonNow       = regexp.MustCompile(`^и ныне(,? бгородичен)?:`)
	// «Благословим Отца…» before the troparion at «Слава» of the 8th ode.
	reCanonTrinity = regexp.MustCompile(`^блгословим оца`)
	reCanonRefrain = regexp.MustCompile(`^припев:`)
	// «мчнком:», «стителем:» — whom the troparion praises (not sung).
	reCanonLabel = regexp.MustCompile(`^(мчнком|мучеником|стителем|прпдбным|пророком|мчнчен|мертвен|покоин):\s*`)
)

// ImportOctoechosSaturdayMatins parses the Saturday matins of all tones
// (oc.<tone>.sat.matins): besides the keys of a weekday,
//
//	praises.sticheron.<N>, praises.theotokion (the martyrs' stichera «на
//	    хвалитех», sung at the aposticha), praises.departed (the departed
//	    one among them, «мертвен», left out)
//	canon.<C>.ode.<O>.glory (the troparion at «Слава»)
//	aposticha.* (the stichera of the departed)
func ImportOctoechosSaturdayMatins(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[7]))
		us, err := parseOctoechosMatins(path, tone, 6)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		units = append(units, us...)
	}
	return units, warnings
}

// ImportOctoechosMatins parses the weekday matins of all tones.
func ImportOctoechosMatins(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		for dow := 2; dow <= 6; dow++ { // Monday..Friday
			path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[dow]))
			us, err := parseOctoechosMatins(path, tone, dow-1)
			if err != nil {
				warnings = append(warnings, err.Error())
			}
			units = append(units, us...)
		}
	}
	return units, warnings
}

// numeralAfter reads the Slavonic numeral of the token at index k of the
// Unicode text's words ("по а҃-мъ" → 1).
func numeralOfWord(w string) int {
	if i := strings.Index(w, "-"); i > 0 {
		w = w[:i]
	}
	return numeralValue(strings.TrimRight(w, ".,:"))
}

func parseOctoechosMatins(path string, tone, weekday int) ([]corpus.Unit, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.%s.matins", tone, weekdayKey[weekday])
	var units []corpus.Unit
	add := func(key, kind, body string, meta map[string]any) {
		units = append(units, corpus.Unit{
			ID: prefix + "." + key, Book: "octoechos", Kind: kind, Tone: tone,
			Lang: "cu", Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: meta,
		})
	}
	started := false
	section := "" // sedalen | canon | aposticha
	kathisma, nSed := 0, 0
	lastSed := ""
	canon, ode, nTrop := 0, 0, 0
	irmos := map[int]string{} // irmos of the first canon by ode
	nApo := 0
	pendingVerse := ""
	wantTheotokion := ""
	odes, nPraises := 0, 0
	sat := weekday == 6
	nextGlory := false
	start := reMatinsStart
	if sat {
		start = reSatMatinsStart
	}
	for _, p := range d.Paras {
		text := strings.TrimLeft(CleanText(pageRef.ReplaceAllString(p.Text(), "")), ". ")
		plain := hip.PlainString(text)
		if !started {
			started = start.MatchString(plain)
			continue
		}
		words := strings.Fields(text)
		if sat && section == "canon" && ode > 0 {
			switch {
			case nextGlory:
				nextGlory = false
				add(fmt.Sprintf("canon.%d.ode.%d.glory", canon, ode), "canon_troparion", text, nil)
				continue
			case reCanonTrinity.MatchString(plain) && len(words) <= 10:
				nextGlory = true
				continue
			case reCanonRefrain.MatchString(plain), strings.Contains(plain, "идеже несть минеи") && !reCanonHead.MatchString(plain):
				continue // a refrain; the rest of the heading of the departed canon
			case reCanonOtherIrm.MatchString(plain):
				canon, nTrop = 2, 0
				add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", text[strings.Index(text, ":")+1:], nil)
				continue
			case reCanonGlory.MatchString(plain) && !reGloryNow.MatchString(plain):
				add(fmt.Sprintf("canon.%d.ode.%d.glory", canon, ode), "canon_troparion", text[strings.Index(text, ":")+1:], nil)
				continue
			case reCanonNow.MatchString(plain):
				add(fmt.Sprintf("canon.%d.ode.%d.theotokion", canon, ode), "theotokion", text[strings.Index(text, ":")+1:], nil)
				continue
			case reCanonLabel.MatchString(plain) && len(words) > 8:
				m := reCanonLabel.FindStringSubmatch(plain)
				nTrop++
				add(fmt.Sprintf("canon.%d.ode.%d.troparion.%d", canon, ode, nTrop), "canon_troparion",
					strings.TrimSpace(text[strings.Index(text, ":")+1:]), map[string]any{"label": m[1]})
				continue
			}
		}
		switch {
		case reOcMatinsEnd.MatchString(plain):
			return units, checkMatinsDay(path, sat, odes, nApo, nPraises)
		case sat && reMatinsPraises.MatchString(plain):
			section = "praises"
			continue
		case sat && (section == "praises" || section == "aposticha") && reCanonGlory.MatchString(plain) && !reGloryNow.MatchString(plain):
			add(section+".glory", "sticheron", afterColon(text, 0), nil) // «слава:» without «и ныне»
			continue
		case sat && (section == "praises" || section == "aposticha") && strings.HasPrefix(plain, "и ныне"):
			add(section+".theotokion", "theotokion", afterColon(text, 0), nil)
			continue
		case reSedalenSec.MatchString(plain) && len(words) > 1:
			section, kathisma, nSed = "sedalen", numeralOfWord(words[1]), 0
			continue
		case reSedalenThird.MatchString(plain):
			section, kathisma, nSed = "sedalen", 3, 0
			continue
		case section == "sedalen" && reSedalenAgain.MatchString(plain):
			if lastSed != "" {
				nSed++
				if pendingVerse != "" {
					add(fmt.Sprintf("sedalen.%d.verse.%d", kathisma, nSed), "verse", pendingVerse, nil)
					pendingVerse = ""
				}
				add(fmt.Sprintf("sedalen.%d.%d", kathisma, nSed), "sedalen", lastSed, map[string]any{"repeat_of": nSed - 1})
			}
			continue
		case reCanonHead.MatchString(plain):
			// The second canon is introduced before (or instead of
			// repeating) its first ode heading.
			if section == "canon" {
				canon, nTrop = 2, 0
			}
			section = "canon"
			continue
		case section != "aposticha" && (reOde.MatchString(plain) && len(words) >= 2 && numeralOfWord(words[1]) > 0 || plain == "песнь"):
			n := nextOde(ode) // a heading without its number
			if len(words) >= 2 {
				n = numeralOfWord(words[1])
			}
			section, nTrop = "canon", 0
			if n == ode && canon == 2 {
				continue // "Песнь а." repeated for the second canon
			}
			ode, canon = n, 1
			odes++
			continue
		case section == "canon" && reIrmosSame.MatchString(plain):
			// "Ин. Ирмос тойже": the second canon has the first one's irmos.
			canon, nTrop = 2, 0
			if irmos[ode] != "" {
				add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", irmos[ode], map[string]any{"same_as": 1})
			}
			continue
		case section == "canon" && reOtherCanon.MatchString(plain):
			canon, nTrop = 2, 0
			continue
		case section == "canon" && reIrmos.MatchString(plain):
			if canon == 1 {
				irmos[ode] = text[strings.Index(text, ":")+1:]
			}
			add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", text[strings.Index(text, ":")+1:], nil)
			continue
		case section == "canon" && reOdeTheot.MatchString(plain):
			add(fmt.Sprintf("canon.%d.ode.%d.theotokion", canon, ode), "theotokion", text[strings.Index(text, ":")+1:], nil)
			continue
		case reMatinsAposti.MatchString(plain):
			section = "aposticha"
			continue
		case section == "":
			continue
		case reGloryNow.MatchString(plain):
			key := section + ".theotokion"
			if section == "sedalen" {
				key = fmt.Sprintf("sedalen.%d.theotokion", kathisma)
			}
			if body := afterColon(text, 6); body != "" {
				add(key, "theotokion", body, nil)
			} else {
				wantTheotokion = key
			}
			continue
		case reVerse.MatchString(plain):
			pendingVerse = verseText(text)
			continue
		case reMatinsSkip.MatchString(plain) && len(words) <= 12:
			continue
		}
		if wantTheotokion != "" {
			add(wantTheotokion, "theotokion", text, nil)
			wantTheotokion = ""
			continue
		}
		body, meta := text, map[string]any(nil)
		if m := reLabel.FindStringSubmatch(plain); m != nil && len(strings.Fields(plain)) > 8 {
			// "мчнчен: …" — a martyrs' hymn among the others
			body, meta = strings.TrimSpace(text[strings.Index(text, ":")+1:]), map[string]any{"label": m[1]}
		}
		if reTrinity.MatchString(plain) {
			body, meta = text[strings.Index(text, ":")+1:], map[string]any{"label": "trinity"}
		}
		if k := strings.LastIndex(body, "["); k > 0 && reTwice.MatchString(hip.PlainString(body[k:])) {
			if meta == nil {
				meta = map[string]any{}
			}
			meta["repeat"] = 2
			body = strings.TrimSpace(body[:k])
		}
		switch section {
		case "sedalen":
			nSed++
			if pendingVerse != "" && nSed > 1 {
				add(fmt.Sprintf("sedalen.%d.verse.%d", kathisma, nSed), "verse", pendingVerse, nil)
			}
			add(fmt.Sprintf("sedalen.%d.%d", kathisma, nSed), "sedalen", body, meta)
			lastSed = body
		case "canon":
			if ode == 0 {
				continue
			}
			nTrop++
			add(fmt.Sprintf("canon.%d.ode.%d.troparion.%d", canon, ode, nTrop), "canon_troparion", body, meta)
		case "praises":
			nPraises++
			if l, _ := meta["label"].(string); l == "мертвен" || l == "покоин" {
				add("praises.departed", "sticheron", body, meta) // left out at matins (Типикон, гл. 12)
				break
			}
			add(fmt.Sprintf("praises.sticheron.%d", nPraises), "sticheron", body, meta)
		case "aposticha":
			nApo++
			if pendingVerse != "" && nApo > 1 {
				add(fmt.Sprintf("aposticha.verse.%d", nApo), "verse", pendingVerse, nil)
			}
			add(fmt.Sprintf("aposticha.sticheron.%d", nApo), "sticheron", body, meta)
		}
		pendingVerse = ""
	}
	if !started {
		return nil, fmt.Errorf("%s: matins not found", filepath.Base(path))
	}
	return units, checkMatinsDay(path, sat, odes, nApo, nPraises)
}

func checkMatinsDay(path string, sat bool, odes, nApo, nPraises int) error {
	if sat && (nPraises < 3 || nPraises > 6) {
		return fmt.Errorf("%s: matins: %d praises stichera", filepath.Base(path), nPraises)
	}
	return checkMatins(path, nil, odes, nApo)
}

func checkMatins(path string, units []corpus.Unit, odes, nApo int) error {
	if odes != 8 || nApo < 3 || nApo > 4 {
		return fmt.Errorf("%s: matins: %d odes, %d aposticha stichera", filepath.Base(path), odes, nApo)
	}
	return nil
}

// nextOde follows the odes of a canon without the second: 1, 3, 4, …, 9.
func nextOde(ode int) int {
	if ode <= 1 {
		return ode + 1 + min(ode, 1)
	}
	return ode + 1
}
