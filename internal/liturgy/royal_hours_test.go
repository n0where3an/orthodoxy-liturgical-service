package liturgy

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func royalCorpus(feast string) mapCorpus {
	rh := "rh." + feast + "."
	ids := []string{rh + "kontakion"}
	tones := map[string]int{}
	for _, h := range []int{1, 3, 6, 9} {
		p := fmt.Sprintf("%s%d.", rh, h)
		ids = append(ids, p+"psalm.1", p+"psalm.2", p+"psalm.3", p+"troparion", p+"theotokion", p+"hymn.1", p+"hymn.2",
			p+"verse.3", p+"hymn.4", p+"canonarch.5", p+"prokeimenon", p+"prokeimenon.verse",
			p+"reading.1", p+"reading.1.title", p+"reading.2", p+"reading.2.title", p+"reading.3", p+"reading.3.title")
		tones[p+"prokeimenon"] = 4
	}
	return mapCorpus{ids: ids, tones: tones}
}

// The Royal Hours: on the eve when it is a weekday, else on the Friday
// before; the hour's psalms, the stichera, the prokeimenon and readings.
func TestRoyalHours(t *testing.T) {
	for _, c := range []struct {
		date  calendar.JDate
		feast string
	}{
		{calendar.MustJDate(2026, time.December, 24), "nativity"}, // Wednesday
		{calendar.MustJDate(2026, time.January, 3), "theophany"},  // Friday: the Theophany on Monday
	} {
		if got := royalHours(calendar.NewDayContext(c.date)); got != c.feast {
			t.Fatalf("%v: royal hours %q", c.date, got)
		}
		p := "rh." + c.feast + ".9."
		plan := planHour(t, royalCorpus(c.feast), c.date, "ninth_hour")
		check(t, plan, map[string]string{
			"royal_psalms": p + "psalm.1," + p + "psalm.2," + p + "psalm.3",
			"troparion":    p + "troparion",
			"kontakion":    "rh." + c.feast + ".kontakion",
			"royal_hymns":  p + "hymn.1," + p + "hymn.2," + p + "verse.3," + p + "hymn.4," + p + "canonarch.5",
		})
		r := refs(plan.Slots["royal_readings"])
		if strings.Count(r, p+"reading.") != 6 || !strings.Contains(r, "sl.lit.wisdom_aright_gospel") {
			t.Errorf("%v readings: %s", c.date, r)
		}
	}
	// Not on a Saturday eve (the hours were on Friday).
	for y := 2020; y < 2040; y++ {
		d := calendar.MustJDate(y, time.December, 24)
		if day := calendar.NewDayContext(d); day.Weekday == time.Saturday {
			if got := royalHours(day); got != "" {
				t.Errorf("%v (Saturday): royal hours %q", d, got)
			}
			break
		}
	}
}

// The typika after the Royal Hours: with the liturgy on the eve (no Creed,
// no «Буди имя»), without it on the Friday before.
func TestRoyalTypika(t *testing.T) {
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		date  calendar.JDate
		feast string
		rule  string
		with  bool
	}{
		{calendar.MustJDate(2026, time.December, 24), "nativity", "typika.royal.eve", true},
		{calendar.MustJDate(2026, time.January, 3), "theophany", "typika.royal", false},
	} {
		plan, err := (&Planner{Rules: rules, Menaion: royalCorpus(c.feast)}).Plan(context.Background(), c.date, "typika", Params{Priest: true})
		if err != nil {
			t.Fatal(err)
		}
		if plan.Rule != c.rule || (len(plan.Slots["with_liturgy"]) > 0) != c.with {
			t.Errorf("%v: rule %s, with liturgy %v", c.date, plan.Rule, plan.Slots["with_liturgy"])
		}
	}
}
