package liturgy

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

func TestSourceBook(t *testing.T) {
	for _, c := range []struct {
		tx   corpus.Text
		want string
	}{
		{corpus.Text{UnitID: "ch.psalm.50", Book: "chasoslov"}, "chasoslov"},
		{corpus.Text{UnitID: "me.11.13.troparion.1", Book: "menaion"}, "menaion"},
		{corpus.Text{UnitID: "oc.4.tue.matins.canon.1.ode.1.irmos", Book: "octoechos"}, "octoechos"},
		// The General Menaion's troparion in a day that lacks its own.
		{corpus.Text{UnitID: "me.12.19.troparion.1", Book: "menaion", Meta: map[string]any{"copied_from": "om.martyr.troparion.1"}}, "general_menaion"},
		// Readings: the Apostle and the Gospel, whether ordinary or the saint's.
		{corpus.Text{UnitID: "lect.ap.24.6", Book: "lectionary"}, "apostle"},
		{corpus.Text{UnitID: "lect.zach.jn.16", Book: "lectionary"}, "gospel"},
		{corpus.Text{UnitID: "me.11.13.lit.apostle", Book: "lectionary", Meta: map[string]any{"copied_from": "lect.zach.ap.318"}}, "apostle"},
		{corpus.Text{UnitID: "me.11.13.lit.gospel", Book: "lectionary", Meta: map[string]any{"copied_from": "lect.zach.jn.36"}}, "gospel"},
		// No book (the fake corpora of the tests): by the prefix.
		{corpus.Text{UnitID: "common.glory"}, "chasoslov"},
		{corpus.Text{UnitID: "oc.1.mon.compline.canon.ode.1.irmos"}, "octoechos"},
	} {
		if got := SourceBook(c.tx); got != c.want {
			t.Errorf("%s: %s, want %s", c.tx.UnitID, got, c.want)
		}
	}
}

// The book in the margin where it changes: the Chasoslov's beginning, the
// Octoechos's canon, the Chasoslov again.
func TestRenderSourceBook(t *testing.T) {
	d := calendar.MustJDate(2026, time.November, 4)
	rules, _ := LoadRules(dataDir())
	feasts, _ := LoadFeasts(dataDir())
	c := complineCorpus(d)
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Feasts: feasts, Menaion: c},
		Texts: kindCorpus{c, map[string]string{"sl.vespers.most_holy_theotokos_save_us": "exclamation",
			"sl.exclamation.": "exclamation", "sl.dismissal.": "exclamation"}}}
	var buf bytes.Buffer
	if _, err := g.Generate(context.Background(), Request{Date: d, Service: "compline", Lang: "cu", Mode: "laity",
		Params: Params{Priest: true}}, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	var books []string
	for _, part := range strings.Split(out, `<span class="src">`)[1:] {
		books = append(books, part[:strings.Index(part, "<")])
	}
	got := strings.Join(books, ",")
	if got != "Часослов,Октоих,Часослов" {
		t.Errorf("books in the margin: %s", got)
	}
	// Not repeated while the book is the same.
	for i := 1; i < len(books); i++ {
		if books[i] == books[i-1] {
			t.Errorf("%s twice in a row", books[i])
		}
	}
}

// kindCorpus gives some texts of a mapCorpus their kind (by id or id
// prefix), as the corpus does.
type kindCorpus struct {
	mapCorpus
	kinds map[string]string
}

func (c kindCorpus) Load(ctx context.Context, lang string, ids []string) (map[string]corpus.Text, error) {
	out, err := c.mapCorpus.Load(ctx, lang, ids)
	for id, t := range out {
		for p, k := range c.kinds {
			if strings.HasPrefix(id, p) {
				t.Kind = k
				out[id] = t
			}
		}
	}
	return out, err
}

// The hours: the usual beginning is marked as the Chasoslov's; after the
// Menaion's kontakion «Господи, помилуй» 40 returns to the Chasoslov.
func TestRenderSourceBookHours(t *testing.T) {
	rules, _ := LoadRules(dataDir())
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Menaion: fakeCorpus{}}, Texts: fakeKinds{fakeCorpus{}}}
	var buf bytes.Buffer
	if _, err := g.Generate(context.Background(), Request{Date: civil(2026, time.November, 12), Service: "third_hour",
		Lang: "cu", Mode: "laity", Params: Params{Priest: true, Standalone: true}}, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	first := strings.Index(out, `<span class="src">`)
	if first < 0 || !strings.HasPrefix(out[first:], `<span class="src">Часослов</span><p>BODY common.glory_to_thee_our_god`) {
		t.Errorf("the beginning is not marked as the Chasoslov's: %.200s", out[max(first, 0):])
	}
	k := strings.LastIndex(out, "BODY me.")
	if k < 0 || !strings.Contains(out[k:], `<span class="src">Часослов</span><p>BODY common.lord_have_mercy<span class="repeat">м҃.`) {
		t.Errorf("«Господи, помилуй» 40 after the kontakion is not marked")
	}
}

// fakeKinds: the fake corpus with the kinds of the exclamations.
type fakeKinds struct{ fakeCorpus }

func (c fakeKinds) Load(ctx context.Context, lang string, ids []string) (map[string]corpus.Text, error) {
	out, err := c.fakeCorpus.Load(ctx, lang, ids)
	for id, t := range out {
		if strings.HasPrefix(id, "sl.exclamation.") || strings.HasPrefix(id, "sl.dismissal.") {
			t.Kind = "exclamation"
			out[id] = t
		}
	}
	return out, err
}
