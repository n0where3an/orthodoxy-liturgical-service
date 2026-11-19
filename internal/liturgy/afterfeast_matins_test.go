package liturgy

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// afterfeastCorpus: 8 August, the afterfeast of the Transfiguration, as the
// Menaion gives it (the feast's canon and the saint's, the feast's sessional
// hymns and aposticha), with the feast day (6 August).
func afterfeastCorpus(saintTroparion bool) mapCorpus {
	me, fe := "me.08.08.", "me.08.06."
	ids := []string{me + "kontakion.1", me + "matins.k_sedalen.1.1", me + "matins.k_sedalen.2.1",
		me + "matins.sedalen.1", me + "matins.sedalen.glory", me + "matins.exapostilarion.1",
		me + "matins.aposticha.sticheron.1", me + "matins.aposticha.verse.2", me + "matins.aposticha.sticheron.2",
		me + "matins.aposticha.verse.3", me + "matins.aposticha.sticheron.3", me + "matins.aposticha.glory_now",
		fe + "troparion.1", fe + "kontakion.1", fe + "matins.ikos.1", fe + "matins.exapostilarion.1"}
	if saintTroparion {
		ids = append(ids, me+"troparion.1")
	}
	for _, o := range odeOrder {
		ids = append(ids, fmt.Sprintf("%smatins.canon.1.ode.%d.irmos", fe, o))
		b := fmt.Sprintf("%smatins.canon.1.ode.%d.", me, o)
		ids = append(ids, b+"irmos", b+"troparion.1", b+"troparion.2", b+"troparion.3")
		if o != 1 { // the saint's canon from the 3rd ode
			b = fmt.Sprintf("%smatins.canon.2.ode.%d.", me, o)
			ids = append(ids, b+"irmos", b+"troparion.1", b+"troparion.2", b+"theotokion")
		}
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 1, fe + "troparion.1": 7}}
}

func TestAfterfeastMatinsPlan(t *testing.T) {
	d := calendar.MustJDate(2026, time.August, 8)
	if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
		t.Fatalf("test date: %v", wd)
	}
	me, fe := "me.08.08.", "me.08.06."
	plan, err := planMatins(t, afterfeastCorpus(true), d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "matins.afterfeast" {
		t.Fatalf("rule %s", plan.Rule)
	}
	want := map[string]string{
		"troparia":      fe + "troparion.1x2,common.glory," + me + "troparion.1,common.and_now," + fe + "troparion.1",
		"sedalny_1":     me + "matins.k_sedalen.1.1,common.glory_now," + me + "matins.k_sedalen.1.1",
		"sedalen":       me + "kontakion.1," + me + "matins.sedalen.1,common.glory," + me + "matins.sedalen.glory",
		"kontakion":     fe + "kontakion.1," + fe + "matins.ikos.1",
		"exapostilaria": me + "matins.exapostilarion.1,common.glory_now," + fe + "matins.exapostilarion.1",
		"aposticha": me + "matins.aposticha.sticheron.1," + me + "matins.aposticha.verse.2," + me + "matins.aposticha.sticheron.2," +
			me + "matins.aposticha.verse.3," + me + "matins.aposticha.sticheron.3,common.glory_now," + me + "matins.aposticha.glory_now",
		"end_troparia": me + "troparion.1,common.glory_now," + fe + "troparion.1",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	// The 3rd ode: the feast's canon with its irmos on 8, the saint's on 4,
	// no Octoechos; the katavasia — the irmos of the feast's canon.
	c := strings.Split(refs(plan.Slots["canon_1_3"]), ",")
	odeThree := strings.Join(c[indexOf(c, "ch.canon.ode.3"):], ",")
	wantThree := "ch.canon.ode.3," + me + "matins.canon.1.ode.3.irmos," +
		strings.Repeat(me+"matins.canon.1.ode.3.troparion.1,"+me+"matins.canon.1.ode.3.troparion.2,"+me+"matins.canon.1.ode.3.troparion.3,", 2) +
		me + "matins.canon.1.ode.3.troparion.1," +
		me + "matins.canon.2.ode.3.troparion.1," + me + "matins.canon.2.ode.3.troparion.2," + me + "matins.canon.2.ode.3.troparion.1," +
		me + "matins.canon.2.ode.3.theotokion,ch.canon.katavasia," + fe + "matins.canon.1.ode.3.irmos"
	if odeThree != wantThree {
		t.Errorf("ode 3:\n got %s\nwant %s", odeThree, wantThree)
	}
	if strings.Contains(refs(plan.Slots["canon_1_3"]), "oc.") {
		t.Error("the Octoechos canon in the afterfeast")
	}

	// The saint without a troparion: the feast's thrice.
	if plan, err = planMatins(t, afterfeastCorpus(false), d); err != nil {
		t.Fatal(err)
	}
	if got := refs(plan.Slots["troparia"]); got != fe+"troparion.1x3" {
		t.Errorf("troparia without the saint's: %s", got)
	}
}

func indexOf(l []string, s string) int {
	for i, x := range l {
		if x == s {
			return i
		}
	}
	return len(l)
}

// «Слава» the saint's, «И ныне» the feast's sticheron, not a theotokion.
func TestApostichaNowOfFeast(t *testing.T) {
	d := calendar.MustJDate(2026, time.September, 16)
	me := "me.09.16."
	f := Facts{Day: calendar.NewDayContext(d), Menaion: map[string][]string{
		"matins.aposticha.sticheron": {me + "matins.aposticha.sticheron.1"},
		"matins.aposticha.glory":     {me + "matins.aposticha.glory"},
		"matins.aposticha.now":       {me + "matins.aposticha.now"},
	}}
	items, err := matinsAposticha(f)
	if err != nil {
		t.Fatal(err)
	}
	want := me + "matins.aposticha.sticheron.1,common.glory," + me + "matins.aposticha.glory,common.and_now," + me + "matins.aposticha.now"
	if got := refs(items); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}
