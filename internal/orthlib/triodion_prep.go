package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The preparatory weeks of the Lenten Triodion (Триодь постная): the Sundays
// of the Publican and the Pharisee, of the Prodigal Son, of the Last
// Judgement (meatfare), of the Expulsion from Paradise (cheesefare), the
// meatfare Saturday of the departed and the cheesefare Saturday of the
// venerable fathers. What their liturgy takes from the Triodion, under
// tp.<day>:
//
//	troparion, kontakion, ikos  the day's hymns (the Saturdays' troparion; the
//	                            Sundays' kontakion — the first, not «вторый»)
//	theotokion                  «Тебе и стену» (the meatfare Saturday)
//	canon.ode.<N>.troparion.<K>, .glory, .theotokion
//	                            the odes of the Triodion's canon the beatitudes
//	                            are sung from («от канона триоди песнь з»)
//	lit.rubric                  «на литургии…» (resolved by ResolveLiturgy into
//	                            lit.prokeimenon, lit.alleluia.verse.N, …)
//
// The files print no cinnabar: the headings are plain («песнь з:», «ирмос:
// житейское море:», «кондак, глас д. подобен: явился еси днесь:»).
var triodionPrepDays = []struct{ id, file string }{
	{"publican", "ned_mytar"},
	{"prodigal", "ned_blud"},
	{"sat_meatfare", "sub_myas"},
	{"meatfare", "ned_myas"},
	{"sat_cheesefare", "sub_syrn"},
	{"cheesefare", "ned_syrn"},
	// The Saturdays and Sundays of Great Lent.
	{"sat_lent1", "sub_1sedm"},
	{"lent1", "ned_1sedm"},
	{"sat_lent2", "sub_2sedm"},
	{"lent2", "ned_2sedm"},
	{"sat_lent3", "sub_3sedm"},
	{"lent3", "ned_krest"},
	{"sat_lent4", "sub_4sedm"},
	{"lent4", "ned_4sedm"},
	{"sat_akathist", "sub_akaf"},
	{"lent5", "ned_5sedm"},
}

var (
	reTPOde        = regexp.MustCompile(`^(?:глас \S+ )?песнь ([а-ѳ])(?:[.:]|$)`)
	reTPProk       = regexp.MustCompile(strings.Replace(reSpecProk.String(), `прокимен,?\s*`, `прокимен,?\s*(?:апсла,?\s*)?`, 1))
	reTPKontakion  = regexp.MustCompile(`^кондак(?:,\s*глас[^:]*)?:\s*(.*)$`)
	reTPTroparion  = regexp.MustCompile(`^(?:и по тристом глаголем )?тропарь,\s*глас [^:]*:\s*(.*)$`)
	reTPStop       = regexp.MustCompile(`^(песнь|кондак|вторый кондак|икос|седален|катаваси|светилен|канон|ин канон|таже|по |на |слава, и ныне|тристую|в настоящий|синаксар|стих|предисловие)`)
	reTPGlory      = regexp.MustCompile(`^слава(?:,\s*тр\S*чен)?:\s*(.+)$`)
	reTPRubricPara = regexp.MustCompile(`^(и повторя|и начинают|таже|кланяющымся|глас \S+[.:]|аще )`)
	reTPSedalen    = regexp.MustCompile(`^(?:и )?седален`)
	reTPSecHead    = regexp.MustCompile(`^(?:и )?(?:седален|светилен|самогласен)`)
	reTPRubricCont = regexp.MustCompile(`^(прокимен|апостол|апсл|вместо|аллилуиа|еиглие)`)
	reTPZach       = regexp.MustCompile(`зачало (\S+?)(?:\s+от полу)?(?:[.,:;]|\s|$)`)
	reTPEpistle    = regexp.MustCompile(`апсл|апостол|\sк[о]? \S+(ном|ю|ем|еем|ям)\b|послания`)
	reTPGospelBook = regexp.MustCompile(`(матфеа|марка|луки|иоанна)[^.:]{0,12}$`)
	reTPNow        = regexp.MustCompile(`^(?:и ныне,?\s*)?(?:бгородичен|бцдичен)[^:]*:\s*(.+)$`)
)

// ImportTriodionPrep parses the six days of the preparatory weeks.
func ImportTriodionPrep(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for _, day := range triodionPrepDays {
		path := filepath.Join(root, "triodion", "pt", day.file+".hip")
		d, err := LoadDoc(path)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		got := parseTriodionPrep(d, "tp."+day.id, path)
		want := []string{"kontakion", "lit.rubric"}
		switch day.id {
		case "sat_lent2", "sat_lent3", "sat_lent4", "sat_akathist":
			// «Со святыми упокой», «Взбранной Воеводе»: cited, given elsewhere.
			want = want[1:]
		}
		for _, k := range want {
			found := false
			for _, u := range got {
				found = found || u.ID == "tp."+day.id+"."+k
			}
			if !found {
				warnings = append(warnings, fmt.Sprintf("%s: no %s", day.file, k))
			}
		}
		units = append(units, got...)
	}
	return units, warnings
}

func parseTriodionPrep(d *Doc, prefix, path string) []corpus.Unit {
	var units []corpus.Unit
	seen := map[string]bool{}
	add := func(key, kind, body string, tone int, meta map[string]any) {
		body = CleanText(body)
		if body == "" || seen[key] {
			return
		}
		seen[key] = true
		m := map[string]any{"file": filepath.Base(path)}
		for k, v := range meta {
			m[k] = v
		}
		units = append(units, corpus.Unit{ID: prefix + "." + key, Book: "triodion", Kind: kind, Tone: tone, Lang: "cu",
			Body: body, SourceID: "orthlib.triodion_postnaya", Meta: m})
	}
	// after: the Unicode text after the colon that ends the plain prefix.
	after := func(text, plain string, n int) string {
		_, pos := hip.Plain(text)
		return uni(text, pos, plain, len(plain)-n, len(plain))
	}
	ode, n := 0, 0
	odesDone := map[int]bool{}
	expect := "" // the next paragraph's text: kontakion | troparion | ikos | a matins key
	expectTone := 0
	lastOde := 0  // the last ode of the Triodion's canon finished
	section := "" // matins: sedalen (after the 3rd ode) | exapostilarion | praises
	nSec := 0     // hymns in the section
	kindOf := func(key string) string {
		switch {
		case strings.HasSuffix(key, "theotokion"):
			return "theotokion"
		case strings.HasPrefix(key, "matins.sedalen"):
			return "sedalen"
		case strings.HasPrefix(key, "matins.exapostilarion"):
			return "exapostilarion"
		case strings.HasPrefix(key, "matins.praises"):
			return "sticheron"
		}
		return key
	}
	for pi, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		if expect != "" {
			if strings.HasPrefix(plain, "подобен") || len(strings.Fields(plain)) < 5 ||
				strings.Contains(plain, "подобен:") && len(strings.Fields(plain)) < 10 { // «Подобен: Два днесь:», «и настоящий.» — the text follows
				continue
			}
			// The hymn itself may begin like a rubric («На высоте…»).
			if !strings.HasPrefix(plain, "икос") && !strings.HasPrefix(plain, "песнь") && !reTPOde.MatchString(plain) {
				add(expect, kindOf(expect), text, expectTone, nil)
			}
			expect = ""
			continue
		}
		// The rubric of the liturgy, inside a long paragraph.
		if i := strings.Index(plain, "на литургии"); i >= 0 && !seen["lit.rubric"] && strings.Contains(triodionRubric(d, pi, plain[i:]), "зачало") {
			_, pos := hip.Plain(text)
			sub := uni(text, pos, plain, i, len(plain))
			// The rubric may go on in the next paragraphs («прокимен…», «апостол…»).
			for j := pi + 1; j < len(d.Paras) && j <= pi+3; j++ {
				nt := CleanText(pageRef.ReplaceAllString(d.Paras[j].Text(), ""))
				if !reTPRubricCont.MatchString(strings.ToLower(hip.PlainString(nt))) {
					break
				}
				sub += " " + nt
			}
			sp := strings.ToLower(hip.PlainString(sub))
			_, spos := hip.Plain(sub)
			parseLiturgyRubricWith(reTPProk, sub, sp, spos, func(key, kind, body string, meta map[string]any) {
				// «аллилуиа, глас в. стих: насаждени…» — the first verse.
				if a, ok := meta["alleluia"].(string); ok {
					if i := strings.Index(a, ":"); i > 0 && strings.HasPrefix(strings.ToLower(hip.PlainString(a[:i])), "стих") {
						meta["alleluia"] = strings.TrimSpace(a[i+1:])
					}
				}
				// «прокимен, глас д. песнь отцев: Благословен еси…»
				if pr, ok := meta["prok"].(string); ok {
					if i := strings.Index(pr, ":"); i > 0 && strings.HasPrefix(strings.ToLower(hip.PlainString(pr[:i])), "песнь") {
						meta["prok"] = strings.TrimSpace(pr[i+1:])
					}
				}
				delete(meta, "beatitudes") // the Triodion's odes: tp.<day>.canon.ode.N
				add("lit."+key, kind, body, 0, meta)
			})
			// The second readings (of the saint, the departed, the Theotokos):
			// copies of the lectionary's by their pericope (ResolveLiturgy).
			for _, x := range triodionExtraReadings(sub, sp, spos) {
				add(x[0], "reading", "-", 0, map[string]any{"copy": x[1]})
				add(x[0]+".announce", "title", "-", 0, map[string]any{"copy": x[1] + ".announce"})
			}
			continue
		}
		// The matins hymns of the Triodion outside the canon's odes.
		if section != "" {
			done := false
			inline := func() string { // the text after the heading's colon, if any
				if i := strings.Index(plain, ":"); i > 0 && len(strings.Fields(plain[i+1:])) > 4 && !strings.HasPrefix(strings.TrimSpace(plain[i+1:]), "подобен") {
					return after(text, plain, len(plain)-i-1)
				}
				return ""
			}
			put := func(key string) {
				if t := inline(); t != "" {
					add(key, kindOf(key), t, toneOf(text), nil)
				} else {
					expect, expectTone = key, toneOf(text)
				}
			}
			switch {
			case strings.HasPrefix(plain, "подобен"), strings.HasPrefix(plain, "стих"):
				done = true
			case reTPOde.MatchString(plain), strings.HasPrefix(plain, "катаваси"), strings.HasPrefix(plain, "кондак"),
				strings.HasPrefix(plain, "икос"), strings.HasPrefix(plain, "славословие"), strings.HasPrefix(plain, "на стиховне"),
				strings.HasPrefix(plain, "на хвалитех"), strings.Contains(plain, "на литургии"), strings.HasPrefix(plain, "в неделю"):
				section = ""
			case strings.HasPrefix(plain, "слава, и ныне") || strings.HasPrefix(plain, "и ныне") || reTPNow.MatchString(plain):
				put("matins." + section + ".theotokion")
				if section != "sedalen" {
					section = ""
				}
				done = true
			case strings.HasPrefix(plain, "слава"):
				if strings.Contains(plain, "тойже") {
					if section == "praises" && nSec > 0 { // «слава, паки тойже»: the last sticheron again
						add("matins.praises.glory", "sticheron", "-", 0, map[string]any{"copy": fmt.Sprintf("%s.matins.praises.sticheron.%d", prefix, nSec)})
						section = ""
					}
				} else if section == "exapostilarion" {
					nSec++
					put(fmt.Sprintf("matins.exapostilarion.%d", nSec))
				} else {
					put("matins." + section + ".glory")
				}
				done = true
			case section == "praises" && (strings.HasPrefix(plain, "воскрсни гдси") || reTPRubricPara.MatchString(plain)):
				done = true // a verse, a rubric («и повторяюще первую стихиру…»)
			case len(strings.Fields(plain)) >= 8 || inline() != "":
				nSec++
				key := map[string]string{"sedalen": "matins.sedalen", "exapostilarion": "matins.exapostilarion",
					"praises": "matins.praises.sticheron"}[section]
				if t := inline(); t != "" && len(strings.Fields(plain)) < 40 && reTPSecHead.MatchString(plain) {
					add(fmt.Sprintf("%s.%d", key, nSec), kindOf(key), t, toneOf(text), nil)
				} else if reTPSecHead.MatchString(plain) {
					nSec--
					expect, expectTone = fmt.Sprintf("%s.%d", key, nSec+1), toneOf(text)
					nSec++
				} else {
					add(fmt.Sprintf("%s.%d", key, nSec), kindOf(key), text, 0, nil)
				}
				done = true
			case reTPSecHead.MatchString(plain): // «седален, глас а.», «светилен.»
				nSec++
				key := map[string]string{"sedalen": "matins.sedalen", "exapostilarion": "matins.exapostilarion",
					"praises": "matins.praises.sticheron"}[section]
				expect, expectTone = fmt.Sprintf("%s.%d", key, nSec), toneOf(text)
				done = true
			}
			if done {
				continue
			}
		}
		if ode == 0 && !seen["matins.sedalen.1"] && lastOde == 3 && reTPSedalen.MatchString(plain) {
			section, nSec = "sedalen", 0
			nSec++
			if i := strings.Index(plain, ":"); i > 0 && len(strings.Fields(plain[i+1:])) > 4 && !strings.HasPrefix(strings.TrimSpace(plain[i+1:]), "подобен") {
				add("matins.sedalen.1", "sedalen", after(text, plain, len(plain)-i-1), toneOf(text), nil)
			} else {
				expect, expectTone = "matins.sedalen.1", toneOf(text)
			}
			continue
		}
		if ode == 0 && lastOde == 9 && !seen["matins.exapostilarion.1"] && (strings.HasPrefix(plain, "светилен") || strings.HasPrefix(plain, "ексапостиларий")) {
			section, nSec = "exapostilarion", 1
			if i := strings.LastIndex(plain, ":"); i > 0 && len(strings.Fields(plain[i+1:])) > 4 {
				add("matins.exapostilarion.1", "exapostilarion", after(text, plain, len(plain)-i-1), 0, nil)
			} else {
				expect = "matins.exapostilarion.1"
			}
			continue
		}
		if strings.HasPrefix(plain, "на хвалитех") && (strings.Contains(plain, "триоди") || strings.Contains(plain, "трипеснц")) && !seen["matins.praises.sticheron.1"] {
			section, nSec = "praises", 0
			continue
		}
		if lastOde > 0 && ode == 0 && strings.HasPrefix(plain, "катаваси") {
			if rest := strings.TrimSpace(plain[strings.Index(plain, ":")+1:]); strings.Contains(plain, ":") && len(strings.Fields(rest)) > 6 {
				add(fmt.Sprintf("canon.ode.%d.katavasia", lastOde), "irmos", after(text, plain, len(rest)), 0, nil)
			}
			continue
		}
		switch {
		case reTPOde.MatchString(plain):
			// The numeral from the Unicode text (ѕ = 6, з = 7: one letter in plain).
			m := reTPOde.FindStringSubmatchIndex(plain)
			_, pos := hip.Plain(text)
			ode, n = uniNumeral(text, pos, plain, m[2], m[3]), 0
			if odesDone[ode] {
				ode = 0 // a second canon's ode: not the Triodion's first
			}
			continue
		case strings.HasPrefix(plain, "ирмос"):
			continue
		case strings.HasPrefix(plain, "вторый кондак"):
			ode = 0
			continue
		case reTPKontakion.MatchString(plain) && !seen["kontakion"]:
			ode = 0
			tone := toneOf(text)
			if m := reTPKontakion.FindStringSubmatch(plain); len(strings.Fields(m[1])) > 4 && !strings.HasPrefix(m[1], "подобен") {
				add("kontakion", "kontakion", after(text, plain, len(m[1])), tone, nil)
			} else {
				expect, expectTone = "kontakion", tone
			}
			continue
		case strings.HasPrefix(plain, "икос:") && seen["kontakion"] && !seen["ikos"]:
			ode = 0
			if rest := strings.TrimSpace(strings.TrimPrefix(plain, "икос:")); len(strings.Fields(rest)) > 4 {
				add("ikos", "ikos", after(text, plain, len(rest)), 0, nil)
			}
			continue
		case reTPTroparion.MatchString(plain) && !seen["troparion"]:
			tone := toneOf(text)
			if m := reTPTroparion.FindStringSubmatch(plain); len(strings.Fields(m[1])) > 4 {
				add("troparion", "troparion", after(text, plain, len(m[1])), tone, nil)
			} else {
				expect, expectTone = "troparion", tone
			}
			continue
		case strings.HasPrefix(plain, "слава, и ныне, бгородичен") && strings.Contains(plain, "тебе и стену, и пристанище имамы"):
			add("theotokion", "theotokion", after(text, plain, len(plain)-strings.Index(plain, "тебе и стену")), 0, nil)
			continue
		}
		if ode == 0 {
			continue
		}
		b := fmt.Sprintf("canon.ode.%d.", ode)
		switch {
		case reTPGlory.MatchString(plain):
			add(b+"glory", "troparion", after(text, plain, len(reTPGlory.FindStringSubmatch(plain)[1])), 0, nil)
		case reTPNow.MatchString(plain):
			add(b+"theotokion", "theotokion", after(text, plain, len(reTPNow.FindStringSubmatch(plain)[1])), 0, nil)
			odesDone[ode], lastOde, ode = true, ode, 0
		case strings.HasPrefix(plain, "катаваси"):
			if rest := strings.TrimSpace(plain[strings.Index(plain, ":")+1:]); strings.Contains(plain, ":") && len(strings.Fields(rest)) > 6 {
				add(b+"katavasia", "irmos", after(text, plain, len(rest)), 0, nil)
			}
			odesDone[ode], lastOde, ode = true, ode, 0
		case reTPSedalen.MatchString(plain) && ode == 3:
			odesDone[ode], lastOde, ode = true, ode, 0
			section, nSec = "sedalen", 1
			if i := strings.Index(plain, ":"); i > 0 && len(strings.Fields(plain[i+1:])) > 4 && !strings.HasPrefix(strings.TrimSpace(plain[i+1:]), "подобен") {
				add("matins.sedalen.1", "sedalen", after(text, plain, len(plain)-i-1), toneOf(text), nil)
			} else {
				expect, expectTone = "matins.sedalen.1", toneOf(text)
			}
		case reTPStop.MatchString(plain), len(strings.Fields(plain)) < 8:
			odesDone[ode], lastOde, ode = true, ode, 0
		default:
			n++
			add(fmt.Sprintf("%stroparion.%d", b, n), "troparion", text, 0, nil)
		}
	}
	return units
}

// triodionRubric: the rubric of the liturgy from a paragraph on, with the
// paragraphs that continue it.
func triodionRubric(d *Doc, pi int, start string) string {
	out := start
	for j := pi + 1; j < len(d.Paras) && j <= pi+3; j++ {
		np := strings.ToLower(hip.PlainString(CleanText(d.Paras[j].Text())))
		if !reTPRubricCont.MatchString(np) {
			break
		}
		out += " " + np
	}
	return out
}

// triodionExtraReadings: the readings after the day's first Apostle and
// Gospel, as (key, lectionary unit): lit.apostle.2 → lect.zach.ap.<N>,
// lit.gospel.2 → lect.zach.<book>.<N>.
func triodionExtraReadings(text, plain string, pos []int) [][2]string {
	var out [][2]string
	nAp, nGos := 0, 0
	lastBook, prev := "", 0 // the last Gospel's book; the end of the last citation
	for _, m := range reTPZach.FindAllStringSubmatchIndex(plain, -1) {
		n := uniNumeral(text, pos, plain, m[2], m[3])
		if n == 0 {
			continue
		}
		before := plain[max(0, m[0]-40):m[0]]
		book := ""
		if g := reTPGospelBook.FindStringSubmatch(before); g != nil {
			book = gospelByName[g[1]]
		} else if lastBook != "" && !reTPEpistle.MatchString(plain[prev:m[0]]) {
			book = lastBook // «еиглие от матфеа, зачало зи. и преподобных, зачало мг»
		}
		prev = m[1]
		if book != "" {
			lastBook = book
			if nGos++; nGos > 1 {
				out = append(out, [2]string{fmt.Sprintf("lit.gospel.%d", nGos), fmt.Sprintf("lect.zach.%s.%d", book, n)})
			}
			continue
		}
		if nAp++; nAp > 1 {
			out = append(out, [2]string{fmt.Sprintf("lit.apostle.%d", nAp), fmt.Sprintf("lect.zach.ap.%d", n)})
		}
	}
	return out
}
