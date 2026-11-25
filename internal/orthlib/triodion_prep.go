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
}

var (
	reTPOde       = regexp.MustCompile(`^(?:глас \S+ )?песнь ([а-ѳ])(?:[.:]|$)`)
	reTPProk      = regexp.MustCompile(strings.Replace(reSpecProk.String(), `прокимен,?\s*`, `прокимен,?\s*(?:апсла,?\s*)?`, 1))
	reTPKontakion = regexp.MustCompile(`^кондак(?:,\s*глас[^:]*)?:\s*(.*)$`)
	reTPTroparion = regexp.MustCompile(`^(?:и по тристом глаголем )?тропарь,\s*глас [^:]*:\s*(.*)$`)
	reTPStop      = regexp.MustCompile(`^(песнь|кондак|вторый кондак|икос|седален|катаваси|светилен|канон|ин канон|таже|по |на |слава, и ныне|тристую|в настоящий|синаксар|стих|предисловие)`)
	reTPGlory     = regexp.MustCompile(`^слава(?:,\s*тр\S*чен)?:\s*(.+)$`)
	reTPNow       = regexp.MustCompile(`^(?:и ныне,?\s*)?(?:бгородичен|бцдичен)[^:]*:\s*(.+)$`)
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
		for _, k := range []string{"kontakion", "lit.rubric"} {
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
	expect := "" // the next paragraph's text: kontakion | troparion | ikos
	expectTone := 0
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		if expect != "" {
			if !reTPStop.MatchString(plain) {
				add(expect, expect, text, expectTone, nil)
			}
			expect = ""
			continue
		}
		// The rubric of the liturgy, inside a long paragraph.
		if i := strings.Index(plain, "на литургии"); i >= 0 && strings.Contains(plain[i:], "зачало") && !seen["lit.rubric"] {
			_, pos := hip.Plain(text)
			sub := uni(text, pos, plain, i, len(plain))
			sp := strings.ToLower(hip.PlainString(sub))
			_, spos := hip.Plain(sub)
			parseLiturgyRubricWith(reTPProk, sub, sp, spos, func(key, kind, body string, meta map[string]any) {
				// «аллилуиа, глас в. стих: насаждени…» — the first verse.
				if a, ok := meta["alleluia"].(string); ok {
					if i := strings.Index(a, ":"); i > 0 && strings.HasPrefix(strings.ToLower(hip.PlainString(a[:i])), "стих") {
						meta["alleluia"] = strings.TrimSpace(a[i+1:])
					}
				}
				delete(meta, "beatitudes") // the Triodion's odes: tp.<day>.canon.ode.N
				add("lit."+key, kind, body, 0, meta)
			})
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
			odesDone[ode], ode = true, 0
		case reTPStop.MatchString(plain), len(strings.Fields(plain)) < 8:
			odesDone[ode], ode = true, 0
		default:
			n++
			add(fmt.Sprintf("%stroparion.%d", b, n), "troparion", text, 0, nil)
		}
	}
	return units
}
