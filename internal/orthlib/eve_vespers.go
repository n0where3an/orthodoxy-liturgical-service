package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The vespers of the eves of the Nativity and the Theophany, joined to the
// liturgy of St Basil the Great (the Menaion, 25 December and 6 January:
// «при часе з-м дне… начинаем вечерню», «и по возгласе тристое. и
// прокимен…»). Units under ev.<feast>:
//
//	lic.<N> (Meta repeat: 2 «дважды»), lic.glory_now   at «Господи, воззвах»
//	reading.<N>, title.<N>, troparion.<N>, verse.<N>, refrain.<N>, mark.<N>
//	    the readings and, after some of them, the reader's troparion with its
//	    verses, the choir singing its end (refrain), «Слава», «И ныне» (mark:
//	    a copy of common.glory, common.and_now); one numbering, in order
//	lit.rubric   the liturgy: prokeimenon, Apostle, alleluia, Gospel,
//	    communion verse (resolved by ResolveLiturgy)
var eveVespersFiles = map[string]string{
	"nativity":  "mineya/dec/dec/Декабрь/25.hip",
	"theophany": "mineya/jan/jan/Январь/06.hip",
}

var (
	reEVStart     = regexp.MustCompile(`начинаем вечерню`)
	reEVEntrance  = regexp.MustCompile(`^вход со еиглием`)
	reEVLiturgy   = regexp.MustCompile(`ектениа.*тристое|тристому:? и тристое`)
	reEVTroparion = regexp.MustCompile(`тропарь( сей)?,? глас \S+?:$|^тропарь,? глас \S+?:$|глаголет тропарь сей:$`)
	reEVVerse     = regexp.MustCompile(`^стих \S+?(?:, псалом \S+?)?:\s*`)
	reEVTitle     = regexp.MustCompile(`чтение\.?$`)
)

// ImportEveVespers parses the vespers of both eves.
func ImportEveVespers(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for feast, file := range eveVespersFiles {
		d, err := LoadDoc(filepath.Join(root, file))
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		us, err := parseEveVespers(d, feast)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", file, err))
		}
		units = append(units, us...)
	}
	return units, warnings
}

func parseEveVespers(d *Doc, feast string) ([]corpus.Unit, error) {
	prefix := "ev." + feast + "."
	var units []corpus.Unit
	add := func(key, kind, body string, meta map[string]any) string {
		units = append(units, corpus.Unit{ID: prefix + key, Book: "menaion", Kind: kind, Lang: "cu", Body: CleanText(body),
			SourceID: "orthlib.mineya", Meta: meta})
		return prefix + key
	}
	section := "" // lic | readings
	nLic, n := 0, 0
	gloryNow, troparionNext, joinNext, skipRefrain := false, false, false, false
	troparion, troparionID, refrainID, title := "", "", "", ""
	item := func(kind, body string, meta map[string]any) string {
		n++
		return add(fmt.Sprintf("%s.%d", kind, n), kind, body, meta)
	}
	mark := func(ref string) { item("mark", "", map[string]any{"copy": ref}) }
	fold := func(s string) string {
		return strings.Trim(hip.Fold(strings.ToLower(hip.PlainString(s))), " .,:;!")
	}
	// tail: the last words of a text, to know the choir's refrain (the end
	// of the troparion) despite misprints («во тме» / «бо тме»).
	tail := func(s string) string {
		w := strings.Fields(strings.NewReplacer(",", " ", ".", " ", ":", " ", ";", " ").Replace(fold(s)))
		return strings.Join(w[max(0, len(w)-3):], " ")
	}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.TrimSpace(strings.ToLower(hip.PlainString(text)))
		if plain == "" {
			continue
		}
		words := len(strings.Fields(plain))
		switch {
		case section == "" && reEVStart.MatchString(plain):
			section = "lic"
			continue
		case section == "":
			continue
		case reEVLiturgy.MatchString(plain):
			_, pos := hip.Plain(text)
			var got []corpus.Unit
			parseLiturgyRubric(text, plain, pos, func(key, kind, body string, meta map[string]any) {
				delete(meta, "beatitudes")
				got = append(got, corpus.Unit{ID: prefix + "lit." + key, Book: "menaion", Kind: kind, Lang: "cu",
					Body: CleanText(body), SourceID: "orthlib.mineya", Meta: meta})
			})
			units = append(units, got...)
			if nLic < 4 || n < 6 {
				return units, fmt.Errorf("eve vespers: %d stichera, %d readings and hymns", nLic, n)
			}
			return append(units, weekendVespersRubric(d, prefix)...), nil
		case reEVEntrance.MatchString(plain):
			section = "readings"
			continue
		}
		if section == "lic" {
			switch {
			case strings.HasPrefix(plain, "дважды") && nLic > 0:
				units[len(units)-1].Meta = map[string]any{"repeat": 2}
			case strings.HasPrefix(plain, "слава, и ныне"):
				gloryNow = true
			case words > 12 && gloryNow:
				add("lic.glory_now", "sticheron", text, nil)
			case words > 12:
				nLic++
				add(fmt.Sprintf("lic.%d", nLic), "sticheron", text, nil)
			}
			continue
		}
		// The readings and the troparia between them.
		if strings.HasPrefix(plain, "[зри]") || strings.HasPrefix(plain, "аще ") || strings.HasPrefix(plain, "со стихи") {
			continue // «аще прилучится навечерие… в пяток, прокимен…»
		}
		// The troparion printed in two paragraphs, the second its end.
		if joinNext {
			joinNext = false
			if words > 6 && !reEVVerse.MatchString(plain) && !reEVTitle.MatchString(plain) {
				units[len(units)-1].Body = CleanText(units[len(units)-1].Body + " " + text)
				troparion += " " + plain
				continue
			}
		}
		switch {
		case reEVTroparion.MatchString(plain):
			troparionNext, troparion, refrainID = true, "", ""
		case troparionNext && words > 6:
			troparionNext = false
			troparion = plain
			troparionID = item("troparion", text, nil)
			joinNext = true
		case reEVVerse.MatchString(plain):
			item("verse", text[strings.Index(text, ":")+1:], nil)
		case troparion != "" && words > 3 && words < len(strings.Fields(troparion)) && tail(text) == tail(troparion):
			// The end of the troparion sung by the choir (given in full the
			// first time, then repeated).
			if skipRefrain {
				skipRefrain = false // the end of the troparion given again in full
				continue
			}
			if refrainID == "" {
				refrainID = item("refrain", text, nil)
			} else {
				item("refrain", "", map[string]any{"copy": refrainID})
			}
		case troparion != "" && strings.Contains(plain, "слава") && strings.Contains(plain, "конец"):
			// «слава: поет лик правый, конец тропаря. и ныне: левый… конец»
			mark("common.glory")
			if refrainID != "" {
				item("refrain", "", map[string]any{"copy": refrainID})
			}
			mark("common.and_now")
			if refrainID != "" {
				item("refrain", "", map[string]any{"copy": refrainID})
			}
		case troparion != "" && strings.HasPrefix(plain, "слава, и ныне"):
			mark("common.glory_now")
		case troparion != "" && words > 6 && fold(text) == fold(troparion) ||
			troparion != "" && words > 6 && strings.HasPrefix(fold(text), fold(troparion)[:min(40, len(fold(troparion)))]):
			// The troparion in full again (at «Слава, и ныне»).
			item("troparion", "", map[string]any{"copy": troparionID})
			skipRefrain = tail(text) != tail(troparion)
		case reEVTitle.MatchString(plain) && words <= 8:
			title = text
			troparion = ""
		case words > 12:
			id := item("reading", reBracketHead.ReplaceAllString(text, ""), nil)
			if title != "" {
				add(fmt.Sprintf("title.%d", n), "announcement", title, map[string]any{"of": id})
			}
			title, troparion = "", ""
		}
	}
	return units, fmt.Errorf("eve vespers: no liturgy rubric")
}

// weekendVespersRubric: the vespers apart when the eve is Saturday or Sunday
// («тристое не поем, но бывает по паремиах ектениа малая, и по возгласе
// прокимен…»): its prokeimenon, Apostle, alleluia and Gospel, under
// <prefix>wv.lit (the Nativity's; the Theophany's are the liturgy's, «яко
// предписася»).
func weekendVespersRubric(d *Doc, prefix string) []corpus.Unit {
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		i := strings.Index(plain, "по возгласе прокимен")
		if i < 0 || !strings.Contains(plain, "тристое не поем") {
			continue
		}
		_, pos := hip.Plain(text)
		sub := uni(text, pos, plain, i, len(plain))
		subPlain := strings.ToLower(hip.PlainString(sub))
		_, subPos := hip.Plain(sub)
		var units []corpus.Unit
		parseLiturgyRubric(sub, subPlain, subPos, func(key, kind, body string, meta map[string]any) {
			delete(meta, "beatitudes")
			delete(meta, "kinonik")
			units = append(units, corpus.Unit{ID: prefix + "wv.lit." + key, Book: "menaion", Kind: kind, Lang: "cu",
				Body: CleanText(body), SourceID: "orthlib.mineya", Meta: meta})
		})
		return units
	}
	return nil
}
