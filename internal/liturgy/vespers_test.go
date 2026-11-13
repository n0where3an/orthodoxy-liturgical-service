package liturgy

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// mapCorpus serves a fixed set of unit ids (any id is loadable).
type mapCorpus struct {
	ids   []string
	tones map[string]int
}

func (c mapCorpus) ListIDs(_ context.Context, prefix string) ([]string, error) {
	var out []string
	for _, id := range c.ids {
		if strings.HasPrefix(id, prefix) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (c mapCorpus) UnitTones(_ context.Context, ids []string) (map[string]int, error) {
	out := map[string]int{}
	for _, id := range ids {
		out[id] = c.tones[id]
	}
	return out, nil
}

func (c mapCorpus) Load(ctx context.Context, lang string, ids []string) (map[string]corpus.Text, error) {
	return fakeCorpus{}.Load(ctx, lang, ids)
}

// weekdayDay builds the corpus of an ordinary weekday d: Octoechos of its
// tone and weekday, Menaion with the given extra ids.
func weekdayCorpus(d calendar.JDate, menaion ...string) mapCorpus {
	day := calendar.NewDayContext(d)
	oc := "oc." + itoa(day.Tone) + "." + WeekdayKey(day.Weekday) + ".vespers."
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1"}
	for i := 1; i <= 6; i++ {
		ids = append(ids, oc+"lic.sticheron."+itoa(i))
	}
	ids = append(ids, oc+"lic.theotokion", oc+"aposticha.sticheron.1", oc+"aposticha.sticheron.2",
		oc+"aposticha.sticheron.3", oc+"aposticha.theotokion")
	for _, m := range menaion {
		ids = append(ids, me+m)
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 4}}
}

func itoa(n int) string { return strings.TrimSpace(strings.Repeat(" ", 0) + string(rune('0'+n))) }
func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n/10) + itoa(n%10)
}

func planVespers(t *testing.T, c mapCorpus, d calendar.JDate) (*Plan, error) {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	return (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, "vespers", Params{Priest: true})
}

func TestDailyVespersSlots(t *testing.T) {
	// Thursday 2026-10-23 O.S. (ordinary period, winter Psalter).
	d := calendar.MustJDate(2026, time.October, 23)
	if d.Weekday() != time.Thursday {
		t.Fatalf("test date is a %v", d.Weekday())
	}
	day := calendar.NewDayContext(d)
	oc := "oc." + itoa(day.Tone) + ".thu.vespers."
	c := weekdayCorpus(d, "vespers.lic.sticheron.1", "vespers.lic.sticheron.2", "vespers.lic.sticheron.3",
		"vespers.lic.glory", "vespers.lic.theotokion")
	plan, err := planVespers(t, c, d)
	if err != nil {
		t.Fatal(err)
	}
	me := "me.10.23."
	want := map[string]string{
		"lord_i_call": strings.Join([]string{oc + "lic.sticheron.1", oc + "lic.sticheron.2", oc + "lic.sticheron.3",
			me + "vespers.lic.sticheron.1", me + "vespers.lic.sticheron.2", me + "vespers.lic.sticheron.3"}, ","),
		"lord_i_call_glory_now": "common.glory," + me + "vespers.lic.glory,common.and_now," + me + "vespers.lic.theotokion",
		"aposticha": strings.Join([]string{oc + "aposticha.sticheron.1", "ch.vespers.aposticha.verse.2", oc + "aposticha.sticheron.2",
			"ch.vespers.aposticha.verse.3", oc + "aposticha.sticheron.3"}, ","),
		"aposticha_glory_now": "common.glory_now," + oc + "aposticha.theotokion",
		"prokeimenon":         "ch.vespers.prokeimenon.thu",
		"troparia":            me + "troparion.1,common.glory_now,ch.theotokion.dismissal.4.thu",
		"kathisma":            "ps.kathisma.18.1,ps.kathisma.18.2,ps.kathisma.18.3", // winter
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
}

func TestVespersSingOnSix(t *testing.T) {
	d := calendar.MustJDate(2026, time.October, 22)
	c := weekdayCorpus(d, "vespers.lic.on.6", "vespers.lic.sticheron.1", "vespers.lic.sticheron.2",
		"vespers.lic.sticheron.3", "vespers.lic.sticheron.4")
	plan, err := planVespers(t, c, d)
	if err != nil {
		t.Fatal(err)
	}
	s := "me.10.22.vespers.lic.sticheron."
	if got, w := refs(plan.Slots["lord_i_call"]), s+"1,"+s+"1,"+s+"2,"+s+"2,"+s+"3,"+s+"4"; got != w {
		t.Errorf("got %s, want %s", got, w)
	}
}

func TestVespersKathismaAndFridayVerses(t *testing.T) {
	cases := []struct {
		date     calendar.JDate
		kathisma string
	}{
		{calendar.MustJDate(2026, time.July, 8), "ps.kathisma.6.1,ps.kathisma.6.2,ps.kathisma.6.3"},      // Tuesday, summer
		{calendar.MustJDate(2026, time.July, 7), ""},                                                     // Monday: no Psalter on Sunday evening
		{calendar.MustJDate(2026, time.July, 12), "ps.kathisma.18.1,ps.kathisma.18.2,ps.kathisma.18.3"},  // Saturday
		{calendar.MustJDate(2026, time.December, 24), "ps.kathisma.9.1,ps.kathisma.9.2,ps.kathisma.9.3"}, // Wednesday after Dec 20
	}
	for _, c := range cases {
		plan, err := planVespers(t, weekdayCorpus(c.date), c.date)
		if err != nil {
			t.Fatalf("%v (%v): %v", c.date, c.date.Weekday(), err)
		}
		if got := refs(plan.Slots["kathisma"]); got != c.kathisma {
			t.Errorf("%v (%v): kathisma %q, want %q", c.date, c.date.Weekday(), got, c.kathisma)
		}
		if c.date.Weekday() == time.Saturday && !strings.Contains(refs(plan.Slots["aposticha"]), "oc.vespers.aposticha.sat.verse.2") {
			t.Errorf("Friday evening aposticha verses: %s", refs(plan.Slots["aposticha"]))
		}
	}
}

func TestVespersUnsupported(t *testing.T) {
	sunday := calendar.MustJDate(2026, time.November, 2)
	great := calendar.MustJDate(2026, time.October, 22)
	for _, c := range []struct {
		d    calendar.JDate
		c    mapCorpus
		rule string
	}{
		{sunday, weekdayCorpus(sunday), "vespers.sunday"},
		{great, weekdayCorpus(great, "vespers.great"), "vespers.great"},
	} {
		_, err := planVespers(t, c.c, c.d)
		var ue *UnsupportedError
		if !errors.As(err, &ue) || ue.Rule != c.rule {
			t.Errorf("%v: want %s, got %v", c.d, c.rule, err)
		}
	}
}

func TestVespersPlaceholders(t *testing.T) {
	d := calendar.MustJDate(2026, time.October, 22)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	c := weekdayCorpus(d)
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c}, Texts: c}
	render := func(mode string, deacon bool) string {
		var buf bytes.Buffer
		if _, err := g.Generate(context.Background(), Request{Date: d, Service: "vespers", Lang: "cu", Mode: mode,
			Params: Params{Priest: true, Deacon: deacon}}, &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	laity, kliros, deacon := render("laity", false), render("kliros", false), render("laity", true)
	lamp := hip.ToUnicode("свjьти'льничныя мл~твы")
	if n := strings.Count(laity, lamp); n != 1 {
		t.Errorf("laity: lamp-lighting prayers placeholder %d times, want 1 (collapsed)", n)
	}
	if strings.Contains(laity, "BODY sl.vespers.lamp_prayer") || strings.Contains(kliros, "sl.vespers.lamp_prayer") || strings.Contains(kliros, lamp) {
		t.Error("secret prayers must not be shown")
	}
	if !strings.Contains(laity, hip.ToUnicode("мл~тву главопреклоне'нiя")) {
		t.Error("no head-bowing placeholder")
	}
	// The placeholder stands before the node it is said during.
	if strings.Index(laity, lamp) > strings.Index(laity, "BODY ch.vespers.psalm.103") {
		t.Error("lamp-lighting placeholder after the psalm")
	}
	if !strings.Contains(deacon, hip.ToUnicode("Дiа'конъ:")+"</span>BODY lit.peace") {
		t.Error("with a deacon, the litany is his")
	}
	if !strings.Contains(laity, hip.ToUnicode("Сщ~е'нникъ:")+"</span>BODY lit.peace") {
		t.Error("without a deacon, the priest says the litany")
	}
}
