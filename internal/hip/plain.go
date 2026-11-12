package hip

import (
	"strings"
	"unicode"
)

// plainMap reduces Church Slavonic letters to the modern Russian alphabet, so
// that anchors and incipits can be written in civil script.
var plainMap = map[rune]string{
	'ѡ': "о", 'ѻ': "о", 'ѽ': "о", 'ѿ': "от", 'ѹ': "у", 'ꙋ': "у", 'є': "е", 'ѥ': "е",
	'ѣ': "е", 'і': "и", 'ї': "и", 'ѵ': "и", 'ѧ': "я", 'ꙗ': "я", 'ѩ': "я", 'ѫ': "у",
	'ѭ': "ю", 'ѯ': "кс", 'ѱ': "пс", 'ѳ': "ф", 'ѕ': "з", 'ꙉ': "г", 'ҁ': "", '҂': "",
	'ё': "е",
}

// combining Cyrillic letters (superscripts) reduce to their base letters.
var combiningLetters = map[rune]string{}

func init() {
	for r, t := range letterTitla {
		combiningLetters[[]rune(t.comb)[0]] = string(r)
	}
}

// Plain returns a lower-case civil-script form of Unicode Church Slavonic
// text: diacritics dropped, archaic letters folded, superscript letters
// restored inline, whitespace collapsed. pos[i] is the index (in runes of s)
// of the character that produced plain rune i.
func Plain(s string) (plain string, pos []int) {
	var b []rune
	rs := []rune(s)
	space := false
	for i, r := range rs {
		if unicode.IsSpace(r) {
			space = len(b) > 0
			continue
		}
		var out string
		if l, ok := combiningLetters[r]; ok {
			out = l
		} else if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			continue
		} else {
			lr := unicode.ToLower(r)
			if m, ok := plainMap[lr]; ok {
				out = m
			} else {
				out = string(lr)
			}
		}
		if !unicode.IsLetter([]rune(out + " ")[0]) {
			b, pos = dropFinalYer(b, pos)
		}
		if space {
			b, pos = dropFinalYer(b, pos)
			b = append(b, ' ')
			pos = append(pos, i)
			space = false
		}
		for _, o := range out {
			b = append(b, o)
			pos = append(pos, i)
		}
	}
	b, pos = dropFinalYer(b, pos)
	return string(b), pos
}

// dropFinalYer removes a word-final hard sign, which civil script omits.
func dropFinalYer(b []rune, pos []int) ([]rune, []int) {
	if n := len(b); n > 1 && b[n-1] == 'ъ' && unicode.IsLetter(b[n-2]) {
		return b[:n-1], pos[:n-1]
	}
	return b, pos
}

// PlainString is Plain without the position map.
func PlainString(s string) string {
	p, _ := Plain(s)
	return p
}

// Fold normalises a civil-script anchor the same way as Plain.
func Fold(s string) string {
	p, _ := Plain(strings.ReplaceAll(s, "ё", "е"))
	return p
}
