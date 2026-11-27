package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Matins of a Menaion service (files with red markup). Keys under
// me.<MM>.<DD>.matins:
//
//	canon.<C>.ode.<O>.irmos | .troparion.<N> | .theotokion | .cross_theotokion
//	canon.<C>.theotokos (a marker: the canon of the Theotokos the Menaion
//	    gives before the saint's, «канон бцды со ирмосом на 6, и стаго на 8»)
//	sedalen.<N>, sedalen.glory, sedalen.theotokion, sedalen.cross_theotokion
//	    (the sessional hymn after the 3rd ode)
//	ikos.<N>                                       (after the 6th ode)
//	exapostilarion.<N>, exapostilarion.theotokion  (after the 9th ode)
//	aposticha.sticheron.<N>, aposticha.verse.<N> (the verse before
//	    sticheron N), aposticha.glory | aposticha.glory_now, aposticha.now
//	    (the feast's at «И ныне»), aposticha.theotokion                       (when the Menaion has them:
//	    an afterfeast sings the feast's stichera instead of the Octoechos's)
//
// The kontakia are imported with the troparia (me.….kontakion.N).
var (
	reMCanon       = regexp.MustCompile(`^(иный |ин |другий )?канон`)
	reMTheotCanon  = regexp.MustCompile(`^канон (бцды|бгородицы|престей бцде|пресвятей бгородице)([,.:]|$| кир| глас| егоже)`)
	reMOde         = regexp.MustCompile(`песнь\s`)
	reMOther       = regexp.MustCompile(`^(ин|иный)\.?\s*ирмос`)
	reMTheot       = regexp.MustCompile(`^(слава,?\s*)?(и ныне,?\s*)?(крсто?)?бгородичен`)
	reMCross       = regexp.MustCompile(`крстобгородичен`)
	reMTrinity     = regexp.MustCompile(`^(слава,?\s*)?тр(о|ои)?и?чен`)
	reMSedalen     = regexp.MustCompile(`^(слава,? и ныне,? |таже,? |ин )?(седален|ипакои)`)
	reMKontakion   = regexp.MustCompile(`^(таже,? |ин )?кондак`)
	reMIkos        = regexp.MustCompile(`^икос`)
	reMSvetilen    = regexp.MustCompile(`^(таже,? |ин )?(светилен|ексапостиларий)`)
	reMAposticha   = regexp.MustCompile(`^на стиховне`)
	reMGlory       = regexp.MustCompile(`^слава`)
	reMAndNow      = regexp.MustCompile(`^и ныне`)
	reMHeadText    = regexp.MustCompile(`^(глас \S+ )?песнь \S+ ирмос:?$|^(ин\.? )?ирмос( тойже)?[.:]?$|^глас \S+ ирмос:?$`)
	reMSame        = regexp.MustCompile(`ирмос тойже`)
	reMIrmosInline = regexp.MustCompile(`^(?:(?:ин|иный)\.?\s*)?ирмос\s*:`)
	reMIrmosOnly   = regexp.MustCompile(`^(глас \S+ )?ирмос:?$`)
	reMCompline    = regexp.MustCompile(`^на повечерии|поется на повечерии`) // «…поется на повечерии»: the canon of compline
	reMMatinsNote  = regexp.MustCompile(`^на утрени`)
	reM9Refrains   = regexp.MustCompile(`^на ф-?й песни`)
	reM9Refrain    = regexp.MustCompile(`^(величай, душе моя|днесь влдка)`)
	reM9Rubric     = regexp.MustCompile(`^(таже|посем|вместо|и поклон|и вторый|и прочии|прочии)`)
	reMOtherSaint  = regexp.MustCompile(`^\S+( \S+)?:$`)
	reMPraises     = regexp.MustCompile(`^на хвалитех`)
	reMStop        = regexp.MustCompile(`^(на литургии|на велицем славословии|на 1-м часе|конец|на утрени|на вечерни|на малей вечерни|на велицей вечерни|месяца|в тойже день)`)
)

func parseMenaionMatins(d *Doc, prefix, path string) []corpus.Unit {
	var units []corpus.Unit
	seen := map[string]bool{}
	add := func(key, kind, heading, body string, meta map[string]any) {
		body = CleanText(body)
		if body == "" || seen[key] {
			return
		}
		seen[key] = true
		m := map[string]any{"heading": CleanText(heading), "file": filepath.Base(path)}
		for k, v := range meta {
			m[k] = v
		}
		units = append(units, corpus.Unit{
			ID: prefix + ".matins." + key, Book: "menaion", Kind: kind, Tone: toneOf(heading),
			Podoben: podobenOf(heading), Lang: "cu", Body: body, SourceID: "orthlib.mineya", Meta: m,
		})
	}
	started := false
	section := ""             // canon | sedalen | kontakion | ikos | svetilen | aposticha
	expect, heading := "", "" // what the next body paragraph is
	canon, canons, ode := 0, 0, 0
	nTrop := map[string]int{}
	nSed, nIkos, nSvet, nApost := 0, 0, 0, 0
	irmos := map[int]string{}
	sequential := false
	compline := false
	odes9 := false // the feast's 9th ode with its refrains
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		red := hip.PlainString(p.RedPrefix())
		plain := hip.PlainString(text)
		// The 9th ode of a twelve feast («на ф-й песни… честнейшую не поем: но
		// поем припевы праздника»; the Theophany prints no heading after
		// it): the refrains are the Irmologion's (ir.pripev.*) — the short
		// lines and the instructions between the troparia are left out;
		// «…втораго канона припев» passes to the second canon.
		if started && (section == "canon" || ode == 9 && odes9) {
			lp := strings.ToLower(plain)
			words := len(strings.Fields(lp))
			head := len([]rune(strings.TrimSpace(red))) > 3 || reMHeadText.MatchString(plain)
			switch {
			case reM9Refrains.MatchString(lp):
				if ode != 9 {
					ode, canon = 9, 1
					odes9 = true
				}
				expect = ""
				continue
			case odes9 && ode == 9 && (strings.Contains(lp, "втораго канона") || strings.Contains(lp, "стаго припевы")) && canons >= 2:
				canon, expect = 2, ""
				continue
			case odes9 && ode == 9 && (lp == "и ирмос:" || strings.HasPrefix(lp, "и ирмос:")):
				expect, heading = "irmos", text
				if strings.Contains(lp, "тойже") {
					expect = ""
				}
				continue
			case odes9 && ode == 9 && reM9Rubric.MatchString(lp) && words <= 12:
				continue // «вместо, славы:», «вместо же, и ныне:» — not another canon
			case odes9 && ode == 9 && strings.Contains(lp, "припев") && words <= 5:
				continue // «другий припев:» — not another canon
			case odes9 && ode == 9 && !head && expect != "irmos" && (words <= 15 || reM9Rubric.MatchString(lp) || reM9Refrain.MatchString(lp)):
				continue
			}
		}
		// «Ирмос: Воду прошед…» — the irmos in the heading's paragraph (its
		// first letter in red with the heading: the paragraph reads as one).
		if lp := strings.ToLower(plain); started && section == "canon" && ode > 0 && reMIrmosInline.MatchString(lp) && !reMSame.MatchString(lp) {
			if k := strings.Index(text, ":"); k > 0 && len(strings.Fields(text[k+1:])) >= 6 {
				if reMOther.MatchString(lp) {
					canon++ // «Ин ирмос: …» — the next canon
				}
				if canon == 1 {
					irmos[ode] = strings.TrimSpace(text[k+1:])
				}
				add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", text[:k+1], strings.TrimSpace(text[k+1:]), nil)
				expect = ""
				continue
			}
		}
		if len([]rune(strings.TrimSpace(red))) > 3 || reMHeadText.MatchString(plain) { // a heading
			// "Канон Октоиха един, на 6: и святых два канона на 8" is the
			// Typikon's instruction, not a canon of this service.
			isCanon := reMCanon.MatchString(plain) && !strings.Contains(plain, "октоиха") &&
				!strings.Contains(plain, "со ирмосом") && strings.Count(plain, "канон") == 1 &&
				!strings.HasPrefix(plain, "канона ")
			if isCanon && compline && !started {
				isCanon = false // the canon of compline, before matins
			}
			if isCanon {
				started, section, expect = true, "canon", ""
				switch {
				case ode <= 1:
					canons++
					canon = canons
				case ode == 9:
					// Canons printed one after another, each with all its odes.
					canons++
					canon, ode, sequential = canons, 0, true
				}
				if reMTheotCanon.MatchString(plain) {
					add(fmt.Sprintf("canon.%d.theotokos", canon), "marker", "", text, nil)
				}
			}
			switch {
			case reMCompline.MatchString(plain):
				compline = true // the canon of compline (forefeasts) is not matins
				continue
			case !started && (compline || !(reMOde.MatchString(plain) && strings.Contains(plain, "ирмос"))):
				if reMMatinsNote.MatchString(plain) {
					compline = false
				}
				continue
			case reMStop.MatchString(plain):
				return units
			case started && reMPraises.MatchString(plain):
				// The praises (imported with the festal matins); the aposticha
				// may follow them (an afterfeast).
				section, expect = "praises", ""
				continue
			case !started && reMPraises.MatchString(plain):
				return units
			case section == "canon" && reMSame.MatchString(plain):
				// "Ин. Ирмос тойже": the irmos of the first canon.
				if !isCanon && reMOther.MatchString(plain) {
					canon++
				}
				if ir := irmos[ode]; ir != "" && canon > 1 {
					add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", text, ir, map[string]any{"same_as": 1})
				}
				expect = ""
				continue
			case reMOde.MatchString(plain) && (section == "canon" || strings.Contains(plain, "ирмос")):
				started = true
				n := odeNumber(text)
				if n == 0 || canons <= 1 && n > 1 && n < ode && ode < 9 {
					n = nextOde(ode) // no number, or a misprint («песнь д.» after the 6th)
				}
				if n != ode {
					ode = n
					switch {
					case isCanon || n == 1: // the canon just opened
						canon = max(canons, 1)
					case !sequential:
						canon = 1
					}
				}
				section, expect, heading = "canon", "", text
				if strings.Contains(plain, "ирмос") {
					expect = "irmos"
				}
				continue
			case (section == "kontakion" || section == "ikos") && ode == 6 && reMOther.MatchString(plain):
				// «Ин. ирмос:» after the kontakion and ikos of the 6th ode: the
				// 7th, its heading missing (12.25).
				section, ode, canon, expect, heading = "canon", 7, 1, "irmos", text
				continue
			case section == "canon" && reMOther.MatchString(plain):
				canon++
				expect, heading = "irmos", text
				continue
			case section == "canon" && reMIrmosOnly.MatchString(plain):
				expect, heading = "irmos", text // "Глас тойже. Ирмос:" after «Ин канон»
				continue
			case isCanon:
				continue
			case section == "canon" && reMTrinity.MatchString(plain):
				expect, heading = "trinity", text
				continue
			case reMSedalen.MatchString(plain):
				section, expect, heading = "sedalen", "sedalen", text
				continue
			case reMKontakion.MatchString(plain):
				section, expect = "kontakion", ""
				continue
			case reMIkos.MatchString(plain):
				section, expect, heading = "ikos", "ikos", text
				continue
			case reMSvetilen.MatchString(plain):
				section, expect, heading = "svetilen", "svetilen", text
				continue
			case reMAposticha.MatchString(plain):
				section, heading = "aposticha", text
				expect = ""
				switch {
				case strings.Contains(plain, "слава"):
					expect = "glory"
				case !strings.Contains(plain, "октоиха"):
					expect = "sticheron" // «на стиховне стихиры, глас в. подобен: …»
				}
				continue
			case section == "aposticha" && strings.HasPrefix(plain, "стих"):
				// «Стих: Твоя суть небеса…» — the verse before the next sticheron.
				if k := strings.Index(text, ":"); k >= 0 && nApost > 0 {
					add(fmt.Sprintf("aposticha.verse.%d", nApost+1), "verse", text, text[k+1:], nil)
				}
				expect = "sticheron"
				continue
			case section == "aposticha" && reMGlory.MatchString(plain) && strings.Contains(plain, "и ныне"):
				expect, heading = "glory_now", text
				continue
			case section == "aposticha" && reMAndNow.MatchString(plain) && !strings.Contains(plain, "бгородичен"):
				// «И ныне, праздника:» — the feast's sticheron, not a theotokion.
				expect, heading = "now", text
				continue
			case reMTheot.MatchString(plain) || reMAndNow.MatchString(plain) && strings.Contains(plain, "бгородичен"):
				expect, heading = "theotokion", text
				if reMCross.MatchString(plain) && !strings.Contains(plain, "или") {
					expect = "cross_theotokion"
				}
				continue
			case reMGlory.MatchString(plain):
				expect, heading = "glory", text
				continue
			case section == "canon" && ode > 0 && reMOtherSaint.MatchString(plain):
				// "Преподобнаго:" — the troparia of the second saint's canon,
				// printed under the same irmos.
				canon++
				expect = ""
				continue
			default:
				expect = ""
				continue
			}
		}
		if !started || expect == "" && section != "canon" {
			continue
		}
		switch section {
		case "canon":
			if ode == 0 {
				continue
			}
			base := fmt.Sprintf("canon.%d.ode.%d", canon, ode)
			switch expect {
			case "irmos":
				if canon == 1 {
					irmos[ode] = text
				}
				add(base+".irmos", "irmos", heading, text, nil)
			case "theotokion", "cross_theotokion":
				add(base+"."+expect, "theotokion", heading, text, nil)
			default:
				nTrop[base]++
				var meta map[string]any
				if expect == "trinity" {
					meta = map[string]any{"label": "trinity"}
				}
				add(fmt.Sprintf("%s.troparion.%d", base, nTrop[base]), "canon_troparion", heading, text, meta)
			}
			expect = ""
		case "sedalen":
			switch expect {
			case "sedalen":
				nSed++
				add(fmt.Sprintf("sedalen.%d", nSed), "sedalen", heading, text, nil)
			case "glory":
				add("sedalen.glory", "sedalen", heading, text, nil)
			case "theotokion", "cross_theotokion":
				add("sedalen."+expect, "theotokion", heading, text, nil)
			}
			expect = ""
		case "ikos":
			nIkos++
			add(fmt.Sprintf("ikos.%d", nIkos), "ikos", heading, text, nil)
			expect = ""
		case "svetilen":
			switch expect {
			case "svetilen", "glory":
				nSvet++
				add(fmt.Sprintf("exapostilarion.%d", nSvet), "exapostilarion", heading, text, nil)
			case "theotokion", "cross_theotokion":
				add("exapostilarion."+expect, "theotokion", heading, text, nil)
			}
			expect = ""
		case "aposticha":
			switch expect {
			case "sticheron":
				nApost++
				add(fmt.Sprintf("aposticha.sticheron.%d", nApost), "sticheron", heading, text, nil)
			case "glory", "glory_now", "now":
				add("aposticha."+expect, "sticheron", heading, text, nil)
			case "theotokion", "cross_theotokion":
				add("aposticha."+expect, "theotokion", heading, text, nil)
			}
			expect = ""
		}
	}
	return units
}

// odeNumber reads the number after «Пѣ́снь» in a Unicode heading.
func odeNumber(text string) int {
	w := strings.Fields(text)
	for i := 0; i+1 < len(w); i++ {
		if hip.PlainString(w[i]) == "песнь" {
			return numeralOfWord(w[i+1])
		}
	}
	return 0
}
