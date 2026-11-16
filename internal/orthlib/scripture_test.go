package orthlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestNumeralValue(t *testing.T) {
	for tok, want := range map[string]int{"а҃": 1, "г҃і": 13, "к҃в": 22, "л҃": 30, "р҃": 100, "сх": 0, "и҆": 0} {
		if got := numeralValue(tok); got != want {
			t.Errorf("%q = %d, want %d", tok, got, want)
		}
	}
}

func TestParseRef(t *testing.T) {
	book, rs, err := ParseRef("Mt_18:18-22, 19:1-2, 13-15")
	if err != nil || book != "Mt" || len(rs) != 3 {
		t.Fatalf("%s %v %v", book, rs, err)
	}
	if r := rs[2]; r.FromChapter != 19 || r.FromVerse != 13 || r.ToVerse != 15 {
		t.Errorf("third range %+v", r)
	}
	_, rs, _ = ParseRef("Eph_5:8b-19")
	if !rs[0].Mid || rs[0].FromVerse != 8 {
		t.Errorf("8b: %+v", rs[0])
	}
	_, rs, _ = ParseRef("Mk_15:43-16:8")
	if r := rs[0]; r.ToChapter != 16 || r.ToVerse != 8 {
		t.Errorf("cross-chapter: %+v", r)
	}
}

// Verse counts of the Slavonic text (a verse whose number is missing in
// the source counts with the previous one).
func TestParseBookChapters(t *testing.T) {
	root := extractedRoot(t)
	for code, chapters := range map[string]int{"Mt": 28, "Mk": 16, "Lk": 24, "Jn": 21, "Acts": 28, "Rom": 16, "Heb": 13, "I Pet": 5} {
		b, err := ParseBook(filepath.Join(root, bookFiles[code]))
		if err != nil {
			t.Fatal(err)
		}
		if len(b.Chapters) != chapters {
			t.Errorf("%s: %d chapters, want %d", code, len(b.Chapters), chapters)
		}
	}
}

func TestReadingIncipits(t *testing.T) {
	root := extractedRoot(t)
	cases := []struct{ ref, start string }{
		{"Mt_18:1-11", "Во время оно, приступиша ученицы"},
		{"Mt_18:10-20", "Рече гдсь: Блюдите"},
		{"Rom_1:18-27", "Братие, открывается"},
		{"Eph_5:8b-19", "Братие, яко чада света ходите"},
		{"Acts_2:14-21", "Во дни оны, став Петр"},
		{"Mk_8:30-34", "Во время оно, запрети иис"},        // numbering of Mk 8 shifted in the source
		{"Mk_14:10-42", "Во время оно, Иуда Искариотский"}, // star merged with the drop cap
		{"Acts_10:34-43", "Во дни оны, отверз Петр"},       // opening star printed in black
	}
	books := map[string]*Book{}
	for _, c := range cases {
		code, rs, err := ParseRef(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		if books[code] == nil {
			if books[code], err = ParseBook(filepath.Join(root, bookFiles[code])); err != nil {
				t.Fatal(err)
			}
		}
		text, err := books[code].Reading(rs)
		if err != nil {
			t.Errorf("%s: %v", c.ref, err)
			continue
		}
		if got, want := hip.Fold(text), hip.Fold(c.start); !strings.HasPrefix(got, want) {
			t.Errorf("%s: starts %q", c.ref, string([]rune(text)[:60]))
		}
	}
}

func TestImportReadings(t *testing.T) {
	root := extractedRoot(t)
	ponomar := os.Getenv("PONOMAR_DIR")
	if ponomar == "" {
		t.Skip("PONOMAR_DIR not set")
	}
	units, warnings, err := ImportReadings(root, filepath.Join(ponomar, "scripts/Perl/data/lectionary.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		// Only the known misprints of the week column are tolerated.
		if !strings.Contains(w, "counted as week") {
			t.Error(w)
		}
	}
	ids := map[string]bool{}
	for _, u := range units {
		if ids[u.ID] {
			t.Errorf("duplicate %s", u.ID)
		}
		ids[u.ID] = true
	}
	for _, id := range []string{"lect.ap.-7.0", "lect.gos.-7.0", "lect.ap.4.1", "lect.gos.5.1", "lect.gos.25.2.announce"} {
		if !ids[id] {
			t.Errorf("missing %s", id)
		}
	}
	if len(units) < 1200 {
		t.Errorf("%d units", len(units))
	}
}
