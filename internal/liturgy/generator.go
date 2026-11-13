package liturgy

import (
	"context"
	"io"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

// TextLoader loads translations (implemented by corpus.Store).
type TextLoader interface {
	Load(ctx context.Context, lang string, ids []string) (map[string]corpus.Text, error)
}

// Generator runs the whole pipeline: day context -> plan -> tree -> mode -> HTML.
type Generator struct {
	DataDir string
	Planner *Planner
	Texts   TextLoader
}

// Request selects what to generate.
type Request struct {
	Date    calendar.JDate
	Service string
	Params  Params
	Lang    string
	Lang2   string // optional second language for parallel output
	Mode    string
}

// Generate renders the requested service as HTML.
func (g *Generator) Generate(ctx context.Context, r Request, w io.Writer) (*Plan, error) {
	plan, err := g.Planner.Plan(ctx, r.Date, r.Service, r.Params)
	if err != nil {
		return nil, err
	}
	t, err := LoadTemplate(g.DataDir, r.Service)
	if err != nil {
		return plan, err
	}
	mode, err := ModeByName(r.Mode)
	if err != nil {
		return plan, err
	}
	view := func(code string) (View, error) {
		lang, err := LoadLang(g.DataDir, code)
		if err != nil {
			return View{}, err
		}
		texts, err := g.loadTexts(ctx, t, plan, lang)
		if err != nil {
			return View{}, err
		}
		nodes, err := Expand(t, plan, texts, lang)
		if err != nil {
			return View{}, err
		}
		return BuildView(nodes, plan, mode, lang), nil
	}
	left, err := view(r.Lang)
	if err != nil {
		return plan, err
	}
	if r.Lang2 == "" {
		return plan, Render(w, left)
	}
	right, err := view(r.Lang2)
	if err != nil {
		return plan, err
	}
	pv, err := NewParallelView(left, right)
	if err != nil {
		return plan, err
	}
	return plan, RenderParallel(w, pv)
}

// loadTexts loads the texts of the service in lang, completing missing ones
// from the language's fallbacks (the text keeps its own Lang).
func (g *Generator) loadTexts(ctx context.Context, t *Template, plan *Plan, lang *Lang) (map[string]corpus.Text, error) {
	ids := t.Refs()
	for _, items := range plan.Slots {
		for _, it := range items {
			ids = append(ids, it.Ref)
		}
	}
	texts, err := g.Texts.Load(ctx, lang.Code, ids)
	if err != nil {
		return nil, err
	}
	for _, fb := range lang.Fallback {
		var missing []string
		for _, id := range ids {
			if _, ok := texts[id]; !ok {
				missing = append(missing, id)
			}
		}
		if len(missing) == 0 {
			break
		}
		more, err := g.Texts.Load(ctx, fb, missing)
		if err != nil {
			return nil, err
		}
		for id, tx := range more {
			texts[id] = tx
		}
	}
	return texts, nil
}
