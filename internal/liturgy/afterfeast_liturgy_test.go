package liturgy

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// The afterfeast of the Transfiguration, 8 August (a Friday in 2026), with
// the feast day's liturgy (6 August).
func afterfeastLiturgyCorpus() mapCorpus {
	me, fe := "me.08.08.", "me.08.06."
	ids := []string{me + "troparion.1", me + "kontakion.1",
		fe + "troparion.1", fe + "kontakion.1", fe + "lit.prokeimenon", fe + "lit.prokeimenon.verse",
		fe + "lit.alleluia.verse.1", fe + "lit.alleluia.verse.2", fe + "lit.kinonik", fe + "lit.zadostoinik"}
	for n := 1; n <= 3; n++ {
		ids = append(ids, fmt.Sprintf("%smatins.canon.1.ode.3.troparion.%d", me, n), fmt.Sprintf("%smatins.canon.2.ode.6.troparion.%d", me, n))
	}
	return mapCorpus{ids: ids, tones: map[string]int{fe + "lit.prokeimenon": 4, fe + "lit.alleluia.verse.1": 4}}
}

func planAfterfeastLiturgy(t *testing.T, temple string) *Plan {
	t.Helper()
	d := calendar.MustJDate(2026, time.August, 8)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	temples, _ := LoadTemples(dataDir())
	plan, err := (&Planner{Rules: rules, Feasts: feasts, Temples: temples, Menaion: afterfeastLiturgyCorpus()}).
		Plan(context.Background(), d, "liturgy", Params{Priest: true, Temple: temple})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestAfterfeastLiturgy(t *testing.T) {
	plan := planAfterfeastLiturgy(t, "")
	if plan.Rule != "liturgy.afterfeast.weekday" {
		t.Fatalf("rule %s", plan.Rule)
	}
	me, fe := "me.08.08.", "me.08.06."
	day := calendar.NewDayContext(calendar.MustJDate(2026, time.August, 8))
	ap := "lect.ap." + itoaN(day.Lections.Apostle.Week) + "." + itoaN(int(day.Lections.Apostle.Weekday))
	want := map[string]string{
		"entrance": "lit.entrance.afterfeast.transfiguration",
		"troparia": fe + "troparion.1," + me + "troparion.1,common.glory," + me + "kontakion.1,common.and_now," + fe + "kontakion.1",
		"prokeimena": "lit.prokeimenon.tone.4," + fe + "lit.prokeimenon," + fe + "lit.prokeimenon," + fe + "lit.prokeimenon.verse," +
			fe + "lit.prokeimenon," + fe + "lit.prokeimenon," + fe + "lit.prokeimenon",
		"alleluia": "lit.alleluia.tone.4,common.alleluia.thrice," + fe + "lit.alleluia.verse.1,common.alleluia.thrice," +
			fe + "lit.alleluia.verse.2,common.alleluia.thrice",
		"apostles":   "lit.wisdom," + ap + ".announce,lit.let_us_attend," + ap,
		"kinonik":    fe + "lit.kinonik,common.alleluia.thrice",
		"truly_meet": "lit.zadostoinik.refrain.transfiguration," + fe + "lit.zadostoinik",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	// The beatitudes: the feast's 3rd ode and the saint's 6th on 8.
	b := refs(plan.Slots["antiphon_3"])
	if want := "ch.typika.beatitudes.5," + me + "matins.canon.1.ode.3.troparion.1,"; !strings.Contains(b, want) {
		t.Errorf("beatitudes: %s", b)
	}
}

// A church of the Theotokos at a feast of the Lord: its troparion after the
// feast's, its kontakion before «Слава»; a church of the Lord: nothing.
func TestAfterfeastLiturgyTemple(t *testing.T) {
	me, fe := "me.08.08.", "me.08.06."
	if got, want := refs(planAfterfeastLiturgy(t, "dormition").Slots["troparia"]),
		fe+"troparion.1,me.08.15.troparion.1,"+me+"troparion.1,me.08.15.kontakion.1,common.glory,"+me+"kontakion.1,common.and_now,"+fe+"kontakion.1"; got != want {
		t.Errorf("church of the Theotokos:\n got %s\nwant %s", got, want)
	}
	if got, want := refs(planAfterfeastLiturgy(t, "exaltation").Slots["troparia"]),
		fe+"troparion.1,"+me+"troparion.1,common.glory,"+me+"kontakion.1,common.and_now,"+fe+"kontakion.1"; got != want {
		t.Errorf("church of the Lord:\n got %s\nwant %s", got, want)
	}
}

// Saturday in the afterfeast (9 August 2026): the feast's prokeimenon and
// communion verse with the saint's, the Saturday's Apostle and Gospel before
// the saint's (not after, as on an ordinary Saturday); no Saturday hymns.
func TestAfterfeastLiturgySaturday(t *testing.T) {
	d := calendar.MustJDate(2026, time.August, 9)
	if d.Weekday() != time.Saturday {
		t.Fatal("test date")
	}
	me, fe := "me.08.09.", "me.08.06."
	c := afterfeastLiturgyCorpus()
	for i, id := range c.ids {
		c.ids[i] = strings.Replace(id, "me.08.08.", me, 1)
	}
	c.ids = append(c.ids, me+"lit.prokeimenon", me+"lit.apostle", me+"lit.gospel", me+"lit.kinonik")
	c.tones[me+"lit.prokeimenon"] = 7
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	plan, err := (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, "liturgy", Params{Priest: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Rule != "liturgy.afterfeast.weekday" {
		t.Fatalf("rule %s", plan.Rule)
	}
	day := calendar.NewDayContext(d)
	ap := "lect.ap." + itoaN(day.Lections.Apostle.Week) + "." + itoaN(int(day.Lections.Apostle.Weekday))
	want := map[string]string{
		"troparia": fe + "troparion.1," + me + "troparion.1,common.glory," + me + "kontakion.1,common.and_now," + fe + "kontakion.1",
		"apostles": "lit.wisdom," + ap + ".announce,lit.let_us_attend," + ap + "," + me + "lit.apostle",
		"kinonik":  fe + "lit.kinonik," + me + "lit.kinonik,common.alleluia.thrice",
	}
	for slot, w := range want {
		if got := refs(plan.Slots[slot]); got != w {
			t.Errorf("%s:\n got %s\nwant %s", slot, got, w)
		}
	}
	if p := refs(plan.Slots["prokeimena"]); !strings.HasPrefix(p, "lit.prokeimenon.tone.4,"+fe) ||
		!strings.HasSuffix(p, "lit.prokeimenon.tone.7,"+me+"lit.prokeimenon,"+me+"lit.prokeimenon") || strings.Contains(p, "sl.prokeimenon.sat") {
		t.Errorf("prokeimena: %s", p)
	}
}
