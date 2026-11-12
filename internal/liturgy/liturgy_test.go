package liturgy

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func dataDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", "data")
}

// fakeCorpus serves every requested id with a synthetic text and lists two
// troparia and one kontakion for every Menaion day.
type fakeCorpus struct{}

func (fakeCorpus) ListIDs(_ context.Context, prefix string) ([]string, error) {
	return []string{prefix + ".troparion.1", prefix + ".troparion.2", prefix + ".kontakion.1"}, nil
}

func (fakeCorpus) Load(_ context.Context, lang string, ids []string) (map[string]corpus.Text, error) {
	out := map[string]corpus.Text{}
	for _, id := range ids {
		out[id] = corpus.Text{UnitID: id, Lang: lang, Body: "BODY " + id, Incipit: "INC " + id}
	}
	return out, nil
}

func civil(y int, m time.Month, d int) calendar.JDate {
	j, err := calendar.FromCivil(y, m, d)
	if err != nil {
		panic(err)
	}
	return j
}

func TestRuleSelection(t *testing.T) {
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &Planner{Rules: rules, Menaion: fakeCorpus{}}
	tests := []struct {
		date        calendar.JDate
		rule        string
		unsupported bool
	}{
		{civil(2026, time.November, 12), "ninth_hour.ordinary", false}, // Thursday
		{civil(2026, time.November, 14), "ninth_hour.ordinary", false}, // Saturday
		{civil(2026, time.November, 15), "ninth_hour.sunday", true},
		{calendar.Pascha(2027).AddDays(3), "ninth_hour.bright_week", true},
		{calendar.Pascha(2027).AddDays(-40), "ninth_hour.lenten_weekday", true}, // Wednesday of Lent
		{calendar.Pascha(2027).AddDays(-43), "ninth_hour.triodion_day", true},   // Saturday of Lent
		{calendar.Pascha(2027).AddDays(20), "ninth_hour.triodion_day", true},    // Pentecostarion weekday
	}
	for _, tt := range tests {
		plan, err := p.Plan(context.Background(), tt.date, "ninth_hour", Params{Priest: true})
		var ue *UnsupportedError
		switch {
		case tt.unsupported:
			if !errors.As(err, &ue) || ue.Rule != tt.rule {
				t.Errorf("%v: want unsupported %s, got plan=%v err=%v", tt.date, tt.rule, plan, err)
			}
		case err != nil:
			t.Errorf("%v: %v", tt.date, err)
		case plan.Rule != tt.rule:
			t.Errorf("%v: rule %s, want %s", tt.date, plan.Rule, tt.rule)
		}
	}
}

func TestHoursAlternation(t *testing.T) {
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &Planner{Rules: rules, Menaion: fakeCorpus{}}
	plan, err := p.Plan(context.Background(), civil(2026, time.November, 12), "ninth_hour", Params{Priest: true})
	if err != nil {
		t.Fatal(err)
	}
	// Two troparia: the ninth hour (4th) takes the second; one kontakion: the first.
	if got := plan.Slots["troparion"]; len(got) != 1 || got[0].Ref != "me.10.30.troparion.2" {
		t.Errorf("troparion slot %v", got)
	}
	if got := plan.Slots["kontakion"]; len(got) != 1 || got[0].Ref != "me.10.30.kontakion.1" {
		t.Errorf("kontakion slot %v", got)
	}
}

func generate(t *testing.T, params Params, mode string) string {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Menaion: fakeCorpus{}}, Texts: fakeCorpus{}}
	var buf bytes.Buffer
	if _, err := g.Generate(context.Background(), Request{
		Date: civil(2026, time.November, 12), Service: "ninth_hour", Params: params, Lang: "cu", Mode: mode,
	}, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestGenerateVariants(t *testing.T) {
	laity := generate(t, Params{Priest: true}, "laity")
	kliros := generate(t, Params{Priest: true}, "kliros")
	reader := generate(t, Params{Priest: false}, "laity")
	standalone := generate(t, Params{Priest: true, Standalone: true}, "laity")

	has := func(name, html, s string, want bool) {
		t.Helper()
		if strings.Contains(html, s) != want {
			t.Errorf("%s: contains %q = %v, want %v", name, s, !want, want)
		}
	}
	// Laity sees exclamations in full, kliros only incipits.
	has("laity", laity, "BODY sl.exclamation.blessed_is_our_god", true)
	has("kliros", kliros, "BODY sl.exclamation.blessed_is_our_god", false)
	has("kliros", kliros, "INC sl.exclamation.blessed_is_our_god…", true)
	// Chant is full in both.
	has("kliros", kliros, "BODY ch.none.psalm.83", true)
	// Rubrics by audience.
	has("laity", laity, hip.ToUnicode("Тропа'рь дне`"), false)
	has("kliros", kliros, hip.ToUnicode("Тропа'рь дне`"), true)
	// Reader's order: no priest exclamations.
	has("reader", reader, "sl.exclamation", false)
	has("reader", reader, "BODY common.prayers_of_holy_fathers", true)
	// Dismissal only when standalone.
	has("laity", laity, "sl.dismissal.daily", false)
	has("standalone", standalone, "BODY sl.dismissal.daily", true)
	// Slots are filled.
	has("laity", laity, "BODY me.10.30.troparion.2", true)
	has("laity", laity, "BODY me.10.30.kontakion.1", true)
}

func TestModes(t *testing.T) {
	cases := []struct {
		n             RNode
		kliros, laity Detail
	}{
		{RNode{Kind: KindText, Performer: Reader}, Full, Full},
		{RNode{Kind: KindText, Performer: Choir}, Full, Full},
		{RNode{Kind: KindText, Performer: Priest, Voice: Aloud}, Cue, Full},
		{RNode{Kind: KindText, Performer: Deacon, Voice: Aloud}, Cue, Full},
		{RNode{Kind: KindText, Performer: Priest, Voice: Secret}, Omit, Placeholder},
		{RNode{Kind: KindRubric, Audience: AudienceKliros}, Full, Omit},
		{RNode{Kind: KindRubric, Audience: AudienceLaity}, Omit, Full},
	}
	for _, c := range cases {
		if d := Kliros.Detail(&c.n); d != c.kliros {
			t.Errorf("kliros %+v: %v, want %v", c.n, d, c.kliros)
		}
		if d := Laity.Detail(&c.n); d != c.laity {
			t.Errorf("laity %+v: %v, want %v", c.n, d, c.laity)
		}
	}
}
