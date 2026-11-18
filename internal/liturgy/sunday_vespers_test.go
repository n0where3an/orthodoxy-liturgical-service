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
