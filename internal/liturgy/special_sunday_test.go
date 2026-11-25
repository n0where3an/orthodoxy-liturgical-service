package liturgy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func planHour(t *testing.T, c mapCorpus, d calendar.JDate, service string) *Plan {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	plan, err := (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, service, Params{Priest: true})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

var hourServices = []string{"first_hour", "third_hour", "sixth_hour", "ninth_hour"}

// Sunday hours (Типикон, гл. 52; гл. 3–5).
func TestSundayHours(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 16) // Sunday
	day := calendar.NewDayContext(d)
	oc, me := "oc."+itoa(day.Tone)+".sun.", "me.11.16."
	hymns := func(p *Plan) string {
		return refs(p.Slots["troparion_first"]) + "|" + refs(p.Slots["troparion"]) + "|" + refs(p.Slots["kontakion"])
	}
	t.Run("saint without a sign", func(t *testing.T) {
		c := sundayCorpus(d, nil, "kontakion.1")
		for _, s := range hourServices {
			p := planHour(t, c, d, s)
			if p.Rule != "hours.sunday" {
				t.Fatalf("rule %s", p.Rule)
			}
			if got, want := hymns(p), oc+"troparion|"+me+"troparion.1|"+oc+"kontakion"; got != want {
				t.Errorf("%s: %s, want %s", s, got, want)
			}
		}
	})
	t.Run("saint of the polyeleos", func(t *testing.T) {
		c := sundayCorpus(d, nil, "kontakion.1")
		c.ids = append(c.ids, "ty.11.16.sign.1")
		c.signs = map[string]string{"ty.11.16.sign.1": "polyeleos"}
		want := []string{me + "kontakion.1", oc + "kontakion", me + "kontakion.1", oc + "kontakion"}
		for i, s := range hourServices {
			if got := refs(planHour(t, c, d, s).Slots["kontakion"]); got != want[i] {
				t.Errorf("%s: kontakion %s, want %s", s, got, want[i])
			}
		}
	})
	t.Run("no saint's troparion", func(t *testing.T) {
		c := sundayCorpus(d, nil)
		c.ids = c.ids[1:] // without me.11.16.troparion.1
		if got := hymns(planHour(t, c, d, "first_hour")); got != oc+"troparion||"+oc+"kontakion" {
			t.Errorf("hymns %s", got)
		}
	})
}

// The Sunday of the Fathers in the forefeast (12.21): «тропарь воскрсн,
// слава, оцев… кондак оцев, и предпразднства, пременяюще»; at the liturgy
// their prokeimenon and readings, no ordinary ones («ряд оставляется»).
func TestFathersSunday(t *testing.T) {
	d := calendar.MustJDate(2026, time.December, 21)
	day := calendar.NewDayContext(d)
	if specialSunday(day) != "fathers" || !day.Lections.Omitted {
		t.Fatalf("test date: %q, omitted %v", specialSunday(day), day.Lections.Omitted)
	}
	oc, me, ms := "oc."+itoa(day.Tone)+".sun.", "me.12.21.", "ms.fathers."
	c := sundayCorpus(d, nil, "troparion.2", "kontakion.1", "kontakion.3")
	c.of = map[string]string{me + "troparion.1": "стыя,", me + "troparion.2": "и ныне", me + "kontakion.1": "стыя,",
		me + "kontakion.3": "предпразднства,"}
	c.ids = append(c.ids, ms+"troparion", ms+"kontakion", ms+"kontakion.forefeast", ms+"lit.prokeimenon",
		ms+"lit.prokeimenon.verse", ms+"lit.alleluia.verse.1", ms+"lit.apostle", ms+"lit.gospel", ms+"lit.kinonik")
	c.tones[ms+"lit.prokeimenon"], c.tones[ms+"lit.alleluia.verse.1"] = 4, 4
	kontakia := []string{ms + "kontakion.forefeast", me + "kontakion.3", ms + "kontakion.forefeast", me + "kontakion.3"}
	for i, s := range hourServices {
		p := planHour(t, c, d, s)
		if got := refs(p.Slots["troparion"]) + "|" + refs(p.Slots["kontakion"]); got != ms+"troparion|"+kontakia[i] {
			t.Errorf("%s: %s", s, got)
		}
	}
	plan, err := planSundayLiturgy(t, c, d, "")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "liturgy.sunday.special" {
		t.Fatalf("rule %s", plan.Rule)
	}
	check(t, plan, map[string]string{
		"troparia": oc + "troparion," + ms + "troparion," + me + "troparion.2," + oc + "kontakion,common.glory," +
			ms + "kontakion.forefeast,common.and_now," + me + "kontakion.3",
		"prokeimena": "lit.prokeimenon.tone.4," + ms + "lit.prokeimenon," + ms + "lit.prokeimenon," + ms + "lit.prokeimenon.verse," +
			ms + "lit.prokeimenon," + ms + "lit.prokeimenon," + ms + "lit.prokeimenon",
		"apostles": "lit.wisdom," + ms + "lit.apostle.announce,lit.let_us_attend," + ms + "lit.apostle",
		"gospels":  ms + "lit.gospel.announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend," + ms + "lit.gospel",
		"alleluia": "lit.alleluia.tone.4,common.alleluia.thrice," + ms + "lit.alleluia.verse.1,common.alleluia.thrice",
		"kinonik":  "sl.kinonik.sun," + ms + "lit.kinonik,common.alleluia.thrice",
	})
}

// The Sunday before the Exaltation (2026: 09.07): «прокимен, апсл,
// аллилуиа, и еиглие прежде недели пред воздвижением, и ряд под зачало.
// таже стаго. прокимен гласа оставляется».
func TestSundayBeforeExaltation(t *testing.T) {
	d := calendar.MustJDate(2026, time.September, 7)
	day := calendar.NewDayContext(d)
	if specialSunday(day) != "before_exaltation" || day.Lections.Omitted {
		t.Fatalf("test date: %q, omitted %v", specialSunday(day), day.Lections.Omitted)
	}
	me, ms := "me.09.07.", "ms.before_exaltation."
	c := sundayCorpus(d, map[string]int{"lit.prokeimenon": 4}, "kontakion.1", "lit.prokeimenon", "lit.apostle", "lit.gospel")
	c.ids = append(c.ids, ms+"lit.prokeimenon", ms+"lit.prokeimenon.verse", ms+"lit.alleluia.verse.1", ms+"lit.apostle", ms+"lit.gospel")
	c.tones[ms+"lit.prokeimenon"], c.tones[ms+"lit.alleluia.verse.1"] = 6, 1
	plan, err := planSundayLiturgy(t, c, d, "")
	if err != nil {
		t.Fatal(err)
	}
	l := day.Lections.Apostle
	ord := fmt.Sprintf("lect.ap.%d.0", l.Week)
	check(t, plan, map[string]string{
		"prokeimena": "lit.prokeimenon.tone.6," + ms + "lit.prokeimenon," + ms + "lit.prokeimenon," + ms + "lit.prokeimenon.verse," +
			ms + "lit.prokeimenon," + ms + "lit.prokeimenon," + ms + "lit.prokeimenon,lit.prokeimenon.tone.4," + me + "lit.prokeimenon," + me + "lit.prokeimenon",
		"apostles": "lit.wisdom," + ms + "lit.apostle.announce,lit.let_us_attend," + ms + "lit.apostle," + ord + "," + me + "lit.apostle",
	})
}

// A feast of the Theotokos on Sunday (the Menaion, 11.21): the
// resurrectional troparion, «Слава» the feast's, the kontakia alternating —
// the resurrectional at the 1st hour, the feast's at the 3rd.
func TestTheotokosFeastSundayHours(t *testing.T) {
	d := calendar.MustJDate(2033, time.November, 21) // Sunday
	day := calendar.NewDayContext(d)
	oc, me := "oc."+itoa(day.Tone)+".sun.", "me.11.21."
	c := sundayCorpus(d, nil, "kontakion.1")
	want := []string{oc + "kontakion", me + "kontakion.1", oc + "kontakion", me + "kontakion.1"}
	for i, s := range hourServices {
		p := planHour(t, c, d, s)
		if p.Rule != "hours.theotokos_feast_on_sunday" {
			t.Fatalf("%s: rule %s", s, p.Rule)
		}
		if got := refs(p.Slots["troparion_first"]) + "|" + refs(p.Slots["troparion"]) + "|" + refs(p.Slots["kontakion"]); got != oc+"troparion|"+me+"troparion.1|"+want[i] {
			t.Errorf("%s: %s", s, got)
		}
	}
}
