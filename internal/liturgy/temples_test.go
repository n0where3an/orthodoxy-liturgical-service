package liturgy

import (
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

func TestLoadTemples(t *testing.T) {
	ts, err := LoadTemples(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[TempleKind]int{}
	for _, x := range ts.List {
		kinds[x.Kind]++
		if !strings.HasPrefix(x.Troparion, "me.") || !strings.HasPrefix(x.Kontakion, "me.") {
			t.Errorf("%s: %s %s", x.ID, x.Troparion, x.Kontakion)
		}
	}
	if kinds[TempleLord] == 0 || kinds[TempleTheotokos] == 0 || kinds[TempleSaint] == 0 {
		t.Errorf("kinds %v", kinds)
	}
	n, err := ts.Find("nicholas")
	if err != nil || n.Kind != TempleSaint || !n.IsFeastDay(12, 6) || n.IsFeastDay(5, 9) {
		t.Errorf("nicholas: %+v, %v", n, err)
	}
	m, err := ts.Find("me.07.19")
	if err != nil || m.Troparion != "me.07.19.troparion.1" || !m.IsFeastDay(7, 19) {
		t.Errorf("me.07.19: %+v, %v", m, err)
	}
	if x, err := ts.Find(""); x != nil || err != nil {
		t.Errorf("empty: %v %v", x, err)
	}
	for _, bad := range []string{"bogus", "me.13.01", "me.12.6"} {
		if _, err := ts.Find(bad); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}

func TestLitTropariaTemple(t *testing.T) {
	ts, err := LoadTemples(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	facts := func(d calendar.JDate, temple string) Facts {
		day := calendar.NewDayContext(d)
		me := "me." + pad(int(day.Menaion.Month)) + "." + pad(day.Menaion.Day) + "."
		tp, err := ts.Find(temple)
		if err != nil {
			t.Fatal(err)
		}
		return Facts{Day: day, Temple: tp, Menaion: map[string][]string{
			"troparion": {me + "troparion.1"}, "kontakion": {me + "kontakion.1"}}}
	}
	tue := calendar.MustJDate(2026, time.November, 4) // Tuesday
	thu := calendar.MustJDate(2026, time.November, 6)
	if tue.Weekday() != time.Tuesday || thu.Weekday() != time.Thursday {
		t.Fatal("test dates")
	}
	const tail = ",common.glory,ch.kontakion.with_the_saints,common.and_now,"
	for _, c := range []struct {
		d            calendar.JDate
		temple, want string
	}{
		{tue, "", "ch.troparion.day.tue,me.11.04.troparion.1,ch.kontakion.day.tue,me.11.04.kontakion.1" + tail + "ch.kontakion.protection"},
		// Lord's: the temple first, troparion and kontakion.
		{tue, "transfiguration", "me.08.06.troparion.1,ch.troparion.day.tue,me.11.04.troparion.1,me.08.06.kontakion.1,ch.kontakion.day.tue,me.11.04.kontakion.1" + tail + "ch.kontakion.protection"},
		// Theotokos': after the day's; its kontakion at «И ныне».
		{tue, "dormition", "ch.troparion.day.tue,me.08.15.troparion.1,me.11.04.troparion.1,ch.kontakion.day.tue,me.11.04.kontakion.1" + tail + "me.08.15.kontakion.1"},
		// A saint's: after the day's, troparion and kontakion.
		{tue, "nicholas", "ch.troparion.day.tue,me.12.06.troparion.1,me.11.04.troparion.1,ch.kontakion.day.tue,me.12.06.kontakion.1,me.11.04.kontakion.1" + tail + "ch.kontakion.protection"},
		// Thursday is St Nicholas's day anyway; the patronal day is the saint's own.
		{thu, "nicholas", "ch.troparion.day.thu,ch.troparion.day.thu.2,me.11.06.troparion.1,ch.kontakion.day.thu,ch.kontakion.day.thu.2,me.11.06.kontakion.1" + tail + "ch.kontakion.protection"},
		{tue, "me.11.04", "ch.troparion.day.tue,me.11.04.troparion.1,ch.kontakion.day.tue,me.11.04.kontakion.1" + tail + "ch.kontakion.protection"},
	} {
		items, err := litTroparia(facts(c.d, c.temple), "")
		if err != nil {
			t.Fatal(err)
		}
		if got := refs(items); got != c.want {
			t.Errorf("%v %q:\n got %s\nwant %s", c.d, c.temple, got, c.want)
		}
	}
	// A saint with a sign in a church of the Theotokos: «И ныне» the temple's.
	items, err := litTroparia(facts(tue, "protection"), "saint")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := refs(items), "me.11.04.troparion.1,me.10.01.troparion.3,common.glory,me.11.04.kontakion.1,common.and_now,me.10.01.kontakion.3"; got != want {
		t.Errorf("saint, protection:\n got %s\nwant %s", got, want)
	}
	// A twelve feast: the feast's only.
	items, _ = litTroparia(facts(tue, "nicholas"), "feast")
	if strings.Contains(refs(items), "12.06") {
		t.Errorf("feast with temple: %s", refs(items))
	}
}
