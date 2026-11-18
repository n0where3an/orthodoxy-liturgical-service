package liturgy

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// sundayMatinsCorpus: the Octoechos Sunday (vespers, liturgy and matins of
// the day's tone), the Sunday appendix and the given Menaion units.
func sundayMatinsCorpus(d calendar.JDate, menaion map[string][]string, tones map[string]int) mapCorpus {
	c := sundayFeastCorpus(d, menaion, nil)
	oc := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun.matins."
	add := func(ids ...string) { c.ids = append(c.ids, ids...) }
	for k := 1; k <= 2; k++ {
		add(fmt.Sprintf("%ssedalen.%d.1", oc, k), fmt.Sprintf("%ssedalen.%d.2", oc, k), fmt.Sprintf("%ssedalen.%d.verse.2", oc, k),
			fmt.Sprintf("%ssedalen.%d.theotokion", oc, k))
	}
	for a := 1; a <= 3; a++ {
		add(fmt.Sprintf("%sstepenna.%d.1", oc, a), fmt.Sprintf("%sstepenna.%d.2", oc, a), fmt.Sprintf("%sstepenna.%d.glory", oc, a))
	}
	add(oc+"hypakoe", oc+"prokeimenon", oc+"prokeimenon.verse", oc+"ikos")
	for cn := 1; cn <= 3; cn++ {
		for _, o := range []int{1, 3, 4, 5, 6, 7, 8, 9} {
			b := fmt.Sprintf("%scanon.%d.ode.%d.", oc, cn, o)
			add(b+"irmos", b+"troparion.1", b+"troparion.2", b+"theotokion")
		}
	}
	for i := 1; i <= 8; i++ {
		add(fmt.Sprintf("%spraises.sticheron.%d", oc, i))
	}
	for k, v := range tones {
		c.tones[k] = v
	}
	return c
}

func planSundayMatins(t *testing.T, c mapCorpus, d calendar.JDate) *Plan {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	kat, err := LoadKatavasia(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	p, err := (&Planner{Rules: rules, Feasts: feasts, Katavasia: kat, Menaion: c}).Plan(context.Background(), d, "matins",
		Params{Priest: true})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestKatavasia(t *testing.T) {
	k, err := LoadKatavasia(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	c := mapCorpus{ids: []string{"me.12.25.matins.canon.1.ode.1.irmos", "me.12.25.matins.canon.1.ode.9.irmos"}}
	// «Христос раждается» from the Entry: the odes the corpus has, the rest «Отверзу».
	got, seasonal, err := k.ForSeason(context.Background(), c, calendar.MonthDay{Month: time.November, Day: 23})
	if err != nil || !seasonal {
		t.Fatal(err, seasonal)
	}
	if got[1] != "me.12.25.matins.canon.1.ode.1.irmos" || got[9] != "me.12.25.matins.canon.1.ode.9.irmos" ||
		got[7] != "me.03.26.matins.canon.2.ode.7.irmos" {
		t.Errorf("Nov 23: %v", got)
	}
	got, seasonal, _ = k.ForSeason(context.Background(), c, calendar.MonthDay{Month: time.October, Day: 26})
	if seasonal || got[4] != "me.03.25.matins.canon.1.ode.4.irmos" || got[1] != "me.03.26.matins.canon.2.ode.1.irmos" {
		t.Errorf("Oct 26: %v %v", got, seasonal)
	}
}

func TestSundayMatins(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 9) // Sunday, tone 8, eothinon 3
	day := calendar.NewDayContext(d)
	if day.Eothinon != 3 || day.Tone != 8 {
		t.Fatalf("eothinon %d tone %d", day.Eothinon, day.Tone)
	}
	oc := "oc.8.sun."
	me := "me.11.09."
	t.Run("saint without a sign", func(t *testing.T) {
		men := []string{"troparion.1", "kontakion.1", "matins.ikos.1", "matins.sedalen.1", "matins.exapostilarion.1",
			"matins.canon.1.ode.1.troparion.1", "matins.canon.1.ode.1.troparion.2", "matins.canon.1.ode.1.theotokion"}
		p := planSundayMatins(t, sundayMatinsCorpus(d, map[string][]string{"me.11.09": men}, map[string]int{me + "troparion.1": 4}), d)
		if p.Rule != "matins.sunday" || p.Template != "great_matins" {
			t.Fatalf("rule %s template %s", p.Rule, p.Template)
		}
		check(t, p, map[string]string{
			"troparia":      oc + "troparionx2,common.glory," + me + "troparion.1,common.and_now,ch.theotokion.dismissal.4.sun",
			"gospel":        "lect.zach.mk.71.announce,sl.lit.glory_to_thee_o_lord,lit.let_us_attend,lect.zach.mk.71,sl.lit.glory_to_thee_o_lord",
			"hypakoe":       oc + "matins.hypakoe",
			"kontakion":     oc + "kontakion," + oc + "matins.ikos",
			"exapostilaria": "oc.eothinon.3.exapostilarion,common.glory," + me + "matins.exapostilarion.1,common.and_now,oc.eothinon.3.theotokion",
			"end_troparia":  "oc.4.sun.matins.end_troparion",
			"magnificat":    "ch.matins.magnificat.1",
		})
		// Ode 1: resurrectional on 4 (irmos, 2 troparia, theotokion), Cross-Resurrection on 3,
		// Theotokos on 3, the Menaion on 4, the katavasia «Отверзу».
		c := refs(p.Slots["canon_1_3"])
		ode1 := c[:strings.Index(c, "ch.canon.ode.3")]
		o := oc + "matins.canon."
		want := "ch.canon.ode.1," + o + "1.ode.1.irmos," + o + "1.ode.1.troparion.1," + o + "1.ode.1.troparion.2," + o + "1.ode.1.theotokion," +
			o + "2.ode.1.troparion.1," + o + "2.ode.1.troparion.2," + o + "2.ode.1.theotokion," +
			o + "3.ode.1.troparion.1," + o + "3.ode.1.troparion.2," + o + "3.ode.1.theotokion," +
			me + "matins.canon.1.ode.1.troparion.1," + me + "matins.canon.1.ode.1.troparion.2," + me + "matins.canon.1.ode.1.troparion.1," +
			me + "matins.canon.1.ode.1.theotokion," + "ch.canon.katavasia,me.03.26.matins.canon.2.ode.1.irmos,"
		if ode1 != want {
			t.Errorf("ode 1:\n got %s\nwant %s", ode1, want)
		}
		if s := refs(p.Slots["sedalen"]); !strings.HasPrefix(s, me+"kontakion.1,"+me+"matins.ikos.1,"+me+"matins.sedalen.1") {
			t.Errorf("after the 3rd ode: %s", s)
		}
		pr := refs(p.Slots["praises"])
		if !strings.Contains(pr, oc+"matins.praises.sticheron.8,common.glory,oc.eothinon.3.sticheron,common.and_now,oc.sun.praises.now") {
			t.Errorf("praises %s", pr)
		}
	})
	t.Run("saint with a sign", func(t *testing.T) {
		men := []string{"troparion.1", "kontakion.1", "vespers.great", "gvespers.paremia.1", "gvespers.paremia.2", "gvespers.paremia.3",
			"matins.canon.1.ode.1.troparion.1", "matins.canon.2.ode.1.troparion.1", "matins.praises.sticheron.1",
			"matins.k_sedalen.p.1"}
		p := planSundayMatins(t, sundayMatinsCorpus(d, map[string][]string{"me.11.09": men}, map[string]int{me + "troparion.1": 4}), d)
		if p.Rule != "matins.sunday.saint" {
			t.Fatalf("rule %s", p.Rule)
		}
		c := refs(p.Slots["canon_1_3"])
		ode1 := c[:strings.Index(c, "ch.canon.katavasia")]
		o := oc + "matins.canon."
		// Resurrectional on 4, of the Theotokos on 2, the saint's two canons on 4 + 4.
		want := "ch.canon.ode.1," + o + "1.ode.1.irmos," + o + "1.ode.1.troparion.1," + o + "1.ode.1.troparion.2," + o + "1.ode.1.theotokion," +
			o + "3.ode.1.troparion.1," + o + "3.ode.1.theotokion," +
			strings.Repeat(me+"matins.canon.1.ode.1.troparion.1,", 4) + strings.Repeat(me+"matins.canon.2.ode.1.troparion.1,", 4)
		if ode1 != want {
			t.Errorf("ode 1:\n got %s\nwant %s", ode1, want)
		}
		if s := refs(p.Slots["sedalny_p"]); !strings.HasPrefix(s, me+"matins.k_sedalen.p.1") {
			t.Errorf("after the polyeleos %s", s)
		}
		pr := refs(p.Slots["praises"])
		if !strings.Contains(pr, oc+"matins.praises.sticheron.4,ch.matins.praises.150.2,"+me+"matins.praises.sticheron.1") {
			t.Errorf("praises: the Octoechos 4 and the saint's 4: %s", pr)
		}
	})
	t.Run("afterfeast of the Entry: «Христос раждается»", func(t *testing.T) {
		d := calendar.MustJDate(2026, time.November, 23)
		c := sundayMatinsCorpus(d, map[string][]string{
			"me.11.21": {"troparion.1", "kontakion.1", "matins.canon.1.ode.1.irmos"},
			"me.11.23": {"troparion.1", "kontakion.1", "matins.canon.1.ode.1.troparion.1"},
			"me.12.25": {"matins.canon.1.ode.1.irmos"},
		}, nil)
		p := planSundayMatins(t, c, d)
		if p.Rule != "matins.sunday.afterfeast" {
			t.Fatalf("rule %s", p.Rule)
		}
		tone := itoa(calendar.NewDayContext(d).Tone)
		check(t, p, map[string]string{
			"troparia": "oc." + tone + ".sun.troparionx2,common.glory,me.11.23.troparion.1,common.and_now,me.11.21.troparion.1",
		})
		if c := refs(p.Slots["canon_1_3"]); !strings.Contains(c, "ch.canon.katavasia,me.12.25.matins.canon.1.ode.1.irmos") {
			t.Errorf("katavasia of ode 1: %s", c)
		}
	})
}
