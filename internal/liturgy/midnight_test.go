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

func midnightCorpus(d calendar.JDate, extra ...string) mapCorpus {
	day := calendar.NewDayContext(d)
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1", me + "kontakion.1"}
	if day.Weekday == time.Sunday {
		b := fmt.Sprintf("oc.%d.sun.", day.Tone)
		ids = append(ids, b+"matins.hypakoe", b+"midnight.sedalen.3", b+"midnight.sedalen.3.theotokion")
		for _, o := range odeOrder {
			c := fmt.Sprintf("%smidnight.canon.ode.%d.", b, o)
			ids = append(ids, c+"irmos", c+"troparion.1", c+"glory", c+"theotokion")
		}
	}
	for _, e := range extra {
		ids = append(ids, me+e)
	}
	return mapCorpus{ids: ids}
}

func planMidnight(t *testing.T, c mapCorpus, d calendar.JDate) (*Plan, error) {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, _ := LoadFeasts(dataDir())
	return (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, "midnight_office", Params{Priest: true})
}

func TestMidnightOfficePlan(t *testing.T) {
	for _, c := range []struct {
		d               calendar.JDate
		rule, template  string
		kathisma, basil string
	}{
		// Tuesday in winter: kathisma 17, the prayers of St Basil.
		{calendar.MustJDate(2026, time.November, 4), "midnight.daily", "", "ps.kathisma.17.1,ps.kathisma.17.2,ps.kathisma.17.3",
			"ch.midnight.prayer.basil,ch.midnight.prayer.we_bless"},
		// Tuesday in summer: without them.
		{calendar.MustJDate(2026, time.July, 7), "midnight.daily", "", "ps.kathisma.17.1,ps.kathisma.17.2,ps.kathisma.17.3", ""},
		// Saturday: kathisma 9.
		{calendar.MustJDate(2026, time.November, 1), "midnight.saturday", "midnight_saturday", "ps.kathisma.9.1,ps.kathisma.9.2,ps.kathisma.9.3",
			"ch.midnight.prayer.basil,ch.midnight.prayer.we_bless"},
	} {
		plan, err := planMidnight(t, midnightCorpus(c.d), c.d)
		if err != nil {
			t.Fatalf("%v: %v", c.d, err)
		}
		if plan.Rule != c.rule || plan.Template != c.template {
			t.Errorf("%v: rule %s, template %q", c.d, plan.Rule, plan.Template)
		}
		if got := refs(plan.Slots["kathisma"]); got != c.kathisma {
			t.Errorf("%v kathisma: %s", c.d, got)
		}
		if got := refs(plan.Slots["basil_prayers"]); got != c.basil {
			t.Errorf("%v Basil's prayers: %s", c.d, got)
		}
	}
}

func TestMidnightOfficeSunday(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 2)
	day := calendar.NewDayContext(d)
	plan, err := planMidnight(t, midnightCorpus(d), d)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "midnight.sunday" || plan.Template != "midnight_sunday" {
		t.Fatalf("rule %s, template %s", plan.Rule, plan.Template)
	}
	b := fmt.Sprintf("oc.%d.sun.midnight.", day.Tone)
	c := refs(plan.Slots["canon"])
	if !strings.HasPrefix(c, "ch.canon.ode.1,"+b+"canon.ode.1.irmos,oc.midnight.refrain,"+b+"canon.ode.1.troparion.1,common.glory,") {
		t.Errorf("canon: %s", c)
	}
	// After the 3rd ode: «Господи, помилуй» thrice, the hymn, «Слава, и ныне», its theotokion.
	if !strings.Contains(c, "common.lord_have_mercyx3,"+b+"sedalen.3,common.glory_now,"+b+"sedalen.3.theotokion,ch.canon.ode.4") {
		t.Errorf("sessional hymn: %s", c)
	}
	if got := refs(plan.Slots["hypakoe"]); got != fmt.Sprintf("oc.%d.sun.matins.hypakoe", day.Tone) {
		t.Errorf("hypakoe: %s", got)
	}
}

func TestMidnightOfficeNotServed(t *testing.T) {
	for _, c := range []struct {
		d     calendar.JDate
		extra []string
		rule  string
	}{
		{calendar.MustJDate(2026, time.September, 8), nil, "midnight.vigil"},
		{calendar.MustJDate(2026, time.November, 13), []string{"vespers.great", "gvespers.paremia.1", "gvespers.litiya.sticheron.1"}, "midnight.vigil"},
		{calendar.MustJDate(2027, time.March, 3), nil, "midnight.triodion"},
	} {
		_, err := planMidnight(t, midnightCorpus(c.d, c.extra...), c.d)
		var ue *UnsupportedError
		if !errors.As(err, &ue) || ue.Rule != c.rule {
			t.Errorf("%v: want %s, got %v", c.d, c.rule, err)
		}
	}
}

func TestMidnightOfficeRender(t *testing.T) {
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	for _, c := range []struct {
		d    calendar.JDate
		want []string
	}{
		{calendar.MustJDate(2026, time.November, 4), []string{"BODY ch.psalm.50", "BODY ch.midnight.troparion.bridegroom",
			"BODY ch.midnight.prayer.master", "BODY ch.midnight.psalm.133", "BODY ch.midnight.troparion.depth",
			"ch.midnight.prayer.remember", "lit.midnight.hierarchy", "ch.midnight.litany.brethren"}},
		{calendar.MustJDate(2026, time.November, 1), []string{"BODY ch.midnight.saturday.troparion.3", "BODY ch.midnight.prayer.eustratius",
			"ch.midnight.prayer.remember"}},
		{calendar.MustJDate(2026, time.November, 2), []string{"BODY ch.midnight.sunday.triadicon.7", "BODY oc.midnight.refrain"}},
	} {
		mc := midnightCorpus(c.d)
		g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: mc}, Texts: mc}
		var buf bytes.Buffer
		if _, err := g.Generate(context.Background(), Request{Date: c.d, Service: "midnight_office", Lang: "cu", Mode: "kliros",
			Params: Params{Priest: true}}, &buf); err != nil {
			t.Fatalf("%v: %v", c.d, err)
		}
		for _, w := range c.want {
			if !strings.Contains(buf.String(), w) {
				t.Errorf("%v: no %s", c.d, w)
			}
		}
	}
}
