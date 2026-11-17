package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The Liturgy of a Menaion service. The Menaion gives it as a rubric
// («На литургии: блаженна от канона…, прокимен, глас з: … апостол к
// евреем, зачало тле. аллилуиа… евангелие луки, зачало кд. причастен…»),
// with hymns cited by their first words, and, for the feasts of the Lord,
// the festal antiphons, the entrance verse and the hymn instead of the
// Trisagion in full. Units under me.<MM>.<DD>.lit:
//
//	rubric (marker; Meta: the parsed citations, resolved by ResolveLiturgy)
//	antiphon.<A>.<K>  verses and refrains of the festal antiphons, in order
//	entrance          the entrance verse
//	trisagion         the hymn sung instead of the Trisagion
//
// Units whose Meta has "copy" take the text of another unit (a refrain
// cited by its first words, «Единородный Сыне», the feast's troparion).
var (
	reLitStart     = regexp.MustCompile(`на литургии`)
	reLitAntiphons = regexp.MustCompile(`на литургии,? антифон`)
	reLitBeat      = regexp.MustCompile(`бла?женна[^.]*`)
	reLitBeatOde   = regexp.MustCompile(`(?:(а|в|г)-го (?:канона )?)?песнь (\S+?)(?:-я|-ю)?[ ,.:]`)
	reLitProk      = regexp.MustCompile(`прокимен,?\s*(?:песнь бцды,? |стаго,?\s*)?(?:глас (\S+?)[.:,]\s*|:\s*)(?:песнь бцды:\s*)?(.+?)\s*стих:\s*(.+?)\s*(?:всякое дыхание|апсл|апостол|еиглие|евангелие)`)
	reLitApostle   = regexp.MustCompile(`(?:апсл|апостол)[^.]*?зачало (\S+?)(\s+от полу)?[.,:]`)
	reLitAlleluia  = regexp.MustCompile(`аллилуиа,? глас (\S+?)[.:]\s*(.+?)\s*(?:стих:\s*(.+?)\s*)?(?:еиглие|евангелие)`)
	reLitGospel    = regexp.MustCompile(`(?:еиглие|евангелие) (?:от )?(матфеа|марка|луки|иоанна)[^.]*?зачало (\S+?)(\s+от полу)?[.,:]`)
	reLitZadost    = regexp.MustCompile(`(?:за достойно|вместо достойно)[^:]*?ирмос:\s*(.+?)[:.]`)
	reLitKinonik   = regexp.MustCompile(`причастен:\s*(.+?)(?::|\.|$)`)
	reLitAntiphonN = regexp.MustCompile(`антифон `)
)

var gospelByName = map[string]string{"матфеа": "mt", "марка": "mk", "луки": "lk", "иоанна": "jn"}

// uni returns the Unicode text of the plain byte range [a, b) of a paragraph.
func uni(text string, pos []int, plain string, a, b int) string {
	ra, rb := len([]rune(plain[:a])), len([]rune(plain[:b]))
	rs := []rune(text)
	if ra >= len(pos) {
		return ""
	}
	end := len(rs)
	if rb < len(pos) {
		end = pos[rb]
	}
	return strings.TrimSpace(string(rs[pos[ra]:end]))
}

// uniNumeral reads a Slavonic numeral cited in the plain range (Unicode
// keeps ѕ = 6 apart from з = 7).
func uniNumeral(text string, pos []int, plain string, a, b int) int {
	w := strings.Fields(uni(text, pos, plain, a, b))
	if len(w) == 0 {
		return 0
	}
	return numeralOfWord(strings.TrimRight(w[0], ".,:"))
}

func parseMenaionLiturgy(d *Doc, prefix, path, day string) []corpus.Unit {
	var units []corpus.Unit
	add := func(key, kind, body string, meta map[string]any) {
		m := map[string]any{"file": filepath.Base(path)}
		for k, v := range meta {
			m[k] = v
		}
		units = append(units, corpus.Unit{ID: prefix + ".lit." + key, Book: "menaion", Kind: kind, Lang: "cu",
			Body: CleanText(body), SourceID: "orthlib.mineya", Meta: m})
	}
	troparion := fmt.Sprintf("me.%s.troparion.1", day)
	antiphon, k := 0, 0
	section := ""          // antiphons | entrance | trisagion
	var sung []corpus.Unit // full texts of the current antiphon, for cues
	rubricDone := false
	nextTroparion := false
	item := func(kind, body string, meta map[string]any) {
		k++
		add(fmt.Sprintf("antiphon.%d.%d", antiphon, k), kind, body, meta)
	}
	// content adds a text line of an antiphon: in full, or cited by its
	// first words (a refrain, the repeated verse, «Единородный Сыне»).
	content := func(text string) {
		plain := hip.PlainString(text)
		if strings.Contains(plain, "единородный сне") {
			item("hymn", text, map[string]any{"copy": "ch.typika.only_begotten"})
			return
		}
		if strings.HasSuffix(strings.TrimSpace(plain), ":") || strings.HasSuffix(strings.TrimSpace(plain), ".") && len(strings.Fields(plain)) <= 4 {
			cue := hip.Fold(strings.TrimRight(strings.TrimSpace(plain), ":."))
			for i := len(sung) - 1; i >= 0; i-- {
				if strings.HasPrefix(hip.Fold(hip.PlainString(sung[i].Body)), cue) {
					item(sung[i].Kind, sung[i].Body, map[string]any{"repeat_of": sung[i].ID})
					return
				}
			}
		}
		item("verse", text, nil)
		sung = append(sung, units[len(units)-1])
	}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if !rubricDone && reLitAntiphons.MatchString(plain) {
			section = "antiphons"
		}
		if !rubricDone && (reLitStart.MatchString(plain) || section != "") && reLitProk.MatchString(plain) ||
			strings.HasPrefix(plain, "тристое: прокимен") {
			if !rubricDone {
				parseLiturgyRubric(text, plain, func() []int { _, pos := hip.Plain(text); return pos }(), add)
				rubricDone = true
			}
			section = ""
			continue
		}
		if section == "" || section == "done" {
			continue
		}
		// A red rubric, possibly ending with the drop cap of the text that
		// follows in black: "Сти́хъ в҃: Д|але́че…", "Тропа́рь: С|пасѝ…".
		red := strings.TrimSpace(p.RedPrefix())
		label, body := "", text
		if len([]rune(hip.PlainString(red))) > 3 {
			label = red
			body = ""
			if i := strings.LastIndex(red, ":"); i >= 0 {
				if cap := strings.TrimSpace(red[i+1:]); cap != "" && len([]rune(hip.PlainString(cap))) <= 2 {
					label = red[:i+1]
					body = cap + strings.TrimPrefix(strings.TrimSpace(text), red)
				}
			}
		}
		lp := strings.ToLower(hip.PlainString(label))
		if m := reLitAntiphonN.FindStringSubmatchIndex(lp); m != nil {
			// "А҆нтїфѡ́нъ в҃…", "Ѻ҆́ба ли̑ка вкꙋ́пѣ. А҆нтїфѡ́нъ г҃…"
			w := strings.Fields(label[strings.Index(strings.ToLower(label), "нтїфѡ́нъ"):])
			if len(w) > 1 {
				if n := numeralOfWord(strings.TrimRight(w[1], ".,:")); n > 0 {
					antiphon, k, sung = n, 0, nil
					section = "antiphons"
				}
			}
		}
		switch {
		case strings.Contains(lp, "входное"):
			section = "entrance"
			continue
		case strings.Contains(lp, "вход"):
			continue // «Таже вход. Диакон речет: Премудрость, прости»: in the template
		case strings.Contains(lp, "тристаго"):
			section = "trisagion"
			continue
		case strings.Contains(lp, "кондак"):
			continue // «Слава, и ныне, кондак» after the entrance: in the template
		}
		switch section {
		case "antiphons":
			if antiphon == 0 {
				continue
			}
			if strings.Contains(lp, "слава, и ныне") {
				item("rubric", "", map[string]any{"copy": "common.glory_now"})
			}
			if strings.Contains(lp, "тропарь") {
				if body == "" {
					nextTroparion = true // «Тропарь, глас з:» — the text follows
					continue
				}
				item("troparion", body, map[string]any{"copy": troparion})
				continue
			}
			if body != "" && nextTroparion {
				item("troparion", body, map[string]any{"copy": troparion})
				nextTroparion = false
				continue
			}
			if body != "" {
				content(body)
			}
		case "entrance":
			if body != "" && !strings.Contains(lp, "тропарь") {
				add("entrance", "verse", body, nil)
				section = "done-entrance"
			}
		case "trisagion":
			if body != "" {
				add("trisagion", "hymn", body, nil)
				section = "done"
			}
		}
	}
	return units
}

// parseLiturgyRubric reads the citations of «На литургии…».
func parseLiturgyRubric(text, plain string, pos []int, add func(key, kind, body string, meta map[string]any)) {
	meta := map[string]any{}
	if m := reLitBeat.FindStringIndex(plain); m != nil {
		var odes [][2]int
		clause := plain[m[0]:m[1]] + "."
		for _, om := range reLitBeatOde.FindAllStringSubmatchIndex(clause, -1) {
			canon := 1
			if om[2] >= 0 {
				canon = map[string]int{"а": 1, "в": 2, "г": 3}[clause[om[2]:om[3]]]
			}
			ode := uniNumeral(text, pos, plain, m[0]+om[4], m[0]+om[5])
			if ode == 7 && !strings.Contains(uni(text, pos, plain, m[0]+om[4], m[0]+om[5]), "з҃") {
				ode = 6
			}
			if ode > 0 {
				odes = append(odes, [2]int{canon, ode})
			}
		}
		// «…песнь г, д-го гласа, и з-я, а-го гласа» — one canon.
		if len(odes) > 0 && strings.Contains(clause, "гласа") {
			for i := range odes {
				odes[i][0] = 1
			}
		}
		if len(odes) > 0 {
			meta["beatitudes"] = odes
		}
	}
	if m := reLitProk.FindStringSubmatchIndex(plain); m != nil {
		if m[2] >= 0 { // else the tone of the cited prokeimenon (ResolveLiturgy)
			meta["prok_tone"] = uniNumeral(text, pos, plain, m[2], m[3])
		}
		meta["prok"] = uni(text, pos, plain, m[4], m[5])
		meta["prok_verse"] = uni(text, pos, plain, m[6], m[7])
	}
	if m := reLitApostle.FindStringSubmatchIndex(plain); m != nil {
		meta["apostle"] = uniNumeral(text, pos, plain, m[2], m[3])
		meta["apostle_half"] = m[4] >= 0
	}
	if m := reLitAlleluia.FindStringSubmatchIndex(plain); m != nil {
		meta["alleluia_tone"] = uniNumeral(text, pos, plain, m[2], m[3])
		meta["alleluia"] = uni(text, pos, plain, m[4], m[5])
		if m[6] >= 0 {
			meta["alleluia_verse"] = uni(text, pos, plain, m[6], m[7])
		}
	}
	if m := reLitGospel.FindStringSubmatchIndex(plain); m != nil {
		meta["gospel_book"] = gospelByName[plain[m[2]:m[3]]]
		meta["gospel"] = uniNumeral(text, pos, plain, m[4], m[5])
		meta["gospel_half"] = m[6] >= 0
	}
	if m := reLitZadost.FindStringSubmatchIndex(plain); m != nil {
		meta["zadostoinik"] = uni(text, pos, plain, m[2], m[3])
	}
	if m := reLitKinonik.FindStringSubmatchIndex(plain); m != nil {
		meta["kinonik"] = uni(text, pos, plain, m[2], m[3])
	}
	add("rubric", "marker", text, meta)
}
