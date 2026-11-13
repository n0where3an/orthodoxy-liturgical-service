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

// Octoechos day files: "<tone>_<n><day>.hip", n = 1 (Sunday) .. 7 (Saturday).
// Each weekday file opens with the vespers of its eve ("В неделю вечера" in
// the Monday file), which belong to that liturgical day.
var octoechosDay = [8]string{"", "1nd", "2pn", "3vt", "4sr", "5ch", "6pt", "7sb"}

// weekdayKey names days in unit keys (0 = Sunday).
var weekdayKey = [7]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

var (
	reLIC      = regexp.MustCompile(`^(вечер,? )?(на гдси,? воззвах|стихиры )`)
	reAposti   = regexp.MustCompile(`на стиховне`)
	reGloryNow = regexp.MustCompile(`^слава,? и ныне`)
	reVerse    = regexp.MustCompile(`^стих:?\s`)
	reEndVesp  = regexp.MustCompile(`ныне отпущаеши|на повечерии`)
	reSkipPara = regexp.MustCompile(`^(подобен|ины стихиры|таже|глас |аще [^:]*минеи)`)
	// a label before a sticheron: "мчнчен:", "мертвен:", "подобен, мчнчен:"
	reLabel = regexp.MustCompile(`^((?:подобен|мчнчен|мученичен|мертвен|покаянен)(?:,\s*\S+)?):\s*`)
)

// ImportOctoechosVespers parses the daily vespers of all tones (weekdays only;
// the Saturday-evening great vespers are not parsed here).
func ImportOctoechosVespers(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		for dow := 2; dow <= 7; dow++ {
			path := filepath.Join(root, "octoih", "oc", fmt.Sprintf("%d_%s.hip", tone, octoechosDay[dow]))
			us, err := parseOctoechosVespers(path, tone, dow-1)
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			units = append(units, us...)
		}
	}
	return units, warnings
}

func parseOctoechosVespers(path string, tone, weekday int) ([]corpus.Unit, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("oc.%d.%s.vespers", tone, weekdayKey[weekday])
	var units []corpus.Unit
	add := func(key, kind, body string, meta map[string]any) {
		units = append(units, corpus.Unit{
			ID: prefix + "." + key, Book: "octoechos", Kind: kind, Tone: tone,
			Lang: "cu", Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: meta,
		})
	}
	section := ""
	nLIC, nApo := 0, 0
	pendingVerse := ""
	wantTheotokion := "" // section whose theotokion text is in the next paragraph
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := hip.PlainString(text)
		switch {
		case reEndVesp.MatchString(plain) && section != "":
			return checkOctoechos(path, units, nLIC, nApo)
		case section == "" && reLIC.MatchString(plain):
			section = "lic"
			continue
		case reAposti.MatchString(plain):
			section = "aposticha"
			continue
		case section == "":
			continue
		case reGloryNow.MatchString(plain):
			// "Слава, и ныне, богородичен: <text>" — text inline or next.
			if body := afterColon(text, 8); body != "" {
				add(section+".theotokion", "theotokion", body, map[string]any{"heading": headOf(text)})
			} else {
				wantTheotokion = section
			}
			continue
		case reVerse.MatchString(plain):
			pendingVerse = verseText(text)
			continue
		case reSkipPara.MatchString(plain) && (len(strings.Fields(plain)) <= 8 || !reLabel.MatchString(plain)):
			continue
		}
		if wantTheotokion != "" {
			add(wantTheotokion+".theotokion", "theotokion", text, nil)
			wantTheotokion = ""
			continue
		}
		body := text
		var meta map[string]any
		if m := reLabel.FindStringSubmatch(plain); m != nil && len(strings.Fields(plain)) > 8 {
			meta = map[string]any{"label": m[1]}
			body = strings.TrimSpace(text[strings.Index(text, ":")+1:])
		}
		switch section {
		case "lic":
			nLIC++
			add(fmt.Sprintf("lic.sticheron.%d", nLIC), "sticheron", body, meta)
		case "aposticha":
			nApo++
			if pendingVerse != "" && nApo > 1 {
				add(fmt.Sprintf("aposticha.verse.%d", nApo), "verse", pendingVerse, nil)
			}
			add(fmt.Sprintf("aposticha.sticheron.%d", nApo), "sticheron", body, meta)
		}
		pendingVerse = ""
	}
	return nil, fmt.Errorf("%s: end of vespers not found", filepath.Base(path))
}

func checkOctoechos(path string, units []corpus.Unit, nLIC, nApo int) ([]corpus.Unit, error) {
	if nLIC < 6 || nApo < 3 {
		return units, fmt.Errorf("%s: %d stichera at «Господи, воззвах», %d at the aposticha", filepath.Base(path), nLIC, nApo)
	}
	return units, nil
}

// verseText strips the "Стих:" label (written with or without a colon).
func verseText(text string) string {
	plain, pos := hip.Plain(text)
	m := reVerse.FindStringIndex(plain)
	rs := []rune(text)
	if m == nil || m[1] >= len(pos) {
		return text
	}
	return strings.TrimSpace(string(rs[pos[len([]rune(plain[:m[1]]))]:]))
}

// afterColon returns the text after the first ':' if it is at least minWords long.
func afterColon(text string, minWords int) string {
	k := strings.Index(text, ":")
	if k < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[k+1:])
	if len(strings.Fields(rest)) < minWords {
		return ""
	}
	return rest
}

func headOf(text string) string {
	if k := strings.Index(text, ":"); k >= 0 {
		return text[:k+1]
	}
	return text
}
