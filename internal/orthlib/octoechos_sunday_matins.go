package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Sunday matins of the Octoechos ("<tone>_1nd.hip", «На утрени по
// шестопсалмии» … the liturgy). Keys under oc.<T>.sun.matins:
//
//	sedalen.<K>.<N>, sedalen.<K>.verse.<N>, sedalen.<K>.theotokion   K = 1, 2
//	hypakoe
//	stepenna.<A>.<N>, stepenna.<A>.glory      the antiphons of the tone
//	prokeimenon, prokeimenon.verse
//	resurrection_beheld                       «Воскресение Христово видевше»
//	ps50.sticheron                            «Воскрес Иисус от гроба»
//	canon.<C>.ode.<O>.irmos | .troparion.<N> | .theotokion
//	    C: 1 resurrectional, 2 of the Cross and Resurrection, 3 of the Theotokos
//	ikos
//	praises.sticheron.<N>, praises.now        «Преблагословенна еси»
//	end_troparion                             «Днесь спасение» / «Воскрес из гроба»
var (
	reSMStart      = regexp.MustCompile(`^в неделю утра по (шестопсалмии|ексапсалмех)|^(в неделю )?на утрени,?( по (шестопсалмии|ексапсалмех))?[,:.]?$|^на утрени,? по (шестопсалмии|ексапсалмех)`)
	reSMEnd        = regexp.MustCompile(`^на литургии`)
	reSMSedalen    = regexp.MustCompile(`^по (\S+?)-?[ймя]?,? (стихословии|стихологии|кафисме)`)
	reSMHypakoe    = regexp.MustCompile(`^(по непорочн\S*,? )?ипакои`)
	reSMAntiphon   = regexp.MustCompile(`(^|[.:] )антифон,? (\S+?)[,.:]`)
	reSMProk       = regexp.MustCompile(`(^|[.:] )прокимен`)
	reSMBeheld     = regexp.MustCompile(`^воскресение хрстово видевше`)
	reSMPs50St     = regexp.MustCompile(`^посем стихира`)
	reSMCanonRes   = regexp.MustCompile(`^канон,? воскрсн`)
	reSMCanonCross = regexp.MustCompile(`^((ин|другий|иный) )?канон,? кре?с?т?овоскре?с`)
	reSMCanonTheot = regexp.MustCompile(`^((ин|другий|иный) )?канон,? (престей|престыя) бцд`)
	reSMOtherIrmos = regexp.MustCompile(`^(ин|иный)[.:,]? ирмос`)
	reSMRefrain    = regexp.MustCompile(`^припев`)
	reSMOther      = regexp.MustCompile(`^(ин|иный)[.:]?$`)
	reSMTroparion  = regexp.MustCompile(`^тропарь:`)
	reSMIkos       = regexp.MustCompile(`^икос:`)
	reSMPraises    = regexp.MustCompile(`^на хвалитех`)
	reSMPraisesNow = regexp.MustCompile(`^и ныне:? преблгословенна`)
	reSMEndTrop    = regexp.MustCompile(`тропарь воскрсн:?\.?$`)
	// The verses of the sessional hymns, sometimes without «Стих:».
	reSMSedVerse = regexp.MustCompile(`^(воскрсни|воскресни) гдси бже мой|^исповемся тебе,? гдси`)
	reSMSkip     = regexp.MustCompile(`^(подобен|таже|аггльский собор|псалом н|и ныне|спси бже|и возглас|каноны|катавасиа|гдси помилуй|по катавасии|ины стихиры|слава,? стихира|кондак|всякое дыхание|глас )`)
)

// ImportOctoechosSundayMatins parses the Sunday matins of all tones.
func ImportOctoechosSundayMatins(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[1]))
		us, err := parseOctoechosSundayMatins(path, tone)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		units = append(units, us...)
	}
	return units, warnings
}

func parseOctoechosSundayMatins(path string, tone int) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.sun.matins.", tone)
	var units []corpus.Unit
	seen := map[string]bool{}
	add := func(key, kind, body string, meta map[string]any) {
		body = strings.TrimSpace(CleanText(body))
		if body == "" || seen[key] {
			return
		}
		seen[key] = true
		m := map[string]any{"file": filepath.Base(path)}
		for k, v := range meta {
			m[k] = v
		}
		units = append(units, corpus.Unit{ID: prefix + key, Book: "octoechos", Kind: kind, Tone: tone, Lang: "cu",
			Body: body, SourceID: "orthlib.octoechos", Meta: m})
	}
	started := false
	section, want, verse := "", "", ""
	kathisma, antiphon, canon, ode := 0, 0, 0, 0
	nextCanon := 0
	n := map[string]int{}
	for _, p := range d.Paras {
		text := strings.TrimLeft(CleanText(pageRef.ReplaceAllString(p.Text(), "")), ". ")
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		if !started {
			started = reSMStart.MatchString(plain)
			continue
		}
		if reSMEnd.MatchString(plain) {
			break
		}
		if want != "" && !reVerse.MatchString(plain) && !reSMSkip.MatchString(plain) {
			kind := map[string]string{"hypakoe": "hypakoe", "prokeimenon": "prokeimenon", "end_troparion": "troparion"}[want]
			if kind == "" {
				kind = "theotokion"
			}
			add(want, kind, text, nil)
			want = ""
			continue
		}
		words := strings.Fields(text) // numerals are read from the Unicode text
		switch {
		case reSMSedalen.MatchString(plain) && len(words) > 1:
			section, kathisma = "sedalen", numeralOfWord(words[1])
			continue
		case reSMHypakoe.MatchString(plain):
			section = "hypakoe"
			if body := afterColon(text, 6); body != "" {
				add("hypakoe", "hypakoe", body, nil)
			} else {
				want = "hypakoe"
			}
			continue
		case reSMAntiphon.MatchString(plain) && (section == "hypakoe" || section == "stepenna"):
			section, antiphon = "stepenna", 0
			for i, w := range words {
				if hip.PlainString(strings.ToLower(w)) == "антифон" || strings.HasPrefix(hip.PlainString(strings.ToLower(w)), "антифон") {
					if i+1 < len(words) {
						antiphon = numeralOfWord(strings.TrimRight(words[i+1], ",.:"))
					}
					break
				}
			}
			continue
		case reSMProk.MatchString(plain) && section != "canon" && section != "praises":
			section = "prokeimenon"
			if body := textAfter(text[strings.Index(text, "рокі"):], "глас", 4); body != "" {
				prok, verse := splitInlineVerse(body)
				add("prokeimenon", "prokeimenon", prok, nil)
				add("prokeimenon.verse", "verse", verse, nil)
			} else {
				want = "prokeimenon"
			}
			continue
		case reSMBeheld.MatchString(plain):
			add("resurrection_beheld", "hymn", text, nil)
			section = "ps50"
			continue
		case reSMPs50St.MatchString(plain):
			add("ps50.sticheron", "sticheron", afterColon(text, 5), nil)
			continue
		case reSMCanonRes.MatchString(plain):
			section, nextCanon = "canon", 1
			continue
		case reSMCanonCross.MatchString(plain):
			nextCanon = 2
			continue
		case reSMCanonTheot.MatchString(plain):
			nextCanon = 3
			continue
		case section == "canon" && reOde.MatchString(plain) && len(words) >= 2 && numeralOfWord(words[1]) > 0:
			ode = numeralOfWord(words[1])
			canon = 1
			if nextCanon > 1 { // «Песнь а, глас тойже. Ирмос: …» of the 2nd/3rd canon
				canon, nextCanon = nextCanon, 0
			}
			if i := strings.Index(plain, "ирмос:"); i >= 0 {
				add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", textAfter(text, "ирмос", 1), map[string]any{"cue": true})
			}
			continue
		case section == "canon" && reSMOther.MatchString(plain): // «Ин.»; the irmos follows
			canon++
			continue
		case section == "canon" && reSMOtherIrmos.MatchString(plain):
			canon++
			add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", textAfter(text, "ирмос", 1), map[string]any{"cue": true})
			continue
		case section == "canon" && reIrmos.MatchString(plain):
			if nextCanon > 1 { // the 3rd canon begins with its irmos, without «Песнь а»
				canon, nextCanon = nextCanon, 0
			}
			add(fmt.Sprintf("canon.%d.ode.%d.irmos", canon, ode), "irmos", afterColon(text, 1), nil)
			continue
		case section == "canon" && reOdeTheot.MatchString(plain):
			add(fmt.Sprintf("canon.%d.ode.%d.theotokion", canon, ode), "theotokion", afterColon(text, 1), nil)
			continue
		case section == "canon" && reSMIkos.MatchString(plain):
			add("ikos", "ikos", afterColon(text, 1), nil)
			continue
		case section == "canon" && reSMRefrain.MatchString(plain):
			continue
		case reSMPraises.MatchString(plain):
			section = "praises"
			continue
		case reSMPraisesNow.MatchString(plain):
			add("praises.now", "theotokion", afterColon(text, 3), nil)
			continue
		case reSMEndTrop.MatchString(plain):
			if body := textAfter(text, "тропарь", 6); body != "" {
				add("end_troparion", "troparion", body, nil)
			} else {
				want = "end_troparion"
			}
			continue
		case reGloryNow.MatchString(plain) && section == "sedalen":
			key := fmt.Sprintf("sedalen.%d.theotokion", kathisma)
			if body := afterColon(text, 6); body != "" {
				add(key, "theotokion", body, nil)
			} else {
				want = key
			}
			continue
		case section == "stepenna" && strings.HasPrefix(plain, "слава"):
			add(fmt.Sprintf("stepenna.%d.glory", antiphon), "verse", afterColon(text, 3), nil)
			continue
		case section == "sedalen" && reSMSedVerse.MatchString(plain):
			verse = text
			continue
		case reVerse.MatchString(plain):
			verse = verseText(text)
			if section == "prokeimenon" {
				v, _ := splitInlineVerse(verse) // «…: всякое дыхание: …»
				add("prokeimenon.verse", "verse", v, nil)
				verse = ""
			}
			continue
		case reSMSkip.MatchString(plain) && len(words) <= 14:
			continue
		}
		body, meta := text, map[string]any(nil)
		switch section {
		case "sedalen":
			k := fmt.Sprintf("sedalen.%d", kathisma)
			n[k]++
			if verse != "" && n[k] > 1 {
				add(fmt.Sprintf("%s.verse.%d", k, n[k]), "verse", verse, nil)
			}
			add(fmt.Sprintf("%s.%d", k, n[k]), "sedalen", body, nil)
		case "stepenna":
			k := fmt.Sprintf("stepenna.%d", antiphon)
			n[k]++
			add(fmt.Sprintf("%s.%d", k, n[k]), "verse", body, nil)
		case "canon":
			if ode == 0 {
				continue
			}
			if reSMTroparion.MatchString(plain) {
				body = afterColon(text, 1)
			}
			if reTrinity.MatchString(plain) {
				body, meta = afterColon(text, 1), map[string]any{"label": "trinity"}
			}
			k := fmt.Sprintf("canon.%d.ode.%d.troparion", canon, ode)
			n[k]++
			add(fmt.Sprintf("%s.%d", k, n[k]), "canon_troparion", body, meta)
		case "praises":
			if n["praises"] >= 8 { // after the stichera: «по славословии тропарь:», «и отпуст»
				if strings.Contains(plain, "тропарь") {
					want = "end_troparion"
				}
				continue
			}
			n["praises"]++
			add(fmt.Sprintf("praises.sticheron.%d", n["praises"]), "sticheron", body, nil)
		}
		verse = ""
	}
	return units, checkSundayMatins(path, prefix, units)
}

func checkSundayMatins(path, prefix string, units []corpus.Unit) error {
	has := map[string]bool{}
	count := map[string]int{}
	for _, u := range units {
		k := strings.TrimPrefix(u.ID, prefix)
		has[k] = true
		if i := strings.LastIndex(k, "."); i > 0 {
			count[k[:i]]++
		}
	}
	var missing []string
	for _, k := range []string{"sedalen.1.theotokion", "sedalen.2.theotokion", "hypakoe", "stepenna.1.glory", "stepenna.3.glory",
		"prokeimenon", "prokeimenon.verse", "ikos"} {
		// «Воскресение Христово видевше», «Воскрес Иисус», «Преблагословенна»
		// and the final troparion are the same every Sunday (of every other
		// tone): given in full in some tones only.
		if !has[k] {
			missing = append(missing, k)
		}
	}
	for c := 1; c <= 3; c++ {
		for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
			if count[fmt.Sprintf("canon.%d.ode.%d.troparion", c, o)] == 0 {
				missing = append(missing, fmt.Sprintf("canon %d ode %d", c, o))
			}
		}
	}
	if count["praises.sticheron"] != 8 {
		missing = append(missing, fmt.Sprintf("%d praises stichera", count["praises.sticheron"]))
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s: Sunday matins: missing %v", filepath.Base(path), missing)
	}
	return nil
}

// splitInlineVerse splits «P. Стих: V. Всякое дыхание: …» into P and V.
func splitInlineVerse(text string) (string, string) {
	plain, pos := hip.Plain(text)
	plain = strings.ToLower(plain)
	cut := func(a, b int) string { return strings.TrimSpace(uni(text, pos, plain, a, b)) }
	end := len(plain)
	if i := strings.Index(plain, "всякое дыхание"); i >= 0 {
		end = i
	}
	if i := strings.Index(plain[:end], "стих:"); i >= 0 {
		return cut(0, i), cut(i+len("стих:"), end)
	}
	return cut(0, end), ""
}

// The Sunday hymns of the Octoechos appendix:
//
//	oc.sun.evlogitaria.verse, .1-4, .glory, .now   «Ангельский собор»
//	                                               («тропари по непорочнах»)
//	oc.sun.praises.now                             «Преблагословенна еси»
//	oc.eothinon.<N>.exapostilarion, .theotokion, .sticheron   N = 1..11
func ImportOctoechosSundayAppendix(root string) ([]corpus.Unit, error) {
	var units []corpus.Unit
	add := func(id, kind, body string, tone int, file string) {
		units = append(units, corpus.Unit{ID: id, Book: "octoechos", Kind: kind, Tone: tone, Lang: "cu",
			Body: strings.TrimSpace(CleanText(body)), SourceID: "orthlib.octoechos", Meta: map[string]any{"file": file}})
	}
	// Evlogitaria.
	d, err := LoadDoc(filepath.Join(root, "octoih", "oc", "tropneporoch.hip"))
	if err != nil {
		return nil, err
	}
	n, verse := 0, false
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		switch {
		case strings.HasPrefix(plain, "блгословен еси гдси"):
			if !verse {
				add("oc.sun.evlogitaria.verse", "verse", text, 5, "tropneporoch.hip")
				verse = true
			}
		case strings.HasPrefix(plain, "слава, трочен:"):
			add("oc.sun.evlogitaria.glory", "troparion", afterColon(text, 3), 5, "tropneporoch.hip")
		case strings.HasPrefix(plain, "и ныне, бгородичен:"):
			add("oc.sun.evlogitaria.now", "theotokion", afterColon(text, 3), 5, "tropneporoch.hip")
		case strings.HasPrefix(plain, "преблгословенна еси"):
			add("oc.sun.praises.now", "theotokion", text, 0, "tropneporoch.hip")
		case verse && n < 4 && len(strings.Fields(plain)) > 10 && !strings.HasPrefix(plain, "по вся недели"):
			n++
			add(fmt.Sprintf("oc.sun.evlogitaria.%d", n), "troparion", text, 5, "tropneporoch.hip")
		}
	}
	if n != 4 {
		return units, fmt.Errorf("tropneporoch.hip: %d evlogitaria, want 4", n)
	}
	// Eothina.
	d, err = LoadDoc(filepath.Join(root, "octoih", "oc", "exapost.hip"))
	if err != nil {
		return units, err
	}
	e, want, tone := 0, false, 0
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		switch {
		case strings.HasPrefix(plain, "ексапостиларий"):
			e++
			add(fmt.Sprintf("oc.eothinon.%d.exapostilarion", e), "exapostilarion", afterColon(text, 3), 0, "exapost.hip")
		case strings.HasPrefix(plain, "бгородичен:") && e > 0:
			add(fmt.Sprintf("oc.eothinon.%d.theotokion", e), "theotokion", afterColon(text, 3), 0, "exapost.hip")
		case strings.HasPrefix(plain, "утренняя стихира"):
			want, tone = true, toneOf(text)
		case want:
			add(fmt.Sprintf("oc.eothinon.%d.sticheron", e), "sticheron", text, tone, "exapost.hip")
			want = false
		}
	}
	if e != 11 {
		return units, fmt.Errorf("exapost.hip: %d eothina, want 11", e)
	}
	return units, nil
}
