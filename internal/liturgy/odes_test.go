package liturgy

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func TestWithOdeVerses(t *testing.T) {
	verses := func(o, n int, set string) []string {
		var out []string
		for i := 1; i <= n; i++ {
			out = append(out, fmt.Sprintf("ir.ode.%d.%s.%d", o, set, i))
		}
		return out
	}
	items := func(n int) []SlotItem {
		var out []SlotItem
		for i := 1; i <= n; i++ {
			out = append(out, SlotItem{Ref: fmt.Sprintf("t%d", i)})
		}
		return out
	}
	// Weekday, on 14 with 16 verses: verses 5..16, «Слава», «И ныне».
	f := Facts{OdeVerses: map[int][]string{1: verses(1, 16, "weekday"), 6: verses(6, 10, "weekday")}}
	got := refs(withOdeVerses(f, "weekday", 1, items(14)))
	if !strings.HasPrefix(got, "ir.ode.1.weekday.5,t1,ir.ode.1.weekday.6,t2,") ||
		!strings.HasSuffix(got, "ir.ode.1.weekday.16,t12,common.glory,t13,common.and_now,t14") {
		t.Errorf("weekday ode 1: %s", got)
	}
	// The 6th ode (10 verses) on 14: the irmos and one troparion without a verse.
	got = refs(withOdeVerses(f, "weekday", 6, items(14)))
	if !strings.HasPrefix(got, "t1,t2,ir.ode.6.weekday.1,t3,") {
		t.Errorf("weekday ode 6: %s", got)
	}
	// Sunday: the second place with «Слава, Господи, святому воскресению Твоему».
	f = Facts{OdeVerses: map[int][]string{3: verses(3, 10, "sunday")}}
	got = refs(withOdeVerses(f, "sunday", 3, items(14)))
	if !strings.HasPrefix(got, "t1,ir.sunday.refrain,t2,ir.ode.3.sunday.1,t3,") ||
		!strings.HasSuffix(got, "ir.ode.3.sunday.10,t12,common.glory,t13,common.and_now,t14") {
		t.Errorf("sunday ode 3: %s", got)
	}
	// On 8: the last six verses.
	got = refs(withOdeVerses(Facts{OdeVerses: map[int][]string{1: verses(1, 16, "weekday")}}, "weekday", 1, items(8)))
	if !strings.HasPrefix(got, "ir.ode.1.weekday.11,t1,") {
		t.Errorf("on 8: %s", got)
	}
}

func TestMatinsTypikonOdeVerses(t *testing.T) {
	d := calendar.MustJDate(2026, time.October, 21) // Wednesday, ordinary
	c := matinsCorpus(d, 1)
	for o := 1; o <= 9; o++ {
		for n := 1; n <= 16; n++ {
			c.ids = append(c.ids, fmt.Sprintf("ir.ode.%d.weekday.%d", o, n))
		}
	}
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, typikon := range []bool{false, true} {
		p, err := (&Planner{Rules: rules, Menaion: c}).Plan(context.Background(), d, "matins", Params{Priest: true, Typikon: typikon})
		if err != nil {
			t.Fatal(err)
		}
		has := strings.Contains(refs(p.Slots["canon_1_3"]), "ir.ode.1.weekday.")
		if has != typikon {
			t.Errorf("typikon %v: verses of the biblical odes %v", typikon, has)
		}
	}
}

// «Егда прилучится празднество… на утрени две кафисмы, а третию рядовую на
// вечерни» (Устав о Псалтири).
func TestKathismataOnFeastDay(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 17) // Monday, winter: kathismata 4, 5, 6
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, festal := range []bool{false, true} {
		f := Facts{Day: calendar.NewDayContext(d), Menaion: map[string][]string{},
			Octoechos: map[string][]string{"matins.sedalen.3": {"oc.1.mon.matins.sedalen.3.1"}}}
		if festal {
			f.Menaion["gvespers.paremia"] = []string{"me.11.17.gvespers.paremia.1", "me.11.17.gvespers.paremia.2",
				"me.11.17.gvespers.paremia.3"}
		}
		slots := map[string][]SlotItem{}
		if err := rules.FillSlotRules("matins", f, slots); err != nil {
			t.Fatal(err)
		}
		if k := refs(slots["kathisma_3"]); (k == "") != festal {
			t.Errorf("festal %v: 3rd kathisma %q at matins", festal, k)
		}
	}
	// The vespers that evening (of Tuesday): the 6th instead of the 18th.
	tue := calendar.MustJDate(2026, time.November, 18)
	vc := weekdayCorpus(tue)
	vc.ids = append(vc.ids, "me.11.17.gvespers.paremia.1")
	v, err := planVespers(t, vc, tue)
	if err != nil {
		t.Fatal(err)
	}
	if k := refs(v.Slots["kathisma"]); !strings.HasPrefix(k, "ps.kathisma.6.") {
		t.Errorf("vespers after the feast: %s", k)
	}
	v, err = planVespers(t, weekdayCorpus(tue), tue)
	if err != nil {
		t.Fatal(err)
	}
	if k := refs(v.Slots["kathisma"]); !strings.HasPrefix(k, "ps.kathisma.18.") {
		t.Errorf("ordinary winter vespers: %s", k)
	}
}

func TestSundaySummerKathisma17(t *testing.T) {
	d := calendar.MustJDate(2027, time.July, 12) // Sunday in summer
	c := sundayMatinsCorpus(d, map[string][]string{"me.07.12": {"troparion.1", "kontakion.1"}}, nil)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	kat, _ := LoadKatavasia(dataDir())
	for _, typikon := range []bool{false, true} {
		p, err := (&Planner{Rules: rules, Feasts: feasts, Katavasia: kat, Menaion: c}).Plan(context.Background(), d, "matins",
			Params{Priest: true, Typikon: typikon})
		if err != nil {
			t.Fatal(err)
		}
		if got := refs(p.Slots["kathisma_17"]) != ""; got != typikon {
			t.Errorf("typikon %v: kathisma 17 %v", typikon, got)
		}
	}
}

func TestOdeNineRefrains(t *testing.T) {
	f := Facts{Feast: &Feast{ID: "entry_of_theotokos"}, Refrains: map[string]bool{}}
	for _, k := range []string{"1.irmos", "1.1", "1.2", "2.irmos", "2.glory", "2.now"} {
		f.Refrains["ir.pripev.entry_of_theotokos."+k] = true
	}
	r := "ir.pripev.entry_of_theotokos."
	items, kat, ok := odeNineRefrains(f, []odeCanon{{irmos: "i1", troparia: []string{"a", "b", "c"}},
		{irmos: "i2", troparia: []string{"x", "y", "z"}}})
	if !ok {
		t.Fatal("no refrains")
	}
	want := r + "1.irmos,i1," + r + "1.irmos,i1," + r + "1.1,a," + r + "1.2,b," + r + "1.1,c," +
		r + "2.irmos,i2," + r + "2.irmos,i2," + r + "2.irmos,x," + r + "2.glory,y," + r + "2.now,z"
	if got := refs(items); got != want {
		t.Errorf("ode 9:\n got %s\nwant %s", got, want)
	}
	if got := refs(kat); got != r+"1.irmos,i1,"+r+"2.irmos,i2" {
		t.Errorf("katavasia %s", got)
	}
	if _, _, ok := odeNineRefrains(Facts{Feast: &Feast{ID: "x"}}, nil); ok {
		t.Error("refrains without the feast's")
	}
}

func TestFestalOdeVerses(t *testing.T) {
	var verses []string
	for i := 1; i <= 10; i++ {
		verses = append(verses, fmt.Sprintf("v%d", i))
	}
	f := Facts{OdeVerses: map[int][]string{1: verses}}
	var items []SlotItem
	for i := 1; i <= 13; i++ {
		items = append(items, SlotItem{Ref: fmt.Sprintf("t%d", i)})
	}
	// «Глаголем ирмос дважды», then the ten verses, «Слава», «И ныне».
	out := withOdeVerses(f, "feast", 1, items)
	if out[0].Ref != "t1" || out[0].Repeat != 2 || refs(out[1:3]) != "v1,t2" ||
		!strings.HasSuffix(refs(out), "v10,t11,common.glory,t12,common.and_now,t13") {
		t.Errorf("feast: %s (first ×%d)", refs(out), out[0].Repeat)
	}
}

// The katavasia: after every ode when the sign is red (the great doxology
// sung), after the 3rd, 6th, 8th and 9th when it is black (six-fold).
func TestKatavasiaByRank(t *testing.T) {
	d := calendar.MustJDate(2026, time.October, 21) // Wednesday
	for _, c := range []struct {
		extra []string
		want  int
	}{
		{nil, 1}, // six-fold: after the 3rd only, in odes 1-3
		{[]string{"matins.praises.sticheron.1"}, 1}, // six-fold with stichera at the praises: still black
		{[]string{"matins.great_doxology"}, 2},      // the red sign: after the 1st and the 3rd
	} {
		corpus := matinsCorpus(d, 1, c.extra...)
		p, err := planMatins(t, corpus, d)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(refs(p.Slots["canon_1_3"]), "ch.canon.katavasia"); got != c.want {
			t.Errorf("%v: %d katavasias in odes 1-3", c.extra, got)
		}
	}
}

func TestGreatDoxologyByRank(t *testing.T) {
	d := calendar.MustJDate(2026, time.October, 21)
	for _, red := range []bool{false, true} {
		extra := []string{"matins.praises.sticheron.1"}
		if red {
			extra = append(extra, "matins.great_doxology")
		}
		p, err := planMatins(t, matinsCorpus(d, 1, extra...), d)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(p.Slots["great_doxology"]) > 0; got != red {
			t.Errorf("red %v: great doxology %v", red, got)
		}
		// The Menaion's stichera at the praises, on 4: read (black) after
		// «Сотворити в них суд» and «Хвалите Бога во святых Его»; sung (red,
		// shortened) from the verse of the first sticheron.
		want := "ch.matins.praises.149.verse,ch.matins.praises.150.1,ch.matins.praises.150.2,"
		if red {
			want = "ch.matins.praises.150.2,"
		}
		if pr := refs(p.Slots["praises"]); !strings.HasPrefix(pr, want) {
			t.Errorf("red %v: praises %s", red, pr)
		}
	}
}

// The great doxology by the sign of the Typikon: red (doxology and higher)
// sung, black or none read; the Menaion's «славословие великое» counts only
// in a fore- or afterfeast (no sign of its own) or without the Typikon.
func TestRedSign(t *testing.T) {
	me := func(ids ...string) map[string][]string {
		m := map[string][]string{}
		for _, id := range ids {
			m["matins.great_doxology"] = append(m["matins.great_doxology"], id)
		}
		return m
	}
	day := calendar.NewDayContext(calendar.MustJDate(2026, time.October, 14))
	marker := "me.10.14.matins.great_doxology"
	for _, c := range []struct {
		name string
		f    Facts
		want bool
	}{
		{"doxology sign", Facts{Day: day, Sign: "doxology"}, true},
		{"polyeleos sign", Facts{Day: day, Sign: "polyeleos"}, true},
		{"six-fold sign, the Menaion says great", Facts{Day: day, Sign: "six_fold", Menaion: me(marker)}, false},
		{"no sign", Facts{Day: day, Sign: "none"}, false},
		{"no Typikon, the Menaion says great", Facts{Day: day, Menaion: me(marker)}, true},
		{"afterfeast, the Menaion says great", Facts{Day: day, Sign: "none", Menaion: me(marker), Feast: &Feast{ID: "x"}}, true},
		{"forefeast, the Menaion says great", Facts{Day: day, Sign: "none", Menaion: me(marker),
			Of: map[string]string{"me.10.14.troparion.1": "предпразднства,"}}, true},
	} {
		if got := c.f.greatDoxology(); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
