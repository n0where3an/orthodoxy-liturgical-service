package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Great vespers of a Menaion service (polyeleos, vigil, feast): a service
// with an entrance and Old Testament readings. When the file also has small
// vespers (vigil), the great vespers start at «На велицей вечерни». Keys
// under me.<MM>.<DD>.gvespers:
//
//	lic.on.<N> (marker: stichera on N), lic.sticheron.<N>, lic.glory, lic.now
//	paremia.<N>.title, paremia.<N>
//	litiya.sticheron.<N>, litiya.glory, litiya.now        (vigil)
//	aposticha.sticheron.<N>, aposticha.verse.<N>, aposticha.glory, aposticha.now
//	loaves (marker: the Menaion's rubric for the blessing of loaves)
//
// A "now" hymn is the theotokion or, at feasts, a hymn of the feast.
var (
	reGVStart    = regexp.MustCompile(`на велицей вечерни`)
	reGVSmall    = regexp.MustCompile(`на малей вечерни`)
	reGVEntrance = regexp.MustCompile(`^(вход|и вход)`)
	reGVReading  = regexp.MustCompile(`чтение[.:]?$`)
	reGVLitiya   = regexp.MustCompile(`^на литии`)
	reGVAposti   = regexp.MustCompile(`^на стиховне`)
	reGVLoaves   = regexp.MustCompile(`^на благословении хлебов`)
	reGVStop     = regexp.MustCompile(`^(на утрени|на повечерии|на полунощнице|тропарь|канон|на литургии|по а-м стихословии)`)
	// "На велицей вечерни, на Господи, воззвах, стихиры…", "И поем стихиры…"
	reGVLIC   = regexp.MustCompile(`на гдси,? воззвах|^(и )?поем стихиры`)
	reGVTwice = regexp.MustCompile(`^(дважды|трижды)\.?$`)
	// «Чтения три. Писаны (ищи) сего месяца в а҃ день»
	reGVSee = regexp.MustCompile(`(писаны|ищи) сего (месяца|мцса|мсца) в `)
	// «Вход. Прокимен дне: и чтения г҃. Исхода чтение.»
	reGVTitleTail = regexp.MustCompile(`([^.:]+ чте́нїе)[.:]?\s*$`)
	reGVChapter   = regexp.MustCompile(`^\s*\[[^\]]*\]\s*`)
)

func parseMenaionGreatVespers(d *Doc, prefix, path string) []corpus.Unit {
	hasSmall, hasGreat := false, false
	for _, p := range d.Paras {
		plain := hip.PlainString(p.Text())
		if reGVSmall.MatchString(plain) {
			hasSmall = true
		}
		if reGVStart.MatchString(plain) {
			hasGreat = true
		}
	}
	started := !(hasSmall && hasGreat)
	var units []corpus.Unit
	seen := map[string]bool{}
	lastTone := 0
	add := func(key, kind, heading, body string, meta map[string]any) *corpus.Unit {
		body = CleanText(body)
		if body == "" || seen[key] {
			return nil
		}
		seen[key] = true
		m := map[string]any{"heading": CleanText(heading), "file": filepath.Base(path)}
		for k, v := range meta {
			m[k] = v
		}
		tone := toneOf(heading)
		if tone == 0 && strings.Contains(hip.PlainString(heading), "тойже") || tone == 0 && kind == "sticheron" {
			tone = lastTone // «Слава, глас тойже», stichera under one heading
		}
		if tone > 0 {
			lastTone = tone
		}
		units = append(units, corpus.Unit{
			ID: prefix + ".gvespers." + key, Book: "menaion", Kind: kind, Tone: tone,
			Podoben: podobenOf(heading), Lang: "cu", Body: body, SourceID: "orthlib.mineya", Meta: m,
		})
		return &units[len(units)-1]
	}
	section, part, heading := "", "", ""
	n := map[string]int{}
	pendingVerse := ""
	paremia := 0
	var paremiaText []string
	flushParemia := func() {
		if paremia > 0 && len(paremiaText) > 0 {
			add(fmt.Sprintf("paremia.%d", paremia), "reading", "", strings.Join(paremiaText, "\n"), nil)
		}
		paremiaText = nil
	}
	readings := 0
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := hip.PlainString(text)
		if !started {
			started = reGVStart.MatchString(plain)
			if !started {
				continue
			}
		}
		red := hip.PlainString(p.RedPrefix())
		isHead := len([]rune(strings.TrimSpace(red))) > 3 && !reGVChapter.MatchString(red)
		if isHead {
			switch {
			case section != "" && (reGVStop.MatchString(plain) || reGVLoaves.MatchString(plain)):
				if reGVLoaves.MatchString(plain) {
					add("loaves", "marker", "", text, nil)
				}
				flushParemia()
				goto done
			case section == "" && (reLIC.MatchString(plain) || reGVLIC.MatchString(plain)):
				section, part, heading = "lic", "sticheron", text
				if m := reSingOn.FindStringSubmatch(text); m != nil && !seen["lic.on"] {
					seen["lic.on"] = true
					units = append(units, corpus.Unit{
						ID: fmt.Sprintf("%s.gvespers.lic.on.%d", prefix, toneLetters[[]rune(m[1])[0]]), Book: "menaion",
						Kind: "marker", Lang: "cu", Body: text, SourceID: "orthlib.mineya",
					})
				}
			case reGVEntrance.MatchString(plain) || section == "lic" && strings.Contains(plain, "вход"):
				section, part = "paremia", ""
				readings = 1
				if reGVSee.MatchString(plain) {
					w := strings.Fields(text)
					for i := range w {
						if hip.PlainString(w[i]) == "в" && i+1 < len(w) {
							if day := numeralOfWord(w[i+1]); day > 0 {
								add("see", "marker", "", text, map[string]any{"day": day, "litiya": strings.Contains(plain, "лити")})
								break
							}
						}
					}
				}
				if m := reGVTitleTail.FindStringSubmatch(text); m != nil && reGVReading.MatchString(plain) {
					paremia++
					add(fmt.Sprintf("paremia.%d.title", paremia), "reading_title", "", strings.TrimSpace(m[1])+".", nil)
				}
			case section == "paremia" && reGVReading.MatchString(plain):
				flushParemia()
				paremia++
				add(fmt.Sprintf("paremia.%d.title", paremia), "reading_title", "", text, nil)
				continue
			case reGVLitiya.MatchString(plain):
				flushParemia()
				section, part, heading = "litiya", "sticheron", text
			case reGVAposti.MatchString(plain):
				flushParemia()
				section, part, heading = "aposticha", "sticheron", text
				if strings.Contains(plain, "слава") {
					part = "glory"
				}
			case section == "":
			case reGVTwice.MatchString(plain):
				// «Дважды»: the sticheron takes two places.
				if last := len(units) - 1; last >= 0 && units[last].Kind == "sticheron" && part == "sticheron" {
					orig := units[last]
					copies := 1
					if strings.HasPrefix(plain, "трижды") {
						copies = 2
					}
					for c := 0; c < copies; c++ {
						n[section]++
						add(fmt.Sprintf("%s.sticheron.%d", section, n[section]), "sticheron", orig.Meta["heading"].(string), orig.Body,
							map[string]any{"repeat_of": orig.ID})
					}
				}
				continue
			case reVerse.MatchString(plain):
				pendingVerse = verseText(text)
				continue
			case reGlory.MatchString(plain) && strings.Contains(plain[:min(len(plain), 30)], "и ныне"):
				part, heading = "glory_now", text
			case reGlory.MatchString(plain):
				part, heading = "glory", text
			case reAndNow.MatchString(plain):
				part, heading = "now", text
				// A cue ("И ныне, богородичен: Никтоже притекаяй к тебе:")
				// points to a hymn given elsewhere.
				// The cue is in black after the red rubric.
				if rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), strings.TrimSpace(p.RedPrefix()))); rest != "" &&
					len(strings.Fields(rest)) >= 2 && len(strings.Fields(rest)) < 8 && strings.HasSuffix(rest, ":") {
					part = ""
				}
			default:
				// «Ины стихиры», «таже стихиры…», «глас д:», authors: same part.
				if part == "" || part == "glory" || part == "now" || part == "glory_now" {
					part = "sticheron"
				}
				if strings.HasPrefix(plain, "подобен") {
					heading += " " + text // the tone stays in the section's heading
				} else {
					heading = text
				}
			}
			if t := toneOf(text); t > 0 {
				lastTone = t
			}
			if _, inline := splitHeading(p, text, true); inline == "" {
				continue
			} else {
				text = inline
			}
		}
		switch section {
		case "", "paremia":
			if section == "paremia" && paremia > 0 {
				paremiaText = append(paremiaText, reGVChapter.ReplaceAllString(text, ""))
			}
			continue
		}
		switch part {
		case "sticheron":
			n[section]++
			k := n[section]
			if section == "aposticha" && pendingVerse != "" {
				add(fmt.Sprintf("%s.verse.%d", section, k), "verse", "", pendingVerse, nil)
			}
			add(fmt.Sprintf("%s.sticheron.%d", section, k), "sticheron", heading, text, nil)
			pendingVerse = ""
		case "glory":
			add(section+".glory", "sticheron", heading, text, nil)
			part = ""
		case "now", "glory_now":
			kind := "sticheron"
			if strings.Contains(hip.PlainString(heading), "бгородичен") {
				kind = "theotokion"
			}
			if u := add(section+".now", kind, heading, text, nil); u != nil && part == "glory_now" {
				u.Meta["glory_now"] = true
			}
			part = ""
		}
	}
	flushParemia()
done:
	if readings == 0 && paremia == 0 {
		return nil // not a great vespers
	}
	return units
}
