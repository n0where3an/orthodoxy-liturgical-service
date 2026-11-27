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

// The Octoechos's hymns of the matins with «Аллилуиа» (the appendix):
//
//   - «Песни троичны» of the eight tones (trsedm.hip): oc.<t>.trinity.1,
//     .glory, .now; the first «во вся седмицы» with the day's ending
//     («в понедельник: предстательствы безплотных твоих помилуй нас…»):
//     oc.<t>.trinity.1.<mon|tue|wed|thu|fri>;
//   - «Светильны подобне по гласу» (svetsedm.hip), printed with Monday's
//     ending («предстательствы безплотных твоих, и спаси мя»), the other
//     days' put in its place: oc.<t>.photagogikon.<day>.
var (
	reOLTrinityHead = regexp.MustCompile(`^песни тр\S*чны\.?\s*глас`)
	reOLSvetHead    = regexp.MustCompile(`^(?:светилен,\s*)?глас \S+:`)
	reOLMonday      = regexp.MustCompile(`предстательствы безплотных твоих,? и спси мя`)
	reOLMonday6     = regexp.MustCompile(`предстательствы гдси,? безплотных твоих`)
)

var lentDays = []struct{ word, id string }{
	{"в понедельник", "mon"}, {"во вторник", "tue"}, {"в среду", "wed"}, {"в четверток", "thu"}, {"пяток", "fri"},
}

// dayEndings: the Unicode text after «в понедельник:» … up to the period,
// by day (Wednesday's serves Friday: «в среду же и пяток»).
func dayEndings(text string) map[string]string {
	plain := strings.ToLower(hip.PlainString(text))
	_, pos := hip.Plain(text)
	out := map[string]string{}
	for _, d := range lentDays {
		i := strings.Index(plain, d.word)
		if i < 0 {
			continue
		}
		j := strings.Index(plain[i:], ":")
		if j < 0 {
			continue
		}
		a := i + j + 1
		b := strings.Index(plain[a:], ".")
		if b < 0 {
			b = len(plain) - a
		}
		out[d.id] = lowerFirst(uni(text, pos, plain, a, a+b))
	}
	if out["fri"] == "" {
		out["fri"] = out["wed"]
	}
	return out
}

// ImportOctoechosLent parses the Trinity hymns and the photagogika.
func ImportOctoechosLent(root string) ([]corpus.Unit, error) {
	var units []corpus.Unit
	add := func(id, kind, file, body string, tone int) {
		units = append(units, corpus.Unit{ID: id, Book: "octoechos", Kind: kind, Tone: tone, Lang: "cu",
			Body: CleanText(body), SourceID: "orthlib.octoechos", Meta: map[string]any{"file": file}})
	}
	// The Trinity hymns.
	path := filepath.Join(root, "octoih", "oc", "trsedm.hip")
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	tone, n := 0, 0
	var endings map[string]string
	first := map[int]string{}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		switch {
		case reOLTrinityHead.MatchString(plain):
			tone, n = toneOf(text), 0
		case strings.HasPrefix(plain, "достоит же глаголати"):
			endings = dayEndings(text)
		case tone > 0 && strings.HasPrefix(plain, "слава"):
			add(fmt.Sprintf("oc.%d.trinity.glory", tone), "troparion", "trsedm.hip", tlInline(text, plain), tone)
		case tone > 0 && strings.HasPrefix(plain, "и ныне"):
			add(fmt.Sprintf("oc.%d.trinity.now", tone), "troparion", "trsedm.hip", tlInline(text, plain), tone)
		case tone > 0 && n == 0:
			n++
			first[tone] = text
			add(fmt.Sprintf("oc.%d.trinity.1", tone), "troparion", "trsedm.hip", text, tone)
		}
	}
	if len(first) != 8 || len(endings) != 5 {
		return units, fmt.Errorf("trsedm.hip: %d tones, %d day endings", len(first), len(endings))
	}
	for t, text := range first {
		base := strings.TrimRight(strings.TrimSpace(text), ".")
		for day, end := range endings {
			add(fmt.Sprintf("oc.%d.trinity.1.%s", t, day), "troparion", "trsedm.hip", base+", "+end+".", t)
		}
	}
	// The photagogika.
	path = filepath.Join(root, "octoih", "oc", "svetsedm.hip")
	if d, err = LoadDoc(path); err != nil {
		return units, err
	}
	endings = map[string]string{}
	got := 0
	pending := 0 // «светилен, глас а:» — the text in the next paragraph
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		for _, dd := range lentDays {
			if strings.HasPrefix(plain, dd.word) {
				for k, v := range dayEndings(text) {
					endings[k] = v
				}
			}
		}
		body, t := "", 0
		switch {
		case pending > 0:
			body, t, pending = text, pending, 0
		case reOLSvetHead.MatchString(plain):
			if b := tlInline(text, plain); b != "" {
				body, t = b, toneOf(text)
			} else {
				pending = toneOf(text)
			}
		}
		if body == "" || t == 0 {
			continue
		}
		bp := strings.ToLower(hip.PlainString(body))
		_, pos := hip.Plain(body)
		for _, dd := range lentDays {
			end := endings[dd.id]
			if dd.id == "fri" && end == "" {
				end = endings["wed"]
			}
			out := body
			if m := reOLMonday.FindStringIndex(bp); m != nil && dd.id != "mon" {
				out = uni(body, pos, bp, 0, m[0]) + " " + strings.TrimRight(end, ".") + uni(body, pos, bp, m[1], len(bp))
			} else if m := reOLMonday6.FindStringIndex(bp); m != nil && dd.id != "mon" {
				// «Предстательствы, Господи, безплотных Твоих, низпосли…»: the
				// day's intercessors without «и спаси мя».
				ep := strings.ToLower(hip.PlainString(end))
				_, epos := hip.Plain(end)
				core := end
				if i := strings.Index(ep, ", и спси мя"); i > 0 {
					core = uni(end, epos, ep, 0, i)
				}
				out = uni(body, pos, bp, 0, m[0]) + " " + core + uni(body, pos, bp, m[1], len(bp))
			}
			add(fmt.Sprintf("oc.%d.photagogikon.%s", t, dd.id), "exapostilarion", "svetsedm.hip", capitalizeFirst(strings.TrimSpace(out)), t)
		}
		got++
	}
	if got != 8 {
		return units, fmt.Errorf("svetsedm.hip: %d photagogika", got)
	}
	return units, nil
}

// lowerFirst lower-cases the first letter (a day's ending put inside a hymn).
func lowerFirst(s string) string {
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsLetter(r) {
			rs[i] = unicode.ToLower(r)
			break
		}
	}
	return string(rs)
}
