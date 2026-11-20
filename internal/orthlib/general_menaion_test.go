package orthlib

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestImportGeneralMenaion(t *testing.T) {
	us, warnings := ImportGeneralMenaion(extractedRoot(t))
	if len(warnings) > 0 {
		t.Errorf("warnings: %v", warnings)
	}
	by := map[string]corpus.Unit{}
	for _, u := range us {
		if _, dup := by[u.ID]; dup {
			t.Errorf("duplicate %s", u.ID)
		}
		by[u.ID] = u
		if u.Book != "general_menaion" || u.SourceID != "orthlib.obschminea" {
			t.Errorf("%s: book %q source %q", u.ID, u.Book, u.SourceID)
		}
	}
	plain := func(id string) string {
		u, ok := by[id]
		if !ok {
			t.Fatalf("no %s", id)
		}
		return strings.ToLower(hip.PlainString(u.Body))
	}
	// Three paremias with titles for every rank of ch. 39.
	for cat := range commonParemiaOf {
		for n := 1; n <= 3; n++ {
			id := fmt.Sprintf("om.%s.gvespers.paremia.%d", cat, n)
			if plain(id) == "" || !strings.HasSuffix(plain(id+".title"), "чтение.") {
				t.Errorf("%s: %q / %q", id, plain(id+".title"), plain(id))
			}
		}
	}
	for id, start := range map[string]string{
		"om.prophet.gvespers.paremia.1":   "првдных душы в руце бжией",
		"om.monk.gvespers.paremia.3":      "првдник аще постигнет",
		"om.hierarchs.gvespers.paremia.1": "память првднаго с похвалами", // «писана святителю»
		"om.martyr.gvespers.paremia.1":    "тако глаголет гдсь: вси языцы",
		"om.martyr.gvespers.paremia.3":    "првдник аще постигнет",    // the hierarchs' third
		"om.martyrs.gvespers.paremia.3":   "праведницы во веки живут", // the martyrs' own
		"om.apostle.gvespers.paremia.1":   "возлюбленнии, аще сердце наше",
		"om.elijah.gvespers.paremia.1":    "бысть слово гдсне ко илии",
	} {
		if !strings.HasPrefix(plain(id), start) {
			t.Errorf("%s = %.60q…, want %q…", id, plain(id), start)
		}
	}
	// Ch. 40: the readings of the liturgy per rank.
	for cat, want := range map[string]string{
		"theotokos_feast": "240 lk 54", "angels": "305 lk 51", "apostle": "131 mt 34", "apostles": "131 lk 50",
		"hierarch": "318 jn 36", "hierarchs": "335 mt 11", "monk": "213 mt 43", "martyr": "292 lk 63",
		"martyrs": "99 mt 36", "monk_martyr": "291 mk 37", "monk_martyrs": "99 mt 38", "nun": "208 mt 104",
		"confessor": "233 lk 64", "unmercenaries": "153 mt 34",
	} {
		m := by["om."+cat+".lit.rubric"].Meta
		if got := fmt.Sprintf("%v %v %v", m["apostle"], m["gospel_book"], m["gospel"]); got != want {
			t.Errorf("%s: readings %s, want %s", cat, got, want)
		}
	}
	// «Прокимен, и аллилуиа, писаны мученицам»; «Прокимен, писан мучеником».
	for cat, from := range commonLiturgyFrom {
		m, f := by["om."+cat+".lit.rubric"].Meta, by["om."+from+".lit.rubric"].Meta
		if m["prok"] != f["prok"] || m["prok_tone"] != f["prok_tone"] {
			t.Errorf("%s: prokeimenon %v (%v), want that of %s", cat, m["prok"], m["prok_tone"], from)
		}
	}
	if m := by["om.theotokos_feast.matins.rubric"].Meta; m["gospel_book"] != "lk" || m["gospel"] != 4 {
		t.Errorf("theotokos matins Gospel %v %v, want lk 4", m["gospel_book"], m["gospel"])
	}
}

func TestCommonRank(t *testing.T) {
	for in, want := range map[string]string{
		"вся прпдбническая":              "monk",
		"стительская":                    "hierarch",
		"стителем":                       "hierarchs",
		"мученическая":                   "martyr",
		"вся бцде":                       "theotokos_feast",
		"вся прпдбней жене":              "nun",
		"праздника и стительская":        "",
		"дне и мчников":                  "",
		"писана царю константину":        "",
		"сего месяца в кг день, стителю": "",
	} {
		if got := commonRank(in); got != want {
			t.Errorf("commonRank(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRankOfTitle(t *testing.T) {
	for in, want := range map[string]string{
		"прпдбнаго и бгоноснаго оца нашего антониа великаго.":                           "monk",
		"стаго апсла и еиглиста марка.":                                                 "apostle",
		"стаго священномученика афиногена, и десяти ученик его.":                        "hieromartyr",
		"стых мчник, тимофеа и маиры: и успение прпдбнаго оца нашего феодосиа, игумена": "monk",
		// «архиепскопа» is misprinted: the martyr's rank.
		"стыя мученицы татианы: в тойже день стаго оца саввы, архиепскопа сербскаго": "woman_martyr",
		"пренесение мощей, иже во стых, оца нашего никифора, патриарха царяграда.":   "hierarch",
		"иже во стых, оца нашего кирилла, архиепскопа иерусалимскаго.":               "hierarch",
		"преподобных оц наших, во обители святаго саввы убиенных.":                   "monks",
		"собор архаггла гавриила.":              "angels",
		"стыя матере нашея матроны селунския.":  "nun",
		"стых апсл, аристарха, пуда и трофима.": "apostles",
		"стых мчниц, агапии, ирины и хионии.":   "women_martyrs",
	} {
		if got := rankOfTitle(in); got != want {
			t.Errorf("rankOfTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSupplementFromGeneralMenaion(t *testing.T) {
	u := func(id string, meta map[string]any) corpus.Unit {
		return corpus.Unit{ID: id, Book: "menaion", Kind: "x", Lang: "cu", Body: id, Meta: meta}
	}
	var units []corpus.Unit
	for _, cat := range []string{"monk", "hierarch"} {
		for n := 1; n <= 3; n++ {
			p := fmt.Sprintf("om.%s.gvespers.paremia.%d", cat, n)
			units = append(units, u(p, nil), u(p+".title", nil))
		}
		units = append(units, u("om."+cat+".matins.prokeimenon", nil), u("om."+cat+".matins.prokeimenon.verse", nil))
		for _, part := range commonLitParts {
			units = append(units, u("om."+cat+".lit."+part, nil))
		}
	}
	units = append(units,
		u("om.monk.lit.rubric", map[string]any{"apostle": 213, "gospel_book": "mt", "gospel": 43}),
		u("om.hierarch.lit.rubric", map[string]any{"apostle": 318, "gospel_book": "jn", "gospel": 36}),
		// 01.17: a vigil with «чтения г: лист рна» — the rank from the title.
		u("me.01.17.gvespers.lic.sticheron.1", map[string]any{"title": "Прпдбнаго оца нашего антониа"}),
		// 06.15: «на литургии служба стителем», «евангелие стительское».
		u("me.06.15.gvespers.lic.sticheron.1", nil),
		u("me.06.15.common.lit", map[string]any{"category": "hierarch"}),
		u("me.06.15.common.matins_gospel", map[string]any{"category": "hierarch"}),
		u("me.06.15.matins.prokeimenon", nil),
		// 09.09: own paremias, the rank from the Apostle and Gospel.
		u("me.09.09.gvespers.lic.sticheron.1", nil),
		u("me.09.09.gvespers.paremia.1", nil),
		u("me.09.09.lit.rubric", map[string]any{"apostle": 213, "gospel_book": "mt", "gospel": 43}),
		u("me.09.09.lit.prokeimenon", nil),
		// A feast of the Theotokos is not guessed from its title.
		u("me.12.26.gvespers.lic.sticheron.1", map[string]any{"title": "Собор престыя бцды: и стаго сщенномученика"}),
	)
	out, warnings := SupplementFromGeneralMenaion(units)
	got := map[string]string{}
	for _, x := range out[len(units):] {
		if strings.HasSuffix(x.ID, ".common.rank") {
			continue // the rank of a saint with the sign (its magnification)
		}
		got[x.ID], _ = x.Meta["copied_from"].(string)
	}
	want := map[string]string{
		"me.01.17.gvespers.paremia.1":       "om.monk.gvespers.paremia.1",
		"me.01.17.gvespers.paremia.3.title": "om.monk.gvespers.paremia.3.title",
		"me.01.17.matins.prokeimenon":       "om.monk.matins.prokeimenon",
		"me.06.15.gvespers.paremia.2":       "om.hierarch.gvespers.paremia.2",
		"me.06.15.lit.apostle":              "om.hierarch.lit.apostle",
		"me.06.15.lit.kinonik":              "om.hierarch.lit.kinonik",
		"me.06.15.matins.gospel":            "om.hierarch.lit.gospel",
		"me.06.15.matins.gospel.announce":   "om.hierarch.lit.gospel.announce",
		"me.09.09.matins.prokeimenon":       "om.monk.matins.prokeimenon",
		"me.09.09.matins.prokeimenon.verse": "om.monk.matins.prokeimenon.verse",
	}
	for id, src := range want {
		if got[id] != src {
			t.Errorf("%s copied from %q, want %q", id, got[id], src)
		}
	}
	for id := range got {
		switch {
		case strings.HasPrefix(id, "me.06.15.matins.prokeimenon"), strings.HasPrefix(id, "me.09.09.gvespers.paremia"),
			strings.HasPrefix(id, "me.09.09.lit."), strings.HasPrefix(id, "me.12.26."):
			t.Errorf("%s: the day has its own or no rank, must not be copied", id)
		}
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "me.12.26") {
		t.Errorf("warnings %v, want one for me.12.26", warnings)
	}
}

// The hymns a day lacks: the troparion, the kontakion, the sessional hymn;
// with a sign the aposticha. Not in a forefeast or afterfeast («тропарь
// праздника»), nor for an additional service (me.MM.DDp).
func TestSupplementHymnsFromGeneralMenaion(t *testing.T) {
	u := func(id string, meta map[string]any) corpus.Unit {
		return corpus.Unit{ID: id, Book: "menaion", Kind: "x", Lang: "cu", Body: id, Tone: 4, Meta: meta}
	}
	var units []corpus.Unit
	for _, id := range []string{"troparion.1", "kontakion.1", "matins.sedalen.1", "matins.sedalen.theotokion",
		"gvespers.aposticha.sticheron.1", "gvespers.aposticha.verse.1", "gvespers.aposticha.glory", "gvespers.aposticha.now"} {
		units = append(units, u("om.martyr."+id, nil))
	}
	martyr := map[string]any{"title": "Стаго мученика вонифатиа"}
	units = append(units,
		// 12.19: a martyr without a troparion and sessional hymn.
		u("me.12.19.kontakion.1", martyr),
		// 05.27: a martyr with a sign, neither troparion nor kontakion nor aposticha.
		u("me.05.27.gvespers.lic.sticheron.1", map[string]any{"title": "Стаго сщенномученика ферапонта"}),
		u("me.05.27.matins.sedalen.1", nil),
		// 06.27: without a sign the first saint's rank, not the last's.
		u("me.06.27.title", map[string]any{"title": "Стаго сщенномученика ферапонта: и обретение мощей прпдбнаго оца нашего нила"}),
		// 09.10: an afterfeast, «тропарь праздника».
		u("me.09.10.feast_troparion", martyr),
		// 07.23p: an additional service.
		u("me.07.23p.kontakion.1", martyr),
	)
	units = append(units, u("om.hieromartyr.troparion.1", nil))
	out, _ := SupplementFromGeneralMenaion(units)
	got := map[string]string{}
	for _, x := range out[len(units):] {
		if strings.HasSuffix(x.ID, ".common.rank") {
			continue
		}
		got[x.ID], _ = x.Meta["copied_from"].(string)
		if x.Tone != 4 || x.Meta["rank_from_title"] != true {
			t.Errorf("%s: tone %d, meta %v", x.ID, x.Tone, x.Meta)
		}
	}
	want := map[string]string{
		"me.12.19.troparion.1":               "om.martyr.troparion.1",
		"me.12.19.matins.sedalen.1":          "om.martyr.matins.sedalen.1",
		"me.12.19.matins.sedalen.theotokion": "om.martyr.matins.sedalen.theotokion",
		"me.05.27.troparion.1":               "om.hieromartyr.troparion.1",
		"me.06.27.troparion.1":               "om.hieromartyr.troparion.1",
		"me.05.27.gvespers.paremia.1":        "",
	}
	for id, src := range want {
		if got[id] != src {
			t.Errorf("%s copied from %q, want %q", id, got[id], src)
		}
	}
	for id := range got {
		switch {
		case id == "me.12.19.kontakion.1", strings.HasPrefix(id, "me.05.27.matins.sedalen"),
			strings.HasPrefix(id, "me.09.10."), strings.HasPrefix(id, "me.07.23p."):
			t.Errorf("%s: must not be copied", id)
		}
	}
}

// A troparion cited by its first words («…: трижды») gets its full text.
func TestCompleteCueTroparion(t *testing.T) {
	root := extractedRoot(t)
	matches, _ := filepath.Glob(filepath.Join(root, "mineya", "aug", "aug", "*", "15.hip"))
	if len(matches) != 1 {
		t.Fatalf("08.15: %v", matches)
	}
	us, err := parseServiceFile(matches[0], "me.08.15", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range us {
		if u.ID == "me.08.15.troparion.1" {
			p := strings.ToLower(hip.PlainString(u.Body))
			if !strings.HasPrefix(p, "в ржстве девство сохранила еси, во успении") || u.Meta["completed_cue"] != true {
				t.Errorf("troparion %q", p)
			}
			return
		}
	}
	t.Error("no me.08.15.troparion.1")
}

// «Слава, и ныне, первую стихиру: Днесь иже на разумных престолех:» — the
// cited sticheron (09.08, great vespers).
func TestGreatVespersCitedGlory(t *testing.T) {
	matches, _ := filepath.Glob(filepath.Join(extractedRoot(t), "mineya", "sep", "sep", "*", "08.hip"))
	if len(matches) != 1 {
		t.Fatalf("09.08: %v", matches)
	}
	us, err := parseServiceFile(matches[0], "me.09.08", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range us {
		if u.ID == "me.09.08.gvespers.lic.now" {
			if u.Meta["repeat_of"] != "me.09.08.gvespers.lic.sticheron.1" || u.Meta["glory_now"] != true || u.Tone != 6 {
				t.Errorf("lic.now meta %v tone %d", u.Meta, u.Tone)
			}
			return
		}
	}
	t.Error("no me.09.08.gvespers.lic.now")
}
