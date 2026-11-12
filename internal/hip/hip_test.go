package hip

import "testing"

func TestToUnicode(t *testing.T) {
	tests := []struct{ in, want string }{
		{"бл~гослове'нъ", "бл҃гослове́нъ"},
		{"гд\\си", "гдⷭ҇и"},
		{"JА='кw", "Ꙗ҆́кѡ"},
		{"w\\т", "ѿ"},
		{"о_у=слы'ши", "ѹ҆слы́ши"},
		{"О_у=кроти'лъ", "Ѹ҆кроти́лъ"},
		{"_е=си`", "є҆сѝ"},
		{"_О='ч~е", "Ѻ҆́ч҃е"},
		{"тебjь`", "тебѣ̀"},
		{"своя^", "своѧ̑"},
		{"Трист~о'е", "Трист҃о́е"},
		{"прiиди`", "прїидѝ"},
		{"i=и~се", "і҆и҃се"},
		{"в~_i", "в҃і"},
		{"пред\\ъ", "пред̾"},
		{"сv'менъ", "сѵ́менъ"},
		{"а=ллилу'iа", "а҆ллилꙋ́їа"},
		{"неsло'бiемъ", "неѕло́бїемъ"},
		{"Ѳ", "Ѳ"},
	}
	for _, tt := range tests {
		if got := ToUnicode(tt.in); got != tt.want {
			t.Errorf("ToUnicode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseParagraphsAndRed(t *testing.T) {
	src := "<::рус>        Библиотека\n<::слав> \n\n%<Тропа'рь, гла'съ и~:%>\n\n%<В%>ъ тебjь` ма'ти,\nи=звjь'стнw спасе'ся.\n\n%<Конда'къ%>"
	ps := Parse(src)
	if len(ps) != 3 {
		t.Fatalf("got %d paragraphs: %#v", len(ps), ps)
	}
	if got := ps[0].RedPrefix(); PlainString(got) != "тропарь, глас и:" {
		t.Errorf("heading red prefix %q (plain %q)", got, PlainString(got))
	}
	if got := PlainString(ps[1].Text()); got != "в тебе мати, известно спасеся." {
		t.Errorf("body %q", got)
	}
	if !ps[1][0].Red || ps[1][1].Red {
		t.Errorf("drop cap colours wrong: %#v", ps[1])
	}
}

func TestPlainPositions(t *testing.T) {
	s := "Ѹ҆слы́ши ѿ"
	p, pos := Plain(s)
	if p != "услыши от" {
		t.Fatalf("plain %q", p)
	}
	if len(pos) != len([]rune(p)) {
		t.Fatalf("pos len %d", len(pos))
	}
	if pos[len(pos)-1] != len([]rune(s))-1 || pos[len(pos)-2] != len([]rune(s))-1 {
		t.Errorf("ѿ positions %v", pos)
	}
}
