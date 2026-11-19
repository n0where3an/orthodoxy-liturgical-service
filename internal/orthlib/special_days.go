package orthlib

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The Saturdays and Sundays with readings of their own around the
// Exaltation, the Nativity and the Theophany («неделя пред ржством хрстовым
// стых отец», «суббота по просвещении»). The Menaion prints them after the
// service of a day (09.08, 09.15, 12.18, 12.26, 01.01, 01.07): a heading and
// the rubric of the liturgy. Units under ms.<id>.lit (rubric: a marker whose
// Meta holds the citations, resolved by ResolveLiturgy into prokeimenon,
// apostle, alleluia.verse.N, gospel, kinonik). The hymns of the Sundays of
// the Fathers and after the Nativity are cut by the manifest (ms.<id>.troparion,
// .kontakion).

// reSpecialHead: the heading of such a Saturday or Sunday (a short paragraph).
var reSpecialHead = regexp.MustCompile(`^(?:\[зри\] )?(?:м\\р )?(неделя|суббота) (пред|по) ([^.:]{3,45})[.:]?$`)

// specialDays maps the headings to the ids of the days.
var specialDays = []struct {
	re *regexp.Regexp
	id string
}{
	{regexp.MustCompile(`^суббота пред воздвиж`), "sat_before_exaltation"},
	{regexp.MustCompile(`^неделя пред воздвиж`), "before_exaltation"},
	{regexp.MustCompile(`^суббота по воздвиж`), "sat_after_exaltation"},
	{regexp.MustCompile(`^неделя по воздвиж`), "after_exaltation"},
	{regexp.MustCompile(`^суббота пред ржств`), "sat_before_nativity"},
	{regexp.MustCompile(`^неделя пред ржств`), "fathers"},
	{regexp.MustCompile(`^суббота по ржств`), "sat_after_nativity"},
	{regexp.MustCompile(`^неделя по ржств`), "after_nativity"},
	{regexp.MustCompile(`^суббота пред бгоявл`), "sat_before_theophany"},
	{regexp.MustCompile(`^неделя пред бгоявл`), "before_theophany"},
	{regexp.MustCompile(`^суббота по просвещ`), "sat_after_theophany"},
	{regexp.MustCompile(`^неделя по просвещ`), "after_theophany"},
}

var specialDayFiles = []string{
	"mineya/sep/sep/Сентябрь/08.hip", "mineya/sep/sep/Сентябрь/15.hip",
	"mineya/dec/dec/Декабрь/18.hip", "mineya/dec/dec/Декабрь/26.hip",
	"mineya/jan/jan/Январь/01.hip", "mineya/jan/jan/Январь/07.hip",
}

// specialHeading returns the id of the special day a paragraph heads.
func specialHeading(plain string) string {
	plain = strings.TrimSpace(plain)
	if !reSpecialHead.MatchString(plain) {
		return ""
	}
	plain = strings.TrimPrefix(strings.TrimPrefix(plain, "[зри] "), `м\р `)
	for _, s := range specialDays {
		if s.re.MatchString(plain) {
			return s.id
		}
	}
	return ""
}

// specialDayStart: the index of the first paragraph of a special day's
// section in a day of the Menaion, 0 if none.
func specialDayStart(d *Doc) int {
	for i, p := range d.Paras {
		if specialHeading(hip.PlainString(p.Text())) != "" {
			return i
		}
	}
	return 0
}

var (
	reSpecKinonik = regexp.MustCompile(`причастен(?: праздника| дне)?:\s*(.+?)[:.]`)
	reSpecOther   = regexp.MustCompile(`другий:\s*(.+?)[:.]`)
)

// ImportSpecialDays parses the liturgy rubrics of the special Saturdays and
// Sundays. Of several rubrics under one heading the fullest is taken (the
// one of the service printed in full gives the verses).
func ImportSpecialDays(root string) ([]corpus.Unit, []string) {
	var warnings []string
	best := map[string]corpus.Unit{}
	var order []string
	for _, file := range specialDayFiles {
		d, err := LoadDoc(filepath.Join(root, file))
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		id := ""
		for _, p := range d.Paras {
			text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
			plain := strings.ToLower(hip.PlainString(text))
			if h := specialHeading(plain); h != "" {
				id = h
				continue
			}
			if id == "" || !strings.Contains(plain, "зачало") {
				continue
			}
			var got []corpus.Unit
			_, pos := hip.Plain(text)
			parseLiturgyRubricWith(reSpecProk, text, plain, pos, func(key, kind, body string, meta map[string]any) {
				got = append(got, corpus.Unit{ID: "ms." + id + ".lit." + key, Book: "menaion", Kind: kind, Lang: "cu",
					Body: CleanText(body), SourceID: "orthlib.mineya", Meta: meta})
			})
			if len(got) != 1 {
				continue
			}
			u := got[0]
			u.Meta["file"] = filepath.Base(file)
			delete(u.Meta, "beatitudes") // the Octoechos's on Sunday
			delete(u.Meta, "zadostoinik")
			// «зачало первое»
			if _, ok := u.Meta["gospel"]; !ok || u.Meta["gospel"] == 0 {
				if b, n := gospelFirst(plain); n > 0 {
					u.Meta["gospel_book"], u.Meta["gospel"], u.Meta["gospel_half"] = b, n, false
				}
			}
			// The communion verse of the day itself: not «Хвалите» (the
			// Sunday's), «дне» or «праздника» (taken from the day).
			delete(u.Meta, "kinonik")
			if m := reSpecKinonik.FindStringSubmatchIndex(plain); m != nil {
				k := plain[m[2]:m[3]]
				a, b := m[2], m[3]
				if strings.HasPrefix(k, "хвалите") {
					a, b = -1, -1
					if o := reSpecOther.FindStringSubmatchIndex(plain[m[3]:]); o != nil {
						a, b = m[3]+o[2], m[3]+o[3]
					}
				}
				if a >= 0 && !strings.HasPrefix(plain[a:b], "дне") && !strings.HasPrefix(plain[a:b], "праздника") {
					u.Meta["kinonik"] = uni(text, pos, plain, a, b)
				}
			}
			if prev, ok := best[u.ID]; !ok {
				order = append(order, u.ID)
				best[u.ID] = u
			} else if len(u.Meta) > len(prev.Meta) {
				best[u.ID] = u
			}
		}
	}
	var units []corpus.Unit
	for _, id := range order {
		units = append(units, best[id])
	}
	return units, warnings
}

var reGospelFirst = regexp.MustCompile(`(?:еиглие|евангелие) (?:от )?(матфеа|марка|луки|иоанна)[^.]*?зачало первое`)

func gospelFirst(plain string) (string, int) {
	if m := reGospelFirst.FindStringSubmatch(plain); m != nil {
		return gospelByName[m[1]], 1
	}
	return "", 0
}
