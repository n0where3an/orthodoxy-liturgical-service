// Package english imports English translations into the corpus under the
// same semantic keys as the Church Slavonic texts.
//
// Sources:
//   - Coverdale Psalter for Orthodox Worship (P. S. Thompson; public domain),
//     as text extracted from the PDF with `pdftotext` (LXX numbering,
//     arranged in kathismata);
//   - Menaion, tr. Isaac Lambertsen, in Markdown (github.com/typiconman/english-md, MIT).
package english

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

const coverdaleSource = "en.coverdale"

// CoverdaleSource describes the Coverdale Psalter.
var CoverdaleSource = corpus.Source{
	ID: coverdaleSource, Title: "Coverdale Psalter for Orthodox Worship",
	Edition: "Philip Silouan Thompson (LXX numbering, kathismata)", License: "public-domain",
	Note: "Released into the public domain by the editor.",
}

var (
	reKathismaEn = regexp.MustCompile(`^(\w+) Kathisma$`)
	reStasisEn   = regexp.MustCompile(`^(First|Second|Third) Stasis$`)
	rePsalmEn    = regexp.MustCompile(`^Psalm (\d+)$`)
)

var ordinals = map[string]int{
	"First": 1, "Second": 2, "Third": 3, "Fourth": 4, "Fifth": 5, "Sixth": 6, "Seventh": 7,
	"Eighth": 8, "Ninth": 9, "Tenth": 10, "Eleventh": 11, "Twelfth": 12, "Thirteenth": 13,
	"Fourteenth": 14, "Fifteenth": 15, "Sixteenth": 16, "Seventeenth": 17, "Eighteenth": 18,
	"Nineteenth": 19, "Twentieth": 20,
}

// Psalter is the parsed Coverdale text: psalms by LXX number (verses), and
// the psalm numbers of each stasis.
type Psalter struct {
	Psalms map[int][]string
	Stases map[[2]int][]int // {kathisma, stasis} -> psalms (a psalm may be split: see Parts)
	// Parts: when a psalm (118) is divided between stases, the verses of each part.
	Parts map[[2]int][]string
}

// ParseCoverdale reads the pdftotext output of the Coverdale Psalter.
func ParseCoverdale(path string) (*Psalter, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ps := &Psalter{Psalms: map[int][]string{}, Stases: map[[2]int][]int{}, Parts: map[[2]int][]string{}}
	kathisma, stasis, psalm := 0, 0, 0
	var cur []string   // verses of the current psalm part
	var verse []string // lines of the current verse
	inBody := false    // after the superscription
	flushVerse := func() {
		if len(verse) > 0 {
			v := strings.Join(verse, " ")
			cur = append(cur, v)
			verse = nil
		}
	}
	flushPart := func() {
		flushVerse()
		if psalm > 0 && len(cur) > 0 {
			ps.Psalms[psalm] = append(ps.Psalms[psalm], cur...)
			key := [2]int{kathisma, stasis}
			ps.Parts[key] = append(ps.Parts[key], cur...)
		}
		cur = nil
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\f", "\n"), "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
			// A blank line ends a superscription.
			if psalm > 0 && !inBody && len(verse) > 0 {
				verse = nil
			}
			if psalm > 0 {
				inBody = true
			}
			continue
		case strings.HasPrefix(line, "Coverdale Psalter for Orthodox Worship"), strings.HasPrefix(line, "This work is hereby released"),
			strings.HasPrefix(line, "To view a copy"), strings.HasPrefix(line, "http://creativecommons"), strings.HasPrefix(line, "Commons, 559"):
			continue
		}
		if m := reKathismaEn.FindStringSubmatch(line); m != nil && ordinals[m[1]] > 0 && ordinals[m[1]] <= kathisma {
			// A misprint in the source ("Second Kathisma" inside the 18th) is
			// a stasis heading.
			flushPart()
			stasis = ordinals[m[1]]
			continue
		}
		if m := reKathismaEn.FindStringSubmatch(line); m != nil && ordinals[m[1]] > 0 {
			flushPart()
			kathisma, stasis, psalm = ordinals[m[1]], 0, 0
			continue
		}
		if m := reStasisEn.FindStringSubmatch(line); m != nil {
			flushPart()
			stasis = ordinals[m[1]]
			continue
		}
		if m := rePsalmEn.FindStringSubmatch(line); m != nil {
			flushPart()
			psalm, _ = strconv.Atoi(m[1])
			key := [2]int{kathisma, stasis}
			ps.Stases[key] = append(ps.Stases[key], psalm)
			// A superscription follows unless the next line already begins
			// the text in capitals ("BLESSED is the man…").
			inBody = startsWithCapsWord(nextNonEmpty(lines, i+1))
			continue
		}
		if psalm == 0 {
			continue
		}
		if !inBody {
			if startsWithCapsWord(line) {
				inBody = true
			} else {
				verse = append(verse, line) // superscription, dropped on the blank line
				continue
			}
		}
		if len(cur) == 0 && len(verse) == 0 {
			line = fixInitialCaps(line)
		}
		verse = append(verse, line)
		if strings.ContainsAny(line[len(line)-1:], ".?!") {
			flushVerse()
		}
	}
	flushPart()
	if len(ps.Psalms) < 150 {
		return ps, fmt.Errorf("coverdale: %d psalms parsed", len(ps.Psalms))
	}
	return ps, nil
}

func nextNonEmpty(lines []string, i int) string {
	for ; i < len(lines); i++ {
		if s := strings.TrimSpace(lines[i]); s != "" {
			return s
		}
	}
	return ""
}

// startsWithCapsWord: the first word of a psalm is set in capitals.
func startsWithCapsWord(s string) bool {
	w := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
	if len(w) == 0 {
		return false
	}
	first := w[0]
	if len(w) > 1 && first == "O" {
		first = w[1]
	}
	return len([]rune(first)) >= 2 && strings.ToUpper(first) == first
}

// fixInitialCaps turns "BLESSED is" into "Blessed is", "O HOW amiable" into
// "O how amiable".
func fixInitialCaps(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		letters := strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) })
		if len([]rune(letters)) < 2 || strings.ToUpper(letters) != letters {
			if i > 1 || (i == 1 && words[0] != "O") {
				break
			}
			continue
		}
		rs := []rune(strings.ToLower(w))
		if i == 0 {
			rs[0] = unicode.ToUpper(rs[0])
		}
		words[i] = string(rs)
		if letters == "LORD" || letters == "GOD" {
			words[i] = strings.Replace(words[i], strings.ToLower(letters), string(letters[0])+strings.ToLower(letters[1:]), 1)
		}
	}
	return strings.Join(words, " ")
}

// coverdaleUnits maps the Psalter to corpus keys: kathisma stases, and the
// psalms read at the hours and vespers.
func coverdaleUnits(ps *Psalter) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	add := func(key string, verses []string) {
		if len(verses) == 0 {
			warnings = append(warnings, "coverdale: nothing for "+key)
			return
		}
		units = append(units, corpus.Unit{ID: key, Lang: "en", Body: strings.Join(verses, "\n"), SourceID: coverdaleSource})
	}
	for k := 1; k <= 20; k++ {
		for s := 1; s <= 3; s++ {
			add(fmt.Sprintf("ps.kathisma.%d.%d", k, s), ps.Parts[[2]int{k, s}])
		}
	}
	for key, n := range map[string]int{
		"ch.prime.psalm.5": 5, "ch.prime.psalm.89": 89, "ch.prime.psalm.100": 100,
		"ch.terce.psalm.16": 16, "ch.terce.psalm.24": 24, "ch.terce.psalm.50": 50,
		"ch.sext.psalm.53": 53, "ch.sext.psalm.54": 54, "ch.sext.psalm.90": 90,
		"ch.none.psalm.83": 83, "ch.none.psalm.84": 84, "ch.none.psalm.85": 85,
		"ch.vespers.psalm.103": 103,
	} {
		add(key, ps.Psalms[n])
	}
	if v := ps.Psalms[85]; len(v) > 0 {
		add("ch.none.psalm.85.repeat", v[len(v)-1:])
	}
	return units, warnings
}

// ImportCoverdale parses the text file and returns the units.
func ImportCoverdale(path string) ([]corpus.Unit, []string, error) {
	ps, err := ParseCoverdale(path)
	if err != nil {
		return nil, nil, err
	}
	units, warnings := coverdaleUnits(ps)
	return units, warnings, nil
}
