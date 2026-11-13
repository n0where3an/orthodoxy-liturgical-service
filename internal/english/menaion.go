package english

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

const lambertsenSource = "en.lambertsen"

// LambertsenSource describes the English Menaion.
var LambertsenSource = corpus.Source{
	ID: lambertsenSource, Title: "Menaion (tr. Isaac E. Lambertsen)",
	Edition: "Markdown edition, github.com/typiconman/english-md", URL: "https://github.com/typiconman/english-md",
	License: "MIT (repository); translation by I. Lambertsen",
	Note:    "Aligned to the Slavonic services by day, tone and order; additional English services are not used.",
}

// hymn is one hymn of an English service.
type hymn struct {
	part string // troparion, kontakion, vespers.lic.sticheron, vespers.lic.glory, …
	tone int
	text string
}

type enService struct {
	month, day int
	title      string
	hymns      []hymn
}

var (
	reToneEn     = regexp.MustCompile(`\bTone ([IVX]+)\b`)
	reLICEn      = regexp.MustCompile(`^On “Lord, I have cried`)
	reAposEn     = regexp.MustCompile(`^(At|On) the aposticha`)
	reStichosEn  = regexp.MustCompile(`^\*Stichos:\*\s*`)
	reTropEn     = regexp.MustCompile(`^(Another )?[Tt]roparion\b`)
	reKontEn     = regexp.MustCompile(`^(Another )?[Kk]ontakion\b`)
	reGloryEn    = regexp.MustCompile(`^Glory…`)
	reNowEn      = regexp.MustCompile(`^Now & ever…`)
	reStopVesp   = regexp.MustCompile(`^(At Matins|Ode I\b|At Compline|At the Liturgy|At Great Vespers|At the Litiya)`)
	reVespStart  = regexp.MustCompile(`^At (Little )?Vespers`)
	reMarkup     = regexp.MustCompile(`\*([^*]+)\*`)
	reTwice      = regexp.MustCompile(`\s*\*(Twice|Thrice)\*\s*$`)
	roman        = map[string]int{"I": 1, "II": 2, "III": 3, "IV": 4, "V": 5, "VI": 6, "VII": 7, "VIII": 8}
	monthsByName = map[string]time.Month{}
)

func init() {
	for m := time.January; m <= time.December; m++ {
		monthsByName[m.String()] = m
	}
}

func toneEn(s string) int {
	if m := reToneEn.FindStringSubmatch(s); m != nil {
		return roman[m[1]]
	}
	return 0
}

func cleanMd(s string) string {
	s = reTwice.ReplaceAllString(s, "")
	s = strings.NewReplacer(`\(`, "(", `\)`, ")", `\*`, "*", `\[`, "[", `\]`, "]").Replace(s)
	s = reMarkup.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// isHeading: a rubric paragraph. Rubrics announcing a hymn end with "—".
func isHeading(p string) bool {
	return strings.HasSuffix(p, "—") || reGloryEn.MatchString(p) || reNowEn.MatchString(p) ||
		reLICEn.MatchString(p) || reAposEn.MatchString(p) || reTropEn.MatchString(p) || reKontEn.MatchString(p) ||
		reVespStart.MatchString(p) || reStopVesp.MatchString(p) || len(strings.Fields(p)) <= 4
}

// parseService reads one chapter (Markdown) of the English Menaion.
func parseService(path string) []hymn {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var paras []string
	for _, p := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n\n") {
		if p = strings.TrimSpace(strings.ReplaceAll(p, "\n", " ")); p != "" {
			paras = append(paras, p)
		}
	}
	var hymns []hymn
	section := "" // "", lic, aposticha
	expect := ""  // part the next text paragraph belongs to
	tone := 0
	vespersDone := false
	inVespers := false
	for _, p := range paras {
		if reVespStart.MatchString(p) && !vespersDone {
			inVespers = true
			continue
		}
		if reStopVesp.MatchString(p) {
			if inVespers {
				vespersDone = true
			}
			inVespers, section = false, ""
		}
		if m := reStichosEn.FindStringIndex(p); m != nil && section == "aposticha" {
			hymns = append(hymns, hymn{part: "vespers.aposticha.verse", text: cleanMd(p[m[1]:])})
			continue
		}
		if isHeading(p) {
			if t := toneEn(p); t > 0 {
				tone = t
			}
			wantsText := strings.HasSuffix(p, "—")
			switch {
			case reTropEn.MatchString(p):
				expect = "troparion"
			case reKontEn.MatchString(p):
				expect = "kontakion"
			case inVespers && reLICEn.MatchString(p):
				section, expect = "lic", "vespers.lic.sticheron"
			case inVespers && reAposEn.MatchString(p):
				section = "aposticha"
				expect = "vespers.aposticha.sticheron"
				if strings.Contains(p, "Octoechos") {
					expect = ""
				}
			case inVespers && section != "" && reGloryEn.MatchString(p):
				switch {
				case strings.Contains(p, "Now & ever") && strings.Contains(p, "Theotokion"):
					expect = "vespers." + section + ".theotokion"
				case strings.Contains(p, "Now & ever"):
					expect = "vespers." + section + ".glory_now"
				default:
					expect = "vespers." + section + ".glory"
				}
			case inVespers && section != "" && reNowEn.MatchString(p):
				expect = "vespers." + section + ".theotokion"
			case strings.Contains(p, "Theotokion") || strings.Contains(p, "theotokion"):
				expect = "" // theotokia outside vespers are not aligned
			default:
				if wantsText && expect == "" {
					expect = "" // other rubric
				}
			}
			if !wantsText && expect != "vespers.lic.sticheron" && expect != "vespers.aposticha.sticheron" {
				expect = ""
			}
			continue
		}
		if expect == "" {
			continue
		}
		hymns = append(hymns, hymn{part: expect, tone: tone, text: cleanMd(p)})
		if !strings.HasSuffix(expect, ".sticheron") {
			expect = ""
		}
	}
	return hymns
}

type manifestXML struct {
	Chapters []struct {
		File string `xml:"file,attr"`
		Name string `xml:"name,attr"`
	} `xml:"manifest>chapter"`
}

var reChapterDate = regexp.MustCompile(`^([A-Z][a-z]+) (\d+):`)

// LoadServices reads all English Menaion services, in manifest order.
func LoadServices(root string) ([]enService, error) {
	dirs, err := filepath.Glob(filepath.Join(root, "MenaionLambertsen*"))
	if err != nil {
		return nil, err
	}
	var out []enService
	for _, dir := range dirs {
		b, err := os.ReadFile(filepath.Join(dir, "manifest.xml"))
		if err != nil {
			return nil, err
		}
		var m manifestXML
		if err := xml.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("%s: %w", dir, err)
		}
		for _, c := range m.Chapters {
			md := reChapterDate.FindStringSubmatch(c.Name)
			if md == nil {
				continue // "Sunday After November 1: …" and other movable services
			}
			mon, ok := monthsByName[md[1]]
			if !ok {
				continue
			}
			d, _ := strconv.Atoi(md[2])
			out = append(out, enService{month: int(mon), day: d, title: c.Name,
				hymns: parseService(filepath.Join(dir, "chapters", c.File+".md"))})
		}
	}
	return out, nil
}

// cuItem is a Slavonic unit to be matched.
type cuItem struct {
	id   string
	part string
	tone int
}

// Align matches English hymns to the Slavonic Menaion units (by day, part,
// tone and order) and returns English translations under the Slavonic keys.
func Align(services []enService, cu []corpus.Unit) ([]corpus.Unit, map[string]int) {
	stats := map[string]int{}
	// Slavonic items by month-day.
	byDay := map[[2]int][]cuItem{}
	for _, u := range cu {
		if u.Book != "menaion" || u.Kind == "marker" {
			continue
		}
		parts := strings.Split(u.ID, ".") // me.MM.DD[sfx].<part…>[.N]
		if len(parts) < 4 {
			continue
		}
		mon, _ := strconv.Atoi(parts[1])
		day, _ := strconv.Atoi(strings.TrimRight(parts[2], "abcdefghijklmnopqrstuvwxyzабвгдежзийклмнопрстуфхцчшщъыьэюя"))
		part := strings.Join(parts[3:], ".")
		if _, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			part = strings.Join(parts[3:len(parts)-1], ".")
		}
		byDay[[2]int{mon, day}] = append(byDay[[2]int{mon, day}], cuItem{id: u.ID, part: part, tone: u.Tone})
	}
	enByDay := map[[2]int][]hymn{}
	for _, s := range services {
		enByDay[[2]int{s.month, s.day}] = append(enByDay[[2]int{s.month, s.day}], s.hymns...)
	}
	var out []corpus.Unit
	for day, items := range byDay {
		en := enByDay[day]
		used := make([]bool, len(en))
		last := map[string]int{} // part -> index of the last English hymn used (keeps order)
		sort.SliceStable(items, func(i, j int) bool { return itemLess(items[i].id, items[j].id) })
		for _, it := range items {
			stats[it.part+".cu"]++
			start := last[it.part]
			match := -1
			for k := start; k < len(en); k++ {
				if used[k] || en[k].part != it.part {
					continue
				}
				if it.tone != 0 && en[k].tone != 0 && en[k].tone != it.tone {
					continue
				}
				match = k
				break
			}
			if match < 0 {
				continue
			}
			used[match] = true
			last[it.part] = match + 1
			stats[it.part+".en"]++
			out = append(out, corpus.Unit{ID: it.id, Lang: "en", Body: en[match].text, SourceID: lambertsenSource})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, stats
}

// itemLess orders Slavonic keys: main service ("25") before additional ones
// ("25p"), then by key with numeric suffixes compared as numbers.
func itemLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	if pa[2] != pb[2] {
		return len(pa[2]) < len(pb[2]) || (len(pa[2]) == len(pb[2]) && pa[2] < pb[2])
	}
	for i := 3; i < len(pa) && i < len(pb); i++ {
		if pa[i] == pb[i] {
			continue
		}
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		if ea == nil && eb == nil {
			return na < nb
		}
		return pa[i] < pb[i]
	}
	return len(pa) < len(pb)
}
