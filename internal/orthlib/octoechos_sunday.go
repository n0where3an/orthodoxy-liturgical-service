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
