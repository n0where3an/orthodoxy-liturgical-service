package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The irmoi of the Irmologion by tone (01glas.hip … 08glas.hip: «начало
// ирмосов перваго гласа. песнь а. …»): ir.irmos.<tone>.<ode>.<n>. The
// Octoechos, the Menaion and the Triodion often give an irmos by its first
// words only («ирмос: Яко по суху:»); SupplementIrmoi completes them.
var (
	reIrmTone     = regexp.MustCompile(`^(\d)glas\.hip$`)
	reIrmOde      = regexp.MustCompile(`^песнь ([а-ѳ])[.:]?\s*$`)
	reIrmNote     = regexp.MustCompile(`^\[[^\]\[]*[\]\[]\s*`) // «[вознесения:[» too
	reIrmHeadOnly = regexp.MustCompile(`^\[[^\]]*\]:?\s*$`)
)

// ImportIrmoi parses the eight tones of the Irmologion.
func ImportIrmoi(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for tone := 1; tone <= 8; tone++ {
		file := fmt.Sprintf("%02dglas.hip", tone)
		path := filepath.Join(root, "irmologij", "irmologij", file)
		d, err := LoadDoc(path)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		ode, n := 0, 0
		for _, p := range d.Paras {
			text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
			plain := strings.ToLower(hip.PlainString(text))
			if plain == "" {
				continue
			}
			if reIrmOde.MatchString(plain) {
				m := reIrmOde.FindStringSubmatchIndex(plain)
				_, pos := hip.Plain(text)
				ode, n = uniNumeral(text, pos, plain, m[2], m[3]), 0
				continue
			}
			if ode == 0 || reIrmHeadOnly.MatchString(plain) || len(strings.Fields(plain)) < 6 {
				continue
			}
			// «[в ржство хрстово:] Христос раждается…» — the note is not the text.
			body := text
			if m := reIrmNote.FindStringIndex(plain); m != nil {
				_, pos := hip.Plain(text)
				body = uni(text, pos, plain, m[1], len(plain))
			}
			n++
			units = append(units, corpus.Unit{ID: fmt.Sprintf("ir.irmos.%d.%d.%d", tone, ode, n), Book: "irmologion", Kind: "irmos",
				Tone: tone, Lang: "cu", Body: strings.TrimSpace(body), SourceID: "orthlib.irmologion",
				Meta: map[string]any{"file": file}})
		}
	}
	return units, warnings
}

// irmosCue: an irmos given by its first words (a short text ending with a
// colon, or a few words).
func irmosCue(body string) bool {
	b := strings.TrimSpace(body)
	w := len(strings.Fields(b))
	return w > 0 && (w <= 6 || strings.HasSuffix(b, ":") && w <= 14)
}

var reUnitOde = regexp.MustCompile(`\.ode\.(\d)\.irmos$`)
var reOctTone = regexp.MustCompile(`^oc\.(\d)\.`)

// SupplementIrmoi completes the irmoi given by their first words from the
// Irmologion: the irmos of the same ode whose text begins with the cue, of
// the unit's tone first (any tone when one only matches). Meta
// "completed_from" names the Irmologion's unit. Returns the number completed
// and the cues left (warnings).
func SupplementIrmoi(units []corpus.Unit) ([]corpus.Unit, []string) {
	byOde := map[int][]irm{}
	for _, u := range units {
		if !strings.HasPrefix(u.ID, "ir.irmos.") {
			continue
		}
		parts := strings.Split(u.ID, ".")
		ode, _ := strconv.Atoi(parts[3])
		byOde[ode] = append(byOde[ode], irm{u.ID, letters(u.Body), u.Tone, u.Body})
	}
	var warnings []string
	for i, u := range units {
		m := reUnitOde.FindStringSubmatch(u.ID)
		if m == nil || strings.HasPrefix(u.ID, "ir.") || !irmosCue(u.Body) {
			continue
		}
		ode, _ := strconv.Atoi(m[1])
		cue := letters(u.Body)
		if len([]rune(cue)) < 5 {
			continue
		}
		tone := u.Tone
		if t := reOctTone.FindStringSubmatch(u.ID); t != nil {
			tone, _ = strconv.Atoi(t[1])
		}
		var same, all []irm
		for _, c := range byOde[ode] {
			if strings.HasPrefix(c.fold, cue) {
				all = append(all, c)
				if c.tone == tone {
					same = append(same, c)
				}
			}
		}
		var got *irm
		switch {
		case len(same) > 0:
			got = &same[0]
		case len(all) > 0 && (tone == 0 || len(all) == 1 || sameIrmos(all)):
			got = &all[0]
		}
		// The cue abbreviates or spells otherwise («прпдбным», «неверным»):
		// its first words, while they still pick one irmos of the tone.
		for w := strings.Fields(cue); got == nil && tone > 0 && len(w) > 2; {
			w = w[:len(w)-1]
			short := strings.Join(w, " ")
			var hit []irm
			for _, c := range byOde[ode] {
				if c.tone == tone && strings.HasPrefix(c.fold, short) {
					hit = append(hit, c)
				}
			}
			if len(hit) == 1 {
				got = &hit[0]
			}
			if len(hit) > 1 {
				break
			}
		}
		if got == nil {
			warnings = append(warnings, fmt.Sprintf("%s: irmos «%s» not in the Irmologion (tone %d, ode %d)", u.ID, strings.TrimSpace(u.Body), tone, ode))
			continue
		}
		units[i].Body = got.body
		if units[i].Meta == nil {
			units[i].Meta = map[string]any{}
		}
		units[i].Meta["completed_from"] = got.id
	}
	return units, warnings
}

// letters: the folded plain text, letters and single spaces only (the cue
// and the full text differ in punctuation: «несть свят, якоже ты»).
func letters(body string) string {
	var b strings.Builder
	space := false
	for _, r := range hip.Fold(strings.ToLower(hip.PlainString(body))) {
		switch {
		case unicode.IsLetter(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			if r == 'ы' { // «юношы» / «юноши»
				r = 'и'
			}
			b.WriteRune(r)
			space = false
		default:
			space = true
		}
	}
	return b.String()
}

// irm: an irmos of the Irmologion (fold: its letters, for matching).
type irm struct {
	id, fold string
	tone     int
	body     string
}

// sameIrmos: the candidates are one irmos printed in several places (their
// first 60 letters agree).
func sameIrmos(cs []irm) bool {
	head := func(s string) string {
		r := []rune(s)
		return string(r[:min(60, len(r))])
	}
	for _, c := range cs[1:] {
		if head(c.fold) != head(cs[0].fold) {
			return false
		}
	}
	return true
}
