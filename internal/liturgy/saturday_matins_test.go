package liturgy

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// Saturday matins with «Бог Господь» (Типикон, гл. 12).
func TestSaturdayMatinsPlan(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 15)
	day := calendar.NewDayContext(d)
	if day.Weekday != time.Saturday || day.Departed != "" {
		t.Fatalf("test date: %v %q", day.Weekday, day.Departed)
	}
	oc := fmt.Sprintf("oc.%d.sat.matins.", day.Tone)
	me := "me.11.15."
	c := matinsCorpus(d, 1, "matins.exapostilarion.1")
	c.ids = append(c.ids, oc+"sedalen.1.3", oc+"sedalen.1.verse.3", oc+"sedalen.2.3", oc+"sedalen.2.verse.3",
		oc+"praises.sticheron.1", oc+"praises.sticheron.2", oc+"praises.sticheron.3", oc+"praises.sticheron.4",
		oc+"praises.departed", oc+"praises.theotokion")
	for _, o := range odeOrder {
		c.ids = append(c.ids, fmt.Sprintf("%scanon.1.ode.%d.glory", oc, o))
	}
	plan, err := planMatins(t, c, d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "matins.saturday" {
		t.Fatalf("rule %s", plan.Rule)
	}
	want := map[string]string{
		"kathisma_1": "ps.kathisma.16.1,ps.kathisma.16.2,ps.kathisma.16.3",
		"kathisma_2": "ps.kathisma.17.1,ps.kathisma.17.2,ps.kathisma.17.3",
		"kathisma_3": "",
		"sedalny_3":  "",
		// After the 16th the martyrs' two only.
		"sedalny_1":     oc + "sedalen.1.1," + oc + "sedalen.1.verse.2," + oc + "sedalen.1.2,common.glory_now," + oc + "sedalen.1.theotokion",
		"sedalny_2":     oc + "sedalen.2.1," + oc + "sedalen.2.verse.2," + oc + "sedalen.2.2," + oc + "sedalen.2.verse.3," + oc + "sedalen.2.3,common.glory_now," + oc + "sedalen.2.theotokion",
		"exapostilaria": me + "matins.exapostilarion.1,common.glory,oc.exapostilarion.sat,common.and_now,oc.exapostilarion.sat.theotokion",
		// The martyrs' stichera of the praises at the aposticha, three.
		"aposticha": oc + "praises.sticheron.1,ch.matins.aposticha.verse.2," + oc + "praises.sticheron.2,ch.matins.aposticha.verse.3," +
			oc + "praises.sticheron.3,common.glory_now," + oc + "praises.theotokion",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	// The saint's canon with its irmos, then the martyrs' of the Octoechos
	// without it; the departed canon is not sung; katavasia: the last irmos.
	c13 := refs(plan.Slots["canon_1_3"])
	ode1 := "ch.canon.ode.1," + me + "matins.canon.1.ode.1.irmos," + me + "matins.canon.1.ode.1.troparion.1,"
	if !strings.HasPrefix(c13, ode1) {
		t.Errorf("canon: the saint's first: %s", c13[:min(len(c13), 200)])
	}
	if strings.Contains(c13, oc+"canon.1.ode.1.irmos,"+oc) || strings.Contains(c13, oc+"canon.2.") {
		t.Errorf("canon: the Octoechos's irmos or departed canon sung: %s", c13)
	}
	octOde1 := me + "matins.canon.1.ode.1.theotokion," + oc + "canon.1.ode.1.troparion.1," + oc + "canon.1.ode.1.troparion.2," +
		oc + "canon.1.ode.1.troparion.3,common.glory," + oc + "canon.1.ode.1.glory,common.and_now," + oc + "canon.1.ode.1.theotokion"
	if !strings.Contains(c13, octOde1) {
		t.Errorf("canon: the martyrs' after the saint's: %s", c13)
	}
	if !strings.HasSuffix(c13, "ch.canon.katavasia,"+oc+"canon.1.ode.3.irmos") {
		t.Errorf("katavasia: %s", c13[max(0, len(c13)-120):])
	}
}

// Saturday with a saint of the great doxology (Типикон, гл. 11): no
// Octoechos canon of the Saturday — the Theotokos's of the past Sunday with
// its irmos, then the saint's; the saint's exapostilarion twice; the
// Menaion's sessional hymns where it gives them; no aposticha.
func TestSaturdayDoxologyMatinsPlan(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 15)
	day := calendar.NewDayContext(d)
	oc := fmt.Sprintf("oc.%d.sat.matins.", day.Tone)
	sun := fmt.Sprintf("oc.%d.sun.matins.", day.Tone)
	me := "me.11.15."
	corpus := func(menCanons int, extra ...string) mapCorpus {
		c := matinsCorpus(d, menCanons, append([]string{"matins.exapostilarion.1", "matins.great_doxology",
			"matins.k_sedalen.1.1", "matins.k_sedalen.1.theotokion"}, extra...)...)
		for _, o := range odeOrder {
			b := fmt.Sprintf("%scanon.3.ode.%d.", sun, o)
			c.ids = append(c.ids, b+"troparion.1", b+"troparion.2", fmt.Sprintf("%scanon.1.ode.%d.irmos", sun, o))
			if o == 1 {
				c.ids = append(c.ids, b+"irmos") // elsewhere «ирмос тойже»: the resurrectional canon's
			}
		}
		return c
	}
	plan, err := planMatins(t, corpus(1), d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "matins.saturday.doxology" {
		t.Fatalf("rule %s", plan.Rule)
	}
	want := map[string]string{
		"kathisma_1":     "ps.kathisma.16.1,ps.kathisma.16.2,ps.kathisma.16.3",
		"kathisma_2":     "ps.kathisma.17.1,ps.kathisma.17.2,ps.kathisma.17.3",
		"sedalny_1":      me + "matins.k_sedalen.1.1,common.glory_now," + me + "matins.k_sedalen.1.theotokion",
		"sedalny_2":      oc + "sedalen.2.1," + oc + "sedalen.2.verse.2," + oc + "sedalen.2.2,common.glory_now," + oc + "sedalen.2.theotokion",
		"exapostilaria":  me + "matins.exapostilarion.1x2,common.glory_now,oc.exapostilarion.sat.theotokion",
		"aposticha":      "",
		"great_doxology": "ch.matins.great_doxology.1",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	c13 := refs(plan.Slots["canon_1_3"])
	ode1 := "ch.canon.ode.1," + sun + "canon.3.ode.1.irmos," + sun + "canon.3.ode.1.troparion.1," + sun + "canon.3.ode.1.troparion.2," +
		me + "matins.canon.1.ode.1.troparion.1," + me + "matins.canon.1.ode.1.troparion.2," + me + "matins.canon.1.ode.1.troparion.3," +
		me + "matins.canon.1.ode.1.theotokion,ch.canon.katavasia," + me + "matins.canon.1.ode.1.irmos,"
	if !strings.HasPrefix(c13, ode1) {
		t.Errorf("canon: the Theotokos's, then the saint's:\n got %s\nwant %s…", c13, ode1)
	}
	if !strings.Contains(c13, "ch.canon.ode.3,"+sun+"canon.1.ode.3.irmos,"+sun+"canon.3.ode.3.troparion.1,") {
		t.Errorf("canon: «ирмос тойже» — the resurrectional canon's irmos: %s", c13)
	}
	if strings.Contains(c13, oc) {
		t.Errorf("canon: the Octoechos's Saturday canon sung: %s", c13)
	}

	// The Menaion's own canon of the Theotokos, first, with its irmos.
	plan, err = planMatins(t, corpus(2, "matins.canon.1.theotokos"), d)
	if err != nil {
		t.Fatal(err)
	}
	c13 = refs(plan.Slots["canon_1_3"])
	ode1 = "ch.canon.ode.1," + me + "matins.canon.1.ode.1.irmos," + me + "matins.canon.1.ode.1.troparion.1,"
	if !strings.HasPrefix(c13, ode1) || strings.Contains(c13, sun) ||
		!strings.Contains(c13, me+"matins.canon.1.ode.1.theotokion,"+me+"matins.canon.2.ode.1.troparion.1,") {
		t.Errorf("canon with the Menaion's of the Theotokos: %s", c13)
	}
	if !strings.Contains(c13, "ch.canon.katavasia,"+me+"matins.canon.2.ode.1.irmos,ch.canon.ode.3") {
		t.Errorf("katavasia: the last canon's irmos: %s", c13)
	}
}
