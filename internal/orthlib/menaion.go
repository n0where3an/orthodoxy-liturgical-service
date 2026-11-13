package orthlib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

var monthDirs = [13][2]string{
	{}, {"jan", "Январь"}, {"feb", "Февраль"}, {"mar", "Март"}, {"apr", "Апрель"},
	{"may", "Май"}, {"jun", "Июнь"}, {"jul", "Июль"}, {"aug", "Август"},
	{"sep", "Сентябрь"}, {"oct", "Октябрь"}, {"nov", "Ноябрь"}, {"dec", "Декабрь"},
}

// MonthDir is the directory of the monthly Menaion in the extracted tree.
func MonthDir(month int) string {
	d := monthDirs[month]
	return filepath.Join("mineya", d[0], d[0], d[1])
}

// monthDirIn finds the month directory under root: archives name the inner
// folder either by the Russian month name or by the Latin abbreviation.
func monthDirIn(root string, month int) string {
	d := monthDirs[month]
	for _, cand := range []string{
		filepath.Join(root, MonthDir(month)),
		filepath.Join(root, "mineya", d[0], d[0], d[0]),
	} {
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
	}
	return filepath.Join(root, MonthDir(month))
}

var dayFile = regexp.MustCompile(`^(\d\d)([a-zа-я]*)\.hip$`)

// ImportMenaion parses all monthly Menaion day files under root.
func ImportMenaion(root string) ([]corpus.Unit, []string, error) {
	var units []corpus.Unit
	var warnings []string
	for m := 1; m <= 12; m++ {
		dir := monthDirIn(root, m)
		entries, err := os.ReadDir(dir)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("month %02d: %v", m, err))
			continue
		}
		for _, e := range entries {
			mm := dayFile.FindStringSubmatch(e.Name())
			if mm == nil {
				continue
			}
			us, err := ParseMenaionDay(filepath.Join(dir, e.Name()), m, mm[1]+mm[2])
			if err != nil {
				warnings = append(warnings, err.Error())
				continue
			}
			if !hasKind(us, "troparion") {
				warnings = append(warnings, fmt.Sprintf("%02d/%s: no troparion found", m, e.Name()))
			}
			units = append(units, us...)
		}
	}
	return units, warnings, nil
}

func hasKind(us []corpus.Unit, kind string) bool {
	for _, u := range us {
		if u.Kind == kind {
			return true
		}
	}
	return false
}

var (
	reTroparion = regexp.MustCompile(`^(?:на [^.:]*?[.:]? )?(?:и )?(?:ин )?тропарь([^:.]*?)глас`)
	reKontakion = regexp.MustCompile(`^(?:на [^.:]*?[.:]? )?(?:и )?(?:ин )?кондак([^:.]*?)глас`)
	reGlorySec  = regexp.MustCompile(`^слава,\s*([^,:]+?),\s*глас`)
	reRubric    = regexp.MustCompile(`^(слава|и ныне|ныне|на |по |стих|канон|песнь|ирмос|икос|седален|светилен|прокимен|подобен|тропар|кондак|ин |богородичен|крестобогородичен|величание|блажен|таже|аще|в той же день|месяца|на литургии|аллилуиа|причастен|задостойник|стихира|самогласен|глас|конец)`)
)

// ParseMenaionDay extracts troparia and kontakia from one day file. service
// is the file stem ("25", "23p").
func ParseMenaionDay(path string, month int, service string) ([]corpus.Unit, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	hasRed := false
	for _, p := range d.Paras {
		for _, s := range p {
			if s.Red {
				hasRed = true
			}
		}
	}
	isHeading := func(p hip.Paragraph) bool {
		if hasRed {
			return len([]rune(hip.PlainString(p.RedPrefix()))) > 3
		}
		return reRubric.MatchString(hip.PlainString(p.Text()))
	}

	prefix := fmt.Sprintf("me.%02d.%s", month, service)
	title := dayTitle(d.Paras)
	var units []corpus.Unit
	counts := map[string]int{}
	lastTone := 0
	prevKind := ""
	seen := map[string]bool{}

	for i := 0; i < len(d.Paras); i++ {
		p := d.Paras[i]
		text := pageRef.ReplaceAllString(p.Text(), "")
		plain := hip.PlainString(text)
		kind, of := "", ""
		// Long paragraphs are Typikon instructions that merely mention a
		// troparion ("на утрени … тропарь воскресен гласа, дважды …").
		const maxHeading = 160
		long := len([]rune(plain)) > maxHeading && !hasInlineHymn(p, text, hasRed)
		switch {
		case long:
			if isHeading(p) {
				prevKind = ""
			}
			continue
		case reTroparion.MatchString(plain):
			kind, of = "troparion", strings.TrimSpace(reTroparion.FindStringSubmatch(plain)[1])
		case reKontakion.MatchString(plain):
			kind, of = "kontakion", strings.TrimSpace(reKontakion.FindStringSubmatch(plain)[1])
		case prevKind != "" && reGlorySec.MatchString(plain):
			// "Слава, преподобнаго, глас тойже:" right after a troparion or
			// kontakion introduces the one of the second saint.
			kind, of = prevKind, strings.TrimSpace(reGlorySec.FindStringSubmatch(plain)[1])
		default:
			if isHeading(p) {
				prevKind = ""
			}
			continue
		}
		heading, inline := splitHeading(p, text, hasRed)
		tone := toneOf(heading)
		if tone == 0 && strings.Contains(hip.PlainString(heading), "тойже") {
			tone = lastTone
		}
		var body []string
		if inline != "" {
			body = append(body, inline)
		}
		j := i + 1
		// A troparion or kontakion is a single paragraph in these files; what
		// follows is the next hymn or a Typikon instruction.
		if inline == "" && j < len(d.Paras) && !isHeading(d.Paras[j]) {
			body = append(body, pageRef.ReplaceAllString(d.Paras[j].Text(), ""))
			j++
		}
		i = j - 1
		b := CleanText(strings.Join(body, "\n"))
		if b == "" {
			continue
		}
		// The same hymn repeated later in the service (e.g. «на благословение
		// хлебов тропарь») is stored once.
		// A cue ("Днесь благоволения Божия предображение: трижды:") points to
		// a hymn given elsewhere; it is not a text.
		if strings.HasSuffix(b, ":") && len(strings.Fields(b)) <= 10 {
			continue
		}
		key := kind + "\x00" + incipitKey(b)
		if seen[key] {
			continue
		}
		seen[key] = true
		counts[kind]++
		lastTone = tone
		prevKind = kind
		u := corpus.Unit{
			ID:       fmt.Sprintf("%s.%s.%d", prefix, kind, counts[kind]),
			Book:     "menaion",
			Kind:     kind,
			Tone:     tone,
			Podoben:  podobenOf(heading),
			Lang:     "cu",
			Body:     b,
			SourceID: "orthlib.mineya",
			Meta: map[string]any{
				"service": service, "heading": CleanText(heading), "title": title, "file": filepath.Base(path),
			},
		}
		if of != "" {
			u.Meta["of"] = of
		}
		units = append(units, u)
	}
	return units, nil
}

// incipitKey identifies a hymn by its first words (variant readings of the
// same hymn differ further on).
func incipitKey(body string) string {
	w := strings.FieldsFunc(hip.PlainString(body), func(r rune) bool { return !unicode.IsLetter(r) })
	if len(w) > 6 {
		w = w[:6]
	}
	return strings.Join(w, " ")
}

// hasInlineHymn reports whether a heading paragraph also carries the hymn.
func hasInlineHymn(p hip.Paragraph, text string, hasRed bool) bool {
	_, inline := splitHeading(p, text, hasRed)
	return inline != ""
}

// splitHeading separates the rubric of a heading paragraph from text that
// follows it in the same paragraph (when the hymn starts right after it).
func splitHeading(p hip.Paragraph, text string, hasRed bool) (heading, inline string) {
	if hasRed {
		red := p.RedPrefix()
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), red))
		// A podoben incipit continues the rubric in black ("Подобен: Д|ева днесь:").
		if len(strings.Fields(rest)) >= 8 && strings.HasSuffix(strings.TrimSpace(red), ":") {
			return red, rest
		}
		return text, ""
	}
	// No colour: the rubric ends at the first ':'.
	if k := strings.Index(text, ":"); k >= 0 {
		rest := strings.TrimSpace(text[k+1:])
		if len(strings.Fields(rest)) >= 8 {
			return text[:k+1], rest
		}
	}
	return text, ""
}

var toneLetters = map[rune]int{'а': 1, 'в': 2, 'г': 3, 'д': 4, 'е': 5, 'є': 5, 'ѕ': 6, 'з': 7, 'и': 8}

// toneOf reads "гла́съ д҃" from a heading (Unicode, before folding, so that
// ѕ = 6 and з = 7 stay distinct).
func toneOf(heading string) int {
	words := strings.Fields(heading)
	for i, w := range words {
		if strings.HasPrefix(hip.PlainString(w), "глас") && i+1 < len(words) {
			var base []rune
			for _, r := range words[i+1] {
				if unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) {
					base = append(base, unicode.ToLower(r))
				}
			}
			if len(base) == 1 {
				return toneLetters[base[0]]
			}
			return 0
		}
	}
	return 0
}

func podobenOf(heading string) string {
	plain, pos := hip.Plain(heading)
	k := strings.Index(plain, "подобен:")
	if k < 0 {
		return ""
	}
	rs := []rune(heading)
	start := pos[len([]rune(plain[:k+len("подобен:")]))-1] + 1
	return strings.Trim(strings.TrimSpace(string(rs[start:])), ":. ")
}

// dayTitle collects the commemoration title: the red paragraphs after the
// "Месяца того же в N-й день" line.
func dayTitle(paras []hip.Paragraph) string {
	var parts []string
	started := false
	for _, p := range paras {
		pl := hip.PlainString(p.Text())
		if strings.HasPrefix(pl, "месяца") {
			started = true
			continue
		}
		if !started {
			continue
		}
		if strings.HasPrefix(pl, "на ") || strings.HasPrefix(pl, "служба") || strings.HasPrefix(pl, "стихиры") || len(parts) >= 3 {
			break
		}
		parts = append(parts, CleanText(p.Text()))
	}
	return strings.Join(parts, " ")
}

// SortUnits orders units by id for stable output.
func SortUnits(us []corpus.Unit) {
	sort.Slice(us, func(i, j int) bool { return us[i].ID < us[j].ID })
}
