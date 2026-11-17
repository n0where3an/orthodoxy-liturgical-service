package liturgy

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// greatCorpus: a Menaion service with great vespers on d; litiya optional.
func greatCorpus(d calendar.JDate, litiya bool) mapCorpus {
	day := calendar.NewDayContext(d)
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	g := me + "gvespers."
	ids := []string{me + "troparion.1", me + "vespers.great", g + "lic.on.8", g + "lic.glory",
		g + "aposticha.sticheron.1", g + "aposticha.verse.2", g + "aposticha.sticheron.2", g + "aposticha.verse.3",
		g + "aposticha.sticheron.3", g + "aposticha.glory", g + "aposticha.now"}
	for _, n := range []string{"1", "2", "3", "4", "5"} {
		ids = append(ids, g+"lic.sticheron."+n)
	}
	for _, n := range []string{"1", "2", "3"} {
		ids = append(ids, g+"paremia."+n, g+"paremia."+n+".title")
	}
	if litiya {
		ids = append(ids, g+"litiya.sticheron.1", g+"litiya.sticheron.2", g+"litiya.glory", g+"litiya.now")
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 4, g + "lic.glory": 6}}
}

func planGreat(t *testing.T, c mapCorpus, d calendar.JDate) *Plan {
	t.Helper()
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	plan, err := (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, "vespers", Params{Priest: true})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestVigilVespersPlan(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 13) // Friday: St John Chrysostom
	plan := planGreat(t, greatCorpus(d, true), d)
	if plan.Rule != "vespers.vigil" || plan.Template != "great_vespers" {
		t.Fatalf("rule %s, template %s", plan.Rule, plan.Template)
	}
	g := "me.11.13.gvespers."
	want := map[string]string{
		// 5 stichera on 8: the first three are repeated.
		"lord_i_call": strings.Join([]string{g + "lic.sticheron.1", g + "lic.sticheron.2", g + "lic.sticheron.3", g + "lic.sticheron.4",
			g + "lic.sticheron.5", g + "lic.sticheron.1", g + "lic.sticheron.2", g + "lic.sticheron.3"}, ","),
		// «И ныне» only cited: the dogmatikon of the tone of «Слава».
		"lord_i_call_glory_now": "common.glory," + g + "lic.glory,common.and_now,me.bogor.6.lic",
		"paremias": "lit.wisdom," + g + "paremia.1.title,lit.let_us_attend," + g + "paremia.1,lit.wisdom," + g + "paremia.2.title,lit.let_us_attend," + g +
			"paremia.2,lit.wisdom," + g + "paremia.3.title,lit.let_us_attend," + g + "paremia.3",
		"litiya":    g + "litiya.sticheron.1," + g + "litiya.sticheron.2,common.glory," + g + "litiya.glory,common.and_now," + g + "litiya.now",
		"aposticha": g + "aposticha.sticheron.1," + g + "aposticha.verse.2," + g + "aposticha.sticheron.2," + g + "aposticha.verse.3," + g + "aposticha.sticheron.3,common.glory," + g + "aposticha.glory,common.and_now," + g + "aposticha.now",
		"troparia":  "me.11.13.troparion.1x2,common.glory_now,ch.theotokos_rejoice",
		"kathisma":  "ps.kathisma.1.1",
		"loaves":    "sl.vespers.loaves_prayer",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
}

func TestPolyeleosAndFeastVespers(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 13)
	plan := planGreat(t, greatCorpus(d, false), d)
	if plan.Rule != "vespers.polyeleos" || len(plan.Slots["litiya"]) != 0 || len(plan.Slots["loaves"]) != 0 {
		t.Errorf("polyeleos: rule %s, litiya %v", plan.Rule, plan.Slots["litiya"])
	}
	if got := refs(plan.Slots["troparia"]); got != "me.11.13.troparion.1,common.glory_now,ch.theotokion.dismissal.4.thu" {
		t.Errorf("polyeleos troparia: %s", got)
	}
	// The Entry of the Theotokos (2026-11-21 O.S., Saturday): the feast's troparion thrice.
	f := calendar.MustJDate(2026, time.November, 21)
	plan = planGreat(t, greatCorpus(f, true), f)
	if plan.Rule != "vespers.vigil.feast" || refs(plan.Slots["troparia"]) != "me.11.21.troparion.1x3" {
		t.Errorf("feast: rule %s, troparia %s", plan.Rule, refs(plan.Slots["troparia"]))
	}
	// The Exaltation on Saturday evening: the feast, not Sunday vespers.
	e := calendar.MustJDate(2026, time.September, 14)
	if e.Weekday() != time.Sunday {
		t.Fatalf("test date is a %v", e.Weekday())
	}
	if plan = planGreat(t, greatCorpus(e, true), e); plan.Rule != "vespers.vigil.lord_feast_on_sunday" {
		t.Errorf("Exaltation on Sunday: rule %s", plan.Rule)
	}
}

func TestGreatVespersRender(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 13)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	render := func(litiya bool) string {
		c := greatCorpus(d, litiya)
		g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c}, Texts: c}
		var buf bytes.Buffer
		if _, err := g.Generate(context.Background(), Request{Date: d, Service: "vespers", Lang: "cu", Mode: "laity",
			Params: Params{Priest: true, Deacon: true}}, &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	vigil, polyeleos := render(true), render(false)
	for _, b := range []string{"BODY lit.vigil.arise", "BODY lit.litiya.save_o_god", "BODY sl.litiya.prayer",
		"BODY sl.vespers.loaves_prayer", "BODY ch.vigil.psalm_33", "BODY lit.augmented.let_us_say", "BODY lit.wisdom_aright"} {
		if !strings.Contains(vigil, b) {
			t.Errorf("vigil: no %s", b)
		}
	}
	// «Приидите, поклонимся» at the vigil is sung by the clergy.
	clergy := hip.ToUnicode("Сщ~е'нно-служи'телiе:")
	if !strings.Contains(vigil, clergy+"</span>BODY common.come_let_us_worship.1") {
		t.Error("vigil: «Приидите, поклонимся» by the clergy")
	}
	// The litiya responses of the Служебник: 40, 50, 30.
	for _, n := range []string{"м~.", "н~.", "л~."} {
		if !strings.Contains(vigil, hip.ToUnicode(n)) {
			t.Errorf("litiya: no «Господи, помилуй» %s", n)
		}
	}
	if strings.Contains(vigil, "BODY sl.dismissal.daily") {
		t.Error("vigil: no dismissal, matins follows")
	}
	if strings.Contains(polyeleos, "BODY lit.litiya.") || !strings.Contains(polyeleos, "BODY sl.dismissal.daily") ||
		!strings.Contains(polyeleos, "BODY sl.exclamation.blessed_is_our_god") {
		t.Error("polyeleos: daily beginning, no litiya, dismissal")
	}
}
