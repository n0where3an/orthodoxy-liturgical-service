package liturgy

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// feastCorpus: a Menaion service with great vespers and its liturgy;
// festal antiphons if festal, the beatitudes from the canon otherwise.
func feastCorpus(d calendar.JDate, festal bool) mapCorpus {
	day := calendar.NewDayContext(d)
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	ids := []string{me + "troparion.1", me + "kontakion.1", me + "vespers.great",
		me + "gvespers.paremia.1", me + "gvespers.paremia.2", me + "gvespers.paremia.3",
		me + "lit.prokeimenon", me + "lit.prokeimenon.verse", me + "lit.alleluia.verse.1", me + "lit.alleluia.verse.2",
		me + "lit.apostle", me + "lit.apostle.announce", me + "lit.gospel", me + "lit.gospel.announce", me + "lit.kinonik",
		me + "lit.zadostoinik"}
	if festal {
		for a := 1; a <= 3; a++ {
			for k := 1; k <= 3; k++ {
				ids = append(ids, fmt.Sprintf("%slit.antiphon.%d.%d", me, a, k))
			}
		}
		ids = append(ids, me+"lit.entrance", me+"lit.trisagion")
	} else {
		for k := 1; k <= 8; k++ {
			ids = append(ids, fmt.Sprintf("%slit.beatitudes.%d", me, k))
		}
		ids = append(ids, me+"lit.beatitudes.theotokion")
	}
	return mapCorpus{ids: ids, tones: map[string]int{me + "troparion.1": 1, me + "lit.prokeimenon": 7, me + "lit.alleluia.verse.1": 1}}
}

func TestFeastLiturgy(t *testing.T) {
	// The Exaltation, 2026-09-14 O.S. — on Sunday: a feast of the Lord.
	d := calendar.MustJDate(2026, time.September, 14)
	plan, err := planLiturgy(t, feastCorpus(d, true), d)
	if err != nil {
		t.Fatal(err)
	}
	me := "me.09.14."
	if plan.Rule != "liturgy.lord_feast_on_sunday" {
		t.Errorf("rule %s", plan.Rule)
	}
	want := map[string]string{
		"antiphon_1": me + "lit.antiphon.1.1," + me + "lit.antiphon.1.2," + me + "lit.antiphon.1.3",
		"entrance":   me + "lit.entrance",
		"troparia":   me + "troparion.1,common.glory_now," + me + "kontakion.1",
		"prokeimena": strings.Join([]string{"lit.prokeimenon.tone.7", me + "lit.prokeimenon", me + "lit.prokeimenon", me + "lit.prokeimenon.verse",
			me + "lit.prokeimenon", me + "lit.prokeimenon", me + "lit.prokeimenon"}, ","),
		"apostles":   "lit.wisdom," + me + "lit.apostle.announce,lit.let_us_attend," + me + "lit.apostle",
		"truly_meet": "lit.zadostoinik.refrain.exaltation," + me + "lit.zadostoinik",
		"kinonik":    me + "lit.kinonik,common.alleluia.thrice",
	}
	want["trisagion"] = me + "lit.trisagionx3,common.glory_now," + me + "lit.trisagion"
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
}

func TestGreatSaintLiturgy(t *testing.T) {
	// St John Chrysostom, Thursday 2026-11-13 O.S.: day and saint.
	d := calendar.MustJDate(2026, time.November, 13)
	plan, err := planLiturgy(t, feastCorpus(d, false), d)
	if err != nil {
		t.Fatal(err)
	}
	me := "me.11.13."
	if plan.Rule != "liturgy.great_saint" {
		t.Fatalf("rule %s", plan.Rule)
	}
	b := refs(plan.Slots["antiphon_3"])
	if !strings.Contains(b, "ch.typika.beatitudes.5,"+me+"lit.beatitudes.1,ch.typika.beatitudes.6,"+me+"lit.beatitudes.2") ||
		!strings.HasSuffix(b, "common.and_now,"+me+"lit.beatitudes.theotokion") {
		t.Errorf("beatitudes on 8: %s", b)
	}
	if got := refs(plan.Slots["troparia"]); got != me+"troparion.1,common.glory,"+me+"kontakion.1,common.and_now,ch.kontakion.protection" {
		t.Errorf("troparia: %s", got)
	}
	ap := refs(plan.Slots["apostles"])
	if !strings.HasPrefix(ap, "lit.wisdom,lect.ap.") || !strings.HasSuffix(ap, me+"lit.apostle") || strings.Contains(ap, me+"lit.apostle.announce") {
		t.Errorf("apostles: the day's announced, the saint's joined to it: %s", ap)
	}
	if g := refs(plan.Slots["gospels"]); strings.Count(g, ".announce") != 1 || !strings.HasSuffix(g, me+"lit.gospel") {
		t.Errorf("gospels: one announcement: %s", g)
	}
	if got := refs(plan.Slots["kinonik"]); got != "sl.kinonik.thu,"+me+"lit.kinonik,common.alleluia.thrice" {
		t.Errorf("kinonik: %s", got)
	}
	if got := refs(plan.Slots["truly_meet"]); got != "ch.lit.it_is_truly_meet" {
		t.Errorf("a saint has no zadostoinik: %s", got)
	}
}
