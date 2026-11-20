package orthlib

import (
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

func TestMagnifications(t *testing.T) {
	units, err := ImportMagnifications(extractedRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	for _, u := range units {
		by[u.ID] = u.Body
	}
	for _, id := range []string{"ir.magnification.12.25", "ir.magnification.12.25.verse.1", "ir.magnification.01.06.verse.2",
		"ir.magnification.06.29", "ir.magnification.06.29.verse.1", "ir.magnification.common.apostle",
		"ir.magnification.common.apostle.verse.1", "ir.magnification.common.bodiless.2", "ir.magnification.pentecost.verse.1"} {
		if by[id] == "" {
			t.Errorf("missing %s", id)
		}
	}
	if a := by["ir.magnification.alleluia"]; !strings.HasPrefix(a, "А҆ллилꙋ́їа") || strings.Contains(a, "Три́жды") {
		t.Errorf("alleluia: %q", a)
	}
	if v := by["ir.magnification.09.08.verse.1"]; !strings.HasSuffix(v, "кро́тость є҆гѡ̀.") {
		t.Errorf("09.08 verse 1: %q", v)
	}
	if by["ir.magnification.09.08.verse.14"] == "" || by["ir.magnification.09.08.verse.15"] != "" {
		t.Errorf("09.08: want 14 verses")
	}
	// Supplemented: the feast's from the Irmologion, a saint's verses from the common.
	men := []corpus.Unit{{ID: "me.12.25.troparion.1", Book: "menaion"},
		{ID: "me.08.28p.matins.magnification", Book: "menaion", Body: by["ir.magnification.common.venerable"]},
		{ID: "me.10.13p.matins.magnification", Book: "menaion", Body: "Велича́емъ тѧ̀, прест҃а́ѧ дв҃о, бг҃оизбра́ннаѧ ѻ҆трокови́це, и҆ чти́мъ ѻ҆́бразъ тво́й."},
		{ID: "me.12.06.title", Book: "menaion", Body: "И҆́же во ст҃ы́хъ ѻ҆тца̀ на́шегѡ нїкола́а, а҆рхїепі́скопа мѷрлѷкі́йскихъ чꙋдотво́рца."},
		{ID: "me.12.06.common.rank", Book: "menaion", Body: "hierarch"}}
	got := map[string]bool{}
	for _, u := range SupplementMagnifications(append(men, units...)) {
		got[u.ID] = true
		if u.ID == "me.12.06.matins.magnification" && !strings.HasPrefix(u.Body, "Велича́емъ тѧ̀, ст҃и́телю ѻ҆́тче нїкола́е, и҆ чти́мъ") {
			t.Errorf("12.06: %q", u.Body)
		}
		if u.ID == "me.10.13p.matins.magnification.verse.1" && u.Body != by["ir.magnification.09.08.verse.1"] {
			t.Errorf("icon: %q", u.Body)
		}
	}
	for _, id := range []string{"me.12.25.matins.magnification", "me.12.25.matins.magnification.verse.1", "me.08.28p.matins.magnification.verse.1",
		"me.12.06.matins.magnification", "me.12.06.matins.magnification.verse.14", "me.10.13p.matins.magnification.verse.14"} {
		if !got[id] {
			t.Errorf("supplement: missing %s", id)
		}
	}
}

func TestSaintName(t *testing.T) {
	for _, c := range []struct{ title, rank, want string }{
		{"И҆́же во ст҃ы́хъ ѻ҆тца̀ на́шегѡ нїкола́а, а҆рхїепі́скопа мѷрлѷкі́йскихъ чꙋдотво́рца.", "hierarch", "нїкола́е"},
		{"Прпⷣбнагѡ и҆ бг҃оно́снагѡ ѻ҆тца̀ на́шегѡ са́ввы ѡ҆свѧще́ннагѡ.", "monk", "са́вво"},
		{"Ст҃а́гѡ а҆пⷭ҇ла и҆ є҆ѵⷢ҇лі́ста лꙋкѝ. И҆ прпⷣбнагѡ і҆ѡ́сифа волоцка́гѡ.", "apostle", "лꙋко̀"},
		{"Ст҃а́гѡ сла́внагѡ великомч҃ника, побѣдоно́сца и҆ чꙋдотво́рца геѡ́ргїа.", "martyr", "геѡ́ргїе"},
		{"Ст҃ы́хъ мч҃никъ, тїмоѳе́а и҆ ма́ѵры: И҆ ѹ҆спе́нїе прпⷣбнагѡ ѻ҆ц҃а̀ на́шегѡ ѳеодо́сїа, и҆гꙋ́мена.", "monk", "ѳеодо́сїе"},
	} {
		if got, ok := saintName(c.title, c.rank); !ok || got != c.want {
			t.Errorf("%s: %q %v, want %q", c.title, got, ok, c.want)
		}
	}
	// Several saints: not the singular common magnification.
	if _, ok := saintName("И҆̀же во ст҃ы́хъ ѻ҆тє́цъ на́шихъ равноапⷭ҇льныхъ меѳо́дїа и҆ кѷрі́лла.", "hierarch"); ok {
		t.Error("plural commemoration taken")
	}
}
