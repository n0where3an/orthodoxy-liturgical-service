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

// The biblical odes (песни библейские) of the Irmologion
// (irmologij/irmologij), the verses sung with the troparia of the canon.
// Keys ir.ode.<O>.<set>.<N>, N from 1 in the order of the book:
//
//	set weekday  «песней стихи, поемии по вся дни к каноном» (16pesnvsedn.hip)
//	set sunday   «песней стихи, поемии в недели и праздники» (17pesnprazd.hip)
//	set lent     «во стую и великую четыредесятницу» (18pesnpost.hip), with
//	             the 2nd ode and the verse numbers of the book
//
// Meta "on" marks the verse from which the canon is sung «на N» (the
// places count «Слава» and «И ныне»: «на 8» is the last six verses);
// «начинай» of the weekday odes is «на 14». The 9th ode is the Song of
// Zacharias (the Magnificat, with «Честнейшую», is in the Chasoslov).
// Ode 8 ends with its own doxology: ir.ode.8.<set>.glory, .now, and the
// refrain «Хвалим, благословим»: ir.ode.8.<set>.refrain. On Sunday the
// troparion after the irmos is sung with «Слава, Господи, святому
// воскресению Твоему» (ir.sunday.refrain) instead of a verse.
var odeOrdinals = map[string]int{"первая": 1, "вторая": 2, "третия": 3, "четвертая": 4, "пятая": 5, "шестая": 6,
	"седмая": 7, "осмая": 8, "девятая": 9}

var (
	reIrOde       = regexp.MustCompile(`^песнь (первая|вторая|третия|четвертая|пятая|шестая|седмая|осмая|девятая)`)
	reIrMagnif    = regexp.MustCompile(`^песнь престыя бцды`)
	reIrZacharias = regexp.MustCompile(`^песнь стаго пророка захарии`)
	reIrOn        = regexp.MustCompile(`^(на (\S+?)|начинай):\s*`)
	reIrRubric    = regexp.MustCompile(`^(первый лик|вторый лик|таже|и паки|от зде|глаголи |подобает|седмая же|и тропарь|и поклон|поклон|и катавасиа|катавасиа|и по ирмосе|в понедельник|во вторник|в среду|в четверток|во второе|в третие|аще|ведомо|сх )`)
	reIrDoxology  = regexp.MustCompile(`^(слава:? и ныне:?$|слава оцу|ныне и присно|и ныне и присно|блгословим оца)`)
)

// ImportIrmologionOdes parses the three sets of the biblical odes.
func ImportIrmologionOdes(root string) ([]corpus.Unit, []string) {
	dir := filepath.Join(root, "irmologij", "irmologij")
	var units []corpus.Unit
	var warnings []string
	for _, s := range []struct{ file, set, start string }{
		{"16pesnvsedn.hip", "weekday", "и абие начинает песни"},
		{"17pesnprazd.hip", "sunday", "таже начинает первый лик стих песни"},
	} {
		us, err := parseIrmologionOdes(filepath.Join(dir, s.file), s.set, s.start)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		units = append(units, us...)
	}
	us, err := parseLentenOdes(filepath.Join(dir, "18pesnpost.hip"))
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	return append(units, us...), warnings
}

func parseIrmologionOdes(path, set, start string) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	var units []corpus.Unit
	add := func(ode int, key, body string, meta map[string]any) {
		m := map[string]any{"file": filepath.Base(path), "set": set}
		for k, v := range meta {
			m[k] = v
		}
		units = append(units, corpus.Unit{ID: fmt.Sprintf("ir.ode.%d.%s.%s", ode, set, key), Book: "irmologion", Kind: "verse",
			Lang: "cu", Body: strings.TrimSpace(body), SourceID: "orthlib.irmologion", Meta: m})
	}
	ode, n := 0, map[int]int{}
	started, magnificat, on14 := false, false, false
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		if !started {
			if strings.HasPrefix(plain, start) {
				started, ode = true, 1
			}
			if set == "sunday" && strings.HasPrefix(plain, "слава гдси, стому воскрснию") {
				// The troparion sung without a verse after the irmos.
				units = append(units, corpus.Unit{ID: "ir.sunday.refrain", Book: "irmologion", Kind: "refrain", Lang: "cu",
					Body: text, SourceID: "orthlib.irmologion", Meta: map[string]any{"file": filepath.Base(path)}})
			}
			continue
		}
		switch {
		case reIrOde.MatchString(plain):
			ode, magnificat = odeOrdinals[reIrOde.FindStringSubmatch(plain)[1]], false
			continue
		case reIrMagnif.MatchString(plain):
			ode, magnificat = 9, true
			continue
		case reIrZacharias.MatchString(plain):
			ode, magnificat = 9, false
			continue
		case magnificat || ode == 9 && n[9] == 0 && strings.HasPrefix(plain, "величит") || ode == 9 && strings.HasPrefix(plain, "яко призре"):
			// The Magnificat of the weekday set comes without its own heading.
			if ode == 9 && !magnificat && !strings.HasPrefix(plain, "величит") && !strings.HasPrefix(plain, "яко призре") {
				break
			}
			magnificat = true
			continue
		case reIrDoxology.MatchString(plain):
			if ode == 8 && strings.HasPrefix(plain, "блгословим оца") {
				add(8, "glory", text, nil)
			} else if ode == 8 && strings.Contains(plain, "ныне и присно") {
				add(8, "now", text, nil)
			}
			if ode == 9 && strings.HasPrefix(plain, "слава") {
				return units, checkIrmologionOdes(path, n)
			}
			continue
		case strings.HasPrefix(plain, "таже глаголем: хвалим"):
			add(8, "refrain", afterColon(text, 3), nil)
			continue
		case strings.HasPrefix(plain, "от зде начинай"):
			on14 = true // «Отзде начинай стихи…»: the next verse, on 14
			continue
		case reIrRubric.MatchString(plain):
			continue
		}
		var meta map[string]any
		if on14 {
			meta, on14 = map[string]any{"on": 14}, false
		}
		if m := reIrOn.FindStringSubmatchIndex(plain); m != nil {
			on := 14
			if m[4] >= 0 {
				_, pos := hip.Plain(text)
				on = uniNumeral(text, pos, plain, m[4], m[5])
			}
			meta = map[string]any{"on": on}
			_, pos := hip.Plain(text)
			text = strings.TrimSpace(uni(text, pos, plain, m[1], len(plain)))
			text = capitalizeFirst(text)
		}
		n[ode]++
		add(ode, fmt.Sprint(n[ode]), text, meta)
	}
	return units, checkIrmologionOdes(path, n)
}

func checkIrmologionOdes(path string, n map[int]int) error {
	var missing []int
	for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
		if n[o] < 6 {
			missing = append(missing, o)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s: odes with fewer than 6 verses: %v (%v)", filepath.Base(path), missing, n)
	}
	return nil
}

// capitalizeFirst upper-cases the first letter of a verse whose «На и:» was cut.
func capitalizeFirst(s string) string {
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsLetter(r) {
			rs[i] = unicode.ToUpper(r)
			break
		}
	}
	return string(rs)
}

// parseLentenOdes: «Во стую и великую четыредесятницу стихословятся песни
// сице»: each paragraph holds several verses, each begun by its number
// («в҃ Помо́щникъ…»); «на ді:» marks the places.
func parseLentenOdes(path string) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	var units []corpus.Unit
	ode := 0
	count := map[int]int{}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if m := reIrOde.FindStringSubmatch(plain); m != nil {
			ode = odeOrdinals[m[1]]
			continue
		}
		if reIrMagnif.MatchString(plain) {
			ode = 0 // the Magnificat (with «Честнейшую»: in the Chasoslov)
			continue
		}
		if ode == 0 || plain == "" || strings.HasPrefix(plain, "слава") || strings.HasPrefix(plain, "подобает") ||
			strings.HasPrefix(plain, "таже") || strings.HasPrefix(plain, "и поклон") {
			continue
		}
		on := 0
		if m := reIrOn.FindStringSubmatchIndex(plain); m != nil && m[4] >= 0 {
			_, pos := hip.Plain(text)
			on = uniNumeral(text, pos, plain, m[4], m[5])
			text = strings.TrimSpace(uni(text, pos, plain, m[1], len(plain)))
		}
		for _, v := range splitNumberedVerses(text) {
			if v.n == 0 {
				continue
			}
			meta := map[string]any{"file": filepath.Base(path), "set": "lent", "n": v.n}
			if on > 0 {
				meta["on"], on = on, 0
			}
			count[ode]++
			units = append(units, corpus.Unit{ID: fmt.Sprintf("ir.ode.%d.lent.%d", ode, v.n), Book: "irmologion", Kind: "verse",
				Lang: "cu", Body: v.text, SourceID: "orthlib.irmologion", Meta: meta})
		}
	}
	var bad []string
	for _, o := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if count[o] < 6 {
			bad = append(bad, fmt.Sprintf("ode %d: %d", o, count[o]))
		}
	}
	if len(bad) > 0 {
		return units, fmt.Errorf("%s: Lenten odes: %v", filepath.Base(path), bad)
	}
	return units, nil
}

type numberedVerse struct {
	n    int
	text string
}

// splitNumberedVerses splits «сⷯ а҃ Пое́мъ… в҃ Помо́щникъ…» at the verse
// numbers: a numeral word (with a titlo) followed by a capitalised word.
func splitNumberedVerses(text string) []numberedVerse {
	words := strings.Fields(text)
	var out []numberedVerse
	cur := -1
	for i, w := range words {
		if strings.HasPrefix(hip.PlainString(w), "сх") && i == 0 {
			continue // «сⷯ» — стих
		}
		if strings.ContainsRune(w, '҃') && i+1 < len(words) && startsUpper(words[i+1]) {
			if v := numeralOfWord(strings.TrimRight(w, ".,:")); v > 0 && len([]rune(w)) <= 4 {
				out = append(out, numberedVerse{n: v})
				cur = len(out) - 1
				continue
			}
		}
		if cur >= 0 {
			if out[cur].text != "" {
				out[cur].text += " "
			}
			out[cur].text += w
		}
	}
	return out
}

func startsUpper(w string) bool {
	for _, r := range w {
		if unicode.IsLetter(r) {
			return unicode.IsUpper(r)
		}
	}
	return false
}
