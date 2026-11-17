package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Verse is one verse of the liturgical Apostol or Gospel.
type Verse struct {
	Text string // book wording
	// Incipit: the liturgical beginning of a reading that starts at this
	// verse ("Братие, …", "Во время оно, …"); it replaces the first Replace
	// runes of Text (0: it is put in front).
	Incipit string
	Replace int
}

// Book is a book of the Apostol or Gospel: chapter -> verse -> text.
type Book struct {
	Name     string
	Chapters map[int]map[int]*Verse
	// Zachala: the pericopes marked in the text, in order.
	Zachala []Zachalo
}

// Zachalo is the start of a pericope (зачало) of the liturgical books.
type Zachalo struct {
	N    int
	Half bool // «от полу»: the second half, marked without a number
	// Start: the first verse.
	Chapter, Verse int
}

// slavonic numeral letters and their values
var numeral = map[rune]int{
	'а': 1, 'в': 2, 'г': 3, 'д': 4, 'є': 5, 'ѕ': 6, 'з': 7, 'и': 8, 'ѳ': 9, 'і': 10,
	'к': 20, 'л': 30, 'м': 40, 'н': 50, 'ѯ': 60, 'ѻ': 70, 'п': 80, 'ч': 90, 'р': 100, 'с': 200, 'т': 300, 'ѵ': 400, 'у': 400,
}

// numeralValue parses a Church Slavonic number written with titlo ("г҃і" = 13).
func numeralValue(tok string) int {
	v, letters, titlo := 0, 0, false
	for _, r := range tok {
		switch {
		case r == '҃':
			titlo = true
		case unicode.Is(unicode.Mn, r):
			return 0
		default:
			n, ok := numeral[r]
			if !ok {
				return 0
			}
			v += n
			letters++
		}
	}
	if !titlo || letters == 0 || letters > 3 {
		return 0
	}
	return v
}

var (
	reZach    = regexp.MustCompile(`зач`)
	reBracket = regexp.MustCompile(`\[([^\]]*)\]`)
)

// ParseBook reads one HIP file of the Apostol or Gospel.
func ParseBook(path string) (*Book, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	b := &Book{Name: filepath.Base(path), Chapters: map[int]map[int]*Verse{}}
	started := false
	chapter, verse := 0, 0
	var cur *Verse
	var text strings.Builder // text of the current verse
	state := 0               // 0 normal, 1 after a star (label expected), 2 collecting incipit
	var incipit strings.Builder
	replace := 0
	flush := func() {
		if cur != nil {
			cur.Text = CleanText(strings.ReplaceAll(text.String(), "*", ""))
		}
		text.Reset()
	}
	var pendingZach *Zachalo
	newVerse := func(n int) {
		flush()
		verse = n
		if pendingZach != nil {
			pendingZach.Chapter, pendingZach.Verse = chapter, n
			b.Zachala = append(b.Zachala, *pendingZach)
			pendingZach = nil
		}
		if b.Chapters[chapter] == nil {
			b.Chapters[chapter] = map[int]*Verse{}
		}
		cur = &Verse{}
		b.Chapters[chapter][verse] = cur
	}
	for _, p := range d.Paras {
		p = splitStars(p)
		for j := 0; j < len(p); j++ {
			seg := p[j]
			plain := hip.PlainString(seg.Text)
			if seg.Red {
				trimmed := strings.TrimSpace(seg.Text)
				switch {
				case strings.HasPrefix(plain, "глава"):
					continue
				case reZach.MatchString(plain) && strings.HasPrefix(trimmed, "("):
					if !started {
						started, chapter = true, 1
					}
					z := Zachalo{}
					if w := strings.Fields(strings.Trim(trimmed, "()")); len(w) > 1 {
						z.N = numeralValue(strings.TrimRight(w[1], ").,"))
					}
					if z.N == 0 && len(b.Zachala) > 0 {
						z.N, z.Half = b.Zachala[len(b.Zachala)-1].N, true
					}
					if z.N > 0 {
						pendingZach = &z
					}
					continue
				case strings.HasPrefix(trimmed, "*"):
					// "*", "**", possibly merged with what follows it in red:
					// "*Понедельник 9 недели: В" (opening star + day label +
					// drop cap of the incipit), "* Б" (closing star + drop cap
					// of the text), "* Сие же и в понедельник…" (a note).
					if !started || cur == nil && state == 0 {
						continue
					}
					rest := strings.TrimLeft(trimmed, "*")
					// An opening star is followed by the day label ("…: В");
					// a closing one ends a collected incipit; any other star
					// is a footnote mark.
					label := ""
					if strings.Contains(rest, ":") {
						label = rest
					} else if j+1 < len(p) && p[j+1].Red && strings.Contains(p[j+1].Text, ":") && !strings.HasPrefix(strings.TrimSpace(p[j+1].Text), "*") {
						label = p[j+1].Text
						j++
					}
					switch {
					case label != "":
						replace = len([]rune(CleanText(strings.ReplaceAll(text.String(), "*", ""))))
						incipit.Reset()
						incipit.WriteString(strings.TrimLeft(label[strings.LastIndex(label, ":")+1:], " "))
						state = 2
					case state == 2:
						if cur != nil && cur.Incipit == "" {
							cur.Incipit = CleanText(incipit.String())
							cur.Replace = replace
						}
						state = 0
						if r := strings.TrimSpace(rest); r != "" && len([]rune(hip.PlainString(r))) <= 2 {
							text.WriteString(" " + r) // drop cap of the following text
						}
					}
					continue
				case state == 1 && strings.Contains(seg.Text, ":"):
					// "Понедельник 9 недели: В" — the drop cap starts the incipit.
					incipit.WriteString(strings.TrimLeft(seg.Text[strings.LastIndex(seg.Text, ":")+1:], " "))
					state = 2
					continue
				case len([]rune(strings.TrimSpace(plain))) <= 2 && !strings.ContainsAny(plain, ":.("):
					// a drop cap: part of the text
					if state == 2 {
						incipit.WriteString(seg.Text)
					} else if started {
						text.WriteString(seg.Text)
					}
					continue
				default:
					continue // rubrics: "Конец среде.", "Преступи вторнику.", …
				}
			}
			if !started {
				continue
			}
			if state == 2 {
				// The closing star is sometimes printed in black, inside the text.
				k := strings.Index(seg.Text, "*")
				if k < 0 {
					incipit.WriteString(seg.Text)
					continue
				}
				incipit.WriteString(seg.Text[:k])
				if cur != nil && cur.Incipit == "" {
					cur.Incipit = CleanText(incipit.String())
					cur.Replace = replace
				}
				state = 0
				seg.Text = strings.TrimLeft(seg.Text[k:], "*")
			}
			// black text: split off verse numbers
			toks := strings.Fields(reBracket.ReplaceAllStringFunc(seg.Text, func(m string) string {
				inner := m[1 : len(m)-1]
				if numeralValue(strings.TrimSpace(inner)) > 0 {
					return " " // section numbers "[л҃з]"
				}
				return inner // editorial insertions "[ѿ]"
			}))
			lead := strings.HasPrefix(seg.Text, " ") || strings.HasPrefix(seg.Text, " ")
			for i, tok := range toks {
				if hip.PlainString(tok) == "сх" { // "сⷯ" = стих
					continue
				}
				if n := numeralValue(tok); n > 0 {
					next := ""
					if i+1 < len(toks) {
						next = toks[i+1]
					}
					// Verse numbering restarting at 1 begins a new chapter.
					if n == 1 && verse > 1 && (next == "" || unicode.IsUpper([]rune(next)[0])) {
						chapter++
						newVerse(1)
						continue
					}
					upper := next == "" || unicode.IsUpper([]rune(next)[0])
					if (n == verse+1 || (verse == 0 && n == 1)) && upper {
						newVerse(n)
						continue
					}
					// Misprints: a skipped verse number (its text stays with
					// the previous verse) or a repeated one ("м҃а" for "м҃в").
					if verse > 0 && upper && (n > verse+1 && n <= verse+3 || n == verse) {
						if n == verse {
							n = verse + 1
						}
						newVerse(n)
						continue
					}
				}
				if cur == nil {
					continue
				}
				if text.Len() > 0 && (i > 0 || lead) {
					text.WriteByte(' ')
				}
				text.WriteString(tok)
			}
			if strings.HasSuffix(seg.Text, " ") {
				text.WriteByte(' ')
			}
		}
		text.WriteByte(' ')
	}
	flush()
	if len(b.Chapters) == 0 {
		return b, fmt.Errorf("%s: no verses parsed", filepath.Base(path))
	}
	return b, nil
}

// VerseRange is a span of verses, possibly across chapters.
type VerseRange struct {
	FromChapter, FromVerse, ToChapter, ToVerse int
	Mid                                        bool // starts in the middle of the first verse ("8b")
}

var reRefPart = regexp.MustCompile(`^(?:(\d+):)?(\d+)([ab]?)(?:-(?:(\d+):)?(\d+)[ab]?)?$`)

// ParseRef parses "Mt_18:18-22, 19:1-2, 13-15" into the book code and ranges.
func ParseRef(ref string) (string, []VerseRange, error) {
	k := strings.LastIndex(ref, "_")
	if k < 0 {
		return "", nil, fmt.Errorf("bad reference %q", ref)
	}
	book := ref[:k]
	var out []VerseRange
	chapter := 0
	for _, part := range strings.Split(ref[k+1:], ",") {
		part = strings.TrimSpace(part)
		m := reRefPart.FindStringSubmatch(part)
		if m == nil {
			return book, nil, fmt.Errorf("bad reference part %q in %q", part, ref)
		}
		if m[1] != "" {
			chapter, _ = strconv.Atoi(m[1])
		}
		if chapter == 0 {
			return book, nil, fmt.Errorf("no chapter in %q", ref)
		}
		r := VerseRange{FromChapter: chapter, Mid: m[3] == "b"}
		r.FromVerse, _ = strconv.Atoi(m[2])
		r.ToChapter, r.ToVerse = chapter, r.FromVerse
		if m[5] != "" {
			if m[4] != "" {
				r.ToChapter, _ = strconv.Atoi(m[4])
				chapter = r.ToChapter
			}
			r.ToVerse, _ = strconv.Atoi(m[5])
		}
		out = append(out, r)
	}
	return book, out, nil
}

// Reading assembles the text of the ranges, starting with the liturgical
// incipit when the source marks one at the first verse.
func (b *Book) Reading(ranges []VerseRange) (string, error) {
	ranges = b.alignStart(ranges)
	var parts []string
	for i, r := range ranges {
		for c := r.FromChapter; c <= r.ToChapter; c++ {
			ch := b.Chapters[c]
			if ch == nil {
				return "", fmt.Errorf("%s: no chapter %d", b.Name, c)
			}
			from, to := 1, 1<<30
			if c == r.FromChapter {
				from = r.FromVerse
			}
			if c == r.ToChapter {
				to = r.ToVerse
			}
			for v := from; v <= to; v++ {
				vs := ch[v]
				if vs == nil {
					if to == 1<<30 {
						break
					}
					// A verse whose number is missing in the source is printed
					// together with the previous one.
					if v > from && ch[v-1] != nil && (ch[v+1] != nil || v == to) {
						continue
					}
					return "", fmt.Errorf("%s: no verse %d:%d", b.Name, c, v)
				}
				t := vs.Text
				if i == 0 && c == r.FromChapter && v == r.FromVerse && r.Mid && vs.Incipit == "" {
					// "8b": the Slavonic verse division differs; the reading
					// starts at the next verse, where its incipit is marked.
					if nx := ch[v+1]; nx != nil && nx.Incipit != "" {
						continue
					}
				}
				first := i == 0 && c == r.FromChapter && (v == r.FromVerse || (r.Mid && v == r.FromVerse+1 && len(parts) == 0))
				if first && vs.Incipit != "" {
					rs := []rune(t)
					rest := ""
					if vs.Replace <= len(rs) {
						rest = strings.TrimSpace(string(rs[vs.Replace:]))
					}
					t = strings.TrimSpace(vs.Incipit + " " + rest)
				}
				parts = append(parts, t)
			}
		}
	}
	return CleanText(strings.Join(parts, " ")), nil
}

// alignStart moves a reading whose first verse has no liturgical beginning
// marked to the neighbouring verse that has one: the verse numbering of the
// Slavonic text sometimes differs by one from the references (a misprinted
// number, a different division). The end moves with it.
func (b *Book) alignStart(ranges []VerseRange) []VerseRange {
	if len(ranges) == 0 || ranges[0].Mid {
		return ranges
	}
	r := ranges[0]
	ch := b.Chapters[r.FromChapter]
	if ch == nil || ch[r.FromVerse] == nil || ch[r.FromVerse].Incipit != "" {
		return ranges
	}
	for _, d := range []int{1, -1} {
		v := ch[r.FromVerse+d]
		if v == nil || v.Incipit == "" {
			continue
		}
		out := append([]VerseRange(nil), ranges...)
		out[0].FromVerse += d
		if r.ToChapter == r.FromChapter && len(ranges) == 1 {
			out[0].ToVerse += d
			if ch[out[0].ToVerse] == nil {
				out[0].ToVerse -= d
			}
		}
		return out
	}
	return ranges
}

var reStars = regexp.MustCompile(`\*+`)

// splitStars puts every run of stars into a red segment of its own: the
// stars that delimit a liturgical beginning are printed now in red, now in
// black, alone or merged with a drop cap ("И҆*") or a label.
func splitStars(p hip.Paragraph) hip.Paragraph {
	var out hip.Paragraph
	for _, seg := range p {
		locs := reStars.FindAllStringIndex(seg.Text, -1)
		if locs == nil {
			out = append(out, seg)
			continue
		}
		prev := 0
		for _, l := range locs {
			if l[0] > prev {
				s := seg
				s.Text = seg.Text[prev:l[0]]
				out = append(out, s)
			}
			s := seg
			s.Text, s.Red = seg.Text[l[0]:l[1]], true
			out = append(out, s)
			prev = l[1]
		}
		if prev < len(seg.Text) {
			s := seg
			s.Text = seg.Text[prev:]
			out = append(out, s)
		}
	}
	return out
}

// ZachaloRanges returns the verses of pericope n (its second half if half).
func (b *Book) ZachaloRanges(n int, half bool) ([]VerseRange, error) {
	for i, z := range b.Zachala {
		if z.N != n || z.Half != half {
			continue
		}
		// It ends where the next numbered pericope begins (a half marker
		// lies within its own pericope).
		end := VerseRange{}
		for _, nx := range b.Zachala[i+1:] {
			if nx.N != n {
				end = VerseRange{ToChapter: nx.Chapter, ToVerse: nx.Verse - 1}
				break
			}
		}
		if end.ToChapter == 0 { // the last pericope: to the end of the book
			end.ToChapter = len(b.Chapters)
			end.ToVerse = len(b.Chapters[end.ToChapter])
		}
		if end.ToVerse == 0 { // the next one starts a chapter
			end.ToChapter--
			end.ToVerse = len(b.Chapters[end.ToChapter])
		}
		return []VerseRange{{FromChapter: z.Chapter, FromVerse: z.Verse, ToChapter: end.ToChapter, ToVerse: end.ToVerse}}, nil
	}
	return nil, fmt.Errorf("%s: no pericope %d (half %v)", b.Name, n, half)
}
