package liturgy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// triodionCorpus: a day of the preparatory weeks with its Triodion units
// (tp.<id>.*): the 3rd and 6th odes, the hymns, the own prokeimenon and
// alleluia when given.
func triodionCorpus(d calendar.JDate, id string, own bool) mapCorpus {
	c := sundayCorpus(d, nil)
	tp := "tp." + id + "."
	for _, o := range []string{"3", "6"} {
		for k := 1; k <= 3; k++ {
			c.ids = append(c.ids, tp+"canon.ode."+o+".troparion."+itoa(k))
		}
		c.ids = append(c.ids, tp+"canon.ode."+o+".glory", tp+"canon.ode."+o+".theotokion")
	}
	c.ids = append(c.ids, tp+"kontakion", tp+"troparion", tp+"theotokion")
	if own {
		c.ids = append(c.ids, tp+"lit.prokeimenon", tp+"lit.prokeimenon.verse", tp+"lit.alleluia.verse.1")
		c.tones[tp+"lit.prokeimenon"], c.tones[tp+"lit.alleluia.verse.1"] = 3, 8
	}
	return c
}

func TestTriodionLiturgy(t *testing.T) {
	pascha := calendar.Pascha(2027)
	oc := func(d calendar.JDate) string { return "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun." }
	t.Run("the Publican", func(t *testing.T) {
		d := pascha.AddDays(-70)
		p, err := planLiturgy(t, triodionCorpus(d, "publican", false), d)
		if err != nil {
			t.Fatal(err)
		}
		if p.Rule != "liturgy.triodion.sunday" {
			t.Fatalf("rule %s", p.Rule)
		}
		o, tp := oc(d), "tp.publican."
		check(t, p, map[string]string{
			// The saint of the Menaion is sung at compline: not here.
			"troparia":   o + "troparion," + o + "kontakion,common.glory," + tp + "kontakion,common.and_now,ch.kontakion.protection",
			"prokeimena": "lit.prokeimenon.tone." + itoa(calendar.NewDayContext(d).Tone) + "," + o + "lit.prokeimenon," + o + "lit.prokeimenon," + o + "lit.prokeimenon.verse," + o + "lit.prokeimenon," + o + "lit.prokeimenon," + o + "lit.prokeimenon",
			"apostles":   "lit.wisdom,lect.ap.33.0.announce,lit.let_us_attend,lect.ap.33.0",
			"gospels":    "lect.gos.33.0.announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend,lect.gos.33.0",
			"kinonik":    "sl.kinonik.sun,common.alleluia.thrice",
		})
		b := refs(p.Slots["antiphon_3"])
		if !strings.Contains(b, o+"lit.beatitudes.4,ch.typika.beatitudes.7,"+tp+"canon.ode.6.troparion.1") ||
			!strings.HasSuffix(b, "common.glory,"+tp+"canon.ode.6.glory,common.and_now,"+tp+"canon.ode.6.theotokion") {
			t.Errorf("beatitudes: %s", b)
		}
	})
	t.Run("meatfare: its own prokeimenon and alleluia", func(t *testing.T) {
		d := pascha.AddDays(-56)
		p, err := planLiturgy(t, triodionCorpus(d, "meatfare", true), d)
		if err != nil {
			t.Fatal(err)
		}
		tp := "tp.meatfare."
		if got := refs(p.Slots["prokeimena"]); !strings.HasPrefix(got, "lit.prokeimenon.tone.3,"+tp+"lit.prokeimenon,") {
			t.Errorf("prokeimenon: %s", got)
		}
		if got := refs(p.Slots["alleluia"]); !strings.HasPrefix(got, "lit.alleluia.tone.8,common.alleluia.thrice,"+tp+"lit.alleluia.verse.1") {
			t.Errorf("alleluia: %s", got)
		}
	})
	t.Run("the meatfare Saturday of the departed", func(t *testing.T) {
		d := pascha.AddDays(-57)
		p, err := planLiturgy(t, triodionCorpus(d, "sat_meatfare", false), d)
		if err != nil {
			t.Fatal(err)
		}
		tp := "tp.sat_meatfare."
		if p.Rule != "liturgy.triodion.saturday" {
			t.Fatalf("rule %s", p.Rule)
		}
		check(t, p, map[string]string{
			"troparia": tp + "troparion,common.glory," + tp + "kontakion,common.and_now," + tp + "theotokion",
			"apostles": "lit.wisdom,lect.ap.35.6.announce,lit.let_us_attend,lect.ap.35.6,lect.zach.ap.270",
			"gospels":  "lect.gos.35.6.announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend,lect.gos.35.6,lect.zach.jn.16",
			"alleluia": "lit.alleluia.tone.6,common.alleluia.thrice,sl.alleluia.sat.dead.verse,common.alleluia.thrice",
			"kinonik":  "sl.kinonik.sat.dead,common.alleluia.thrice",
		})
		// On 8: the 3rd and the 6th odes.
		if b := refs(p.Slots["antiphon_3"]); !strings.Contains(b, "ch.typika.beatitudes.3,"+tp+"canon.ode.3.troparion.1") ||
			!strings.Contains(b, tp+"canon.ode.6.troparion.1") {
			t.Errorf("beatitudes: %s", b)
		}
	})
	t.Run("no liturgy on Wednesday and Friday of cheesefare week", func(t *testing.T) {
		for _, off := range []int{-53, -51} {
			d := pascha.AddDays(off)
			_, err := planLiturgy(t, sundayCorpus(d, nil), d)
			var ue *UnsupportedError
			if !errors.As(err, &ue) || ue.Rule != "liturgy.triodion.cheesefare_no_liturgy" {
				t.Errorf("%v: %v", d, err)
			}
		}
	})
}

func TestLentWeekends(t *testing.T) {
	pascha := calendar.Pascha(2027)
	t.Run("the Sunday of the Cross: St Basil's liturgy", func(t *testing.T) {
		d := pascha.AddDays(-28)
		c := triodionCorpus(d, "lent3", true)
		p, err := planLiturgy(t, c, d)
		if err != nil {
			t.Fatal(err)
		}
		if p.Rule != "liturgy.triodion.lent_sunday" || len(p.Slots["basil"]) == 0 {
			t.Fatalf("rule %s, basil %v", p.Rule, p.Slots["basil"])
		}
		o, tp := "oc."+itoa(calendar.NewDayContext(d).Tone)+".sun.", "tp.lent3."
		check(t, p, map[string]string{
			"troparia":  o + "troparion," + tp + "troparion," + o + "kontakion,common.glory," + tp + "kontakion,common.and_now,ch.kontakion.protection",
			"trisagion": "me.09.14.lit.trisagionx3,common.glory_now,me.09.14.lit.trisagion",
			"kinonik":   "me.09.14.lit.kinonik,common.alleluia.thrice",
		})
	})
	t.Run("a parental Saturday", func(t *testing.T) {
		d := pascha.AddDays(-36)
		p, err := planLiturgy(t, triodionCorpus(d, "sat_lent2", false), d)
		if err != nil {
			t.Fatal(err)
		}
		check(t, p, map[string]string{
			"troparia": "ch.troparion.day.sat,ch.troparion.day.sat.2,common.glory,ch.kontakion.with_the_saints,common.and_now,tp.sat_meatfare.theotokion",
			"apostles": "lit.wisdom,lect.ap.38.6.announce,lit.let_us_attend,lect.ap.38.6,lect.zach.ap.270",
			"gospels":  "lect.gos.38.6.announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend,lect.gos.38.6,lect.zach.jn.16",
			"kinonik":  "sl.kinonik.sat,sl.kinonik.sat.dead,common.alleluia.thrice",
		})
		if pk := refs(p.Slots["prokeimena"]); !strings.HasSuffix(pk, "lit.prokeimenon.tone.6,sl.prokeimenon.sat.dead,sl.prokeimenon.sat.dead") {
			t.Errorf("prokeimena: %s", pk)
		}
	})
	t.Run("the hours", func(t *testing.T) {
		d := pascha.AddDays(-42)
		o := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun."
		p := planHour(t, triodionCorpus(d, "lent1", false), d, "third_hour")
		if got := refs(p.Slots["troparion_first"]) + "|" + refs(p.Slots["troparion"]) + "|" + refs(p.Slots["kontakion"]); got != o+"troparion|tp.lent1.troparion|tp.lent1.kontakion" {
			t.Errorf("Sunday: %s", got)
		}
		d = pascha.AddDays(-29)
		p = planHour(t, triodionCorpus(d, "sat_lent3", false), d, "sixth_hour")
		if got := refs(p.Slots["troparion_first"]) + "|" + refs(p.Slots["troparion"]) + "|" + refs(p.Slots["kontakion"]); got != "ch.troparion.day.sat|ch.troparion.day.sat.2|ch.kontakion.with_the_saints" {
			t.Errorf("parental Saturday: %s", got)
		}
	})
}

func TestLentSundayVespers(t *testing.T) {
	d := calendar.Pascha(2027).AddDays(-28) // the Sunday of the Cross
	c := triodionCorpus(d, "lent3", false)
	tp := "tp.lent3."
	for n := 1; n <= 4; n++ {
		c.ids = append(c.ids, tp+"gvespers.lic.sticheron."+itoa(n))
	}
	c.ids = append(c.ids, tp+"gvespers.lic.glory", tp+"gvespers.aposticha.glory_now")
	p, err := (&Planner{Rules: mustRules(t), Feasts: mustFeasts(t), Menaion: c}).Plan(context.Background(), d, "vespers", Params{Priest: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Rule != "vespers.triodion.lent_sunday" {
		t.Fatalf("rule %s", p.Rule)
	}
	lic := refs(p.Slots["lord_i_call"])
	if strings.Count(lic, tp+"gvespers.lic.sticheron.") != 4 || strings.Count(lic, "gvespers.lic.sticheron.") != 10 {
		t.Errorf("«Господи, воззвах» on 10, the Triodion's on 4: %s", lic)
	}
	check(t, p, map[string]string{
		"troparia": "ch.theotokos_rejoicex2," + tp + "troparion",
	})
	if a := refs(p.Slots["aposticha"]); !strings.HasSuffix(a, "common.glory_now,"+tp+"gvespers.aposticha.glory_now") {
		t.Errorf("aposticha: %s", a)
	}
}

func TestLentSundayMatins(t *testing.T) {
	d := calendar.Pascha(2027).AddDays(-42) // the Sunday of Orthodoxy
	c := sundayMatinsCorpus(d, nil, nil)
	tp := "tp.lent1."
	for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
		b := fmt.Sprintf("%scanon.ode.%d.", tp, o)
		c.ids = append(c.ids, b+"troparion.1", b+"troparion.2", b+"theotokion", b+"katavasia")
	}
	c.ids = append(c.ids, tp+"troparion", tp+"kontakion", tp+"ikos", tp+"matins.sedalen.1", tp+"matins.sedalen.theotokion",
		tp+"matins.exapostilarion.1", tp+"matins.exapostilarion.theotokion", tp+"matins.praises.sticheron.1",
		tp+"matins.praises.sticheron.2", tp+"matins.praises.sticheron.3", tp+"matins.praises.sticheron.4", tp+"matins.praises.glory")
	c.tones[tp+"troparion"] = 2
	p := planSundayMatins(t, c, d)
	if p.Rule != "matins.triodion.lent_sunday" {
		t.Fatalf("rule %s", p.Rule)
	}
	check(t, p, map[string]string{
		"ps50":      "ch.psalm.50,common.glory,tp.ps50.glory,common.and_now,tp.ps50.now,ch.matins.ps50.verse,tp.ps50.sticheron",
		"kontakion": tp + "kontakion," + tp + "ikos",
		"sedalen":   tp + "matins.sedalen.1,common.glory_now," + tp + "matins.sedalen.theotokion",
	})
	if k := refs(p.Slots["kathisma_17"]); !strings.HasPrefix(k, "ps.kathisma.17.1") {
		t.Errorf("«Непорочны»: %s", k)
	}
	c13 := refs(p.Slots["canon_1_3"])
	if !strings.Contains(c13, "ch.canon.katavasia,"+tp+"canon.ode.1.katavasia") || !strings.Contains(c13, tp+"canon.ode.1.troparion.1") {
		t.Errorf("canon: %s", c13)
	}
	if pr := refs(p.Slots["praises"]); !strings.Contains(pr, "common.glory,"+tp+"matins.praises.glory,") || strings.Contains(pr, tp+"matins.praises.sticheron.4") {
		t.Errorf("praises: %s", pr)
	}
}
