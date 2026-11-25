package liturgy

import (
	"errors"
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
