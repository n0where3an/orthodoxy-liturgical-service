package liturgy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// complineCorpus: the Octoechos canon of compline of the day and the
// Menaion day's troparion and kontakion.
func complineCorpus(d calendar.JDate, extra ...string) mapCorpus {
	day := calendar.NewDayContext(d)
	oc := fmt.Sprintf("oc.%d.%s.compline.", day.Tone, WeekdayKey(day.Weekday))
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1", me + "kontakion.1", oc + "sedalen",
		fmt.Sprintf("oc.%d.%s.kontakion", day.Tone, WeekdayKey(day.Weekday))}
	for _, o := range odeOrder {
		b := fmt.Sprintf("%scanon.ode.%d.", oc, o)
		ids = append(ids, b+"irmos", b+"troparion.1", b+"troparion.2", b+"glory", b+"theotokion")
	}
	for _, e := range extra {
		ids = append(ids, me+e)
	}
	return mapCorpus{ids: ids}
}

func planCompline(t *testing.T, c mapCorpus, d calendar.JDate, temple string) (*Plan, error) {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, _ := LoadFeasts(dataDir())
	temples, _ := LoadTemples(dataDir())
	p := &Planner{Rules: rules, Feasts: feasts, Temples: temples, Menaion: c}
	return p.Plan(context.Background(), d, "compline", Params{Priest: true, Temple: temple})
}

func TestComplineCanon(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 3) // Monday: the compline of Sunday evening
	day := calendar.NewDayContext(d)
	plan, err := planCompline(t, complineCorpus(d), d, "")
	if err != nil {
		t.Fatal(err)
	}
	b := fmt.Sprintf("oc.%d.mon.compline.", day.Tone)
	c := refs(plan.Slots["canon"])
	ode1 := "ch.canon.ode.1," + b + "canon.ode.1.irmos,sl.vespers.most_holy_theotokos_save_us," + b + "canon.ode.1.troparion.1," +
		"sl.vespers.most_holy_theotokos_save_us," + b + "canon.ode.1.troparion.2,common.glory," + b + "canon.ode.1.glory," +
		"common.and_now," + b + "canon.ode.1.theotokion,"
	if !strings.HasPrefix(c, ode1) {
		t.Errorf("ode 1:\n got %s\nwant %s…", c, ode1)
	}
	if !strings.Contains(c, b+"canon.ode.6.theotokion,common.lord_have_mercyx3,common.glory_now,"+b+"sedalen,ch.canon.ode.7") {
		t.Errorf("no sessional hymn after the 6th ode: %s", c)
	}
}

func TestComplineTroparia(t *testing.T) {
	const tail = "ch.compline.troparion.fathers,ch.compline.troparion.martyrs,common.glory,ch.kontakion.with_the_saints,common.and_now,ch.compline.theotokion"
	for _, c := range []struct {
		d      calendar.JDate
		temple string
		want   string
	}{
		// Sunday evening: «Небесных воинств».
		{calendar.MustJDate(2026, time.November, 3), "", "ch.troparion.day.mon," + tail},
		{calendar.MustJDate(2026, time.November, 3), "transfiguration", "me.08.06.troparion.1,ch.troparion.day.mon," + tail},
		{calendar.MustJDate(2026, time.November, 3), "archangel_michael", "ch.troparion.day.mon," + tail},
		{calendar.MustJDate(2026, time.November, 3), "nicholas", "ch.troparion.day.mon,me.12.06.troparion.1," + tail},
		// Tuesday evening: «Спаси, Господи»; no church of the Lord.
		{calendar.MustJDate(2026, time.November, 5), "transfiguration", "ch.troparion.day.wed," + tail},
		{calendar.MustJDate(2026, time.November, 5), "dormition", "ch.troparion.day.wed,me.08.15.troparion.1," + tail},
		// Wednesday evening: the apostles and St Nicholas.
		{calendar.MustJDate(2026, time.November, 6), "", "ch.troparion.day.thu,ch.troparion.day.thu.2," + tail},
		{calendar.MustJDate(2026, time.November, 6), "nicholas", "ch.troparion.day.thu,ch.troparion.day.thu.2," + tail},
		{calendar.MustJDate(2026, time.November, 6), "elijah", "ch.troparion.day.thu,ch.troparion.day.thu.2,me.07.20.troparion.1," + tail},
		// Friday evening: «Апостоли, мученицы», «Яко начатки»; the saint has a troparion.
		{calendar.MustJDate(2026, time.November, 1), "",
			"ch.troparion.day.sat,common.glory,ch.kontakion.with_the_saints,common.and_now,ch.kontakion.day.sat"},
	} {
		plan, err := planCompline(t, complineCorpus(c.d), c.d, c.temple)
		if err != nil {
			t.Fatalf("%v %s: %v", c.d, c.temple, err)
		}
		if got := refs(plan.Slots["troparia"]); got != c.want {
			t.Errorf("%v (%v) %q:\n got %s\nwant %s", c.d, c.d.Weekday(), c.temple, got, c.want)
		}
	}
	// Saturday evening: the kontakion of the tone.
	sun := calendar.MustJDate(2026, time.November, 2)
	plan, err := planCompline(t, complineCorpus(sun), sun, "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := refs(plan.Slots["troparia"]), fmt.Sprintf("oc.%d.sun.kontakion", calendar.NewDayContext(sun).Tone); got != want {
		t.Errorf("Saturday evening: %s, want %s", got, want)
	}
	// The afterfeast: the feast's kontakion.
	af := calendar.MustJDate(2026, time.August, 10)
	if plan, err = planCompline(t, complineCorpus(af), af, ""); err != nil {
		t.Fatal(err)
	}
	if got := refs(plan.Slots["troparia"]); got != "me.08.06.kontakion.1" {
		t.Errorf("afterfeast: %s", got)
	}
}

func TestComplineNotServed(t *testing.T) {
	for _, c := range []struct {
		d    calendar.JDate
		rule string
	}{
		{calendar.MustJDate(2026, time.September, 8), "compline.vigil"}, // a twelve feast
		{calendar.MustJDate(2026, time.December, 24), "compline.great"}, // the eve of the Nativity
		{calendar.MustJDate(2027, time.March, 3), "compline.triodion"},  // Great Lent
	} {
		_, err := planCompline(t, complineCorpus(c.d), c.d, "")
		var ue *UnsupportedError
		if !errors.As(err, &ue) || ue.Rule != c.rule {
			t.Errorf("%v: want %s, got %v", c.d, c.rule, err)
		}
	}
	// A saint with a vigil.
	d := calendar.MustJDate(2026, time.November, 13)
	_, err := planCompline(t, complineCorpus(d, "vespers.great", "gvespers.paremia.1", "gvespers.litiya.sticheron.1"), d, "")
	var ue *UnsupportedError
	if !errors.As(err, &ue) || ue.Rule != "compline.vigil" {
		t.Errorf("vigil saint: %v", err)
	}
}

func TestComplineRender(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 4)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	c := complineCorpus(d)
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c}, Texts: c}
	var buf bytes.Buffer
	if _, err := g.Generate(context.Background(), Request{Date: d, Service: "compline", Lang: "cu", Mode: "kliros",
		Params: Params{Priest: true}}, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, b := range []string{"BODY ch.psalm.50", "BODY ch.compline.psalm.101", "BODY common.creed", "BODY ch.lit.it_is_truly_meet",
		"BODY ch.compline.troparion.fathers", "BODY common.prayer_of_the_hours", "BODY ch.compline.prayer.my_hope", "sl.dismissal.daily"} {
		if !strings.Contains(out, b) {
			t.Errorf("no %s", b)
		}
	}
}
