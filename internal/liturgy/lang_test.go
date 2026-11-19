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

// halfEnglish has every text in cu, and in en only the psalms.
type halfEnglish struct{ fakeCorpus }

func (halfEnglish) Load(_ context.Context, lang string, ids []string) (map[string]corpus.Text, error) {
	out := map[string]corpus.Text{}
	for _, id := range ids {
		if lang == "en" && !strings.Contains(id, "psalm") {
			continue
		}
		out[id] = corpus.Text{UnitID: id, Lang: lang, Body: strings.ToUpper(lang) + " " + id, Incipit: id}
	}
	return out, nil
}

func TestFallbackAndParallel(t *testing.T) {
	rules, _ := LoadRules(dataDir())
	g := &Generator{DataDir: dataDir(), Planner: &Planner{Rules: rules, Menaion: fakeCorpus{}}, Texts: halfEnglish{}}
	gen := func(lang, lang2 string) string {
		var buf bytes.Buffer
		if _, err := g.Generate(context.Background(), Request{Date: calendar.MustJDate(2026, time.October, 30),
			Service: "ninth_hour", Params: Params{Priest: true}, Lang: lang, Lang2: lang2, Mode: "laity"}, &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	en := gen("en", "")
	if !strings.Contains(en, "EN ch.none.psalm.83") {
		t.Error("English psalm missing")
	}
	// The Church Slavonic block is marked and not offered for machine translation.
	if !strings.Contains(en, `fallback notranslate" lang="cu" translate="no"><p>`) || !strings.Contains(en, "CU common.trisagion") {
		t.Error("untranslated texts must fall back to Church Slavonic, marked")
	}
	if strings.Contains(en, `<html lang="en" translate="no"`) {
		t.Error("an English page may be translated")
	}
	if cu := gen("cu", ""); !strings.Contains(cu, `<html lang="cu" translate="no" class="notranslate">`) ||
		!strings.Contains(cu, `<meta name="google" content="notranslate">`) {
		t.Error("a Church Slavonic page must not be offered for translation")
	}
	if !strings.Contains(en, "Reader:") {
		t.Error("English performer labels")
	}
	par := gen("cu", "en")
	if strings.Count(par, `<div class="row">`) < 40 {
		t.Error("parallel page has too few rows")
	}
	if !strings.Contains(par, "CU ch.none.psalm.83") || !strings.Contains(par, "EN ch.none.psalm.83") {
		t.Error("both languages expected side by side")
	}
}
