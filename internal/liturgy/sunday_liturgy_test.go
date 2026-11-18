package liturgy

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func planSundayLiturgy(t *testing.T, c mapCorpus, d calendar.JDate, temple string) (*Plan, error) {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	temples, err := LoadTemples(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	return (&Planner{Rules: rules, Feasts: feasts, Temples: temples, Menaion: c}).Plan(context.Background(), d, "liturgy",
		Params{Priest: true, Temple: temple})
}

func TestSundayLiturgy(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 9) // Sunday, tone 8
	day := calendar.NewDayContext(d)
	oc := "oc." + itoa(day.Tone) + ".sun."
	me := "me.11.09."
	b := func(i int) string { return "ch.typika.beatitudes." + itoa(i) }
	// b(10) has two digits
	b10 := "ch.typika.beatitudes.10"

	t.Run("saint", func(t *testing.T) {
		c := sundayCorpus(d, nil, "kontakion.1", "matins.canon.1.ode.3.troparion.1", "matins.canon.1.ode.3.troparion.2",
			"matins.canon.1.ode.3.troparion.3")
		plan, err := planSundayLiturgy(t, c, d, "")
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != "liturgy.sunday" {
			t.Fatalf("rule %s", plan.Rule)
		}
		odes := me + "matins.canon.1.ode.3.troparion."
		check(t, plan, map[string]string{
			// On 8: the Octoechos on 4, the saint's on 4 (3 given: the first repeated).
			"antiphon_3": strings.Join([]string{b(0), b(1), b(2), b(3), oc + "lit.beatitudes.1", b(4), oc + "lit.beatitudes.2",
				b(5), oc + "lit.beatitudes.3", b(6), oc + "lit.beatitudes.4", b(7), odes + "1", b(8), odes + "2", b(9), odes + "3",
				b10, odes + "1", "common.glory", oc + "lit.beatitudes.glory", "common.and_now", oc + "lit.beatitudes.now"}, ","),
			"entrance": "ch.lit.come_let_us_worship.sunday",
			"troparia": oc + "troparion," + me + "troparion.1," + oc + "kontakion,common.glory," + me + "kontakion.1,common.and_now,ch.kontakion.protection",
			"prokeimena": "lit.prokeimenon.tone.8," + oc + "lit.prokeimenon," + oc + "lit.prokeimenon," + oc + "lit.prokeimenon.verse," +
				oc + "lit.prokeimenon," + oc + "lit.prokeimenon," + oc + "lit.prokeimenon",
			"alleluia": "lit.alleluia.tone.8,common.alleluia.thrice," + oc + "lit.alleluia.verse.1,common.alleluia.thrice," + oc +
				"lit.alleluia.verse.2,common.alleluia.thrice",
			"kinonik":          "sl.kinonik.sun,common.alleluia.thrice",
			"dismissal_prefix": "sl.dismissal.prefix.sunday",
		})
		if a := refs(plan.Slots["apostles"]); !strings.HasSuffix(a, ".0") {
			t.Errorf("apostle %s: want the Sunday reading", a)
		}
	})
	t.Run("no Menaion, church of the Theotokos", func(t *testing.T) {
		c := sundayCorpus(d, nil)
		c.ids = c.ids[1:] // without the Menaion troparion
		plan, err := planSundayLiturgy(t, c, d, "dormition")
		if err != nil {
			t.Fatal(err)
		}
		check(t, plan, map[string]string{
			"antiphon_3": strings.Join([]string{b(0), b(1), b(2), b(3), b(4), b(5), oc + "lit.beatitudes.1", b(6), oc + "lit.beatitudes.2",
				b(7), oc + "lit.beatitudes.3", b(8), oc + "lit.beatitudes.4", b(9), oc + "lit.beatitudes.5", b10, oc + "lit.beatitudes.6",
				"common.glory", oc + "lit.beatitudes.glory", "common.and_now", oc + "lit.beatitudes.now"}, ","),
			"troparia": oc + "troparion,me.08.15.troparion.1," + oc + "kontakion,common.glory_now,me.08.15.kontakion.1",
		})
	})
	t.Run("church of a saint", func(t *testing.T) {
		plan, err := planSundayLiturgy(t, sundayCorpus(d, nil, "kontakion.1"), d, "nicholas")
		if err != nil {
			t.Fatal(err)
		}
		check(t, plan, map[string]string{
			"troparia": oc + "troparion,me.12.06.troparion.1," + me + "troparion.1," + oc + "kontakion,me.12.06.kontakion.1,common.glory," +
				me + "kontakion.1,common.and_now,ch.kontakion.protection",
		})
	})
	t.Run("saint with a sign", func(t *testing.T) {
		c := sundayCorpus(d, map[string]int{"lit.prokeimenon": 4}, "kontakion.1", "vespers.great", "gvespers.paremia.1",
			"gvespers.paremia.2", "gvespers.paremia.3", "lit.beatitudes.1", "lit.beatitudes.2", "lit.prokeimenon",
			"lit.prokeimenon.verse", "lit.apostle", "lit.gospel", "lit.alleluia.verse.1", "lit.kinonik")
		plan, err := planSundayLiturgy(t, c, d, "")
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != "liturgy.sunday" {
			t.Fatalf("rule %s", plan.Rule)
		}
		for slot, want := range map[string]string{
			"prokeimena": ",lit.prokeimenon.tone.4," + me + "lit.prokeimenon," + me + "lit.prokeimenon",
			"apostles":   "," + me + "lit.apostle",
			"gospels":    "," + me + "lit.gospel",
			"kinonik":    "sl.kinonik.sun," + me + "lit.kinonik,common.alleluia.thrice",
		} {
			if got := refs(plan.Slots[slot]); !strings.HasSuffix(got, want) {
				t.Errorf("%s: %s, want …%s", slot, got, want)
			}
		}
		if a := refs(plan.Slots["antiphon_3"]); !strings.Contains(a, b(9)+","+me+"lit.beatitudes.1,"+b10+","+me+"lit.beatitudes.2,common.glory") {
			t.Errorf("beatitudes %s: the saint's 2 on 4 (repeated)", a)
		}
	})
	t.Run("saint with a sign, readings not given", func(t *testing.T) {
		_, err := planSundayLiturgy(t, sundayCorpus(d, nil, "vespers.great"), d, "")
		var ue *UnsupportedError
		if !asUnsupported(err, &ue) || ue.Rule != "liturgy.sunday.unsupported" {
			t.Errorf("got %v", err)
		}
	})
}

func asUnsupported(err error, ue **UnsupportedError) bool { return errors.As(err, ue) }

// sundayFeastCorpus: a Sunday of the Octoechos with the units of the given
// Menaion days (ids relative to me.MM.DD).
func sundayFeastCorpus(d calendar.JDate, days map[string][]string, of map[string]string) mapCorpus {
	c := sundayCorpus(d, nil)
	c.ids = c.ids[1:] // the Menaion of the day comes from days
	for day, units := range days {
		for _, u := range units {
			c.ids = append(c.ids, day+"."+u)
		}
	}
	c.of = of
	return c
}

func TestSundayLiturgyFeasts(t *testing.T) {
	t.Run("afterfeast of the Dormition", func(t *testing.T) {
		d := calendar.MustJDate(2027, time.August, 16)
		if d.Weekday() != time.Sunday {
			t.Fatalf("%v", d.Weekday())
		}
		oc := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun."
		c := sundayFeastCorpus(d, map[string][]string{
			"me.08.15": {"troparion.1", "kontakion.1", "matins.canon.1.ode.9.irmos", "gvespers.paremia.1"},
			"me.08.16": {"troparion.1", "kontakion.1", "kontakion.2", "lit.beatitudes.1"},
		}, map[string]string{"me.08.16.kontakion.1": "праздника бцды,"})
		plan, err := planSundayLiturgy(t, c, d, "nicholas")
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != "liturgy.sunday.afterfeast" {
			t.Fatalf("rule %s", plan.Rule)
		}
		// The feast's kontakion copied into the day is not the saint's; no temple in the afterfeast.
		for slot, want := range map[string]string{
			"troparia":         oc + "troparion,me.08.15.troparion.1,me.08.16.troparion.1," + oc + "kontakion,common.glory,me.08.16.kontakion.2,common.and_now,me.08.15.kontakion.1",
			"truly_meet":       "lit.zadostoinik.refrain.dormition,me.08.15.matins.canon.1.ode.9.irmos",
			"entrance":         "ch.lit.come_let_us_worship.sunday",
			"dismissal_prefix": "sl.dismissal.prefix.sunday",
		} {
			if got := refs(plan.Slots[slot]); got != want {
				t.Errorf("%s:\n got %s\nwant %s", slot, got, want)
			}
		}
	})
	t.Run("apodosis of the Exaltation", func(t *testing.T) {
		d := calendar.MustJDate(2026, time.September, 21)
		c := sundayFeastCorpus(d, map[string][]string{
			"me.09.14": {"troparion.1", "kontakion.1", "lit.trisagion", "matins.canon.1.ode.9.irmos"},
			"me.09.21": {"troparion.1", "kontakion.1"},
		}, map[string]string{"me.09.21.troparion.1": "праздника,", "me.09.21.kontakion.1": "праздника,"})
		plan, err := planSundayLiturgy(t, c, d, "")
		if err != nil {
			t.Fatal(err)
		}
		oc := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun."
		for slot, want := range map[string]string{
			// «Кресту Твоему» on the apodosis.
			"trisagion": "me.09.14.lit.trisagionx3,common.glory_now,me.09.14.lit.trisagion",
			"troparia":  oc + "troparion,me.09.14.troparion.1," + oc + "kontakion,common.glory_now,me.09.14.kontakion.1",
		} {
			if got := refs(plan.Slots[slot]); got != want {
				t.Errorf("%s:\n got %s\nwant %s", slot, got, want)
			}
		}
	})
	t.Run("Nativity of the Theotokos on Sunday", func(t *testing.T) {
		d := calendar.MustJDate(2031, time.September, 8)
		c := sundayFeastCorpus(d, map[string][]string{
			"me.09.08": {"troparion.1", "kontakion.1", "gvespers.paremia.1", "gvespers.paremia.2", "gvespers.paremia.3",
				"lit.beatitudes.1", "lit.apostle", "lit.gospel", "matins.canon.1.ode.9.irmos"},
		}, nil)
		plan, err := planSundayLiturgy(t, c, d, "dormition")
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != "liturgy.sunday.theotokos_feast" {
			t.Fatalf("rule %s", plan.Rule)
		}
		oc := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun."
		for slot, want := range map[string]string{
			// No temple's hymns at a twelve feast.
			"troparia":   oc + "troparion,me.09.08.troparion.1,common.glory," + oc + "kontakion,common.and_now,me.09.08.kontakion.1",
			"truly_meet": "lit.zadostoinik.refrain.nativity_of_theotokos,me.09.08.matins.canon.1.ode.9.irmos",
		} {
			if got := refs(plan.Slots[slot]); got != want {
				t.Errorf("%s:\n got %s\nwant %s", slot, got, want)
			}
		}
		if a := refs(plan.Slots["apostles"]); !strings.HasSuffix(a, ".0,me.09.08.lit.apostle") {
			t.Errorf("apostles %s: the Sunday's, then the feast's", a)
		}
	})
}
