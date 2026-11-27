package liturgy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func TestLentWeekday(t *testing.T) {
	for off, want := range map[int]string{-48: "w1_mon", -44: "w1_fri", -43: "", -40: "w2_tue", -9: "w6_fri", -8: "", -49: ""} {
		if got := lentWeekday(off); got != want {
			t.Errorf("%d: %q, want %q", off, got, want)
		}
	}
}

// The weekday matins of Great Lent with «Аллилуиа» (Типикон, гл. 49).
func TestLentenWeekdayMatins(t *testing.T) {
	d := calendar.Pascha(2027).AddDays(-40) // Tuesday of the 2nd week
	day := calendar.NewDayContext(d)
	tp := "tp.w2_tue."
	me := fmt.Sprintf("me.%s.%s.", pad(int(day.Menaion.Month)), pad(day.Menaion.Day))
	c := matinsCorpus(d, 1)
	c.ids = append(c.ids, tp+"matins.sedalen.k2.1", tp+"matins.sedalen.k2.glory", tp+"matins.sedalen.k2.theotokion",
		tp+"matins.sedalen.k3.1", tp+"matins.sedalen.k3.theotokion",
		tp+"matins.aposticha.idiomelon", tp+"matins.aposticha.martyricon", tp+"matins.aposticha.theotokion")
	for _, o := range []int{2, 8, 9} {
		b := fmt.Sprintf("%scanon.1.ode.%d.", tp, o)
		c.ids = append(c.ids, b+"irmos", b+"1.troparion", b+"2.theotokion")
		b = fmt.Sprintf("%scanon.2.ode.%d.", tp, o)
		c.ids = append(c.ids, b+"irmos", b+"katavasia", b+"1.troparion")
		if o == 8 { // «Благословим Отца…» before the triadikon
			c.ids = append(c.ids, b+"2.refrain", b+"3.triadikon", b+"4.theotokion", b+"5.refrain")
		} else {
			c.ids = append(c.ids, b+"2.triadikon", b+"3.theotokion", b+"4.refrain", b+"5.troparion")
		}
	}
	for o := 1; o <= 9; o++ {
		for v := 1; v <= 6; v++ {
			c.ids = append(c.ids, fmt.Sprintf("ir.ode.%d.lent.%d", o, v))
		}
	}
	plan, err := (&Planner{Rules: mustRules(t), Feasts: mustFeasts(t), Menaion: c}).Plan(context.Background(), d, "matins",
		Params{Priest: true, Typikon: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "matins.triodion.lenten_weekday" || plan.Template != "lenten_matins" {
		t.Fatalf("rule %s, template %s", plan.Rule, plan.Template)
	}
	tone := day.Tone
	check(t, plan, map[string]string{
		"alleluia": fmt.Sprintf("lit.alleluia.tone.%d,ch.matins.alleluia.verse.1,common.alleluia.thrice,ch.matins.alleluia.verse.2,"+
			"common.alleluia.thrice,ch.matins.alleluia.verse.3,common.alleluia.thrice,ch.matins.alleluia.verse.4,common.alleluia.thrice", tone),
		"troparia":   fmt.Sprintf("oc.%d.trinity.1.tue,common.glory,oc.%d.trinity.glory,common.and_now,oc.%d.trinity.now", tone, tone, tone),
		"kathisma_1": "ps.kathisma.10.1,ps.kathisma.10.2,ps.kathisma.10.3",
		"kathisma_3": "ps.kathisma.12.1,ps.kathisma.12.2,ps.kathisma.12.3",
		"sedalny_2": tp + "matins.sedalen.k2.1,common.glory," + tp + "matins.sedalen.k2.glory,common.and_now," +
			tp + "matins.sedalen.k2.theotokion",
		"sedalny_3":     tp + "matins.sedalen.k3.1,common.glory_now," + tp + "matins.sedalen.k3.theotokion",
		"exapostilaria": fmt.Sprintf("oc.%d.photagogikon.tuex3", tone),
		"end_troparia":  "ch.matins.in_the_temple",
		"kontakion":     me + "kontakion.1," + me + "matins.ikos.1",
		"aposticha": tp + "matins.aposticha.idiomelon,ch.matins.aposticha.verse.2," + tp + "matins.aposticha.idiomelon," +
			"ch.matins.aposticha.verse.3," + tp + "matins.aposticha.martyricon,common.glory_now," + tp + "matins.aposticha.theotokion",
	})
	c13 := refs(plan.Slots["canon_1_3"])
	for _, w := range []string{
		// The 2nd ode: the triode's irmos first, the verses on the places up
		// to the triadikon, «Слава Тебе, Боже наш» before the last troparion,
		// the irmos in full as the katavasia.
		"ch.canon.ode.2,ir.ode.2.lent.3," + tp + "canon.1.ode.2.irmos,ir.ode.2.lent.4," + tp + "canon.1.ode.2.1.troparion",
		"ir.ode.2.lent.6," + tp + "canon.2.ode.2.1.troparion,common.glory," + tp + "canon.2.ode.2.2.triadikon,common.and_now," +
			tp + "canon.2.ode.2.3.theotokion," + tp + "canon.2.ode.2.4.refrain," + tp + "canon.2.ode.2.5.troparion,ch.canon.katavasia," +
			tp + "canon.2.ode.2.katavasia",
		// The 3rd ode, the Menaion's alone: its irmos only after it.
		"ch.canon.ode.3,ir.ode.3.lent.5," + me + "matins.canon.1.ode.3.troparion.1",
		me + "matins.canon.1.ode.3.theotokion,ch.canon.katavasia," + me + "matins.canon.1.ode.3.irmos",
	} {
		if !strings.Contains(c13, w) {
			t.Errorf("canon_1_3 lacks %s:\n%s", w, c13)
		}
	}
	if strings.Contains(c13, "oc.") {
		t.Errorf("the Octoechos canon in Lent: %s", c13)
	}
	c78 := refs(plan.Slots["canon_7_8"])
	// «Благословим Отца» in the place of «Слава».
	if w := tp + "canon.2.ode.8.2.refrain," + tp + "canon.2.ode.8.3.triadikon,common.and_now," + tp + "canon.2.ode.8.4.theotokion"; !strings.Contains(c78, w) {
		t.Errorf("canon_7_8 lacks %s:\n%s", w, c78)
	}
	c46 := refs(plan.Slots["canon_4_6"])
	if !strings.Contains(c46, "ch.canon.ode.4,ir.ode.4.lent.4,"+me+"matins.canon.1.ode.4.irmos") {
		t.Errorf("canon_4_6: %s", c46)
	}
}
