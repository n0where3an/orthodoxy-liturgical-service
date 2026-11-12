// Package hip converts Church Slavonic texts in the HIP-7 encoding (as used by
// orthlib.ru) to Unicode in the conventions of the Ponomar Unicode fonts.
//
// Only the modern (level 0) character set is mapped exactly; rarer level 1-2
// symbols fall back to their nearest modern letter. Language switches to
// <::рус>, <::лат> and <::греч> are honoured by dropping that text, since the
// orthlib files use them only for headers.
package hip

import (
	"bytes"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Segment is a run of text with uniform colour.
type Segment struct {
	Text string
	Red  bool // cinnabar (киноварь)
}

// Paragraph is a sequence of segments; text segments are separated by single
// spaces already.
type Paragraph []Segment

// Text returns the paragraph text without colour information.
func (p Paragraph) Text() string {
	var b strings.Builder
	for _, s := range p {
		b.WriteString(s.Text)
	}
	return b.String()
}

// RedPrefix returns the red text at the start of the paragraph (possibly
// spanning several segments separated only by spaces).
func (p Paragraph) RedPrefix() string {
	var b strings.Builder
	for _, s := range p {
		if !s.Red {
			if strings.TrimSpace(s.Text) == "" {
				b.WriteString(s.Text)
				continue
			}
			break
		}
		b.WriteString(s.Text)
	}
	return strings.TrimSpace(b.String())
}

// DecodeCP1251 decodes the byte encoding of orthlib .hip files.
func DecodeCP1251(b []byte) (string, error) {
	r := charmap.Windows1251.NewDecoder().Reader(bytes.NewReader(b))
	out, err := io.ReadAll(r)
	return string(out), err
}

// ParseFile decodes a cp1251 .hip file and parses it.
func ParseFile(b []byte) ([]Paragraph, error) {
	s, err := DecodeCP1251(b)
	if err != nil {
		return nil, err
	}
	return Parse(s), nil
}

// ToUnicode converts a HIP fragment (already UTF-8) to a single Unicode string;
// paragraphs are joined by "\n".
func ToUnicode(s string) string {
	ps := Parse(s)
	lines := make([]string, len(ps))
	for i, p := range ps {
		lines[i] = p.Text()
	}
	return strings.Join(lines, "\n")
}

// Combining marks (Unicode).
const (
	acute      = "́"
	grave      = "̀"
	kamora     = "̑"
	psili      = "҆" // zvatelnoe
	dasia      = "҅"
	titlo      = "҃"
	pokrytie   = "҇"
	payerok    = "̾"
	doubleGrav = "̏"
	dotAbove   = "̇"
	diaeresis  = "̈"
	breve      = "̆"
)

// letterTitla maps letters used as superscripts (буквенные титла) to the
// Unicode combining letters; the bool is whether the letter is written with
// a vzmet (pokrytie) by default.
var letterTitla = map[rune]struct {
	comb  string
	vzmet bool
}{
	'а': {"ⷶ", false}, 'б': {"ⷠ", true}, 'в': {"ⷡ", true},
	'г': {"ⷢ", true}, 'д': {"ⷣ", false}, 'ж': {"ⷤ", false},
	'з': {"ⷥ", false}, 'и': {"ꙵ", false}, 'к': {"ⷦ", true},
	'л': {"ⷧ", true}, 'м': {"ⷨ", false}, 'н': {"ⷩ", true},
	'о': {"ⷪ", true}, 'п': {"ⷫ", true}, 'р': {"ⷬ", true},
	'с': {"ⷭ", true}, 'т': {"ⷮ", false}, 'у': {"ꙷ", false},
	'х': {"ⷯ", false}, 'ц': {"ⷰ", true}, 'ч': {"ⷱ", true},
	'ш': {"ⷲ", true}, 'щ': {"ⷳ", true}, 'f': {"ⷴ", true},
	'я': {"ꙶ", false}, 'е': {"ⷷ", false},
}

// simple letters in <::слав> mode that differ from their HIP spelling.
var slavLetters = map[rune]string{
	'я': "ѧ", 'Я': "Ѧ",
	'у': "ꙋ", 'У': "Ꙋ",
	's': "ѕ", 'S': "Ѕ",
	'f': "ѳ", 'F': "Ѳ",
	'v': "ѵ", 'V': "Ѵ",
	'u': "ѫ", 'U': "Ѫ",
	'w': "ѡ", 'W': "Ѡ",
	'q': "ҁ", 'Q': "Ҁ",
	'g': "ꙉ", 'G': "Ꙉ",
	'#': "҂",
}

// named <...> and _x symbols.
var named = map[string]string{
	"е": "є", "Е": "Є", "о": "ѻ", "О": "Ѻ", "у": "ꙋ", "У": "Ꙋ",
	"i": "і", "I": "І", "w": "ѽ", "W": "Ѽ",
	"кс": "ѯ", "Кс": "Ѯ", "КС": "Ѯ", "пс": "ѱ", "Пс": "Ѱ", "ПС": "Ѱ",
	"с": "с", "С": "С",
	"jь": "ѣ", "Jь": "Ѣ", "JЬ": "Ѣ",
	"ъ": "ъ", "Ъ": "Ъ", "ы": "ы", "Ы": "Ы", "т": "т", "Т": "Т",
	"з": "з", "З": "З", "г": "г", "Г": "Г", "ч": "ч", "Ч": "Ч", "ц": "ц", "Ц": "Ц",
	"п": "п", "П": "П", "в": "в", "В": "В", "ю": "ю", "Ю": "Ю", "я": "ѧ", "Я": "Ѧ",
	"s": "ѕ", "S": "Ѕ", "f": "ѳ", "F": "Ѳ", "u": "ѫ", "U": "Ѫ",
	"-":  "—",
	"((": "«", "))": "»",
	".": "·",
	"*": "*",
	"_": " ",
	"":  "\n", // paragraph
}

// iotated letters after the J prefix.
var iotated = map[rune]string{
	'а': "ꙗ", 'А': "Ꙗ", 'е': "ѥ", 'Е': "Ѥ", 'ь': "ѣ", 'Ь': "Ѣ",
	'я': "ѩ", 'Я': "Ѩ", 'u': "ѭ", 'U': "Ѭ",
}

type parser struct {
	src   []rune
	i     int
	lang  string
	red   bool
	paras []Paragraph
	cur   Paragraph
	buf   strings.Builder // current segment
	space bool            // a space is pending
}

// Parse parses HIP text (UTF-8) into paragraphs.
func Parse(s string) []Paragraph {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	p := &parser{src: []rune(s), lang: "слав"}
	p.run()
	return p.paras
}

func (p *parser) peek(k int) rune {
	if p.i+k < len(p.src) {
		return p.src[p.i+k]
	}
	return 0
}

func (p *parser) hasPrefix(s string) bool {
	r := []rune(s)
	if p.i+len(r) > len(p.src) {
		return false
	}
	for k, c := range r {
		if p.src[p.i+k] != c {
			return false
		}
	}
	return true
}

func (p *parser) flushSeg() {
	if p.buf.Len() == 0 {
		return
	}
	p.cur = append(p.cur, Segment{Text: p.buf.String(), Red: p.red})
	p.buf.Reset()
}

func (p *parser) setRed(red bool) {
	if red == p.red {
		return
	}
	p.flushSeg()
	p.red = red
}

func (p *parser) endPara() {
	p.flushSeg()
	p.space = false
	// trim trailing space of the last segment
	if n := len(p.cur); n > 0 {
		p.cur[n-1].Text = strings.TrimRight(p.cur[n-1].Text, " ")
		if p.cur[n-1].Text == "" {
			p.cur = p.cur[:n-1]
		}
	}
	if len(p.cur) > 0 {
		p.paras = append(p.paras, p.cur)
	}
	p.cur = nil
}

// emit writes visible text, inserting a pending space first.
func (p *parser) emit(s string) {
	if p.space {
		if len(p.cur) > 0 || p.buf.Len() > 0 {
			p.buf.WriteByte(' ')
		}
		p.space = false
	}
	p.buf.WriteString(s)
}

func (p *parser) run() {
	nl := 0
	for p.i < len(p.src) {
		c := p.src[p.i]
		switch {
		case c == '\n':
			nl++
			p.i++
			if nl >= 2 {
				p.endPara()
			} else {
				p.space = true
			}
			continue
		case c == ' ' || c == '\t':
			p.space = true
			p.i++
			continue
		}
		nl = 0
		if p.hasPrefix("<::") {
			end := p.find('>')
			p.lang = string(p.src[p.i+3 : end])
			p.i = end + 1
			continue
		}
		if p.lang != "слав" {
			// Drop non-Slavonic text up to the end of the line.
			for p.i < len(p.src) && p.src[p.i] != '\n' && !p.hasPrefix("<::") {
				p.i++
			}
			continue
		}
		if c == '%' {
			p.comment()
			continue
		}
		if c == '<' && (p.peek(1) == '>') {
			p.i += 2
			p.endPara()
			continue
		}
		if c == '_' && p.peek(1) == '/' {
			p.i += 2
			p.endPara()
			continue
		}
		p.word()
	}
	p.endPara()
}

func (p *parser) find(r rune) int {
	for k := p.i; k < len(p.src); k++ {
		if p.src[k] == r {
			return k
		}
	}
	return len(p.src) - 1
}

func (p *parser) comment() {
	n := p.peek(1)
	switch n {
	case '<':
		p.setRed(true)
		p.i += 2
	case '>':
		p.setRed(false)
		p.i += 2
	case '{':
		depth := 0
		for p.i++; p.i < len(p.src); p.i++ {
			if p.src[p.i] == '{' {
				depth++
			} else if p.src[p.i] == '}' {
				depth--
				if depth == 0 {
					p.i++
					return
				}
			}
		}
	default:
		p.i += 2
	}
}

// word consumes one base sign with its accents.
func (p *parser) word() {
	c := p.src[p.i]
	var base string
	var baseRune rune // for i-with-accent handling
	switch {
	case c == '&' || c == '@':
		p.i++
		return
	case c == '{':
		// grouped text: just parse inner content inline
		p.i++
		return
	case c == '}':
		p.i++
		return
	case c == '\\':
		// accent without base (e.g. after a brace group): treat as accent
		p.accents()
		return
	case (c == 'о' || c == 'О') && (p.hasPrefixAt(1, "_у") || p.hasPrefixAt(1, "<у>") || p.hasPrefixAt(1, "_У") || p.hasPrefixAt(1, "<У>")):
		if c == 'О' {
			base = "Ѹ"
		} else {
			base = "ѹ"
		}
		if p.hasPrefixAt(1, "_") {
			p.i += 3
		} else {
			p.i += 4
		}
	case c == 'j' || c == 'J':
		k := 1
		for p.peek(k) == 'j' || p.peek(k) == 'J' {
			k++
		}
		next := p.peek(k)
		if s, ok := iotated[next]; ok {
			if c == 'J' && unicode.IsLower(next) {
				s = string(unicode.ToUpper([]rune(s)[0]))
			}
			base = s
			p.i += k + 1
		} else {
			p.i++
			return
		}
	case (c == 'w' || c == 'W') && p.hasPrefixAt(1, "\\т"):
		if c == 'W' {
			base = "Ѿ"
		} else {
			base = "ѿ"
		}
		p.i += 3
	case c == '_':
		matched := false
		for _, name := range []string{"кс", "Кс", "КС", "пс", "Пс", "ПС", "е", "Е", "о", "О", "у", "У", "i", "I", "w", "W", "с", "С", "_"} {
			if p.hasPrefixAt(1, name) {
				base = named[name]
				p.i += 1 + utf8.RuneCountInString(name)
				matched = true
				if name == "i" || name == "I" {
					baseRune = 'i'
				}
				break
			}
		}
		if !matched {
			p.i++
			return
		}
	case c == '<':
		end := p.find('>')
		name := string(p.src[p.i+1 : end])
		p.i = end + 1
		if s, ok := named[name]; ok {
			base = s
			if name == "i" || name == "I" {
				baseRune = 'i'
			}
		} else {
			// Unknown symbol: keep its letters.
			base = name
		}
	case c == 'i' || c == 'I':
		baseRune = c
		p.i++
		if p.accentFollows() {
			base = map[rune]string{'i': "і", 'I': "І"}[c]
		} else {
			base = map[rune]string{'i': "ї", 'I': "Ї"}[c]
		}
	default:
		p.i++
		if s, ok := slavLetters[c]; ok {
			base = s
		} else {
			base = string(c)
		}
	}
	_ = baseRune
	p.emit(base)
	p.accents()
}

func (p *parser) hasPrefixAt(off int, s string) bool {
	p.i += off
	ok := p.hasPrefix(s)
	p.i -= off
	return ok
}

func (p *parser) accentFollows() bool {
	switch p.peek(0) {
	case '\'', '`', '^', '=', '$', '~', '\\', '"':
		return true
	}
	return false
}

// accents consumes accent signs following a base sign.
func (p *parser) accents() {
	for p.i < len(p.src) {
		c := p.src[p.i]
		switch c {
		case '\'':
			p.buf.WriteString(acute)
		case '`':
			p.buf.WriteString(grave)
		case '^':
			p.buf.WriteString(kamora)
		case '=':
			p.buf.WriteString(psili)
		case '$':
			p.buf.WriteString(dasia)
		case '~':
			p.buf.WriteString(titlo)
		case '"':
			p.buf.WriteString(doubleGrav)
		case '\\':
			n := p.peek(1)
			switch {
			case n == '{':
				// \{x} : superscript letter without vzmet (optionally \^ after)
				end := p.findFrom(p.i+2, '}')
				inner := []rune(string(p.src[p.i+2 : end]))
				p.i = end + 1
				vz := false
				if p.hasPrefix("\\^") {
					vz = true
					p.i += 2
				}
				for _, r := range inner {
					if t, ok := letterTitla[unicode.ToLower(r)]; ok {
						p.buf.WriteString(t.comb)
					}
				}
				if vz {
					p.buf.WriteString(pokrytie)
				}
				continue
			case n == 'ъ' || n == 'Ъ':
				p.buf.WriteString(payerok)
			case n == '.':
				p.buf.WriteString(dotAbove)
			case n == ':':
				p.buf.WriteString(diaeresis)
			case n == '-':
				p.buf.WriteString(pokrytie)
			case n == '@':
				p.buf.WriteString(breve)
			case n == '^':
				// explicit vzmet after a letter-titlo: already emitted by default
			default:
				if t, ok := letterTitla[unicode.ToLower(n)]; ok {
					p.buf.WriteString(t.comb)
					if t.vzmet {
						p.buf.WriteString(pokrytie)
					}
				}
			}
			p.i += 2
			continue
		default:
			return
		}
		p.i++
	}
}

func (p *parser) findFrom(start int, r rune) int {
	for k := start; k < len(p.src); k++ {
		if p.src[k] == r {
			return k
		}
	}
	return len(p.src) - 1
}
