package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The magnifications with the selected psalms (Ирмологий, «Псалми
// избраннии», 23izbrpsalm.hip): «сии избранныя от псалмов стихи поются по
// полиелеи во владычныя праздники, и во вся святыя имущыя полиелей». Keys
// ir.magnification.<key>:
//
//	<key>          the magnification («начинает первый лик, поя»)
//	<key>.2        another one (the bodiless powers)
//	<key>.verse.N  the verses of the selected psalm, sung by the two liks
//	               in turn (а҃, в҃ in the book)
//	alleluia       «Слава, и ныне: Аллилуиа, аллилуиа, аллилуиа, слава
//	               Тебе, Боже» (трижды), the end of each
//
// <key> is the Menaion day (MM.DD) of a fixed feast, palm, thomas,
// ascension, pentecost, or common.<apostle|martyr|hierarch|venerable|
// bodiless>.
var magnificationHeadings = []struct{ heading, key string }{
	{"на ржство бцды", "09.08"},
	{"в воздвижение", "09.14"},
	{"на вход бцды", "11.21"},
	{"на ржство гдса", "12.25"},
	{"на стое бгоявление", "01.06"},
	{"на сретение", "02.02"},
	{"на блговещение", "03.25"},
	{"в неделю цветную", "palm"},
	{"в новую неделю", "thomas"},
	{"на вознесение", "ascension"},
	{"в неделю и-ю", "pentecost"},
	{"на ржство прдтечево", "06.24"},
	{"на преображение", "08.06"},
	{"на успение", "08.15"},
	{"на усекновение", "08.29"},
	{"стым апслом петру", "06.29"},
	{"общий апслом", "common.apostle"},
	{"на памяти стых мчник", "common.martyr"},
	{"стителем общий", "common.hierarch"},
	{"на памяти прпдбных", "common.venerable"},
	{"стым безплотным", "common.bodiless"},
}

var (
	reMagRubric  = regexp.MustCompile(`^(\[|начинает|первый лик|таже|паки|тойже лик|ведомо|в праздники|и во дни|псалми избраннии|общий апслом)`)
	reMagMarker  = regexp.MustCompile(`(?:^|\s)[ав]҃\s+`)
	reMagCommons = []struct {
		re  *regexp.Regexp
		key string
	}{
		{regexp.MustCompile(`прпдбн`), "venerable"},
		{regexp.MustCompile(`стителю|стителие`), "hierarch"},
		{regexp.MustCompile(`стрстотерп|страстотерп|мчнк|мчнц|мученик|великомчн`), "martyr"},
		{regexp.MustCompile(`апсле|апсли|апостол`), "apostle"},
		{regexp.MustCompile(`аггл`), "bodiless"},
		// A feast of an icon of the Theotokos: the selected psalm of her Nativity.
		{regexp.MustCompile(`(^|\s)(дво|бце|бгородице|мти|влдчце)[\s,]`), "09.08"},
	}
	// The commons of the Irmologion by the rank of the saint (the General
	// Menaion's categories); the feminine and the plural ranks have none.
	magnificationOfRank = map[string]string{"monk": "venerable", "hierarch": "hierarch", "martyr": "martyr",
		"hieromartyr": "martyr", "monk_martyr": "martyr", "apostle": "apostle", "angels": "bodiless"}
	reMagAnchor = regexp.MustCompile(`^(нашего|о?т?ца|оца|апсла|апостола|стителя|святителя|прпдбнаго|преподобнаго|.*мученика|.*мчника|архистратига|князя|кнзя)$`)
	reMagPlural = regexp.MustCompile(`наших|отец|отцев|мчников|мучеников|мчниц|мучениц|апслов|апостолов|стителей|прпдбных|преподобных`)
	reMagName   = regexp.MustCompile(`,? ?\[[^\]]*\]`)
)

// ImportMagnifications parses the magnifications and the selected psalms.
func ImportMagnifications(root string) ([]corpus.Unit, error) {
	path := filepath.Join(root, "irmologij", "irmologij", "23izbrpsalm.hip")
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	var units []corpus.Unit
	seen := map[string]bool{}
	add := func(id, kind, body string) {
		if seen[id] || body == "" {
			return
		}
		seen[id] = true
		units = append(units, corpus.Unit{ID: id, Book: "irmologion", Kind: kind, Lang: "cu", Body: body,
			SourceID: "orthlib.irmologion", Meta: map[string]any{"file": filepath.Base(path)}})
	}
	key, alias, mags, alleluia := "", "", 0, false
	verses := func(k, text string) {
		parts := reMagMarker.Split(text, -1)
		n := 0
		for _, v := range parts {
			if v = strings.TrimSpace(v); v != "" {
				n++
				add(fmt.Sprintf("ir.magnification.%s.verse.%d", k, n), "verse", v)
			}
		}
	}
	for _, p := range d.Paras {
		text := CleanText(p.Text())
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		switched := false
		for _, h := range magnificationHeadings {
			if strings.HasPrefix(plain, h.heading) {
				if h.key == "common.apostle" && key == "06.29" {
					alias = key // Ss Peter and Paul: the apostles' verses
				} else {
					alias = ""
				}
				key, mags, switched = h.key, 0, true
				break
			}
		}
		if h := strings.Index(plain, "общий апслом"); switched && h > 0 || key == "" {
			continue
		}
		switch {
		case strings.HasPrefix(plain, "слава, и ныне"):
			alleluia = !strings.Contains(plain, "аллилуиа")
			if alleluia || seen["ir.magnification.alleluia"] {
				continue
			}
			fallthrough
		case alleluia:
			alleluia = false
			if i := strings.Index(plain, "аллилуиа"); i >= 0 {
				_, pos := hip.Plain(text)
				end := len(plain)
				if j := strings.Index(plain, "трижды"); j > i {
					end = j
				}
				add("ir.magnification.alleluia", "refrain", uni(text, pos, plain, i, end))
			}
		case switched, reMagRubric.MatchString(plain):
		case reMagMarker.MatchString(text):
			verses(key, text)
			if alias != "" {
				verses(alias, text)
			}
		default:
			mags++
			id := "ir.magnification." + key
			if mags > 1 {
				id = fmt.Sprintf("%s.%d", id, mags)
			}
			add(id, "magnification", text)
		}
	}
	return units, nil
}

// SupplementMagnifications gives the Menaion feasts the magnification and
// the selected psalm of the Irmologion (the Menaion prints neither for the
// great feasts), and the Menaion's own magnifications of the saints the
// verses of the common selected psalm (of the venerable, the hierarchs,
// the martyrs, the apostles, the bodiless powers), when it has none:
// me.<day>.matins.magnification, me.<day>.matins.magnification.verse.N.
func SupplementMagnifications(units []corpus.Unit) []corpus.Unit {
	ir := map[string]corpus.Unit{}
	have := map[string]bool{}
	days := map[string]bool{}    // me.MM.DD[suffix]
	var own, ranks []corpus.Unit // the Menaion's magnifications, the ranks of its saints
	titles := map[string]string{}
	for _, u := range units {
		switch {
		case strings.HasPrefix(u.ID, "ir.magnification."):
			ir[u.ID] = u
		case strings.HasPrefix(u.ID, "me."):
			have[u.ID] = true
			parts := strings.SplitN(u.ID, ".", 4)
			if len(parts) == 4 {
				days[strings.Join(parts[:3], ".")] = true
				switch parts[3] {
				case "matins.magnification":
					own = append(own, u)
				case "common.rank":
					ranks = append(ranks, u)
				case "title":
					titles[strings.Join(parts[:3], ".")] = u.Body
				}
			}
		}
	}
	copyUnit := func(id, from string) { // the Irmologion's text, under the day's key
		u := ir[from]
		units = append(units, corpus.Unit{ID: id, Book: u.Book, Kind: u.Kind, Lang: u.Lang, Body: u.Body,
			SourceID: u.SourceID, Meta: map[string]any{"from": from}})
		have[id] = true
	}
	copyVerses := func(day, key string) {
		if have[day+".matins.magnification.verse.1"] {
			return
		}
		for n := 1; ; n++ {
			from := fmt.Sprintf("ir.magnification.%s.verse.%d", key, n)
			if _, ok := ir[from]; !ok {
				return
			}
			copyUnit(fmt.Sprintf("%s.matins.magnification.verse.%d", day, n), from)
		}
	}
	for _, h := range magnificationHeadings {
		day := "me." + h.key
		if !days[day] || strings.Contains(h.key, "common") {
			continue
		}
		if _, ok := ir["ir.magnification."+h.key]; ok && !have[day+".matins.magnification"] {
			copyUnit(day+".matins.magnification", "ir.magnification."+h.key)
		}
		copyVerses(day, h.key)
	}
	// The saints with the sign the Menaion gives no magnification: the
	// common one of their rank, with the name.
	for _, u := range ranks {
		day := strings.TrimSuffix(u.ID, ".common.rank")
		common := magnificationOfRank[u.Body]
		m, ok := ir["ir.magnification.common."+common]
		if !ok || have[day+".matins.magnification"] {
			continue
		}
		body := m.Body
		if common != "bodiless" {
			name, ok := saintName(titles[day], u.Body)
			if !ok || name == "" {
				continue // no saint of the rank, or none named: not guessed
			}
			body = reMagName.ReplaceAllString(body, " "+name+",")
		}
		units = append(units, corpus.Unit{ID: day + ".matins.magnification", Book: m.Book, Kind: m.Kind, Lang: m.Lang,
			Body: body, SourceID: m.SourceID, Meta: map[string]any{"from": m.ID}})
		have[day+".matins.magnification"] = true
		copyVerses(day, "common."+common)
	}
	for _, u := range own {
		day := strings.TrimSuffix(u.ID, ".matins.magnification")
		plain := strings.ToLower(hip.PlainString(u.Body))
		for _, c := range reMagCommons {
			if c.re.MatchString(plain) {
				key := c.key
				if !strings.Contains(key, ".") {
					key = "common." + key
				}
				copyVerses(day, key)
				break
			}
		}
	}
	return units
}

// saintName: the name, in the vocative, of the saint of the rank in a
// Menaion day's title («…ѻ҆ц҃а̀ на́шегѡ нїкола́а, а҆рхїепі́скопа…» →
// «нїкола́е»): the commemoration naming the rank (the last; the only one),
// the word after «отца нашего» or the rank, past the epithets. ok is false
// when the title names no such saint, or several («отец наших», «мучеников»):
// the common magnification is not theirs. name is "" when it is not found
// (the magnification is then not given).
func saintName(title, rank string) (name string, ok bool) {
	segs := strings.FieldsFunc(title, func(r rune) bool { return r == ':' || r == ';' })
	seg := ""
	for i := len(segs) - 1; i >= 0 && seg == ""; i-- {
		if rankOfSegment(strings.ToLower(hip.PlainString(segs[i]))) == rank {
			seg = segs[i]
		}
	}
	if seg == "" && len(segs) == 1 {
		seg = segs[0]
	}
	if seg == "" || reMagPlural.MatchString(strings.ToLower(hip.PlainString(seg))) {
		return "", false
	}
	words := strings.Fields(seg)
	for j, w := range words {
		pw := strings.Trim(strings.ToLower(hip.PlainString(w)), ",.")
		if !reMagAnchor.MatchString(pw) {
			continue
		}
		for _, n := range words[j+1:] {
			pn := strings.Trim(strings.ToLower(hip.PlainString(n)), ",.")
			if reMagAnchor.MatchString(pn) || magEpithet(pn) {
				continue
			}
			return vocative(strings.Trim(n, ",.:")), true
		}
		break
	}
	return "", true
}

func magEpithet(w string) bool {
	switch w {
	case "и", "игумена", "архидиакона", "первомученика", "священника", "диакона", "исповедника", "бгоносного", "богоносного",
		"наперсника", "девственника", "благоверного", "блговернаго", "князя", "кнзя":
		return true
	}
	if strings.Contains(w, "глист") || strings.Contains(w, "евангелист") {
		return true
	}
	return strings.HasSuffix(w, "аго") || strings.HasSuffix(w, "ого") || strings.HasSuffix(w, "яго") || strings.HasSuffix(w, "ца")
}

// vocative: the vocative of a masculine name in the genitive («нїкола́а» →
// «нїкола́е», «са́ввы» → «са́вво», «лꙋкѝ» → «лꙋко̀», «петра̀» → «петрѐ»);
// "" for another ending.
func vocative(gen string) string {
	rs := []rune(gen)
	i := len(rs) - 1
	for i > 0 && unicode.Is(unicode.Mn, rs[i]) {
		i--
	}
	if i < 1 {
		return ""
	}
	switch rs[i] {
	case 'а':
		rs[i] = 'е'
	case 'ы':
		rs[i] = 'о'
	case 'и', 'і', 'ѝ': // «лꙋкѝ»: ѝ precomposed or not
		switch rs[i-1] {
		case 'к', 'г', 'х':
			if rs[i] == 'ѝ' {
				return string(rs[:i]) + "о\u0300" + string(rs[i+1:])
			}
			rs[i] = 'о'
		default:
			return ""
		}
	default:
		return ""
	}
	return string(rs)
}
