package liturgy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// feastCorpus: a Menaion service with great vespers and its liturgy;
// festal antiphons if festal, the beatitudes from the canon otherwise.
func feastCorpus(d calendar.JDate, festal bool) mapCorpus {
	day := calendar.NewDayContext(d)
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1", me + "kontakion.1", me + "vespers.great",
		me + "gvespers.paremia.1", me + "gvespers.paremia.2", me + "gvespers.paremia.3",
		me + "lit.prokeimenon", me + "lit.prokeimenon.verse", me + "lit.alleluia.verse.1", me + "lit.alleluia.verse.2",
		me + "lit.apostle", me + "lit.apostle.announce", me + "lit.gospel", me + "lit.gospel.announce", me + "lit.kinonik",
		me + "lit.zadostoinik"}
	if festal {
		for a := 1; a <= 3; a++ {
			for k := 1; k <= 3; k++ {
				ids = append(ids, fmt.Sprintf("%slit.antiphon.%d.%d", me, a, k))
			}
		}
		ids = append(ids, me+"lit.entrance", me+"lit.trisagion")
	} else {
		for k := 1; k <= 8; k++ {
			ids = append(ids, fmt.Sprintf("%slit.beatitudes.%d", me, k))
		}
		ids = append(ids, me+"lit.beatitudes.theotokion")
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 1, me + "lit.prokeimenon": 7, me + "lit.alleluia.verse.1": 1}}
}

func TestFeastLiturgy(t *testing.T) {
	// The Exaltation, 2026-09-14 O.S. — on Sunday: a feast of the Lord.
	d := calendar.MustJDate(2026, time.September, 14)
	plan, err := planLiturgy(t, feastCorpus(d, true), d)
	if err != nil {
		t.Fatal(err)
	}
	me := "me.09.14."
	if plan.Rule != "liturgy.lord_feast_on_sunday" {
		t.Errorf("rule %s", plan.Rule)
	}
	want := map[string]string{
		"antiphon_1": me + "lit.antiphon.1.1," + me + "lit.antiphon.1.2," + me + "lit.antiphon.1.3",
		"entrance":   me + "lit.entrance",
		"troparia":   me + "troparion.1,common.glory_now," + me + "kontakion.1",
		"prokeimena": strings.Join([]string{"lit.prokeimenon.tone.7", me + "lit.prokeimenon", me + "lit.prokeimenon", me + "lit.prokeimenon.verse",
			me + "lit.prokeimenon", me + "lit.prokeimenon", me + "lit.prokeimenon"}, ","),
		"apostles":   "lit.wisdom," + me + "lit.apostle.announce,lit.let_us_attend," + me + "lit.apostle",
		"truly_meet": "lit.zadostoinik.refrain.exaltation," + me + "lit.zadostoinik",
		"kinonik":    me + "lit.kinonik,common.alleluia.thrice",
	}
	want["trisagion"] = me + "lit.trisagionx3,common.glory_now," + me + "lit.trisagion"
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
}

func TestGreatSaintLiturgy(t *testing.T) {
	// St John Chrysostom, Thursday 2026-11-13 O.S.: day and saint.
	d := calendar.MustJDate(2026, time.November, 13)
	plan, err := planLiturgy(t, feastCorpus(d, false), d)
	if err != nil {
		t.Fatal(err)
	}
	me := "me.11.13."
	if plan.Rule != "liturgy.great_saint" {
		t.Fatalf("rule %s", plan.Rule)
	}
	b := refs(plan.Slots["antiphon_3"])
	if !strings.Contains(b, "ch.typika.beatitudes.5,"+me+"lit.beatitudes.1,ch.typika.beatitudes.6,"+me+"lit.beatitudes.2") ||
		!strings.HasSuffix(b, "common.and_now,"+me+"lit.beatitudes.theotokion") {
		t.Errorf("beatitudes on 8: %s", b)
	}
	if got := refs(plan.Slots["troparia"]); got != me+"troparion.1,common.glory,"+me+"kontakion.1,common.and_now,ch.kontakion.protection" {
		t.Errorf("troparia: %s", got)
	}
	ap := refs(plan.Slots["apostles"])
	if !strings.HasPrefix(ap, "lit.wisdom,lect.ap.") || !strings.HasSuffix(ap, me+"lit.apostle") || strings.Contains(ap, me+"lit.apostle.announce") {
		t.Errorf("apostles: the day's announced, the saint's joined to it: %s", ap)
	}
	if g := refs(plan.Slots["gospels"]); strings.Count(g, ".announce") != 1 || !strings.HasSuffix(g, me+"lit.gospel") {
		t.Errorf("gospels: one announcement: %s", g)
	}
	if got := refs(plan.Slots["kinonik"]); got != "sl.kinonik.thu,"+me+"lit.kinonik,common.alleluia.thrice" {
		t.Errorf("kinonik: %s", got)
	}
	if got := refs(plan.Slots["truly_meet"]); got != "ch.lit.it_is_truly_meet" {
		t.Errorf("a saint has no zadostoinik: %s", got)
	}
}

// The Nativity and the Theophany: St John Chrysostom's liturgy on the feast
// when the eve is a weekday; the feast's own troparion (the Theophany's is
// the Menaion's third), the zadostoinik with its refrain; St Basil's
// (TestBasilLiturgy) when the feast falls on Sunday or Monday; the eve.
func TestNativityTheophanyLiturgy(t *testing.T) {
	for _, c := range []struct {
		date            calendar.JDate
		rule, troparion string
		unsupported     bool
	}{
		{calendar.MustJDate(2026, time.December, 25), "liturgy.nativity_theophany", "me.12.25.troparion.1", false}, // Thursday
		{calendar.MustJDate(2027, time.January, 6), "liturgy.nativity_theophany", "me.01.06.troparion.3", false},   // Tuesday
	} {
		day := calendar.NewDayContext(c.date)
		me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
		corpus := feastCorpus(c.date, true)
		corpus.ids = append(corpus.ids, me+"troparion.2", me+"troparion.3")
		plan, err := planLiturgy(t, corpus, c.date)
		if c.unsupported {
			var ue *UnsupportedError
			if !errors.As(err, &ue) || ue.Rule != c.rule {
				t.Errorf("%v: want unsupported %s, got %v", c.date, c.rule, err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		feast := map[string]string{"me.12.25.": "nativity", "me.01.06.": "theophany"}[me]
		check(t, plan, map[string]string{
			"troparia":         c.troparion + ",common.glory_now," + me + "kontakion.1",
			"truly_meet":       "ir.pripev." + feast + ".1.irmos,lit.zadostoinik." + feast,
			"dismissal_prefix": "sl.dismissal.prefix." + feast,
		})
		if plan.Rule != c.rule {
			t.Errorf("%v: rule %s", c.date, plan.Rule)
		}
	}
}

// The liturgy of St Basil the Great: the feast of the Nativity or the
// Theophany on Sunday or Monday, 1 January — slot basil; «О Тебе радуется»
// instead of «Достойно есть», a feast's zadostoinik kept.
func TestBasilLiturgy(t *testing.T) {
	for _, c := range []struct {
		date  calendar.JDate
		basil bool
		truly string
	}{
		{calendar.MustJDate(2026, time.January, 6), true, "ir.pripev.theophany.1.irmos,lit.zadostoinik.theophany"},  // Monday
		{calendar.MustJDate(2027, time.January, 6), false, "ir.pripev.theophany.1.irmos,lit.zadostoinik.theophany"}, // Tuesday
		{calendar.MustJDate(2027, time.January, 1), true, "ch.lit.all_of_creation"},
		{calendar.MustJDate(2026, time.November, 13), false, "ch.lit.it_is_truly_meet"},
	} {
		corpus := feastCorpus(c.date, c.date.Day() != 13)
		day := calendar.NewDayContext(c.date)
		me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
		corpus.ids = append(corpus.ids, me+"troparion.3")
		if c.date.Day() == 1 {
			corpus.ids = removeID(corpus.ids, me+"lit.zadostoinik")
		}
		plan, err := planLiturgy(t, corpus, c.date)
		if err != nil {
			t.Fatalf("%v: %v", c.date, err)
		}
		if got := len(plan.Slots["basil"]) > 0; got != c.basil {
			t.Errorf("%v: basil %v", c.date, got)
		}
		if got := refs(plan.Slots["truly_meet"]); got != c.truly {
			t.Errorf("%v: truly_meet %s, want %s", c.date, got, c.truly)
		}
	}
}

func removeID(ids []string, id string) []string {
	var out []string
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

// eveCorpus: the vespers of the eve with the liturgy (ev.<feast>.*).
func eveCorpus(d calendar.JDate, feast string) mapCorpus {
	c := feastCorpus(d, false)
	var ids []string
	for _, id := range c.ids { // the eve has no great vespers of its own
		if !strings.Contains(id, "vespers.great") && !strings.Contains(id, "gvespers.paremia") {
			ids = append(ids, id)
		}
	}
	c.ids = ids
	ev := "ev." + feast + "."
	for n := 1; n <= 4; n++ {
		c.ids = append(c.ids, fmt.Sprintf("%slic.%d", ev, n))
	}
	c.ids = append(c.ids, ev+"lic.glory_now", ev+"reading.1", ev+"title.1", ev+"troparion.2", ev+"verse.3", ev+"refrain.4",
		ev+"mark.5", ev+"lit.prokeimenon", ev+"lit.prokeimenon.verse", ev+"lit.alleluia.verse.1", ev+"lit.apostle",
		ev+"lit.gospel", ev+"lit.kinonik")
	c.tones[ev+"lit.prokeimenon"], c.tones[ev+"lit.alleluia.verse.1"] = 1, 5
	return c
}

// The eve of the Nativity or the Theophany on a weekday: vespers joined to
// St Basil's liturgy; on Friday the Nativity's eve has the great prokeimenon.
func TestEveLiturgy(t *testing.T) {
	ev := "ev.nativity."
	d := calendar.MustJDate(2026, time.December, 24) // Wednesday
	plan, err := planLiturgy(t, eveCorpus(d, "nativity"), d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "liturgy.nativity_theophany.eve" || len(plan.Slots["basil"]) == 0 {
		t.Fatalf("rule %s, basil %v", plan.Rule, plan.Slots["basil"])
	}
	lic := refs(plan.Slots["eve_lord_i_call"])
	if !strings.HasPrefix(lic, "ch.vespers.lic.verse.23,"+ev+"lic.1,ch.vespers.lic.verse.24,"+ev+"lic.1,ch.vespers.lic.verse.25,"+ev+"lic.2") ||
		!strings.HasSuffix(lic, "common.glory_now,"+ev+"lic.glory_now") {
		t.Errorf("Lord, I call: %s", lic)
	}
	check(t, plan, map[string]string{
		"eve_prokeimenon": "ch.vespers.prokeimenon.thu,ch.vespers.prokeimenon.thu,ch.vespers.prokeimenon.thu.verse," +
			"ch.vespers.prokeimenon.thu,ch.vespers.prokeimenon.thu,ch.vespers.prokeimenon.thu",
		"eve_readings": "lit.wisdom," + ev + "title.1,lit.let_us_attend," + ev + "reading.1," + ev + "troparion.2," + ev + "verse.3," +
			ev + "refrain.4," + ev + "mark.5",
		"apostles":   "lit.wisdom," + ev + "lit.apostle.announce,lit.let_us_attend," + ev + "lit.apostle",
		"truly_meet": "ch.lit.all_of_creation",
		"kinonik":    ev + "lit.kinonik,common.alleluia.thrice",
	})
	// On Friday: «кто бог велий».
	for y := 2020; y < 2040; y++ {
		d := calendar.MustJDate(y, time.December, 24)
		if calendar.NewDayContext(d).Weekday != time.Friday {
			continue
		}
		plan, err := planLiturgy(t, eveCorpus(d, "nativity"), d)
		if err != nil {
			t.Fatal(err)
		}
		if got := refs(plan.Slots["eve_prokeimenon"]); !strings.HasPrefix(got, "lit.prokeimenon.tone.7,ch.vespers.prokeimenon.great.who,") {
			t.Errorf("%v: %s", d, got)
		}
		break
	}
	// On Sunday the eve's liturgy is St John Chrysostom's: the Sunday's
	// (before the Theophany).
	for y := 2020; y < 2040; y++ {
		d := calendar.MustJDate(y, time.January, 5)
		if calendar.NewDayContext(d).Weekday != time.Sunday {
			continue
		}
		c := eveCorpus(d, "theophany")
		c.ids = append(c.ids, sundayCorpus(d, nil).ids...)
		c.ids = append(c.ids, "ms.before_theophany.lit.apostle", "ms.before_theophany.lit.gospel")
		plan, err := planLiturgy(t, c, d)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != "liturgy.sunday.special" || len(plan.Slots["basil"]) > 0 {
			t.Errorf("%v: rule %s, basil %v", d, plan.Rule, plan.Slots["basil"])
		}
		break
	}
	// On Saturday: the day's, the readings of the Saturday before the feast
	// first, the forefeast's hymns.
	for y := 2020; y < 2040; y++ {
		d := calendar.MustJDate(y, time.December, 24)
		day := calendar.NewDayContext(d)
		if day.Weekday != time.Saturday {
			continue
		}
		c := eveCorpus(d, "nativity")
		c.ids = append(c.ids, "ms.sat_before_nativity.lit.apostle", "ms.sat_before_nativity.lit.gospel")
		plan, err := planLiturgy(t, c, d)
		if err != nil {
			t.Fatal(err)
		}
		if a := refs(plan.Slots["apostles"]); !strings.HasPrefix(a, "lit.wisdom,ms.sat_before_nativity.lit.apostle.announce,") {
			t.Errorf("%v: apostles %s", d, a)
		}
		if tr := refs(plan.Slots["troparia"]); strings.Contains(tr, "day.sat") {
			t.Errorf("%v: the Saturday's hymns in the forefeast: %s", d, tr)
		}
		// The vespers apart, that evening.
		v, err := (&Planner{Rules: mustRules(t), Feasts: mustFeasts(t), Menaion: c}).Plan(context.Background(), d.AddDays(1), "vespers", Params{Priest: true})
		if err != nil {
			t.Fatal(err)
		}
		if v.Rule != "vespers.nativity_theophany.weekend" || len(v.Slots["eve_gospel"]) == 0 || len(v.Slots["kathisma"]) != 3 ||
			refs(v.Slots["eve_after"]) != "me.12.25.troparion.1,common.glory_now,me.12.25.kontakion.1" {
			t.Errorf("%v vespers: %s %v", d, v.Rule, v.Slots["eve_after"])
		}
		break
	}
}

func mustRules(t *testing.T) *Rules {
	t.Helper()
	r, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func mustFeasts(t *testing.T) *Feasts {
	t.Helper()
	f, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	return f
}
