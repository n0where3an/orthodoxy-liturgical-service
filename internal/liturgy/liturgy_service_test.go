package liturgy

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func planLiturgy(t *testing.T, c mapCorpus, d calendar.JDate) (*Plan, error) {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	return (&Planner{Rules: rules, Feasts: feasts, Menaion: c}).Plan(context.Background(), d, "liturgy", Params{Priest: true})
}

func menaionDay(d calendar.JDate) mapCorpus {
	day := calendar.NewDayContext(d)
	me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
	return mapCorpus{ids: []string{me + "troparion.1", me + "kontakion.1"}}
}

func TestWeekdayLiturgySlots(t *testing.T) {
	cases := []struct {
		date           calendar.JDate
		troparia       string
		kontakia       string
		prokeimenon    string
		tone, alleluia string
	}{
		// Tuesday 2026-11-04 O.S.: the Forerunner.
		{calendar.MustJDate(2026, time.November, 4), "ch.troparion.day.tue", "ch.kontakion.day.tue",
			"sl.prokeimenon.tue", "lit.prokeimenon.tone.7", "lit.alleluia.tone.4"},
		// Thursday: the Apostles and St Nicholas.
		{calendar.MustJDate(2026, time.November, 6), "ch.troparion.day.thu,ch.troparion.day.thu.2",
			"ch.kontakion.day.thu,ch.kontakion.day.thu.2", "sl.prokeimenon.thu", "lit.prokeimenon.tone.8", "lit.alleluia.tone.1"},
		// Friday: the Cross (texts of Wednesday).
		{calendar.MustJDate(2026, time.November, 7), "ch.troparion.day.wed", "ch.kontakion.day.wed",
			"sl.prokeimenon.fri", "lit.prokeimenon.tone.7", "lit.alleluia.tone.1"},
	}
	for _, c := range cases {
		plan, err := planLiturgy(t, menaionDay(c.date), c.date)
		if err != nil {
			t.Fatalf("%v: %v", c.date, err)
		}
		day := calendar.NewDayContext(c.date)
		me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
		ap, gos := day.Lections.Apostle, day.Lections.Gospel
		want := map[string]string{
			"day_troparia":     c.troparia,
			"saint_troparion":  me + "troparion.1",
			"day_kontakia":     c.kontakia,
			"saint_kontakion":  me + "kontakion.1",
			"prokeimenon":      c.prokeimenon,
			"prokeimenon_tone": c.tone,
			"alleluia_tone":    c.alleluia,
			"apostle":          "lect.ap." + itoaN(ap.Week) + "." + itoaN(int(ap.Weekday)),
			"apostle_announce": "lect.ap." + itoaN(ap.Week) + "." + itoaN(int(ap.Weekday)) + ".announce",
			"gospel":           "lect.gos." + itoaN(gos.Week) + "." + itoaN(int(gos.Weekday)),
		}
		for slot, w := range want {
			if got := refs(plan.Slots[slot]); got != w {
				t.Errorf("%v %s:\n got %s\nwant %s", c.date, slot, got, w)
			}
		}
	}
}

func itoaN(n int) string {
	if n < 0 {
		return "-" + itoaN(-n)
	}
	if n < 10 {
		return itoa(n)
	}
	return itoaN(n/10) + itoa(n%10)
}

func TestLiturgyUnsupported(t *testing.T) {
	for _, c := range []struct {
		d    calendar.JDate
		rule string
	}{
		{calendar.MustJDate(2026, time.November, 2), "liturgy.sunday"},
		{calendar.MustJDate(2026, time.November, 8), "liturgy.saturday"},
		{calendar.MustJDate(2027, time.January, 7), "liturgy.feast"},        // afterfeast of the Theophany
		{calendar.MustJDate(2027, time.March, 3), "liturgy.lenten_weekday"}, // Wednesday of Great Lent
	} {
		_, err := planLiturgy(t, menaionDay(c.d), c.d)
		var ue *UnsupportedError
		if !errors.As(err, &ue) || ue.Rule != c.rule {
			t.Errorf("%v (%v): want %s, got %v", c.d, c.d.Weekday(), c.rule, err)
		}
	}
	great := calendar.MustJDate(2026, time.November, 13) // Friday; St John Chrysostom
	c := menaionDay(great)
	c.ids = append(c.ids, "me.11.13.vespers.great")
	_, err := planLiturgy(t, c, great)
	var ue *UnsupportedError
	if !errors.As(err, &ue) || ue.Rule != "liturgy.great_saint" {
		t.Errorf("saint with a sign: got %v", err)
	}
}

func TestLiturgyRender(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 4)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	c := menaionDay(d)
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c}, Texts: c}
	render := func(mode string, deacon bool) string {
		var buf bytes.Buffer
		if _, err := g.Generate(context.Background(), Request{Date: d, Service: "liturgy", Lang: "cu", Mode: mode,
			Params: Params{Priest: true, Deacon: deacon}}, &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	laity, kliros, deacon := render("laity", false), render("kliros", false), render("laity", true)
	if strings.Contains(kliros, "BODY sl.lit.anaphora_prayer") || strings.Contains(laity, "BODY sl.lit.anaphora_prayer") {
		t.Error("secret prayers of the anaphora must not be shown")
	}
	epiclesis := hip.ToUnicode("мл~твы призыва'нiя ст~а'гw дх~а и= преложе'нiя ст~ы'хъ дарw'въ")
	if n := strings.Count(laity, epiclesis); n != 1 {
		t.Errorf("laity: epiclesis placeholder %d times, want 1 (collapsed)", n)
	}
	if strings.Index(laity, epiclesis) > strings.Index(laity, "BODY ch.lit.we_hymn_thee") {
		t.Error("epiclesis placeholder after «We hymn Thee»")
	}
	for _, body := range []string{"BODY ch.typika.psalm_102", "BODY ch.lit.cherubic.1", "BODY common.creed",
		"BODY common.our_father", "BODY sl.dismissal.liturgy_chrysostom", "BODY lect.gos."} {
		if !strings.Contains(laity, body) {
			t.Errorf("laity: no %s", body)
		}
	}
	for _, own := range []string{"bless_master", "bless_entrance", "o_lord_save_the_pious", "bless_evangelist"} {
		if strings.Contains(laity, "BODY lit.deacon."+own) {
			t.Errorf("the deacon's own line %s without a deacon", own)
		}
	}
	if !strings.Contains(deacon, "BODY lit.deacon.bless_master") || !strings.Contains(deacon, "BODY lit.great.peace_from_above") {
		t.Error("with a deacon: his opening and the peace petitions of the second litany of the faithful")
	}
	if strings.Count(laity, "BODY lit.great.peace_from_above") != 1 {
		t.Error("without a deacon, the peace petitions are said only in the great litany")
	}
}
