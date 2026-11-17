package orthlib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestZachalaIndex(t *testing.T) {
	root := extractedRoot(t)
	lk, err := ParseBook(filepath.Join(root, bookFiles["Lk"]))
	if err != nil {
		t.Fatal(err)
	}
	r, err := lk.ZachaloRanges(24, false)
	if err != nil || r[0].FromChapter != 6 || r[0].FromVerse != 17 || r[0].ToVerse != 23 {
		t.Errorf("Lk zach. 24: %+v %v", r, err)
	}
	heb, _ := ParseBook(filepath.Join(root, bookFiles["Heb"]))
	if r, err := heb.ZachaloRanges(335, false); err != nil || r[0].FromChapter != 13 || r[0].FromVerse != 17 {
		t.Errorf("Heb zach. 335: %+v %v", r, err)
	}
	units, warnings, err := ImportZachala(root)
	if err != nil || len(warnings) > 0 || len(units) < 1400 {
		t.Errorf("%d units, %v, %v", len(units), warnings, err)
	}
}

func TestMenaionLiturgyRubric(t *testing.T) {
	d, err := LoadDoc(filepath.Join(extractedRoot(t), "mineya", "dec", "dec", "Декабрь", "06.hip"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	for _, u := range parseMenaionLiturgy(d, "me.12.06", "06.hip", "12.06") {
		if strings.HasSuffix(u.ID, ".lit.rubric") {
			m = u.Meta
		}
	}
	if m == nil {
		t.Fatal("no rubric")
	}
	if m["apostle"] != 335 || m["gospel"] != 24 || m["gospel_book"] != "lk" || m["prok_tone"] != 7 || m["alleluia_tone"] != 4 {
		t.Errorf("rubric: %v", m)
	}
	if odes, _ := m["beatitudes"].([][2]int); len(odes) != 2 || odes[0] != [2]int{1, 3} || odes[1] != [2]int{2, 6} {
		t.Errorf("beatitudes: %v", m["beatitudes"])
	}
}

func TestFestalAntiphons(t *testing.T) {
	d, err := LoadDoc(filepath.Join(extractedRoot(t), "mineya", "sep", "sep", "Сентябрь", "14.hip"))
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{}
	for _, u := range parseMenaionLiturgy(d, "me.09.14", "14.hip", "09.14") {
		keys[strings.TrimPrefix(u.ID, "me.09.14.lit.")] = u.Body
		if c, _ := u.Meta["copy"].(string); c != "" {
			keys[strings.TrimPrefix(u.ID, "me.09.14.lit.")] = "copy:" + c
		}
	}
	// «Сти́хъ в҃: Д|але́че…» — a verse with its label in red.
	if !strings.HasPrefix(keys["antiphon.1.3"], "Дале́че") {
		t.Errorf("antiphon 1, verse 2: %q", keys["antiphon.1.3"])
	}
	if keys["antiphon.2.10"] != "copy:ch.typika.only_begotten" || keys["antiphon.3.2"] != "copy:me.09.14.troparion.1" {
		t.Errorf("antiphons 2, 3: %q %q", keys["antiphon.2.10"], keys["antiphon.3.2"])
	}
	if !strings.HasPrefix(keys["trisagion"], "Крⷭ҇тꙋ̀ твоемꙋ̀") || !strings.HasPrefix(keys["entrance"], "Возноси́те") {
		t.Errorf("trisagion %q, entrance %q", keys["trisagion"], keys["entrance"])
	}
}

func TestVersePool(t *testing.T) {
	pool, err := loadVersePool(extractedRoot(t))
	if err != nil || len(pool.prok) < 20 || len(pool.kinonik) < 20 {
		t.Fatalf("%v", err)
	}
	if got := complete("Въ па́мѧть вѣ́чнꙋю:", pool.kinonik); !strings.Contains(got, "бꙋ́детъ") {
		t.Errorf("kinonik cue: %q", got)
	}
	var at []string
	for _, a := range pool.alleluia {
		at = append(at, a.Text)
	}
	// Written in full in the Menaion, under a titlo in the Служебник.
	if got := complete("Свѧще́нницы твоѝ ѡ҆блекꙋ́тсѧ въ пра́вдꙋ:", at); !strings.Contains(got, "возра́дꙋютсѧ") {
		t.Errorf("alleluia cue: %q", got)
	}
}
