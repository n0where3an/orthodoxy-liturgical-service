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

// The signs of the services in the Typikon's calendar (Типикон, гл. 48,
// «Месяцеслов», files 48_<MM>.hip): every commemoration begins with its
// sign, black or red. Their meaning, by ch. 47 «О знамениях»:
//
//	%<<(+)>%>    «крст окружен»: the great feasts (of the Lord, of the
//	             Theotokos, the Forerunner's two, Peter and Paul) — vigil
//	%<<\+/>%>    «крст с полукружием»: the middle feasts, of saints — vigil
//	             («бдение совершается»), and the canon of the Theotokos added
//	             at matins; small vespers as the day's text gives it
//	%<<+>%>      «крст един точию»: «бдение не совершается» — Ps 1, stichera
//	             on 6 (8), three readings; at matins the polyeleos, the
//	             Gospel, the canon on 8, the great doxology
//	%<<(:.>%>    red, «три точки несовершенно окружены»: the doxology service —
//	             the ordinary kathisma, stichera on 6, the canon on 6, the
//	             great doxology
//	<(:.>        black: the six-fold — stichera on 6, the canon on 6
//
// and none: stichera on 3, the canon on 4. Keys ty.<MM>.<DD>.sign.<N> (N
// the commemoration of the day in order); Meta: sign (none, six_fold,
// doxology, polyeleos, saint_vigil, vigil), red.
var (
	reTySign = regexp.MustCompile(`^(%<)?<(\(:\.|\+|\(\+\)|\\\+/)>\s*(%>)?\s*`)
	reTyDay  = regexp.MustCompile(`^(\S{1,8}?)\.\s+`)
	// A numbered line of a service's rubric, not a day of the month.
	reTyRubric = regexp.MustCompile(`^(слава|и ныне|на |по |таже|аще|стих|глас)`)
)

var tySigns = map[string]string{`(:.`: "six_fold", `+`: "polyeleos", `\+/`: "saint_vigil", `(+)`: "vigil"}

// SignRank orders the signs: none < six_fold < doxology < polyeleos <
// saint_vigil (a middle feast) < vigil (a great feast).
var SignRank = map[string]int{"none": 0, "six_fold": 1, "doxology": 2, "polyeleos": 3, "saint_vigil": 4, "vigil": 5}

// ImportTypikonSigns reads the signs of the twelve months.
func ImportTypikonSigns(root string) ([]corpus.Unit, error) {
	dir := filepath.Join(root, "tipikon", "tipik", "Типикон")
	var units []corpus.Unit
	var errs []string
	for m := 1; m <= 12; m++ {
		path := filepath.Join(dir, fmt.Sprintf("48_%02d.hip", m))
		us, err := parseTypikonSigns(path, m)
		if err != nil {
			errs = append(errs, err.Error())
		}
		units = append(units, us...)
	}
	if len(errs) > 0 {
		return units, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return units, nil
}

func parseTypikonSigns(path string, month int) ([]corpus.Unit, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text, err := hip.DecodeCP1251(b)
	if err != nil {
		return nil, err
	}
	var units []corpus.Unit
	day, n := 0, 0
	add := func(sign string, red bool, rest string) {
		n++
		title := strings.TrimSpace(CleanText(hip.ToUnicode(rest)))
		units = append(units, corpus.Unit{ID: fmt.Sprintf("ty.%02d.%02d.sign.%d", month, day, n), Book: "typikon", Kind: "marker",
			Lang: "cu", Body: title, SourceID: "orthlib.tipikon",
			Meta: map[string]any{"sign": sign, "red": red, "file": filepath.Base(path)}})
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		// «s~_i. <(:.> Ст~ы'я…»: a day begins with its numeral, the next day's.
		if m := reTyDay.FindStringSubmatch(line); m != nil {
			v := numeralOfWord(strings.TrimRight(hip.ToUnicode(m[1]), "."))
			rest := line[len(m[0]):]
			signed := reTySign.MatchString(rest)
			restPlain := strings.ToLower(hip.PlainString(hip.ToUnicode(rest)))
			switch {
			case v == day+1 && v <= 31 && (signed || startsUpper(hip.ToUnicode(rest)) && !reTyRubric.MatchString(restPlain)):
				day, n = v, 0
				line = rest
			case v == day && signed && n == 1 && units[len(units)-1].Meta["sign"] == "none":
				// «г~. Слава…» inside a service taken for the day: the day is this one.
				units, n = units[:len(units)-1], 0
				line = rest
			default:
				continue
			}
		} else if day == 0 || !reTySign.MatchString(line) {
			continue // a further commemoration of the day begins with its sign
		}
		if m := reTySign.FindStringSubmatch(line); m != nil {
			sign := tySigns[m[2]]
			red := m[1] != ""
			if sign == "six_fold" && red {
				sign = "doxology" // the same sign in red
			}
			add(sign, red, line[len(m[0]):])
		} else {
			add("none", false, line)
		}
	}
	if day < 28 {
		return units, fmt.Errorf("%s: %d days", filepath.Base(path), day)
	}
	return units, nil
}
