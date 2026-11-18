package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The great vespers of Sunday (Saturday evening) in the Octoechos, file
// "<tone>_1nd.hip", section «В субботу на велицей вечерни». Keys under
// oc.<T>.sun:
//
//	gvespers.lic.sticheron.1-3   resurrectional stichera
//	gvespers.lic.sticheron.4-7   «ины стихиры анатолиевы»
//	gvespers.lic.theotokos.1-3   of the Theotokos (Paul of Amorion), «идеже
//	                             несть минеи»
//	gvespers.lic.dogmatikon      «Слава, и ныне» (after the Menaion's «Слава»)
//	gvespers.aposticha.sticheron.1-4, gvespers.aposticha.verse.2-4
//	gvespers.aposticha.theotokion
//	troparion                    the resurrectional troparion of the tone
var (
	reSVStart     = regexp.MustCompile(`^в субботу на велицей вечерни`)
	reSVAnat      = regexp.MustCompile(`^ины стихиры,? (творение )?анатоли`)
	reSVTheotokos = regexp.MustCompile(`^ины стихиры,? (престей |престыя )?бцде`)
	reSVAposti    = regexp.MustCompile(`^на стиховне стихиры|входим во храм, поюще стихиру`)
	reSVAlphabet  = regexp.MustCompile(`^(ины стихиры,? )?по алфавиту`)
	reSVGloryNow  = regexp.MustCompile(`^слава,? (минеи\.? )?и ныне,? бгородичен`)
	reSVTroparion = regexp.MustCompile(`тропарь`)
	reSVSkip      = regexp.MustCompile(`^(на гдси|стихиры|ины стихиры|подобен|вход|таже|по обычных|и по возгласе|по предначинательном|слава,? (и ныне|минеи)|бгородичен|глас )`)
)

// ImportOctoechosSundayVespers parses the Sunday great vespers of all tones.
func ImportOctoechosSundayVespers(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[1]))
		us, err := parseOctoechosSundayVespers(path, tone)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		units = append(units, us...)
	}
	return units, warnings
}

func parseOctoechosSundayVespers(path string, tone int) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.sun.", tone)
	var units []corpus.Unit
	add := func(key, kind, body string) {
		units = append(units, corpus.Unit{ID: prefix + key, Book: "octoechos", Kind: kind, Tone: tone, Lang: "cu",
			Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: map[string]any{"file": filepath.Base(path)}})
	}
	n := map[string]int{}
	section, want, verse := "", "", ""
	started := false
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		if !started {
			started = reSVStart.MatchString(plain)
			if started {
				section = "res"
			}
			continue
		}
		// The text announced by the previous paragraph.
		if want != "" && !reSVSkip.MatchString(plain) {
			switch want {
			case "dogmatikon":
				add("gvespers.lic.dogmatikon", "theotokion", text)
			case "aposticha.theotokion":
				add("gvespers.aposticha.theotokion", "theotokion", text)
			case "troparion":
				add("troparion", "troparion", text)
				return checkSundayVespers(path, units, n)
			}
			want = ""
			continue
		}
		switch {
		case strings.Contains(plain, "повечерии"):
			return checkSundayVespers(path, units, n)
		case section == "aposticha" && reSVTroparion.MatchString(plain) && len([]rune(plain)) < 150:
			// «Тропарь воскресен, глас …:», «…по Отче наш: тропарь …»
			want = "troparion"
			continue
		case reSVAnat.MatchString(plain):
			section = "anat"
			continue
		case reSVTheotokos.MatchString(plain):
			section = "theotokos"
			continue
		case reSVAposti.MatchString(plain):
			section = "aposticha"
			continue
		case reSVAlphabet.MatchString(plain):
			continue
		case reSVGloryNow.MatchString(plain):
			key := "dogmatikon"
			if section == "aposticha" {
				key = "aposticha.theotokion"
			}
			if body := afterColon(text, 8); body != "" {
				if key == "dogmatikon" {
					add("gvespers.lic.dogmatikon", "theotokion", body)
				} else {
					add("gvespers.aposticha.theotokion", "theotokion", body)
				}
			} else {
				want = key
			}
			continue
		case reVerse.MatchString(plain):
			verse = verseText(text)
			continue
		case reSVSkip.MatchString(plain):
			continue
		}
		switch section {
		case "res", "anat":
			n["lic"]++
			add(fmt.Sprintf("gvespers.lic.sticheron.%d", n["lic"]), "sticheron", text)
		case "theotokos":
			n["theotokos"]++
			add(fmt.Sprintf("gvespers.lic.theotokos.%d", n["theotokos"]), "sticheron", text)
		case "aposticha":
			n["aposticha"]++
			if n["aposticha"] > 1 && verse != "" {
				add(fmt.Sprintf("gvespers.aposticha.verse.%d", n["aposticha"]), "verse", verse)
			}
			add(fmt.Sprintf("gvespers.aposticha.sticheron.%d", n["aposticha"]), "sticheron", text)
		}
		verse = ""
	}
	return checkSundayVespers(path, units, n)
}

func checkSundayVespers(path string, units []corpus.Unit, n map[string]int) ([]corpus.Unit, error) {
	has := map[string]bool{}
	for _, u := range units {
		has[u.ID[strings.Index(u.ID[3:], ".")+4:]] = true
	}
	if n["lic"] != 7 || n["theotokos"] != 3 || n["aposticha"] != 4 || !has["sun.gvespers.lic.dogmatikon"] ||
		!has["sun.gvespers.aposticha.theotokion"] || !has["sun.troparion"] {
		return units, fmt.Errorf("%s: Sunday vespers: %v stichera, dogmatikon %v, aposticha theotokion %v, troparion %v",
			filepath.Base(path), n, has["sun.gvespers.lic.dogmatikon"], has["sun.gvespers.aposticha.theotokion"], has["sun.troparion"])
	}
	return units, nil
}

// The liturgy and the kontakion of Sunday in the Octoechos (the end of
// "<tone>_1nd.hip", «На литургии блаженны, глас …»). Keys under oc.<T>.sun:
//
//	kontakion                       the resurrectional kontakion (after the 6th ode)
//	lit.beatitudes.1-6, lit.beatitudes.glory, lit.beatitudes.now
//	lit.prokeimenon, lit.prokeimenon.verse
//	lit.alleluia.verse.1, lit.alleluia.verse.2
var (
	reSLStart     = regexp.MustCompile(`^на литургии.*бла?женн`)
	reSLKontakion = regexp.MustCompile(`^кондак,? глас`)
	reSLProk      = regexp.MustCompile(`^прокимен`)
	reSLAlleluia  = regexp.MustCompile(`^аллилуиа`)
	reSLGlory     = regexp.MustCompile(`^слава`)
	reSLNow       = regexp.MustCompile(`^и ныне`)
)

// ImportOctoechosSundayLiturgy parses the Sunday liturgy and kontakion of all tones.
func ImportOctoechosSundayLiturgy(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[1]))
		us, err := parseOctoechosSundayLiturgy(path, tone)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		units = append(units, us...)
	}
	return units, warnings
}

// textAfter returns the text after the colon that follows the plain marker
// (e.g. "глас") of a paragraph, or "" when it is shorter than minWords.
func textAfter(text, marker string, minWords int) string {
	plain, pos := hip.Plain(text)
	plain = strings.ToLower(plain)
	g := strings.Index(plain, marker)
	if g < 0 {
		g = 0
	}
	c := strings.Index(plain[g:], ":")
	if c < 0 {
		return ""
	}
	body := strings.TrimSpace(uni(text, pos, plain, g+c+1, len(plain)))
	if len(strings.Fields(body)) < minWords {
		return ""
	}
	return body
}

func parseOctoechosSundayLiturgy(path string, tone int) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.sun.", tone)
	var units []corpus.Unit
	add := func(key, kind, body string, t int) {
		units = append(units, corpus.Unit{ID: prefix + key, Book: "octoechos", Kind: kind, Tone: t, Lang: "cu",
			Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: map[string]any{"file": filepath.Base(path)}})
	}
	kontakion, want := "", ""
	section := ""
	n := 0
	var tail []string // from «прокимен» to the end: the prokeimenon and the alleluia
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		if section == "" {
			switch {
			case reSLStart.MatchString(plain):
				section = "beatitudes"
			case want == "kontakion":
				kontakion, want = text, ""
			case reSLKontakion.MatchString(plain):
				// The last one before the liturgy is that of the Sunday canon
				// (the first may be of the compline canon).
				if body := textAfter(text, "глас", 8); body != "" {
					kontakion = body
				} else {
					want = "kontakion"
				}
			}
			continue
		}
		if section == "prokeimenon" || reSLProk.MatchString(plain) {
			section = "prokeimenon"
			tail = append(tail, text)
			continue
		}
		if want != "" {
			add(want, map[bool]string{true: "theotokion", false: "troparion"}[want == "lit.beatitudes.now"], text, tone)
			want = ""
			continue
		}
		switch {
		case reVerse.MatchString(plain):
		case reSLGlory.MatchString(plain):
			if body := afterColon(text, 5); body != "" {
				add("lit.beatitudes.glory", "troparion", body, tone)
			} else {
				want = "lit.beatitudes.glory"
			}
		case reSLNow.MatchString(plain):
			if body := afterColon(text, 5); body != "" {
				add("lit.beatitudes.now", "theotokion", body, tone)
			} else {
				want = "lit.beatitudes.now"
			}
		default:
			n++
			add(fmt.Sprintf("lit.beatitudes.%d", n), "troparion", text, tone)
		}
	}
	// «Прокимен, глас …: P. Стих: V. Аллилуиа: A. Стих: B.» — in one
	// paragraph or several.
	if len(tail) > 0 {
		text := strings.Join(tail, " ")
		plain, pos := hip.Plain(text)
		plain = strings.ToLower(plain)
		g := strings.Index(plain, "глас")
		a := strings.Index(plain, "аллилуиа")
		if g >= 0 && a > g {
			c := g + strings.Index(plain[g:], ":") + 1
			cut := func(from, to int) (string, string) { // text, its «стих:» part
				seg := plain[from:to]
				if v := strings.Index(seg, "стих:"); v >= 0 {
					return strings.TrimSpace(uni(text, pos, plain, from, from+v)), strings.TrimSpace(uni(text, pos, plain, from+v+len("стих:"), to))
				}
				return strings.TrimSpace(uni(text, pos, plain, from, to)), ""
			}
			prok, pv := cut(c, a)
			ab := a + len("аллилуиа")
			for ab < len(plain) && (plain[ab] == ':' || plain[ab] == '.' || plain[ab] == ',' || plain[ab] == ' ') {
				ab++
			}
			al, av := cut(ab, len(plain))
			prok = strings.TrimRight(prok, ":") // «…людем своим даст: …миром:»
			if !strings.HasSuffix(prok, ".") {
				prok += "."
			}
			add("lit.prokeimenon", "prokeimenon", prok, tone)
			add("lit.prokeimenon.verse", "verse", pv, 0)
			add("lit.alleluia.verse.1", "verse", al, tone)
			add("lit.alleluia.verse.2", "verse", av, 0)
		}
	}
	if kontakion != "" {
		add("kontakion", "kontakion", kontakion, tone)
	}
	has := map[string]bool{}
	for _, u := range units {
		has[strings.TrimPrefix(u.ID, prefix)] = true
	}
	for _, k := range []string{"kontakion", "lit.beatitudes.6", "lit.beatitudes.glory", "lit.beatitudes.now", "lit.prokeimenon",
		"lit.prokeimenon.verse", "lit.alleluia.verse.1", "lit.alleluia.verse.2"} {
		if !has[k] {
			return units, fmt.Errorf("%s: Sunday liturgy: no %s (%d beatitudes)", filepath.Base(path), k, n)
		}
	}
	if n != 6 {
		return units, fmt.Errorf("%s: Sunday liturgy: %d beatitudes, want 6", filepath.Base(path), n)
	}
	return units, nil
}
