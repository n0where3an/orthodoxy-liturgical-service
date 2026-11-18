package liturgy

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// sundayCorpus: the Octoechos Sunday great vespers of the day's tone and
// the given Menaion units.
func sundayCorpus(d calendar.JDate, tones map[string]int, menaion ...string) mapCorpus {
	day := calendar.NewDayContext(d)
	oc := "oc." + itoa(day.Tone) + ".sun."
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1", oc + "gvespers.lic.dogmatikon", oc + "gvespers.aposticha.theotokion", oc + "troparion"}
	for i := 1; i <= 7; i++ {
		ids = append(ids, oc+"gvespers.lic.sticheron."+itoa(i))
	}
	for i := 1; i <= 3; i++ {
		ids = append(ids, oc+"gvespers.lic.theotokos."+itoa(i))
	}
	for i := 1; i <= 4; i++ {
		ids = append(ids, oc+"gvespers.aposticha.sticheron."+itoa(i))
		if i > 1 {
			ids = append(ids, oc+"gvespers.aposticha.verse."+itoa(i))
		}
	}
	ids = append(ids, oc+"kontakion", oc+"lit.beatitudes.glory", oc+"lit.beatitudes.now", oc+"lit.prokeimenon",
		oc+"lit.prokeimenon.verse", oc+"lit.alleluia.verse.1", oc+"lit.alleluia.verse.2")
	for i := 1; i <= 6; i++ {
		ids = append(ids, oc+"lit.beatitudes."+itoa(i))
	}
	for _, m := range menaion {
		ids = append(ids, me+m)
	}
	t := map[string]int{me + "troparion.1": 4}
	for k, v := range tones {
		t[me+k] = v
	}
	return mapCorpus{ids: ids, tones: t}
}

func TestSundayVespers(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 9) // Sunday, ordinary period
	if d.Weekday() != time.Sunday {
		t.Fatalf("test date is a %v", d.Weekday())
	}
	day := calendar.NewDayContext(d)
	oc := "oc." + itoa(day.Tone) + ".sun.gvespers."
	me := "me.11.09."
	verses := func(refs ...string) string {
		var out []string
		for i, r := range refs {
			out = append(out, fmt.Sprintf("ch.vespers.lic.verse.%d", 21+i), r)
		}
		return strings.Join(out, ",")
	}
	t.Run("saint", func(t *testing.T) {
		c := sundayCorpus(d, map[string]int{"vespers.aposticha.glory": 6}, "vespers.lic.sticheron.1", "vespers.lic.sticheron.2",
			"vespers.lic.glory", "vespers.aposticha.glory")
		plan, err := planVespers(t, c, d)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != "vespers.sunday" || plan.Template != "great_vespers" {
			t.Fatalf("rule %s, template %s", plan.Rule, plan.Template)
		}
		want := map[string]string{
			// 3 + 4 of the Octoechos, 3 of the Menaion (2 given: the first repeated).
			"lord_i_call": verses(oc+"lic.sticheron.1", oc+"lic.sticheron.2", oc+"lic.sticheron.3", oc+"lic.sticheron.4",
				oc+"lic.sticheron.5", oc+"lic.sticheron.6", oc+"lic.sticheron.7",
				me+"vespers.lic.sticheron.1", me+"vespers.lic.sticheron.2", me+"vespers.lic.sticheron.1"),
			"lord_i_call_glory_now": "common.glory," + me + "vespers.lic.glory,common.and_now," + oc + "lic.dogmatikon",
			"prokeimenon_verse": "ch.vespers.prokeimenon.sun.verse,ch.vespers.prokeimenon.sun,ch.vespers.prokeimenon.sun.verse.2," +
				"ch.vespers.prokeimenon.sun,ch.vespers.prokeimenon.sun.verse.3",
			"aposticha": oc + "aposticha.sticheron.1," + oc + "aposticha.verse.2," + oc + "aposticha.sticheron.2," + oc + "aposticha.verse.3," +
				oc + "aposticha.sticheron.3," + oc + "aposticha.verse.4," + oc + "aposticha.sticheron.4,common.glory," +
				me + "vespers.aposticha.glory,common.and_now,me.bogor.6.aposticha",
			"troparia": "ch.theotokos_rejoicex3",
			"paremias": "",
			"loaves":   "",
		}
		check(t, plan, want)
	})
	t.Run("no Menaion", func(t *testing.T) {
		plan, err := planVespers(t, sundayCorpus(d, nil), d)
		if err != nil {
			t.Fatal(err)
		}
		check(t, plan, map[string]string{
			"lord_i_call": verses(oc+"lic.sticheron.1", oc+"lic.sticheron.2", oc+"lic.sticheron.3", oc+"lic.sticheron.4",
				oc+"lic.sticheron.5", oc+"lic.sticheron.6", oc+"lic.sticheron.7",
				oc+"lic.theotokos.1", oc+"lic.theotokos.2", oc+"lic.theotokos.3"),
			"lord_i_call_glory_now": "common.glory_now," + oc + "lic.dogmatikon",
		})
		if a := refs(plan.Slots["aposticha"]); !strings.HasSuffix(a, "common.glory_now,"+oc+"aposticha.theotokion") {
			t.Errorf("aposticha %s", a)
		}
	})
}

func check(t *testing.T, plan *Plan, want map[string]string) {
	t.Helper()
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
}

func TestSundayVespersCombined(t *testing.T) {
	plan := func(t *testing.T, c mapCorpus, d calendar.JDate) *Plan {
		t.Helper()
		p, err := planVespers(t, c, d)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	verses := func(refs ...string) string {
		var out []string
		for i, r := range refs {
			out = append(out, fmt.Sprintf("ch.vespers.lic.verse.%d", 21+i), r)
		}
		return strings.Join(out, ",")
	}
	oct := func(d calendar.JDate, n int) []string {
		oc := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun.gvespers.lic.sticheron."
		var out []string
		for i := 1; i <= n; i++ {
			out = append(out, oc+itoa(i))
		}
		return out
	}

	t.Run("saint with a sign", func(t *testing.T) {
		d := calendar.MustJDate(2026, time.October, 26) // Sunday, St Demetrius
		oc := "oc." + itoa(calendar.NewDayContext(d).Tone) + ".sun.gvespers."
		me := "me.10.26."
		c := sundayFeastCorpus(d, map[string][]string{"me.10.26": {"troparion.1", "vespers.great", "gvespers.lic.sticheron.1",
			"gvespers.lic.sticheron.2", "gvespers.lic.sticheron.3", "gvespers.lic.glory", "gvespers.paremia.1",
			"gvespers.paremia.2", "gvespers.paremia.3", "gvespers.litiya.sticheron.1", "gvespers.litiya.glory",
			"gvespers.litiya.now", "gvespers.aposticha.glory"}}, nil)
		c.tones[me+"gvespers.aposticha.glory"] = 8
		p := plan(t, c, d)
		if p.Rule != "vespers.sunday.saint" {
			t.Fatalf("rule %s", p.Rule)
		}
		g := me + "gvespers."
		check(t, p, map[string]string{
			// The Octoechos 6 (3 + 3 of Anatolius), the saint's 4.
			"lord_i_call":           verses(append(oct(d, 6), g+"lic.sticheron.1", g+"lic.sticheron.2", g+"lic.sticheron.3", g+"lic.sticheron.1")...),
			"lord_i_call_glory_now": "common.glory," + g + "lic.glory,common.and_now," + oc + "lic.dogmatikon",
			"aposticha": oc + "aposticha.sticheron.1," + oc + "aposticha.verse.2," + oc + "aposticha.sticheron.2," + oc + "aposticha.verse.3," +
				oc + "aposticha.sticheron.3," + oc + "aposticha.verse.4," + oc + "aposticha.sticheron.4,common.glory," + g +
				"aposticha.glory,common.and_now,me.bogor.8.aposticha",
			"troparia": "ch.theotokos_rejoicex2," + me + "troparion.1",
			"loaves":   "sl.vespers.loaves_prayer",
		})
		if !strings.Contains(refs(p.Slots["paremias"]), g+"paremia.3") {
			t.Errorf("paremias %s", refs(p.Slots["paremias"]))
		}
	})
	t.Run("apodosis: the feast's stichera", func(t *testing.T) {
		d := calendar.MustJDate(2026, time.September, 21)
		c := sundayFeastCorpus(d, map[string][]string{
			"me.09.14": {"troparion.1", "kontakion.1", "gvespers.lic.sticheron.1", "gvespers.lic.sticheron.2", "gvespers.lic.now",
				"gvespers.aposticha.now", "gvespers.litiya.sticheron.1"},
			"me.09.21": {"troparion.1"},
		}, map[string]string{"me.09.21.troparion.1": "праздника,"})
		p := plan(t, c, d)
		if p.Rule != "vespers.sunday.afterfeast" {
			t.Fatalf("rule %s", p.Rule)
		}
		g := "me.09.14.gvespers."
		check(t, p, map[string]string{
			"lord_i_call": verses(append(oct(d, 4), g+"lic.sticheron.1", g+"lic.sticheron.2", g+"lic.sticheron.1",
				g+"lic.sticheron.2", g+"lic.sticheron.1", g+"lic.sticheron.2")...),
			"lord_i_call_glory_now": "common.glory_now," + g + "lic.now",
			// No litiya on the apodosis's own day: «Богородице Дево» thrice.
			"troparia": "ch.theotokos_rejoicex3",
			"loaves":   "",
		})
	})
	t.Run("feast of the Theotokos", func(t *testing.T) {
		d := calendar.MustJDate(2031, time.September, 8)
		c := sundayFeastCorpus(d, map[string][]string{"me.09.08": {"troparion.1", "kontakion.1", "gvespers.lic.sticheron.1",
			"gvespers.lic.now", "gvespers.paremia.1", "gvespers.paremia.2", "gvespers.paremia.3", "gvespers.litiya.sticheron.1",
			"gvespers.litiya.now", "gvespers.aposticha.now"}}, nil)
		p := plan(t, c, d)
		if p.Rule != "vespers.sunday.theotokos_feast" {
			t.Fatalf("rule %s", p.Rule)
		}
		g := "me.09.08.gvespers."
		st := g + "lic.sticheron.1"
		check(t, p, map[string]string{
			"lord_i_call":           verses(append(oct(d, 4), st, st, st, st, st, st)...),
			"lord_i_call_glory_now": "common.glory_now," + g + "lic.now",
			"troparia":              "me.09.08.troparion.1x3",
			"loaves":                "sl.vespers.loaves_prayer",
		})
		if a := refs(p.Slots["aposticha"]); !strings.HasSuffix(a, "common.glory_now,"+g+"aposticha.now") {
			t.Errorf("aposticha %s", a)
		}
	})
}
