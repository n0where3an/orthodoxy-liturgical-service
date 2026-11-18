package orthlib

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestTrailingHymn(t *testing.T) {
	for _, c := range []struct {
		text, kind, of string
	}{
		{"и҆ ны́нѣ, бг҃оро́диченъ: на стїхо́внѣ ѻ҆ктѡ́иха. тропа́рь, гла́съ а҃:", "troparion", ""},
		{"и҆ ны́нѣ, пра́здника. тропа́рь, гла́съ г҃:", "troparion", ""},
		{"и҆ ны́нѣ, бг҃оро́диченъ. конда́къ, гла́съ и҃. подо́бенъ: ꙗ҆́кѡ нача̑тки:", "kontakion", ""},
		{"тропа́рь пра́здника: спаси̑, гдⷭ҇и, лю́ди твоѧ̑:", "", ""},                      // a cue, no tone
		{"тропа́рь, гла́съ г҃:", "troparion", ""},                                         // at the start: the ordinary heading
		{"а҆́ще ли нѣ́сть ѹ҆ста́ва, глаго́ли се́й конда́къ, гла́съ в҃.", "kontakion", ""}, // the General Menaion
		{"а҆́ще ли нѣ́сть ѹ҆ста́ва, глаго́ли се́й тропа́рь, гла́съ д҃:", "troparion", ""},
		{"по г҃-й пѣ́сни конда́къ и҆ і҆́косъ.", "", ""},
	} {
		got := trailingHymn(c.text)
		if c.kind == "" {
			if got != nil {
				t.Errorf("%q: %v", c.text, got)
			}
			continue
		}
		if got == nil || got[0] != c.kind || got[1] != c.of || !strings.Contains(got[2], "гла́съ") {
			t.Errorf("%q: %q", c.text, got)
		}
	}
}

// The troparion whose heading ends a paragraph of rubrics.
func TestMenaionTrailingTroparion(t *testing.T) {
	root := extractedRoot(t)
	for _, c := range []struct {
		file    string
		month   int
		service string
		incipit string
		tone    int
	}{
		{"mineya/dec/dec/Декабрь/11.hip", 12, "11", "терпения столп", 1},
		{"mineya/oct/oct/Октябрь/06.hip", 10, "06", "апсле стый фомо", 3},
		{"mineya/may/may/Май/02.hip", 5, "02", "столп был еси православия", 3},
	} {
		units, err := ParseMenaionDay(filepath.Join(root, c.file), c.month, c.service)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, u := range units {
			if strings.HasSuffix(u.ID, "."+c.service+".troparion.1") {
				found = true
				if pl := strings.ToLower(hip.PlainString(u.Body)); !strings.HasPrefix(pl, c.incipit) || u.Tone != c.tone {
					t.Errorf("%s: tone %d, %q", u.ID, u.Tone, pl)
				}
			}
		}
		if !found {
			t.Errorf("%s: no troparion", c.file)
		}
	}
}
