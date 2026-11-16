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
	reOcMatinsEnd  = regexp.MustCompile(`^(таже[,:]? бла?го есть|в \S+ бла?женны|в \S+ вечера)`)
	reTwice        = regexp.MustCompile(`\s*\[?(дважды|трижды)\.?\]?\.?\s*$`)
	reMatinsSkip   = regexp.MustCompile(`^(подобен|таже|глас |творение|его же краегранесие)`)
)

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
	odes := 0
	for _, p := range d.Paras {
		text := strings.TrimLeft(CleanText(pageRef.ReplaceAllString(p.Text(), "")), ". ")
		plain := hip.PlainString(text)
		if !started {
			started = reMatinsStart.MatchString(plain)
			continue
		}
		words := strings.Fields(text)
		switch {
		case reOcMatinsEnd.MatchString(plain):
			return units, checkMatins(path, units, odes, nApo)
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
	return units, checkMatins(path, units, odes, nApo)
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
