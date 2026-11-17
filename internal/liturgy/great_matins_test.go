package liturgy

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// festalMatinsCorpus: a Menaion service with great vespers (litiya if
// vigil) and its festal matins: two canons, sessional hymns, prokeimenon,
// Gospel, the praises on 4.
func festalMatinsCorpus(d calendar.JDate, vigil bool) mapCorpus {
	day := calendar.NewDayContext(d)
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	oc := fmt.Sprintf("oc.%d.%s.matins.", day.Tone, WeekdayKey(day.Weekday))
	ids := []string{me + "troparion.1", me + "kontakion.1", me + "vespers.great",
		me + "gvespers.paremia.1", me + "gvespers.paremia.2", me + "gvespers.paremia.3",
		me + "matins.k_sedalen.1.1", me + "matins.k_sedalen.2.1", me + "matins.k_sedalen.p.1", me + "matins.k_sedalen.p.glory",
		me + "matins.magnification", me + "matins.prokeimenon", me + "matins.prokeimenon.verse",
		me + "matins.gospel", me + "matins.gospel.announce", me + "matins.ps50.sticheron",
		me + "matins.sedalen.1", me + "matins.ikos.1", me + "matins.exapostilarion.1", me + "matins.exapostilarion.theotokion",
		me + "matins.praises.on.4", me + "matins.praises.sticheron.1", me + "matins.praises.sticheron.2",
		me + "matins.praises.sticheron.3", me + "matins.praises.glory", me + "matins.praises.now"}
	if vigil {
		ids = append(ids, me+"gvespers.litiya.sticheron.1")
	}
	for _, o := range odeOrder {
		for c := 1; c <= 2; c++ {
			b := fmt.Sprintf("%smatins.canon.%d.ode.%d.", me, c, o)
			ids = append(ids, b+"irmos", b+"troparion.1", b+"troparion.2", b+"theotokion")
			ob := fmt.Sprintf("%scanon.%d.ode.%d.", oc, c, o)
			ids = append(ids, ob+"irmos", ob+"troparion.1", ob+"theotokion")
		}
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 4, me + "matins.prokeimenon": 4}}
}

func TestFestalMatinsPlan(t *testing.T) {
	// The Entry of the Theotokos, 2026-11-21 O.S. (Saturday): a feast.
	d := calendar.MustJDate(2026, time.November, 21)
	plan, err := planMatins(t, festalMatinsCorpus(d, true), d)
	if err != nil {
		t.Fatal(err)
	}
	me := "me.11.21."
	if plan.Rule != "matins.feast" || plan.Template != "great_matins" {
		t.Fatalf("rule %s, template %s", plan.Rule, plan.Template)
	}
	want := map[string]string{
		"troparia":  me + "troparion.1x3",
		"sedalny_p": me + "matins.k_sedalen.p.1,common.glory," + me + "matins.k_sedalen.p.glory",
		"prokeimenon": "lit.prokeimenon.tone.4," + me + "matins.prokeimenon," + me + "matins.prokeimenon," + me + "matins.prokeimenon.verse," +
			me + "matins.prokeimenon," + me + "matins.prokeimenon," + me + "matins.prokeimenon",
		"praises": "ch.matins.praises.150.2," + me + "matins.praises.sticheron.1,ch.matins.praises.150.3," + me + "matins.praises.sticheron.2," +
			"ch.matins.praises.150.4," + me + "matins.praises.sticheron.3,ch.matins.praises.150.5," + me + "matins.praises.sticheron.1," +
			"common.glory," + me + "matins.praises.glory,common.and_now," + me + "matins.praises.now",
		"end_troparia": me + "troparion.1",
		"magnificat":   "",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	// A feast: the Menaion's canons only, katavasia after every ode.
	c := refs(plan.Slots["canon_1_3"])
	if strings.Contains(c, "oc.") || strings.Count(c, "ch.canon.katavasia") != 2 {
		t.Errorf("feast canon: %s", c)
	}
}

func TestSaintMatins(t *testing.T) {
	// St John Chrysostom, Thursday 2026-11-13 O.S.
	d := calendar.MustJDate(2026, time.November, 13)
	for _, c := range []struct {
		vigil bool
		rule  string
	}{{true, "matins.vigil"}, {false, "matins.polyeleos"}} {
		plan, err := planMatins(t, festalMatinsCorpus(d, c.vigil), d)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != c.rule {
			t.Errorf("rule %s, want %s", plan.Rule, c.rule)
		}
		if got := refs(plan.Slots["troparia"]); got != "me.11.13.troparion.1x2,common.glory_now,ch.theotokion.dismissal.4.sun" {
			t.Errorf("troparia: %s", got)
		}
		if len(plan.Slots["magnificat"]) == 0 {
			t.Error("a saint: «Честнейшую» is sung")
		}
		if (len(plan.Slots["plain_beginning"]) == 0) != c.vigil {
			t.Errorf("%s: the beginning of matins", c.rule)
		}
	}
}

func TestFestalMatinsRender(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 13)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	c := festalMatinsCorpus(d, true)
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c}, Texts: c}
	var buf bytes.Buffer
	if _, err := g.Generate(context.Background(), Request{Date: d, Service: "matins", Lang: "cu", Mode: "laity",
		Params: Params{Priest: true, Deacon: true}}, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, b := range []string{"BODY ps.psalm.134", "BODY oc.stepenna.4.1.1", "BODY lit.matins.every_breath",
		"BODY me.11.13.matins.gospel", "BODY ch.matins.great_doxology.1", "BODY ch.matins.magnificat.1"} {
		if !strings.Contains(out, b) {
			t.Errorf("no %s", b)
		}
	}
	if strings.Contains(out, "BODY ch.matins.psalm.19") {
		t.Error("vigil: matins begins with the Six Psalms")
	}
}
