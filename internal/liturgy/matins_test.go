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
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// matinsCorpus: Octoechos weekday matins of the day's tone and the Menaion
// matins with menCanons canons (3 troparia and a theotokion per ode).
func matinsCorpus(d calendar.JDate, menCanons int, extra ...string) mapCorpus {
	day := calendar.NewDayContext(d)
	oc := fmt.Sprintf("oc.%d.%s.matins.", day.Tone, WeekdayKey(day.Weekday))
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1", me + "kontakion.1", me + "matins.ikos.1", me + "matins.sedalen.1", me + "matins.sedalen.theotokion"}
	for k := 1; k <= 3; k++ {
		ids = append(ids, fmt.Sprintf("%ssedalen.%d.1", oc, k), fmt.Sprintf("%ssedalen.%d.2", oc, k),
			fmt.Sprintf("%ssedalen.%d.verse.2", oc, k), fmt.Sprintf("%ssedalen.%d.theotokion", oc, k))
	}
	for _, o := range odeOrder {
		for c := 1; c <= 2; c++ {
			b := fmt.Sprintf("%scanon.%d.ode.%d.", oc, c, o)
			ids = append(ids, b+"irmos", b+"troparion.1", b+"troparion.2", b+"troparion.3", b+"theotokion")
		}
		for c := 1; c <= menCanons; c++ {
			b := fmt.Sprintf("%smatins.canon.%d.ode.%d.", me, c, o)
			ids = append(ids, b+"irmos", b+"troparion.1", b+"troparion.2", b+"troparion.3", b+"theotokion")
		}
	}
	ids = append(ids, oc+"aposticha.sticheron.1", oc+"aposticha.sticheron.2", oc+"aposticha.sticheron.3", oc+"aposticha.theotokion")
	for _, e := range extra {
		ids = append(ids, me+e)
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 4}}
}

func planMatins(t *testing.T, c mapCorpus, d calendar.JDate) (*Plan, error) {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	return (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, "matins", Params{Priest: true})
}

func TestWeekdayMatinsPlan(t *testing.T) {
	// Tuesday 2026-11-04 O.S. (winter Psalter: three kathismata).
	d := calendar.MustJDate(2026, time.November, 4)
	day := calendar.NewDayContext(d)
	oc := fmt.Sprintf("oc.%d.tue.matins.", day.Tone)
	me := "me.11.04."
	plan, err := planMatins(t, matinsCorpus(d, 2, "matins.exapostilarion.1"), d)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"kathisma_1":    "ps.kathisma.7.1,ps.kathisma.7.2,ps.kathisma.7.3",
		"kathisma_3":    "ps.kathisma.9.1,ps.kathisma.9.2,ps.kathisma.9.3",
		"sedalny_1":     oc + "sedalen.1.1," + oc + "sedalen.1.verse.2," + oc + "sedalen.1.2,common.glory_now," + oc + "sedalen.1.theotokion",
		"sedalny_3":     oc + "sedalen.3.1," + oc + "sedalen.3.verse.2," + oc + "sedalen.3.2,common.glory_now," + oc + "sedalen.3.theotokion",
		"troparia":      me + "troparion.1x2,common.glory_now,ch.theotokion.dismissal.4.tue",
		"sedalen":       me + "matins.sedalen.1,common.glory_now," + me + "matins.sedalen.theotokion",
		"kontakion":     me + "kontakion.1," + me + "matins.ikos.1",
		"exapostilaria": "oc.exapostilarion.tue,common.glory," + me + "matins.exapostilarion.1,common.and_now,oc.exapostilarion.mon.theotokion",
		"aposticha": oc + "aposticha.sticheron.1,ch.matins.aposticha.verse.2," + oc + "aposticha.sticheron.2,ch.matins.aposticha.verse.3," +
			oc + "aposticha.sticheron.3,common.glory_now," + oc + "aposticha.theotokion",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	if plan.Slots["troparia"][0].Repeat != 2 {
		t.Error("the troparion at «Бог Господь» is sung twice")
	}
	// Two Menaion canons: one Octoechos canon; katavasia after the 3rd ode.
	c13 := refs(plan.Slots["canon_1_3"])
	if strings.Contains(c13, oc+"canon.2.") {
		t.Error("with two Menaion canons, only the first Octoechos canon")
	}
	if strings.Contains(c13, me+"matins.canon.1.ode.1.irmos") {
		t.Error("the Menaion irmos is not sung (only the first canon's)")
	}
	if !strings.HasSuffix(c13, "ch.canon.katavasia,"+me+"matins.canon.2.ode.3.irmos") {
		t.Errorf("katavasia: the irmos of the last canon: %s", c13[max(0, len(c13)-120):])
	}
}

func TestMatinsOneMenaionCanon(t *testing.T) {
	d := calendar.MustJDate(2026, time.July, 8) // Tuesday, summer: two kathismata
	plan, err := planMatins(t, matinsCorpus(d, 1), d)
	if err != nil {
		t.Fatal(err)
	}
	day := calendar.NewDayContext(d)
	oc := fmt.Sprintf("oc.%d.tue.matins.", day.Tone)
	if !strings.Contains(refs(plan.Slots["canon_9"]), oc+"canon.2.ode.9.troparion.1") {
		t.Error("with one Menaion canon, both Octoechos canons")
	}
	if len(plan.Slots["kathisma_3"]) != 0 || len(plan.Slots["sedalny_3"]) != 0 {
		t.Error("summer: no third kathisma")
	}
	if got := refs(plan.Slots["kathisma_2"]); got != "ps.kathisma.8.1,ps.kathisma.8.2,ps.kathisma.8.3" {
		t.Errorf("kathisma 2: %s", got)
	}
}

func TestMatinsUnsupported(t *testing.T) {
	for _, c := range []struct {
		d    calendar.JDate
		rule string
	}{
		{calendar.Pascha(2027).AddDays(-7), "matins.sunday.unsupported"},            // Palm Sunday
		{calendar.MustJDate(2026, time.October, 25), "matins.saturday.unsupported"}, // Dimitrievskaya
		{calendar.MustJDate(2027, time.March, 3), "matins.triodion"},
	} {
		_, err := planMatins(t, matinsCorpus(c.d, 1), c.d)
		var ue *UnsupportedError
		if !errors.As(err, &ue) || ue.Rule != c.rule {
			t.Errorf("%v (%v): want %s, got %v", c.d, c.d.Weekday(), c.rule, err)
		}
	}
}

func TestMatinsRender(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 4)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	c := matinsCorpus(d, 2)
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c}, Texts: c}
	render := func(mode string, p Params) string {
		var buf bytes.Buffer
		if _, err := g.Generate(context.Background(), Request{Date: d, Service: "matins", Lang: "cu", Mode: mode, Params: p}, &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	laity := render("laity", Params{Priest: true})
	prayers := hip.ToUnicode("мл~твы о_у='треннiя")
	if n := strings.Count(laity, prayers); n != 1 {
		t.Errorf("morning prayers placeholder %d times, want 1", n)
	}
	if strings.Index(laity, prayers) > strings.Index(laity, "BODY ch.hexapsalm.psalm.87") {
		t.Error("morning prayers placeholder after Psalm 87")
	}
	for _, b := range []string{"BODY ch.hexapsalm.psalm.3", "BODY ch.psalm.50", "BODY ch.matins.magnificat.1",
		"BODY sl.matins.exclamation.ode9", "BODY ch.matins.doxology.1", "BODY lit.confirm_the_faith"} {
		if !strings.Contains(laity, b) {
			t.Errorf("no %s", b)
		}
	}
	if strings.Contains(laity, "BODY sl.dismissal.daily") {
		t.Error("joined to the first hour: no dismissal")
	}
	if !strings.Contains(render("laity", Params{Priest: true, Standalone: true}), "BODY sl.dismissal.daily") {
		t.Error("standalone matins ends with the dismissal")
	}
	reader := render("laity", Params{})
	if strings.Contains(reader, "BODY lit.") || strings.Contains(reader, "BODY sl.matins.exclamation") {
		t.Error("reader's order: no litanies or exclamations")
	}
}
