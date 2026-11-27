package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The weekday matins of Great Lent in the Triodion (Mon–Fri of the six
// weeks): the sessional hymns after the 2nd and 3rd kathismata, the triodes
// (the canons of three or four odes: Joseph's and Theodore's), the
// idiomelon and martyricon of the aposticha. Units tp.<day>.*:
//
//	matins.sedalen.k2.N, .k2.glory, .k2.theotokion (k3 likewise)
//	canon.<c>.ode.<o>.irmos             — the irmos (often its first words)
//	canon.<c>.ode.<o>.<k>.<kind>        — the places in the book's order,
//	                                      kind troparion | triadikon |
//	                                      theotokion | refrain («Слава
//	                                      Тебе, Боже наш, слава Тебе»,
//	                                      «Благословим Отца», «Хвалим»)
//	canon.<c>.ode.<o>.katavasia          — the irmos sung after the ode
//	matins.aposticha.idiomelon, .martyricon, .theotokion
//
// Not imported: Wednesday and Friday of the 4th week (the Cross's own
// canon) and Thursday of the 5th (the Great Canon).
var triodionLentFiles = []struct {
	file string
	week int
}{
	{"pn_1sedm", 1}, {"vt_1sedm", 1}, {"sr_1sedm", 1}, {"cht_1sedm", 1}, {"pt_1sedm", 1},
	{"2sedm", 2}, {"3sedm", 3}, {"4sedm", 4}, {"5sedm", 5}, {"sedm_vayi", 6},
}

var lentDayNames = []struct{ word, id string }{
	{"понедельник", "mon"}, {"вторник", "tue"}, {"среду", "wed"}, {"четверток", "thu"}, {"пяток", "fri"},
}

// TriodionLentSkip: the days whose matins is not the triodes' order.
var TriodionLentSkip = map[string]bool{"w4_wed": true, "w4_fri": true, "w5_thu": true}

var (
	reTLDay     = regexp.MustCompile(`^во? (понедельник|вторник|среду|четверток|пяток)`)
	reTLRefrain = regexp.MustCompile(`^(слава тебе,? бже наш|блгословим оца|хвалим,? блгословим)`)
	reTLRubric  = regexp.MustCompile(`^(по |зде |таже|сице|и стихослов|стихослов|и поем|поем |подобен|канон|и паки|чтение|зри|и повторя|паки |стих[: ]|ин канон|и абие)`)
	reTLGlory   = regexp.MustCompile(`^(?:слава,?\s*)?тр\S*чен\s*:`)
	reTLTheot   = regexp.MustCompile(`^(?:и ныне,?\s*)?(?:крсто)?(?:бгородичен|бцдичен)`)
	reTLOther   = regexp.MustCompile(`^(иный|ин[.,\s]|другий)`)
	reTLOde     = regexp.MustCompile(`(?:^|\.\s)(?:глас \S+ )?песнь ([а-ѳ])(?:[.,:]|\s|$)`)
)

// ImportTriodionLent parses the weekday matins of the Lenten Triodion.
func ImportTriodionLent(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for _, lf := range triodionLentFiles {
		path := filepath.Join(root, "triodion", "pt", lf.file+".hip")
		d, err := LoadDoc(path)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		// The day segments: a heading «в понедельник …» (not the evening's).
		type seg struct {
			id   string
			from int
		}
		var segs []seg
		for i, p := range d.Paras {
			plain := strings.ToLower(hip.PlainString(CleanText(p.Text())))
			m := reTLDay.FindStringSubmatch(plain)
			if m == nil || len(strings.Fields(plain)) > 12 {
				continue
			}
			if strings.Contains(plain, "вечер") {
				segs = append(segs, seg{"", i}) // the evening: ends the matins
				continue
			}
			for _, dn := range lentDayNames {
				if dn.word == m[1] {
					segs = append(segs, seg{fmt.Sprintf("w%d_%s", lf.week, dn.id), i})
				}
			}
		}
		for si, s := range segs {
			if s.id == "" || TriodionLentSkip[s.id] {
				continue
			}
			to := len(d.Paras)
			if si+1 < len(segs) {
				to = segs[si+1].from
			}
			got, w := parseTriodionLentDay(d, s.from, to, "tp."+s.id, path)
			units = append(units, got...)
			warnings = append(warnings, w...)
		}
	}
	return units, warnings
}

// tlInline: the Unicode text after the heading's colons («и ныне,
// бгородичен. подобен: Треволненьми…»), "" when only cues follow.
func tlInline(text, plain string) string {
	for i := strings.Index(plain, ":"); i > 0; {
		rest := strings.TrimSpace(plain[i+1:])
		if len(strings.Fields(rest)) > 4 && !reTPRubricHead.MatchString(rest) {
			_, pos := hip.Plain(text)
			return uni(text, pos, plain, i+1, len(plain))
		}
		j := strings.Index(plain[i+1:], ":")
		if j < 0 {
			return ""
		}
		i += 1 + j
	}
	return ""
}

var (
	reTLKathisma = regexp.MustCompile(`по ([вг])-\s?м|по (втором|третием)`)
	reTLMartyr   = regexp.MustCompile(`^(?:и |таже,?\s*)?(?:мч|муч)`)
	reTLMartyrL  = regexp.MustCompile(`(?:мч|муч)\S*`)
	reTLTheotL   = regexp.MustCompile(`(?:крсто)?(?:бгородичен|бцдичен)`)
	reTLHours    = regexp.MustCompile(`^(на [агзф]-?\s?м часе|тропарь пророчества|об третием)`)
	reTLTwice    = regexp.MustCompile(`\s*\[?дважды\.?\]?\s*$`)
)

// tlLabelled: the text after a label («бгородичен», «мчнчен»), its tone
// and cue skipped («бгородичен, глас тойже. подобен: …:»); "" when the text
// is in the next paragraph.
func tlLabelled(text, plain string, label *regexp.Regexp) string {
	m := label.FindStringIndex(plain)
	if m == nil {
		return ""
	}
	i := m[1]
	rest := plain[i:]
	for {
		r := strings.TrimLeft(rest, " .,:;")
		switch {
		case strings.HasPrefix(r, "глас"):
			// «глас тойже», «глас д» — up to the next separator.
			j := strings.IndexAny(r, ".:")
			if j < 0 {
				return ""
			}
			r = r[j+1:]
		case strings.HasPrefix(r, "подобен"):
			r = r[len("подобен"):]
		default:
			if len(strings.Fields(r)) < 6 {
				return ""
			}
			_, pos := hip.Plain(text)
			return uni(text, pos, plain, len(plain)-len(r), len(plain))
		}
		rest = r
	}
}

// tlStripTwice drops a closing «дважды» / «[дважды.]»; true when found.
func tlStripTwice(text string) (string, bool) {
	plain := strings.ToLower(hip.PlainString(text))
	m := reTLTwice.FindStringIndex(plain)
	if m == nil || m[0] == 0 {
		return text, false
	}
	_, pos := hip.Plain(text)
	return uni(text, pos, plain, 0, m[0]), true
}

func tlTwice(text string) string { t, _ := tlStripTwice(text); return t }

func parseTriodionLentDay(d *Doc, from, to int, prefix, path string) ([]corpus.Unit, []string) {
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
	var (
		sec        string // k2 | k3 | canon | svetilen | aposticha | done
		expect     string // the key of the next paragraph's text
		expectKind string
		nSed       = map[string]int{}
		tones      = map[int]int{} // the tone of each triode
		ode, c, k  int
		places     = map[string]int{} // canon.c.ode.o → places so far
	)
	sedalen := func(text string) {
		nSed[sec]++
		t, twice := tlStripTwice(text)
		key := fmt.Sprintf("matins.sedalen.%s.%d", sec, nSed[sec])
		add(key, "sedalen", t, 0, nil)
		if twice {
			add(fmt.Sprintf("matins.sedalen.%s.glory", sec), "sedalen", "-", 0, map[string]any{"copy": prefix + "." + key})
		}
	}
	// put: the labelled text here, or the next paragraph's.
	put := func(key, kind, text, plain string, label *regexp.Regexp) {
		if strings.HasSuffix(key, "sedalen."+sec+".theotokion") {
			sec = "" // the section ends with its theotokion
		}
		if t := tlLabelled(text, plain, label); t != "" {
			add(key, kind, t, 0, nil)
		} else {
			expect, expectKind = key, kind
		}
	}
	apostichaStart := func(text, plain string) {
		sec = "aposticha"
		i := strings.Index(plain, "на стиховне")
		_, pos := hip.Plain(text)
		rest := uni(text, pos, plain, i, len(plain))
		rp := strings.ToLower(hip.PlainString(rest))
		if t := tlInline(rest, rp); t != "" {
			add("matins.aposticha.idiomelon", "sticheron", tlTwice(t), toneOf(rest), nil)
		} else {
			expect, expectKind = "matins.aposticha.idiomelon", "sticheron"
		}
	}
	for pi := from; pi < to && sec != "done"; pi++ {
		text := CleanText(pageRef.ReplaceAllString(d.Paras[pi].Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		words := len(strings.Fields(plain))
		if expect != "" {
			if words < 5 || words < 8 && (strings.HasPrefix(plain, "глас") || strings.HasPrefix(plain, "подобен")) {
				// «подобен: Скоро предвари:», «глас а. подобен: Гроб Твой:», «творение Льва царя.»
				continue
			}
			t := text
			if strings.HasPrefix(expect, "matins.aposticha.idiomelon") {
				t = tlTwice(t)
			}
			add(expect, expectKind, t, 0, nil)
			expect = ""
			continue
		}
		if reTLHours.MatchString(plain) || reTLDay.MatchString(plain) && pi > from {
			break
		}
		// The sessional hymns after the 2nd and 3rd kathismata.
		if (sec == "" || sec == "k2" || sec == "k3") && reTLKathisma.MatchString(plain) {
			ms := reTLKathisma.FindAllStringSubmatchIndex(plain, -1)
			m := ms[len(ms)-1]
			sec = "k2"
			if m[2] >= 0 && plain[m[2]:m[3]] == "г" || m[4] >= 0 && plain[m[4]:m[5]] == "третием" {
				sec = "k3"
			}
			_, pos := hip.Plain(text)
			head := uni(text, pos, plain, m[0], len(plain))
			hp := strings.ToLower(hip.PlainString(head))
			if strings.Contains(hp, "седал") {
				if t := tlInline(head, hp); t != "" {
					sedalen(t)
				} else {
					expect, expectKind = fmt.Sprintf("matins.sedalen.%s.%d", sec, nSed[sec]+1), "sedalen"
					nSed[sec]++
				}
			}
			continue
		}
		if sec == "k2" || sec == "k3" {
			switch {
			case strings.HasPrefix(plain, "подобен") && words < 6:
				continue
			case strings.HasPrefix(plain, "слава"):
				head, now, _ := strings.Cut(plain, "и ныне")
				switch {
				case regexp.MustCompile(`^слава,?\s*(?:паки\s*)?тойже`).MatchString(head):
					add(fmt.Sprintf("matins.sedalen.%s.glory", sec), "sedalen", "-", 0,
						map[string]any{"copy": fmt.Sprintf("%s.matins.sedalen.%s.%d", prefix, sec, max(nSed[sec], 1))})
				case now == "" || strings.Contains(plain, "слава, и ныне"):
					if strings.Contains(plain, "слава, и ныне") {
						put(fmt.Sprintf("matins.sedalen.%s.theotokion", sec), "theotokion", text, plain, reTLTheotL)
						continue
					}
					if t := tlInline(text, plain); t != "" {
						add(fmt.Sprintf("matins.sedalen.%s.glory", sec), "sedalen", t, toneOf(text), nil)
					} else {
						expect, expectKind = fmt.Sprintf("matins.sedalen.%s.glory", sec), "sedalen"
					}
					continue
				}
				if strings.Contains(plain, "и ныне") {
					i := strings.Index(plain, "и ныне")
					_, pos := hip.Plain(text)
					rest := uni(text, pos, plain, i, len(plain))
					put(fmt.Sprintf("matins.sedalen.%s.theotokion", sec), "theotokion", rest, strings.ToLower(hip.PlainString(rest)), reTLTheotL)
				}
				continue
			case reTLTheot.MatchString(plain):
				put(fmt.Sprintf("matins.sedalen.%s.theotokion", sec), "theotokion", text, plain, reTLTheotL)
				continue
			case words >= 8 && !reTLRubric.MatchString(plain) && !reTPOde.MatchString(plain):
				sedalen(text)
				continue
			}
		}
		// The canon.
		if strings.HasPrefix(plain, "канон") {
			sec, tones[1] = "canon", toneOf(text)
			if !reTLOde.MatchString(plain) { // «канон минеи и триоди. песнь в. глас в.»
				continue
			}
		}
		if m := reTLOde.FindStringSubmatchIndex(plain); m != nil && (sec == "canon" || sec == "k2" || sec == "k3" || sec == "") {
			_, pos := hip.Plain(text)
			sec, c, k = "canon", 1, 0
			ode = uniNumeral(text, pos, plain, m[2], m[3])
			if t := toneOf(text); t > 0 {
				tones[1] = t
			}
			// «песнь а. ирмос: Тристаты крепкия:», «песнь и. Вавилонская пещь:»
			rest := strings.TrimLeft(plain[m[1]:], " .:")
			off := len(plain) - len(rest)
			if strings.HasPrefix(rest, "ирмос") {
				j := strings.Index(rest, ":")
				rest = strings.TrimLeft(rest[j+1:], " ")
				off = len(plain) - len(rest)
			}
			if rest != "" && !strings.HasPrefix(rest, "глас") {
				add(fmt.Sprintf("canon.1.ode.%d.irmos", ode), "irmos", uni(text, pos, plain, off, len(plain)), tones[1], nil)
			}
			continue
		}
		if sec == "canon" && ode > 0 {
			base := fmt.Sprintf("canon.%d.ode.%d", c, ode)
			place := func(kind, body string) {
				k++
				places[base] = k
				add(fmt.Sprintf("%s.%d.%s", base, k, kind), kind, body, tones[c], nil)
			}
			switch {
			case strings.Contains(plain, "на стиховне"):
				apostichaStart(text, plain)
				continue
			case strings.HasPrefix(plain, "светилен") || strings.HasPrefix(plain, "ексапостиларий") ||
				strings.HasPrefix(plain, "таже") && strings.Contains(plain, "светилен"):
				sec = "svetilen"
				if strings.HasPrefix(plain, "светилен") || strings.HasPrefix(plain, "ексапостиларий") {
					if t := tlInline(text, plain); t != "" {
						add("matins.exapostilarion.1", "exapostilarion", t, 0, nil)
					}
				}
				continue
			case reTLOther.MatchString(plain) && words < 12:
				c, k = c+1, 0
				tones[c] = tones[c-1]
				if t := toneOf(text); t > 0 {
					tones[c] = t
				}
				_, pos := hip.Plain(text)
				if strings.Contains(plain, "ирмос тойже") {
					add(fmt.Sprintf("canon.%d.ode.%d.irmos", c, ode), "irmos", "-", tones[c],
						map[string]any{"copy": fmt.Sprintf("%s.canon.%d.ode.%d.irmos", prefix, c-1, ode)})
				} else if i := strings.Index(plain, "ирмос:"); i > 0 {
					add(fmt.Sprintf("canon.%d.ode.%d.irmos", c, ode), "irmos", uni(text, pos, plain, i+len("ирмос:"), len(plain)), tones[c], nil)
				}
				continue
			case strings.HasPrefix(plain, "катаваси"):
				_, pos := hip.Plain(text)
				if i := strings.Index(plain, ":"); i > 0 {
					add(base+".katavasia", "irmos", uni(text, pos, plain, i+1, len(plain)), tones[c], nil)
				}
				continue
			case strings.HasPrefix(plain, "ирмос тойже") && places[base] > 0:
				// The second triode, its irmos the first's («ирмос тойже.»).
				c, k = c+1, 0
				tones[c] = tones[c-1]
				add(fmt.Sprintf("canon.%d.ode.%d.irmos", c, ode), "irmos", "-", tones[c],
					map[string]any{"copy": fmt.Sprintf("%s.canon.%d.ode.%d.irmos", prefix, c-1, ode)})
				continue
			case strings.HasPrefix(plain, "ирмос"):
				key := base + ".irmos"
				if strings.Contains(plain, "тойже") {
					add(key, "irmos", "-", tones[c], map[string]any{"copy": fmt.Sprintf("%s.canon.%d.ode.%d.irmos", prefix, max(c-1, 1), ode)})
					continue
				}
				_, pos := hip.Plain(text)
				t := uni(text, pos, plain, strings.Index(plain, ":")+1, len(plain))
				if places[base] == 0 && !seen[key] {
					add(key, "irmos", t, tones[c], nil)
				} else {
					add(base+".katavasia", "irmos", t, tones[c], nil)
				}
				continue
			case reTLRefrain.MatchString(plain):
				place("refrain", text)
				continue
			case reTLGlory.MatchString(plain):
				if t := tlInline(text, plain); t != "" {
					place("triadikon", t)
				}
				continue
			case reTLTheot.MatchString(plain):
				if t := tlLabelled(text, plain, reTLTheotL); t != "" {
					place("theotokion", t)
				}
				continue
			case words >= 6 && !reTLRubric.MatchString(plain):
				place("troparion", text)
				continue
			}
			continue
		}
		if sec == "svetilen" && strings.Contains(plain, "на стиховне") {
			apostichaStart(text, plain)
			continue
		}
		if sec == "aposticha" {
			switch {
			case strings.HasPrefix(plain, "стих") || strings.HasPrefix(plain, "паки") || strings.HasPrefix(plain, "и повторя") ||
				strings.HasPrefix(plain, "подобен") && words < 6:
			case strings.HasPrefix(plain, "слава") && !strings.Contains(plain, "и ныне"):
				// «Слава» an idiomelon of its own (the 6th week's Friday).
				if t := tlInline(text, plain); t != "" {
					add("matins.aposticha.glory", "sticheron", t, toneOf(text), nil)
				} else {
					expect, expectKind = "matins.aposticha.glory", "sticheron"
				}
			case strings.HasPrefix(plain, "и ныне") && strings.Contains(plain, "тойже") && seen["matins.aposticha.glory"]:
				add("matins.aposticha.theotokion", "sticheron", "-", 0, map[string]any{"copy": prefix + ".matins.aposticha.glory"})
			case strings.HasPrefix(plain, "слава") || strings.HasPrefix(plain, "и ныне"):
				if t := tlLabelled(text, plain, reTLTheotL); t != "" {
					add("matins.aposticha.theotokion", "theotokion", t, 0, nil)
				} else if t := tlInline(text, plain); t != "" { // «слава, и ныне, глас тойже: Архангельское слово…»
					add("matins.aposticha.theotokion", "theotokion", t, 0, nil)
				} else {
					expect, expectKind = "matins.aposticha.theotokion", "theotokion"
				}
			case reTLMartyr.MatchString(plain):
				put("matins.aposticha.martyricon", "sticheron", text, plain, reTLMartyrL)
			case !seen["matins.aposticha.idiomelon"] && words >= 8:
				add("matins.aposticha.idiomelon", "sticheron", tlTwice(text), 0, nil)
			}
			if seen["matins.aposticha.theotokion"] {
				sec = "done"
			}
		}
	}
	// A sessional hymn and its theotokion printed without the label («по в-м
	// стихословии седальны…»: the hymn, then «Безневестная Мати Твоя…»).
	for _, s := range []string{"k2", "k3"} {
		th := fmt.Sprintf("matins.sedalen.%s.theotokion", s)
		last := fmt.Sprintf("matins.sedalen.%s.%d", s, nSed[s])
		if !seen[th] && nSed[s] >= 2 && !seen[fmt.Sprintf("matins.sedalen.%s.glory", s)] {
			for i := range units {
				if units[i].ID == prefix+"."+last {
					units[i].ID, units[i].Kind = prefix+"."+th, "theotokion"
					seen[th] = true
				}
			}
		}
	}
	var warnings []string
	day := strings.TrimPrefix(prefix, "tp.")
	for _, key := range []string{"matins.sedalen.k2.1", "matins.sedalen.k2.theotokion", "matins.sedalen.k3.1",
		"matins.sedalen.k3.theotokion", "matins.aposticha.idiomelon", "matins.aposticha.martyricon", "matins.aposticha.theotokion"} {
		if !seen[key] {
			warnings = append(warnings, fmt.Sprintf("%s (%s): no %s", day, filepath.Base(path), key))
		}
	}
	var odes []string
	for b := range places {
		odes = append(odes, b)
	}
	sort.Strings(odes)
	if len(odes) < 3 {
		warnings = append(warnings, fmt.Sprintf("%s (%s): triodes %v", day, filepath.Base(path), odes))
	}
	for _, b := range odes {
		if !seen[b+".katavasia"] && !seen[b+".irmos"] {
			warnings = append(warnings, fmt.Sprintf("%s: %s without an irmos", day, b))
		}
	}
	return units, warnings
}
