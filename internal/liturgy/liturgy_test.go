package liturgy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

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
		{civil(2026, time.November, 12), "hours.ordinary", false}, // Thursday
		{civil(2026, time.November, 14), "hours.ordinary", false}, // Saturday
		{civil(2026, time.November, 15), "hours.sunday", true},
		{calendar.Pascha(2027).AddDays(3), "hours.bright_week", true},
		{calendar.Pascha(2027).AddDays(-40), "hours.lenten_weekday", true}, // Wednesday of Lent
		{calendar.Pascha(2027).AddDays(-43), "hours.triodion_day", true},   // Saturday of Lent
		{calendar.Pascha(2027).AddDays(20), "hours.triodion_day", true},    // Pentecostarion weekday
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
	// The fake day has two troparia and one kontakion: the hours alternate
	// the troparia (1, 2, 1, 2) and all take the only kontakion.
	want := map[string]string{"first_hour": "1", "third_hour": "2", "sixth_hour": "1", "ninth_hour": "2"}
	for service, n := range want {
		plan, err := p.Plan(context.Background(), civil(2026, time.November, 12), service, Params{Priest: true})
		if err != nil {
			t.Fatal(err)
		}
		if got := plan.Slots["troparion"]; len(got) != 1 || got[0].Ref != "me.10.30.troparion."+n {
			t.Errorf("%s: troparion slot %v", service, got)
		}
		if got := plan.Slots["kontakion"]; len(got) != 1 || got[0].Ref != "me.10.30.kontakion.1" {
			t.Errorf("%s: kontakion slot %v", service, got)
		}
	}
}

func generate(t *testing.T, params Params, mode string) string {
	t.Helper()
	return generateService(t, "ninth_hour", params, mode)
}

func generateService(t *testing.T, service string, params Params, mode string) string {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Menaion: fakeCorpus{}}, Texts: fakeCorpus{}}
	var buf bytes.Buffer
	if _, err := g.Generate(context.Background(), Request{
		Date: civil(2026, time.November, 12), Service: service, Params: params, Lang: "cu", Mode: mode,
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

func TestHourVariants(t *testing.T) {
	has := func(name, html, s string, want bool) {
		t.Helper()
		if strings.Contains(html, s) != want {
			t.Errorf("%s: contains %q = %v, want %v", name, s, !want, want)
		}
	}
	// First hour: joined to matins (no usual beginning), always a dismissal;
	// parish: priest reads «Христе, Свете истинный», then «Взбранной Воеводе».
	first := generateService(t, "first_hour", Params{Priest: true}, "laity")
	has("1st", first, "sl.exclamation.blessed_is_our_god", false)
	has("1st", first, "BODY common.come_let_us_worship.1", true)
	has("1st", first, "BODY sl.dismissal.daily", true)
	has("1st", first, "BODY tp.akathist.kontakion.1", true)
	has("1st", first, `class="item text priest"><p><span class="who">`+hip.ToUnicode("Сщ~е'нникъ:")+`</span>BODY ch.prime.prayer`, true)
	firstT := generateService(t, "first_hour", Params{Priest: true, Typikon: true}, "laity")
	has("1st typikon", firstT, "tp.akathist", false)
	has("1st typikon", firstT, `class="item text priest"><p><span class="who">`+hip.ToUnicode("Сщ~е'нникъ:")+`</span>BODY ch.prime.prayer`, false)
	has("1st typikon", firstT, "BODY ch.prime.prayer", true)
	firstS := generateService(t, "first_hour", Params{Priest: true, Standalone: true}, "laity")
	has("1st standalone", firstS, "BODY sl.exclamation.blessed_is_our_god", true)

	// Third hour: usual beginning, no dismissal (the sixth follows).
	third := generateService(t, "third_hour", Params{Priest: true}, "laity")
	has("3rd", third, "BODY sl.exclamation.blessed_is_our_god", true)
	has("3rd", third, "sl.dismissal", false)
	has("3rd", third, "BODY ch.terce.prayer_of_mardarius", true)
	thirdS := generateService(t, "third_hour", Params{Priest: true, Standalone: true}, "laity")
	has("3rd standalone", thirdS, "BODY sl.dismissal.daily", true)

	// Sixth hour: joined to the third, no dismissal (the liturgy follows).
	sixth := generateService(t, "sixth_hour", Params{Priest: true}, "laity")
	has("6th", sixth, "sl.exclamation.blessed_is_our_god", false)
	has("6th", sixth, "sl.dismissal", false)
	has("6th", sixth, "BODY ch.sext.prayer_of_basil", true)

	// Every hour: «Слава» before the troparion, «И ныне» before the theotokion.
	for _, svc := range []string{"first_hour", "third_hour", "sixth_hour", "ninth_hour"} {
		h := generateService(t, svc, Params{Priest: true}, "laity")
		g, tr, n := strings.Index(h, "BODY common.glory<"), strings.Index(h, "BODY me.10.30.troparion"), strings.Index(h, "BODY common.and_now")
		if g < 0 || tr < 0 || n < 0 || !(g < tr && tr < n) {
			t.Errorf("%s: glory/troparion/and-now order %d %d %d", svc, g, tr, n)
		}
	}
}

// TestTemplateRefsInManifest: every fixed text used by a template is cut by
// the import manifest (so the corpus can serve it).
func TestTemplateRefsInManifest(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(dataDir(), "orthlib", "units.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Units []struct{ Key string } `yaml:"units"`
	}
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, u := range m.Units {
		keys[u.Key] = true
	}
	for _, svc := range []string{"first_hour", "third_hour", "sixth_hour", "ninth_hour"} {
		tpl, err := LoadTemplate(dataDir(), svc)
		if err != nil {
			t.Fatal(err)
		}
		for _, ref := range tpl.Refs() {
			if !keys[ref] {
				t.Errorf("%s: %s is not in the import manifest", svc, ref)
			}
		}
	}
}

func seasonSlots(t *testing.T, d calendar.JDate, service string) map[string][]SlotItem {
	t.Helper()
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	day := calendar.NewDayContext(d)
	slots := map[string][]SlotItem{}
	if err := rules.FillSlotRules(service, Facts{Day: day, Feast: feasts.At(day)}, slots); err != nil {
		t.Fatal(err)
	}
	return slots
}

func refs(items []SlotItem) string {
	var out []string
	for _, it := range items {
		s := it.Ref
		if it.Repeat > 1 {
			s += fmt.Sprintf("x%d", it.Repeat)
		}
		out = append(out, s)
	}
	return strings.Join(out, ",")
}

func TestSeasonSlots(t *testing.T) {
	j := func(m time.Month, d int) calendar.JDate { return calendar.MustJDate(2026, m, d) }
	p := calendar.Pascha(2027)
	const (
		akathist = "tp.akathist.kontakion.1"
		ordinary = "common.glory_to_thee_our_god,common.heavenly_king"
		risen    = "tc.pascha.troparion x3"
	)
	tests := []struct {
		name                  string
		date                  calendar.JDate
		closing, opening, pre string // first hour: closing kontakion, opening prayers, paschal opening
	}{
		{"ordinary day", j(time.October, 30), akathist, ordinary, ""},
		{"Nativity of the Theotokos", j(time.September, 8), "me.09.08.kontakion.1", ordinary, ""},
		{"its apodosis", j(time.September, 12), "me.09.08.kontakion.1", ordinary, ""},
		{"after the apodosis", j(time.September, 13), akathist, ordinary, ""},
		{"Exaltation afterfeast", j(time.September, 20), "me.09.14.kontakion.1", ordinary, ""},
		{"Entry apodosis", j(time.November, 25), "me.11.21.kontakion.1", ordinary, ""},
		{"Nativity afterfeast", j(time.December, 31), "me.12.25.kontakion.1", ordinary, ""},
		{"Transfiguration apodosis", j(time.August, 13), "me.08.06.kontakion.1", ordinary, ""},
		{"Dormition apodosis", j(time.August, 23), "me.08.15.kontakion.1", ordinary, ""},
		{"Thomas Sunday", p.AddDays(7), "tc.pascha.kontakion", risen, risen},
		{"apodosis of Pascha", p.AddDays(38), "tc.pascha.kontakion", risen, risen},
		{"Ascension", p.AddDays(39), "tc.ascension.kontakion", "", ""},
		{"Saturday before Pentecost", p.AddDays(48), akathist, "", ""},
		{"Pentecost", p.AddDays(49), "tc.pentecost.kontakion", "", ""},
		{"Pentecost apodosis", p.AddDays(55), "tc.pentecost.kontakion", ordinary, ""},
		{"All Saints", p.AddDays(56), akathist, ordinary, ""},
	}
	for _, tt := range tests {
		s := seasonSlots(t, tt.date, "first_hour")
		if got := refs(s["closing_kontakion"]); got != tt.closing {
			t.Errorf("%s (%v): closing kontakion %q, want %q", tt.name, tt.date, got, tt.closing)
		}
		if got := refs(s["opening_prayers"]); got != strings.ReplaceAll(tt.opening, " ", "") {
			t.Errorf("%s: opening prayers %q, want %q", tt.name, got, tt.opening)
		}
		if got := refs(s["paschal_opening"]); got != strings.ReplaceAll(tt.pre, " ", "") {
			t.Errorf("%s: paschal opening %q, want %q", tt.name, got, tt.pre)
		}
	}
	// The closing kontakion is a first-hour slot only; the third hour has
	// no paschal_opening slot (it starts with the usual beginning).
	s := seasonSlots(t, p.AddDays(10), "third_hour")
	if _, ok := s["closing_kontakion"]; ok {
		t.Error("third hour got a closing kontakion")
	}
	if _, ok := s["paschal_opening"]; ok {
		t.Error("third hour got a paschal opening")
	}
	if got := refs(s["opening_prayers"]); got != "tc.pascha.troparionx3" {
		t.Errorf("third hour in the Paschal period: opening %q", got)
	}
}

func TestFeastKontakionInFirstHour(t *testing.T) {
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: fakeCorpus{}}, Texts: fakeCorpus{}}
	var buf bytes.Buffer
	plan, err := g.Generate(context.Background(), Request{
		Date: calendar.MustJDate(2026, time.September, 10), Service: "first_hour",
		Params: Params{Priest: true}, Lang: "cu", Mode: "laity",
	}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Day.Feast != "nativity_of_theotokos" {
		t.Errorf("feast %q", plan.Day.Feast)
	}
	if !strings.Contains(buf.String(), "BODY me.09.08.kontakion.1") || strings.Contains(buf.String(), "tp.akathist") {
		t.Error("first hour in the afterfeast should end with the kontakion of the feast")
	}
}

func TestFeastAndAfterfeastHours(t *testing.T) {
	rules, err := LoadRules(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	feasts, err := LoadFeasts(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &Planner{Rules: rules, Feasts: feasts, Menaion: fakeCorpus{}}
	plan := func(d calendar.JDate, svc string) *Plan {
		t.Helper()
		pl, err := p.Plan(context.Background(), d, svc, Params{Priest: true})
		if err != nil {
			t.Fatalf("%v %s: %v", d, svc, err)
		}
		return pl
	}
	// Find a weekday in the afterfeast of the Nativity of the Theotokos (Sep 9-12).
	var after calendar.JDate
	for d := 9; d <= 12; d++ {
		if j := calendar.MustJDate(2026, time.September, d); j.Weekday() != time.Sunday {
			after = j
			break
		}
	}
	feastDay := calendar.MustJDate(2026, time.September, 8)
	for _, svc := range []string{"first_hour", "third_hour", "sixth_hour", "ninth_hour"} {
		pl := plan(feastDay, svc)
		if pl.Rule != "hours.feast_day" || refs(pl.Slots["troparion"]) != "me.09.08.troparion.1" ||
			refs(pl.Slots["kontakion"]) != "me.09.08.kontakion.1" {
			t.Errorf("feast day %s: rule %s, slots %v", svc, pl.Rule, pl.Slots)
		}
	}
	// Afterfeast: feast, saint 1, saint 2, feast (the fake day has two troparia).
	_, _, dd := after.Date()
	saint := fmt.Sprintf("me.09.%02d.troparion.", dd)
	want := map[string]string{
		"first_hour": "me.09.08.troparion.1", "third_hour": saint + "1",
		"sixth_hour": saint + "2", "ninth_hour": "me.09.08.troparion.1",
	}
	for svc, w := range want {
		pl := plan(after, svc)
		if pl.Rule != "hours.afterfeast" || refs(pl.Slots["troparion"]) != w {
			t.Errorf("afterfeast %s: rule %s, troparion %s, want %s", svc, pl.Rule, refs(pl.Slots["troparion"]), w)
		}
	}
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
