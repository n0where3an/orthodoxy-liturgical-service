package liturgy

import (
	"context"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func TestSaturdayLiturgyTroparia(t *testing.T) {
	ts, err := LoadTemples(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	sat := calendar.MustJDate(2026, time.November, 1) // Saturday; Ss Cosmas and Damian
	if sat.Weekday() != time.Saturday {
		t.Fatal("test date")
	}
	day := calendar.NewDayContext(sat)
	facts := func(temple, sign string, saint bool) Facts {
		f := Facts{Day: day, Sign: sign, Menaion: map[string][]string{}}
		if saint {
			f.Menaion["troparion"] = []string{"me.11.01.troparion.1"}
			f.Menaion["kontakion"] = []string{"me.11.01.kontakion.1"}
		}
		if temple != "" {
			if f.Temple, err = ts.Find(temple); err != nil {
				t.Fatal(err)
			}
		}
		return f
	}
	const tail = "common.glory,ch.kontakion.with_the_saints,common.and_now,ch.kontakion.day.sat"
	for _, c := range []struct {
		name         string
		temple, sign string
		saint        bool
		want         string
	}{
		{"saint", "", "none", true, "ch.troparion.day.sat,me.11.01.troparion.1,me.11.01.kontakion.1," + tail},
		{"no troparion: «Помяни, Господи»", "", "none", false, "ch.troparion.day.sat,ch.troparion.day.sat.2," + tail},
		// The Lord's and the Theotokos' temple: the troparion first, no kontakion.
		{"Lord's temple", "transfiguration", "none", true,
			"me.08.06.troparion.1,ch.troparion.day.sat,me.11.01.troparion.1,me.11.01.kontakion.1," + tail},
		{"Theotokos' temple", "dormition", "six_fold", true,
			"me.08.15.troparion.1,ch.troparion.day.sat,me.11.01.troparion.1,me.11.01.kontakion.1," + tail},
		// A saint's temple: nothing — «вси стии именовани суть в дневном тропаре».
		{"saint's temple", "nicholas", "none", true, "ch.troparion.day.sat,me.11.01.troparion.1,me.11.01.kontakion.1," + tail},
		// The great doxology: «Слава» the saint's, «И ныне» the temple's or «Предстательство».
		{"doxology", "", "doxology", true,
			"ch.troparion.day.sat,me.11.01.troparion.1,common.glory,me.11.01.kontakion.1,common.and_now,ch.kontakion.protection"},
		{"doxology, Theotokos' temple", "dormition", "doxology", true,
			"me.08.15.troparion.1,ch.troparion.day.sat,me.11.01.troparion.1,common.glory,me.11.01.kontakion.1,common.and_now,me.08.15.kontakion.1"},
	} {
		items, err := litTroparia(facts(c.temple, c.sign, c.saint), "")
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := refs(items); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

func TestSaturdayLiturgyPlan(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 1) // Saturday
	day := calendar.NewDayContext(d)
	ap, gos := day.Lections.Apostle, day.Lections.Gospel
	apRef := "lect.ap." + itoaN(ap.Week) + "." + itoaN(int(ap.Weekday))
	gosRef := "lect.gos." + itoaN(gos.Week) + "." + itoaN(int(gos.Weekday))
	if ap.Weekday != time.Saturday {
		t.Fatalf("Saturday readings: %+v", ap)
	}

	// A saint without readings of his own: the day's only.
	plan, err := planLiturgy(t, menaionDay(d), d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "liturgy.weekday" {
		t.Errorf("rule %s", plan.Rule)
	}
	want := map[string]string{
		"prokeimena": "lit.prokeimenon.tone.8,sl.prokeimenon.sat,sl.prokeimenon.sat,sl.prokeimenon.sat.verse,sl.prokeimenon.sat," +
			"sl.prokeimenon.sat,sl.prokeimenon.sat",
		"apostles": "lit.wisdom," + apRef + ".announce,lit.let_us_attend," + apRef,
		"alleluia": "lit.alleluia.tone.4,common.alleluia.thrice,sl.alleluia.sat.verse.1,common.alleluia.thrice," +
			"sl.alleluia.sat.verse.2,common.alleluia.thrice",
		"kinonik": "sl.kinonik.sat,common.alleluia.thrice",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}

	// The saint's own: before the day's (Типикон, гл. 12).
	me := "me.11.01."
	c := menaionDay(d)
	c.ids = append(c.ids, me+"lit.prokeimenon", me+"lit.prokeimenon.verse", me+"lit.apostle", me+"lit.alleluia.verse",
		me+"lit.gospel", me+"lit.kinonik")
	c.tones = map[string]int{me + "lit.prokeimenon": 7, me + "lit.alleluia.verse": 1}
	if plan, err = planLiturgy(t, c, d); err != nil {
		t.Fatal(err)
	}
	want = map[string]string{
		"prokeimena": "lit.prokeimenon.tone.7," + me + "lit.prokeimenon," + me + "lit.prokeimenon," + me + "lit.prokeimenon.verse," +
			me + "lit.prokeimenon," + me + "lit.prokeimenon," + me + "lit.prokeimenon," +
			"lit.prokeimenon.tone.8,sl.prokeimenon.sat,sl.prokeimenon.sat",
		"apostles": "lit.wisdom," + me + "lit.apostle.announce,lit.let_us_attend," + me + "lit.apostle," + apRef,
		"alleluia": "lit.alleluia.tone.1,common.alleluia.thrice," + me + "lit.alleluia.verse,common.alleluia.thrice," +
			"sl.alleluia.sat.verse.1,common.alleluia.thrice",
		"gospels": me + "lit.gospel.announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend," + me + "lit.gospel," + gosRef,
		"kinonik": me + "lit.kinonik,sl.kinonik.sat,common.alleluia.thrice",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("saint's readings, %s:\n got %s\nwant %s", slot, got, w)
		}
	}
}

func TestDepartedSaturdayLiturgy(t *testing.T) {
	d := calendar.MustJDate(2026, time.October, 25) // Saturday before St Demetrius
	day := calendar.NewDayContext(d)
	if day.Departed != calendar.DepartedDemetrius {
		t.Fatalf("departed %q", day.Departed)
	}
	ap, gos := day.Lections.Apostle, day.Lections.Gospel
	apRef := "lect.ap." + itoaN(ap.Week) + "." + itoaN(int(ap.Weekday))
	gosRef := "lect.gos." + itoaN(gos.Week) + "." + itoaN(int(gos.Weekday))
	plan, err := planLiturgy(t, menaionDay(d), d)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"prokeimena": "lit.prokeimenon.tone.8,sl.prokeimenon.sat,sl.prokeimenon.sat,sl.prokeimenon.sat.verse,sl.prokeimenon.sat," +
			"sl.prokeimenon.sat,sl.prokeimenon.sat,lit.prokeimenon.tone.7,sl.prokeimenon.sat.dead,sl.prokeimenon.sat.dead",
		"apostles": "lit.wisdom," + apRef + ".announce,lit.let_us_attend," + apRef + ",lect.zach.ap.270",
		"gospels":  gosRef + ".announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend," + gosRef + ",lect.zach.jn.16",
		"alleluia": "lit.alleluia.tone.4,common.alleluia.thrice,sl.alleluia.sat.verse.1,common.alleluia.thrice," +
			"sl.alleluia.sat.verse.2,common.alleluia.thrice,sl.alleluia.sat.dead.verse,common.alleluia.thrice",
		"kinonik": "sl.kinonik.sat,sl.kinonik.sat.dead,common.alleluia.thrice",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	// An ordinary Saturday: nothing for the departed.
	d = calendar.MustJDate(2026, time.October, 18)
	if plan, err = planLiturgy(t, menaionDay(d), d); err != nil {
		t.Fatal(err)
	}
	if got := refs(plan.Slots["kinonik"]); got != "sl.kinonik.sat,common.alleluia.thrice" {
		t.Errorf("ordinary Saturday kinonik: %s", got)
	}
}

func TestDepartedSaturdayMoved(t *testing.T) {
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	departedOn := func(c mapCorpus, d calendar.JDate) string {
		p := &Planner{Rules: rules, Feasts: feasts, Menaion: c}
		got, err := p.departed(context.Background(), d)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	// 2023: 22 October (the Kazan icon, polyeleos) is the calendar's Saturday.
	kazan, before := calendar.MustJDate(2023, time.October, 22), calendar.MustJDate(2023, time.October, 15)
	if kazan.Weekday() != time.Saturday {
		t.Fatal("test date")
	}
	c := mapCorpus{ids: []string{"ty.10.22.sign.1", "ty.10.22.sign.2"},
		signs: map[string]string{"ty.10.22.sign.1": "none", "ty.10.22.sign.2": "polyeleos"}}
	if got := departedOn(c, kazan); got != "" {
		t.Errorf("on the Kazan icon: %q", got)
	}
	if got := departedOn(c, before); got != calendar.DepartedDemetrius {
		t.Errorf("a week earlier: %q", got)
	}
	// Without the feast it stays.
	if got := departedOn(mapCorpus{}, kazan); got != calendar.DepartedDemetrius {
		t.Errorf("no feast: %q", got)
	}
	if got := departedOn(mapCorpus{}, before); got != "" {
		t.Errorf("no feast, a week earlier: %q", got)
	}
	// 2021: 24 October, «Всех скорбящих Радость» — great vespers in the Menaion only.
	joy := calendar.MustJDate(2021, time.October, 24)
	c = mapCorpus{ids: []string{"me.10.24p.vespers.great"}}
	if got := departedOn(c, joy); got != "" {
		t.Errorf("on «Всех скорбящих Радость»: %q", got)
	}
	if got := departedOn(c, joy.AddDays(-7)); got != calendar.DepartedDemetrius {
		t.Errorf("a week before «Всех скорбящих Радость»: %q", got)
	}
}
