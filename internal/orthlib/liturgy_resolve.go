package orthlib

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// psalmVerse is a prokeimenon or an alleluia verse with its verse, or a
// communion verse, in full (Служебник: «Прокимны и аллилуиа дневнии»,
// «…общии святым»). The Menaion cites them by their first words.
type psalmVerse struct {
	Tone        int
	Text, Verse string
}

type versePool struct {
	prok, alleluia []psalmVerse
	kinonik        []string
}

// loadVersePool reads the daily and the common prokeimena of the Служебник.
func loadVersePool(root string) (*versePool, error) {
	pool := &versePool{}
	for _, f := range []string{"17prok_dnevn.hip", "18prok_obsch.hip"} {
		d, err := LoadDoc(filepath.Join(root, "sluzhebnik", "sl", f))
		if err != nil {
			return nil, err
		}
		kind, tone := "", 0
		for _, p := range d.Paras {
			text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
			plain, pos := hip.Plain(text)
			plain = strings.ToLower(plain)
			body := ""
			switch {
			case strings.Contains(plain, "причастен:"):
				i := strings.Index(plain, "причастен:") + len("причастен:")
				pool.kinonik = append(pool.kinonik, uni(text, pos, plain, i, len(plain)))
				kind = ""
				continue
			case strings.Contains(plain, "прокимен") && strings.Contains(plain, "глас"),
				strings.HasPrefix(plain, "аллилуиа") && strings.Contains(plain, "глас"):
				kind = "prok"
				if strings.HasPrefix(plain, "аллилуиа") {
					kind = "alleluia"
				}
				g := strings.Index(plain, "глас ")
				tone = uniNumeral(text, pos, plain, g+len("глас "), len(plain))
				if c := strings.Index(plain[g:], ":"); c >= 0 {
					body = uni(text, pos, plain, g+c+1, len(plain))
				}
				if len(strings.Fields(body)) < 3 {
					continue // the text is in the next paragraph
				}
			case kind != "":
				body = text
			default:
				continue
			}
			bp, bpos := hip.Plain(body)
			bp = strings.ToLower(bp)
			v := psalmVerse{Tone: tone, Text: body}
			if i := strings.Index(bp, "стих:"); i >= 0 {
				v.Text = uni(body, bpos, bp, 0, i)
				v.Verse = uni(body, bpos, bp, i+len("стих:"), len(bp))
			}
			if kind == "prok" {
				pool.prok = append(pool.prok, v)
			} else {
				pool.alleluia = append(pool.alleluia, v)
			}
			kind = ""
		}
	}
	return pool, nil
}

// precomposed letters with a grave accent (the corpus writes them with a
// combining accent, typed text may not).
var precomposed = strings.NewReplacer("ѝ", "и", "ѐ", "е", "Ѝ", "И", "Ѐ", "Е")

// cueKey folds the first words of a cited text for matching.
func cueKey(s string, words int) string {
	s = precomposed.Replace(s)
	w := strings.FieldsFunc(hip.Fold(hip.PlainString(s)), func(r rune) bool { return r == ' ' || r == ',' || r == ':' || r == '.' || r == ';' })
	if len(w) > words {
		w = w[:words]
	}
	return strings.Join(w, " ")
}

func isCue(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasSuffix(s, ":") || strings.HasSuffix(s, ";")
}

// sameStart: the first words agree, allowing for one word written in full
// in one book and under a titlo in the other («Свѧще́нницы» / «Сщ҃е́нницы»).
func sameStart(a, b string) bool {
	wa, wb := strings.Fields(cueKey(a, 5)), strings.Fields(cueKey(b, 5))
	n := min(len(wa), len(wb))
	if n < 2 {
		return n == 1 && wa[0] == wb[0] && len(wa) == len(wb)
	}
	same := 0
	for i := 0; i < n; i++ {
		if wa[i] == wb[i] {
			same++
		}
	}
	return same >= n-1 && wa[0][:2] == wb[0][:2] && (same >= 3 || same == n)
}

// complete returns the full text of a cited one.
func complete(cue string, candidates []string) string {
	if !isCue(cue) {
		return cue
	}
	for _, c := range candidates {
		if !isCue(c) && sameStart(cue, c) {
			return c
		}
	}
	return strings.TrimRight(strings.TrimSpace(cue), ":;")
}

// ResolveLiturgy turns the Menaion's liturgy rubrics into texts: the
// prokeimenon with its verse, the alleluia verses, the communion verse, the
// Apostle and the Gospel (from the pericopes, lect.zach.*), the beatitudes
// troparia (from the matins canon), the zadostoinik (an irmos of the canon),
// and fills the units that copy another one.
func ResolveLiturgy(root string, units []corpus.Unit) ([]corpus.Unit, []string) {
	pool, err := loadVersePool(root)
	if err != nil {
		return units, []string{err.Error()}
	}
	byID := map[string]int{}
	for i, u := range units {
		byID[u.ID] = i
	}
	var warnings []string
	// Copies.
	for i, u := range units {
		src, _ := u.Meta["copy"].(string)
		if src == "" {
			continue
		}
		if j, ok := byID[src]; ok {
			units[i].Body = units[j].Body
		} else {
			warnings = append(warnings, fmt.Sprintf("%s: no %s to copy", u.ID, src))
		}
	}
	var add []corpus.Unit
	put := func(id, kind string, tone int, body string, meta map[string]any) {
		if strings.TrimSpace(body) == "" {
			return
		}
		add = append(add, corpus.Unit{ID: id, Book: "menaion", Kind: kind, Tone: tone, Lang: "cu", Body: CleanText(body),
			SourceID: "orthlib.mineya", Meta: meta})
	}
	copyUnit := func(id, src string) bool {
		j, ok := byID[src]
		if !ok {
			return false
		}
		c := units[j]
		c.ID, c.Meta = id, map[string]any{"copied_from": src}
		add = append(add, c)
		return true
	}
	prokTexts := func() (texts, verses []string, tones []int) {
		for _, p := range pool.prok {
			texts, verses, tones = append(texts, p.Text), append(verses, p.Verse), append(tones, p.Tone)
		}
		return
	}
	pt, pv, _ := prokTexts()
	var at []string
	for _, a := range pool.alleluia {
		at = append(at, a.Text)
	}
	for _, u := range units {
		if !strings.HasSuffix(u.ID, ".lit.rubric") {
			continue
		}
		base := strings.TrimSuffix(u.ID, "rubric") // me.MM.DD.lit.
		day := strings.TrimSuffix(base, "lit.")    // me.MM.DD.
		m := u.Meta
		if prok, _ := m["prok"].(string); prok != "" {
			tone, _ := m["prok_tone"].(int)
			full := complete(prok, pt)
			verse, _ := m["prok_verse"].(string)
			if isCue(verse) || verse == "" {
				for i := range pt {
					if sameStart(pt[i], full) && pv[i] != "" {
						verse = pv[i]
						break
					}
				}
			}
			put(base+"prokeimenon", "prokeimenon", tone, full, nil)
			put(base+"prokeimenon.verse", "verse", 0, strings.TrimRight(strings.TrimSpace(verse), ":;")+terminal(verse), nil)
		}
		if al, _ := m["alleluia"].(string); al != "" {
			tone, _ := m["alleluia_tone"].(int)
			full := complete(al, at)
			verse, _ := m["alleluia_verse"].(string)
			for _, a := range pool.alleluia {
				if sameStart(a.Text, full) {
					if verse == "" || isCue(verse) {
						verse = a.Verse
					}
					break
				}
			}
			put(base+"alleluia.verse.1", "verse", tone, full, nil)
			put(base+"alleluia.verse.2", "verse", 0, strings.TrimRight(strings.TrimSpace(verse), ":;"), nil)
		}
		if k, _ := m["kinonik"].(string); k != "" {
			put(base+"kinonik", "kinonik", 0, complete(k+":", pool.kinonik), nil)
		}
		if n, _ := m["apostle"].(int); n > 0 {
			src := fmt.Sprintf("lect.zach.ap.%d", n)
			if h, _ := m["apostle_half"].(bool); h {
				src += ".half"
			}
			if !copyUnit(base+"apostle", src) || !copyUnit(base+"apostle.announce", src+".announce") {
				warnings = append(warnings, fmt.Sprintf("%s: no Apostle pericope %d", u.ID, n))
			}
		}
		if n, _ := m["gospel"].(int); n > 0 {
			src := fmt.Sprintf("lect.zach.%s.%d", m["gospel_book"], n)
			if h, _ := m["gospel_half"].(bool); h {
				src += ".half"
			}
			if !copyUnit(base+"gospel", src) || !copyUnit(base+"gospel.announce", src+".announce") {
				warnings = append(warnings, fmt.Sprintf("%s: no Gospel pericope %s %d", u.ID, m["gospel_book"], n))
			}
		}
		if odes, ok := m["beatitudes"].([][2]int); ok {
			n := 0
			for _, co := range odes {
				for t := 1; t <= 6; t++ {
					src := fmt.Sprintf("%smatins.canon.%d.ode.%d.troparion.%d", day, co[0], co[1], t)
					if _, ok := byID[src]; !ok {
						break
					}
					n++
					copyUnit(fmt.Sprintf("%sbeatitudes.%d", base, n), src)
				}
			}
			// «И ныне»: the theotokion of the last ode.
			if len(odes) > 0 {
				last := odes[len(odes)-1]
				copyUnit(base+"beatitudes.theotokion", fmt.Sprintf("%smatins.canon.%d.ode.%d.theotokion", day, last[0], last[1]))
			}
			if n == 0 {
				warnings = append(warnings, fmt.Sprintf("%s: no canon troparia for the beatitudes", u.ID))
			}
		}
		if z, _ := m["zadostoinik"].(string); z != "" {
			var irmoi []string
			ids := map[string]string{}
			for c := 1; c <= 3; c++ {
				id := fmt.Sprintf("%smatins.canon.%d.ode.9.irmos", day, c)
				if j, ok := byID[id]; ok {
					irmoi = append(irmoi, units[j].Body)
					ids[units[j].Body] = id
				}
			}
			if full := complete(z+":", irmoi); ids[full] != "" {
				copyUnit(base+"zadostoinik", ids[full])
			} else {
				warnings = append(warnings, fmt.Sprintf("%s: zadostoinik «%s» not found among the irmoi", u.ID, z))
			}
		}
	}
	sort.SliceStable(add, func(i, j int) bool { return add[i].ID < add[j].ID })
	return append(units, add...), warnings
}

// terminal keeps a final full stop when the cited verse is shortened.
func terminal(verse string) string {
	if isCue(verse) {
		return "."
	}
	return ""
}
