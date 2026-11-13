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

// Kathisma numbers in Church Slavonic numerals, as folded by hip.Plain
// (ѕ folds to з, ѳ to ф): "кафисма ии." is the 18th.
var kathismaNumber = map[string]int{
	"а": 1, "в": 2, "г": 3, "д": 4, "е": 5, "з": 6, "з2": 7, "и": 8, "ф": 9, "і": 10,
	"аи": 11, "ви": 12, "ги": 13, "ди": 14, "еи": 15, "зи": 16, "зи2": 17, "ии": 18, "фи": 19, "к": 20,
}

var (
	reKathisma = regexp.MustCompile(`^кафисма ([а-яі]+)\.?$`)
	reStasis   = regexp.MustCompile(`^слава:?$`)
)

// ImportPsalter cuts the 20 kathismata into stases (ps.kathisma.N.K, K=1..3)
// from the Следованная Псалтирь. Kathismata are numbered by order of
// appearance, since 6/7 and 16/17 fold to the same letters.
func ImportPsalter(root string) ([]corpus.Unit, []string) {
	path := filepath.Join(root, "sledpsalt", "sp", "01ps.hip")
	d, err := LoadDoc(path)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var units []corpus.Unit
	var warnings []string
	kathisma, stasis := 0, 0
	var lines []string
	flush := func() {
		if kathisma == 0 || stasis == 0 || stasis > 3 || len(lines) == 0 {
			lines = nil
			return
		}
		units = append(units, corpus.Unit{
			ID: fmt.Sprintf("ps.kathisma.%d.%d", kathisma, stasis), Book: "psalter", Kind: "stasis",
			Lang: "cu", Body: strings.Join(lines, "\n"), SourceID: "orthlib.psalter",
			Meta: map[string]any{"kathisma": kathisma, "stasis": stasis},
		})
		lines = nil
	}
	for _, p := range d.Paras {
		text := CleanText(p.Text())
		plain := hip.PlainString(text)
		if reKathisma.MatchString(plain) {
			flush()
			kathisma++
			stasis = 1
			if kathisma > 20 {
				break
			}
			continue
		}
		if kathisma == 0 || stasis > 3 {
			continue
		}
		if reStasis.MatchString(plain) {
			flush()
			stasis++
			continue
		}
		if isSuperscription(text) {
			continue // "Псалом Давиду, а.", "Песнь степеней, рк."
		}
		lines = append(lines, stripVerseNumber(text))
	}
	flush()
	if kathisma != 20 {
		warnings = append(warnings, fmt.Sprintf("psalter: found %d kathismata", kathisma))
	}
	return units, warnings
}

// isSuperscription: a short line ending with the psalm number under titlo,
// not starting with a verse number.
func isSuperscription(text string) bool {
	w := strings.Fields(text)
	if len(w) == 0 || len(w) > 24 || stripVerseNumber(text) != text {
		return false
	}
	last := strings.TrimSuffix(w[len(w)-1], ".")
	return strings.ContainsRune(last, '\u0483') && len([]rune(last)) <= 6 && strings.HasSuffix(w[len(w)-1], ".")
}

// stripVerseNumber removes a leading verse number ("а҃. ", "і҃в. ").
func stripVerseNumber(s string) string {
	rs := []rune(s)
	i := 0
	for i < len(rs) && i < 8 && (unicode.IsLetter(rs[i]) || unicode.Is(unicode.Mn, rs[i])) {
		i++
	}
	if i > 0 && i+1 < len(rs) && rs[i] == '.' && rs[i+1] == ' ' && strings.ContainsRune(string(rs[:i]), '҃') {
		return string(rs[i+2:])
	}
	return s
}
