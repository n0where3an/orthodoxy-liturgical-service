package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The General Menaion (Минея общая, М., 2002): common services by the rank
// of the saint, with their readings and prokeimena. Keys om.<category>.*,
// built like a day of the Menaion (troparion, kontakion, vespers, gvespers,
// matins, lit).
var generalMenaionChapters = []struct{ File, Category string }{
	{"01", "lord_feast"}, {"02", "theotokos_feast"}, {"03", "cross"}, {"04", "angels"}, {"05", "forerunner"},
	{"06", "fathers"}, {"07", "prophet"}, {"08", "apostle"}, {"09", "apostles"}, {"10", "hierarch"},
	{"11", "hierarchs"}, {"12", "monk"}, {"13", "monks"}, {"14", "martyr"}, {"15", "martyrs"},
	{"16", "hieromartyr"}, {"17", "hieromartyrs"}, {"18", "monk_martyr"}, {"19", "monk_martyrs"},
	{"20", "woman_martyr"}, {"21", "women_martyrs"}, {"22", "nun"}, {"23", "nuns"}, {"24", "nun_martyr"},
	{"25", "confessor"}, {"26", "unmercenaries"}, {"27", "fools"},
}

// ImportGeneralMenaion parses the common services of the General Menaion.
func ImportGeneralMenaion(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for _, ch := range generalMenaionChapters {
		us, err := parseServiceFile(filepath.Join(root, "obschminea", "om", ch.File+".hip"), "om."+ch.Category, ch.Category)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		for i := range us {
			us[i].Book, us[i].SourceID = "general_menaion", "orthlib.obschminea"
		}
		units = append(units, us...)
	}
	par, err := parseCommonParemias(filepath.Join(root, "obschminea", "om", "39.hip"))
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	lit, err := parseCommonLiturgy(filepath.Join(root, "obschminea", "om", "40.hip"))
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	// The rubrics of ch. 40 replace whatever the services themselves cite.
	have := map[string]bool{}
	for _, u := range lit {
		have[u.ID] = true
	}
	kept := units[:0]
	for _, u := range units {
		if !have[u.ID] {
			kept = append(kept, u)
		}
	}
	units = append(append(kept, par...), lit...)
	for i := range units {
		units[i].Book, units[i].SourceID = "general_menaion", "orthlib.obschminea"
	}
	return units, warnings
}

// Ch. 39 «Паремии общыя коемуждо святому»: three readings for each rank;
// the other ranks take another's («что и мученикам»).
var (
	reOMParemiasHead = regexp.MustCompile(`^(.+?) паремии[.,]`)
	reOMReading      = regexp.MustCompile(`^[абвг][)\]]`)
	reOMLetter       = regexp.MustCompile(`^\s*[абвг]҃?\s*[)\]]\s*`)
	reOMBracket      = regexp.MustCompile(`\s*\[[^\]]*\]\.?`)
)

var commonParemiaSets = map[string]string{
	"пророку": "prophet", "апслу единому": "apostle", "апслом": "apostles", "стителю": "hierarch",
	"стителем": "hierarchs", "мчнком": "martyrs",
}

// The category → the set of ch. 39 it reads.
var commonParemiaOf = map[string]string{
	"prophet": "prophet", "monk": "prophet", "monks": "prophet", "monk_martyr": "prophet", "monk_martyrs": "prophet",
	"confessor": "prophet", "fools": "prophet", "nun": "prophet", "nuns": "prophet",
	"apostle": "apostle", "apostles": "apostles", "hierarch": "hierarch", "hierarchs": "hierarchs",
	"martyr": "martyr", "martyrs": "martyrs", "hieromartyr": "martyrs", "hieromartyrs": "martyrs",
	"woman_martyr": "martyrs", "women_martyrs": "martyrs", "nun_martyr": "martyrs", "unmercenaries": "martyrs",
	"elijah": "elijah",
}

// The days whose own paremias are in ch. 39.
var commonParemiasOfDay = map[string]string{"me.07.20": "elijah"}

type commonReading struct{ Title, Body string }

func parseCommonParemias(path string) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	sets := map[string][]commonReading{}
	set := ""
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		switch {
		case plain == "":
		case strings.HasPrefix(plain, "паремии пророку илии"):
			set = "elijah" // of one saint, July 20; its titles have no letters
		case strings.HasPrefix(plain, "а мчнку г-я"):
			// The martyr's third is the hierarchs' third.
			set = "martyr"
			sets[set] = append([]commonReading(nil), sets["martyrs"][:2]...)
		case set == "martyr":
			sets[set] = append(sets[set], sets["hierarchs"][2])
			set = ""
		case reOMParemiasHead.MatchString(plain) && len([]rune(plain)) < 40:
			name := reOMParemiasHead.FindStringSubmatch(plain)[1]
			set = commonParemiaSets[name]
		case set == "":
		case reOMReading.MatchString(plain) || set == "elijah" && strings.HasSuffix(reOMBracket.ReplaceAllString(plain, ""), "чтение"):
			title := strings.TrimSpace(reOMBracket.ReplaceAllString(reOMLetter.ReplaceAllString(text, ""), ""))
			title = strings.TrimRight(title, ".:") + "."
			if strings.Contains(plain, "писана стителю") { // the hierarch's first
				sets[set] = append(sets[set], sets["hierarch"][0])
				continue
			}
			sets[set] = append(sets[set], commonReading{Title: title})
		default:
			if r := sets[set]; len(r) > 0 {
				if r[len(r)-1].Body != "" {
					r[len(r)-1].Body += "\n"
				}
				r[len(r)-1].Body += text
			}
		}
	}
	var units []corpus.Unit
	cats := append([]struct{ File, Category string }(nil), generalMenaionChapters...)
	cats = append(cats, struct{ File, Category string }{"", "elijah"})
	for _, ch := range cats {
		set := commonParemiaOf[ch.Category]
		if set == "" {
			continue
		}
		if len(sets[set]) != 3 {
			return units, fmt.Errorf("%s: %d paremias of %q", path, len(sets[set]), set)
		}
		for i, r := range sets[set] {
			id := fmt.Sprintf("om.%s.gvespers.paremia.%d", ch.Category, i+1)
			meta := map[string]any{"set": set, "file": filepath.Base(path)}
			units = append(units,
				corpus.Unit{ID: id + ".title", Kind: "reading_title", Lang: "cu", Body: r.Title, Meta: meta},
				corpus.Unit{ID: id, Kind: "reading", Lang: "cu", Body: CleanText(r.Body), Meta: meta})
		}
	}
	return units, nil
}

// Ch. 40 «Прокимени, апостоли, аллилуиаре, евангелия и причастны общи
// святым»: per «Служба …» the prokeimenon, the Apostle, the alleluia, the
// Gospel and the communion verse, read as a Menaion liturgy rubric.
var commonLiturgyBlocks = map[string][]string{
	"служба престей бцде":                        {"theotokos_feast"},
	"служба стым нбсным безплотным силам":        {"angels"},
	"служба стым пророком обща":                  {"prophet"},
	"служба стым апслом обща":                    {"apostle", "apostles"},
	"служба стителю":                             {"hierarch"},
	"служба стителей обща":                       {"hierarchs"},
	"служба прпдбных и хрста ради юродивых обща": {"monk", "monks", "fools"},
	"служба мчнка":                               {"martyr"},
	"служба мчнков обща":                         {"martyrs"},
	"служба сщенномчнка":                         {"hieromartyr"},
	"служба сщенномчнков обща":                   {"hieromartyrs"},
	"служба прпдбномчнков обща":                  {"monk_martyr", "monk_martyrs"},
	"служба стых мчнц обща":                      {"woman_martyr", "women_martyrs", "nun_martyr"},
	"служба прпдбных жен обща":                   {"nun", "nuns"},
	"служба исповедников":                        {"confessor"},
	"служба безмездников":                        {"unmercenaries"},
}

// What a block lacks and takes from another («писаны мученицам»), and the
// readings of one of the ranks a block shares («Евангелие апостолу единому
// …, апостолом обще …»).
var (
	commonLiturgyFrom = map[string]string{"nun": "women_martyrs", "nuns": "women_martyrs",
		"confessor": "monk_martyrs", "unmercenaries": "martyrs"}
	commonLiturgyFix = map[string]map[string]any{
		"apostle":      {"gospel_book": "mt", "gospel": 34, "gospel_half": false},
		"monk_martyr":  {"gospel_book": "mk", "gospel": 37, "gospel_half": false},
		"monk_martyrs": {"apostle": 99, "apostle_half": false, "gospel_book": "mt", "gospel": 38, "gospel_half": false},
	}
)

func parseCommonLiturgy(path string) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	type block struct{ cats, paras []string }
	var blocks []*block
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.TrimRight(strings.ToLower(hip.PlainString(text)), ".")
		if plain == "" || strings.HasPrefix(plain, "(л") {
			continue
		}
		if strings.HasPrefix(plain, "служба ") {
			cats := commonLiturgyBlocks[plain]
			if cats == nil {
				return nil, fmt.Errorf("%s: unknown block %q", path, plain)
			}
			blocks = append(blocks, &block{cats: cats})
			continue
		}
		if len(blocks) > 0 {
			blocks[len(blocks)-1].paras = append(blocks[len(blocks)-1].paras, text)
		}
	}
	metas := map[string]map[string]map[string]any{} // category → lit|matins → meta
	for _, b := range blocks {
		parts := map[string]string{"lit": strings.Join(b.paras, " ")}
		// «На утрени прокимен … на литургии прокимен …»
		lp := strings.ToLower(hip.PlainString(parts["lit"]))
		if strings.HasPrefix(lp, "на утрени") {
			for i, t := range b.paras {
				if strings.HasPrefix(strings.ToLower(hip.PlainString(t)), "на литургии") {
					parts["matins"] = strings.Join(b.paras[:i], " ")
					parts["lit"] = strings.Join(b.paras[i:], " ")
				}
			}
		}
		for service, text := range parts {
			_, pos := hip.Plain(text)
			plain := strings.ToLower(hip.PlainString(text))
			var meta map[string]any
			collect := func(_, _, _ string, m map[string]any) { meta = m }
			if service == "matins" {
				parseMatinsRubric(text, plain, func(_, _, _, _ string, m map[string]any) { meta = m })
			} else {
				parseLiturgyRubric(text, plain, pos, collect)
			}
			for _, c := range b.cats {
				if metas[c] == nil {
					metas[c] = map[string]map[string]any{}
				}
				m := map[string]any{"text": text}
				for k, v := range meta {
					m[k] = v
				}
				for k, v := range commonLiturgyFix[c] {
					if service == "lit" {
						m[k] = v
					}
				}
				metas[c][service] = m
			}
		}
	}
	var units []corpus.Unit
	for _, ch := range generalMenaionChapters {
		for _, service := range []string{"matins", "lit"} {
			m := metas[ch.Category][service]
			if m == nil {
				continue
			}
			if from := commonLiturgyFrom[ch.Category]; from != "" && service == "lit" {
				for k, v := range metas[from]["lit"] {
					if _, ok := m[k]; !ok && k != "text" {
						m[k] = v
					}
				}
			}
			m["file"] = filepath.Base(path)
			text, _ := m["text"].(string)
			delete(m, "text")
			units = append(units, corpus.Unit{ID: fmt.Sprintf("om.%s.%s.rubric", ch.Category, service), Kind: "marker",
				Lang: "cu", Body: text, Meta: m})
		}
	}
	return units, nil
}

// References of a Menaion day to the General Menaion: «на литургии служба
// преподобническая», «чтения три стительская». Markers under
// me.<MM>.<DD>.common: lit, paremias (Meta: category).
var (
	reCommonLit      = regexp.MustCompile(`на литургии,?:? (?:прочая )?(?:вся )?служба (?:вся )?([^.,:]+)`)
	reCommonParemias = regexp.MustCompile(`чтения (?:три|г:?)?,? ?([^.,:\[(]+)`)
	reCommonGospel   = regexp.MustCompile(`еиглие,? (\S+?ское)[.,:]`)
)

// commonRank: the category of a rank named in a reference; "" for a
// reference elsewhere («служба праздника», «сего месяца»).
func commonRank(words string) string {
	w := strings.TrimPrefix(strings.TrimSpace(words), "вся ")
	switch {
	case strings.Contains(w, "праздника") || strings.Contains(w, "дне") || strings.Contains(w, "месяца") ||
		strings.Contains(w, "мцса") || strings.Contains(w, "писан"):
		return ""
	case strings.Contains(w, "прпдбней жене"):
		return "nun"
	case strings.HasPrefix(w, "прпдбническ"), strings.HasPrefix(w, "преподобническ"), strings.HasPrefix(w, "прпдбнаго"),
		strings.HasPrefix(w, "прпдбнническ"),
		strings.HasPrefix(w, "прпдбных"):
		return "monk"
	case strings.HasPrefix(w, "стителем"), strings.HasPrefix(w, "иерархов"):
		return "hierarchs"
	case strings.HasPrefix(w, "стительск"), strings.HasPrefix(w, "стителю"):
		return "hierarch"
	case strings.HasPrefix(w, "мученическ"), strings.HasPrefix(w, "мчническ"):
		return "martyr"
	case strings.HasPrefix(w, "бцд"):
		return "theotokos_feast"
	}
	return ""
}

func parseCommonRefs(d *Doc, prefix, path string) []corpus.Unit {
	var units []corpus.Unit
	seen := map[string]bool{}
	for _, p := range d.Paras {
		plain := strings.ToLower(hip.PlainString(CleanText(pageRef.ReplaceAllString(p.Text(), ""))))
		for key, re := range map[string]*regexp.Regexp{"lit": reCommonLit, "paremias": reCommonParemias, "matins_gospel": reCommonGospel} {
			m := re.FindStringSubmatch(plain)
			if m == nil || seen[key] {
				continue
			}
			// «…всякое дыхание: евангелие святительское» (of matins).
			if key == "matins_gospel" && (strings.HasPrefix(plain, "на литургии") || !strings.Contains(plain, "прокимен")) {
				continue
			}
			if cat := commonRank(m[1]); cat != "" {
				seen[key] = true
				units = append(units, corpus.Unit{ID: prefix + ".common." + key, Book: "menaion", Kind: "marker", Lang: "cu",
					Body: m[0], SourceID: "orthlib.mineya", Meta: map[string]any{"category": cat, "file": filepath.Base(path)}})
			}
		}
	}
	return units
}

// The parts a service with a sign takes from the General Menaion when its
// own Menaion day lacks them.
var commonLitParts = []string{"prokeimenon", "prokeimenon.verse", "apostle", "apostle.announce", "alleluia.verse.1",
	"alleluia.verse.2", "gospel", "gospel.announce", "kinonik"}

// SupplementFromGeneralMenaion: what a day of the Menaion lacks is taken
// from the General Menaion, by the rank of the saint:
//
//   - every day: the troparion and the kontakion (not where the Menaion
//     says «тропарь праздника»: a forefeast or afterfeast), the sessional
//     hymn after the 3rd ode;
//   - a service with a sign (great vespers): the paremias, the aposticha,
//     the prokeimenon of matins;
//   - the readings of the liturgy and the matins Gospel where the day refers
//     to the General Menaion.
//
// The rank comes from the day's reference to the General Menaion, else from
// its own Apostle and Gospel, else from its title. Run it after
// ResolveLiturgy.
func SupplementFromGeneralMenaion(units []corpus.Unit) ([]corpus.Unit, []string) {
	byID := map[string]int{}
	for i, u := range units {
		byID[u.ID] = i
	}
	type day struct {
		gvespers, paremias, matinsProk, matinsGospel, litProk    bool
		troparion, kontakion, feastTroparion, sedalen, aposticha bool
		ref, refParemias, refGospel, title                       string
		lit                                                      map[string]any
	}
	days := map[string]*day{}
	var order []string
	for _, u := range units {
		if !strings.HasPrefix(u.ID, "me.") {
			continue
		}
		parts := strings.SplitN(u.ID, ".", 4) // me MM DD rest
		if len(parts) < 4 {
			continue
		}
		p := strings.Join(parts[:3], ".")
		dd := days[p]
		if dd == nil {
			dd = &day{}
			days[p] = dd
			order = append(order, p)
		}
		rest := parts[3]
		switch {
		case rest == "troparion.1":
			dd.troparion = true
		case rest == "kontakion.1":
			dd.kontakion = true
		case rest == "feast_troparion":
			dd.feastTroparion = true
		case strings.HasPrefix(rest, "matins.sedalen."):
			dd.sedalen = true
		case strings.HasPrefix(rest, "gvespers.paremia."):
			dd.paremias = true
		case strings.HasPrefix(rest, "gvespers.lic."):
			dd.gvespers = true
		case strings.HasPrefix(rest, "gvespers.aposticha."):
			dd.gvespers, dd.aposticha = true, true
		case rest == "matins.prokeimenon":
			dd.matinsProk = true
		case rest == "matins.gospel":
			dd.matinsGospel = true
		case rest == "common.matins_gospel":
			dd.refGospel, _ = u.Meta["category"].(string)
		case rest == "lit.prokeimenon":
			dd.litProk = true
		case rest == "lit.rubric":
			dd.lit = u.Meta
		case rest == "common.lit":
			dd.ref, _ = u.Meta["category"].(string)
		case rest == "common.paremias":
			dd.refParemias, _ = u.Meta["category"].(string)
		}
		if t, _ := u.Meta["title"].(string); t != "" && dd.title == "" {
			dd.title = strings.ToLower(hip.PlainString(t))
		}
	}
	// The ranks by their Apostle and Gospel of the liturgy.
	type reading struct {
		ap   int
		book string
		gos  int
	}
	byReading := map[reading]string{}
	for _, ch := range generalMenaionChapters {
		j, ok := byID["om."+ch.Category+".lit.rubric"]
		if !ok {
			continue
		}
		m := units[j].Meta
		ap, _ := m["apostle"].(int)
		gos, _ := m["gospel"].(int)
		book, _ := m["gospel_book"].(string)
		r := reading{ap, book, gos}
		if _, dup := byReading[r]; !dup {
			byReading[r] = ch.Category
		}
	}
	var add []corpus.Unit
	var warnings []string
	copyUnit := func(id, src string) bool {
		j, ok := byID[src]
		if !ok {
			return false
		}
		c := units[j]
		c.ID, c.Book, c.SourceID = id, "menaion", "orthlib.obschminea"
		c.Meta = map[string]any{"copied_from": src}
		add = append(add, c)
		return true
	}
	guessed := false
	copyGuess := func(id, src string) {
		if copyUnit(id, src) && guessed {
			add[len(add)-1].Meta["rank_from_title"] = true
		}
	}
	// The General Menaion's units by their parent («om.martyr.matins.sedalen»).
	omParts := map[string][]string{}
	for _, u := range units {
		if strings.HasPrefix(u.ID, "om.") {
			if k := strings.LastIndex(u.ID, "."); k > 0 {
				omParts[u.ID[:k]] = append(omParts[u.ID[:k]], u.ID)
			}
		}
	}
	// copyAll copies the units under om.<cat>.<part> (and one level deeper:
	// gvespers.aposticha.sticheron.N) into the day.
	copyAll := func(p, cat, part string) {
		src := "om." + cat + "." + part
		for parent, ids := range omParts {
			if parent != src && !strings.HasPrefix(parent, src+".") {
				continue
			}
			for _, id := range ids {
				copyGuess(p+strings.TrimPrefix(id, "om."+cat), id)
			}
		}
	}
	for _, p := range order {
		dd := days[p]
		cat := dd.ref
		if cat == "" && dd.lit != nil {
			ap, _ := dd.lit["apostle"].(int)
			gos, _ := dd.lit["gospel"].(int)
			book, _ := dd.lit["gospel_book"].(string)
			cat = byReading[reading{ap, book, gos}]
		}
		guessed = false
		// A feast of the Lord or of the Theotokos has its own order.
		feastDay := strings.Contains(dd.title, "бцды") || strings.Contains(dd.title, "гдса нашего")
		if cat == "" && !feastDay {
			cat, guessed = rankOfTitle(dd.title), true
			if !dd.gvespers {
				// A day without a sign: the service is the first saint's.
				cat = rankOfFirst(dd.title)
			}
		}
		if dd.gvespers && cat != "" {
			// The rank of the saint with the sign (its magnification, SupplementMagnifications).
			add = append(add, corpus.Unit{ID: p + ".common.rank", Book: "menaion", Kind: "marker", Lang: "cu", Body: cat,
				SourceID: "orthlib.mineya", Meta: map[string]any{"category": cat, "rank_from_title": guessed}})
		}
		if dd.ref != "" && !dd.litProk {
			for _, part := range commonLitParts {
				copyUnit(p+".lit."+part, "om."+dd.ref+".lit."+part)
			}
		}
		// The troparion, kontakion and sessional hymn of the rank (the main
		// service of the day only, not the additional ones, «me.MM.DDp»).
		// In a forefeast or afterfeast («тропарь праздника») the feast's
		// hymns stand for them.
		main := len(p) == len("me.MM.DD")
		if main && cat != "" && !dd.feastTroparion {
			if !dd.troparion {
				copyGuess(p+".troparion.1", "om."+cat+".troparion.1")
			}
			if !dd.kontakion {
				copyGuess(p+".kontakion.1", "om."+cat+".kontakion.1")
			}
			if !dd.sedalen {
				copyAll(p, cat, "matins.sedalen")
			}
			if dd.gvespers && !dd.aposticha {
				copyAll(p, cat, "gvespers.aposticha")
			}
		}
		// The matins Gospel of the rank, which is that of its liturgy.
		if dd.refGospel != "" && !dd.matinsGospel {
			copyUnit(p+".matins.gospel", "om."+dd.refGospel+".lit.gospel")
			copyUnit(p+".matins.gospel.announce", "om."+dd.refGospel+".lit.gospel.announce")
		}
		if !dd.gvespers {
			continue
		}
		if !dd.paremias {
			pc, g := dd.refParemias, guessed
			if own := commonParemiasOfDay[p]; own != "" {
				pc, guessed = own, false
			}
			if pc == "" {
				pc = cat
			} else {
				guessed = false
			}
			if pc == "" {
				warnings = append(warnings, fmt.Sprintf("%s: no paremias and no rank to take them from the General Menaion", p))
			} else {
				for n := 1; n <= 3; n++ {
					copyGuess(fmt.Sprintf("%s.gvespers.paremia.%d.title", p, n), fmt.Sprintf("om.%s.gvespers.paremia.%d.title", pc, n))
					copyGuess(fmt.Sprintf("%s.gvespers.paremia.%d", p, n), fmt.Sprintf("om.%s.gvespers.paremia.%d", pc, n))
				}
			}
			guessed = g
		}
		if !dd.matinsProk && cat != "" {
			copyGuess(p+".matins.prokeimenon", "om."+cat+".matins.prokeimenon")
			copyGuess(p+".matins.prokeimenon.verse", "om."+cat+".matins.prokeimenon.verse")
		}
	}
	return append(units, add...), warnings
}

// rankOfTitle guesses the rank of the saint with the sign from a day's
// title: the last commemoration that names a rank — the Russian Menaion
// adds its own saints, who have the festal service, after the Greek ones.
func rankOfTitle(title string) string {
	segs := strings.FieldsFunc(title, func(r rune) bool { return r == ':' || r == ';' })
	for i := len(segs) - 1; i >= 0; i-- {
		if r := rankOfSegment(segs[i]); r != "" {
			return r
		}
	}
	return ""
}

// rankOfFirst: the rank of the first commemoration of the title that names
// one — a day without a sign, whose service is the first saint's.
func rankOfFirst(title string) string {
	for _, seg := range strings.FieldsFunc(title, func(r rune) bool { return r == ':' || r == ';' }) {
		if r := rankOfSegment(seg); r != "" {
			return r
		}
	}
	return ""
}

func rankOfSegment(title string) string {
	ranks := []struct{ word, cat string }{
		{"прпдбномученицы", "nun_martyr"}, {"прпдбномчницы", "nun_martyr"},
		{"прпдбномученик", "monk_martyr"}, {"прпдбномчник", "monk_martyr"},
		{"сщенномученик", "hieromartyr"}, {"священномученик", "hieromartyr"}, {"сщенномчник", "hieromartyr"},
		{"мученицы", "woman_martyr"}, {"мчницы", "woman_martyr"}, {"мучениц ", "women_martyrs"}, {"мчниц ", "women_martyrs"},
		{"мучениц,", "women_martyrs"}, {"мчниц,", "women_martyrs"},
		{"мучеников", "martyrs"}, {"мученик ", "martyrs"}, {"мученик,", "martyrs"}, {"мчник ", "martyrs"}, {"мчник,", "martyrs"},
		{"мученик", "martyr"}, {"мчник", "martyr"},
		{"безмездник", "unmercenaries"}, {"юродив", "fools"}, {"исповедник", "confessor"},
		{"апсла", "apostle"}, {"апслов", "apostles"}, {"апсл ", "apostles"}, {"апсл,", "apostles"}, {"апсл:", "apostles"},
		{"пророка", "prophet"}, {"архаггла", "angels"}, {"архистратига", "angels"}, {"безплотных", "angels"},
		{"архиепископа", "hierarch"}, {"епископа", "hierarch"}, {"митрополита", "hierarch"}, {"стителя", "hierarch"},
		{"патриарха", "hierarch"}, {"иже во стых", "hierarch"},
		{"прпдбныя", "nun"}, {"преподобныя", "nun"}, {"матере нашея", "nun"},
		{"прпдбнаго", "monk"}, {"преподобнаго", "monk"}, {"прпдбных", "monks"}, {"преподобных", "monks"},
	}
	best, at := "", len(title)+1
	for _, r := range ranks {
		if i := strings.Index(title, r.word); i >= 0 && i < at {
			best, at = r.cat, i
		}
	}
	return best
}
