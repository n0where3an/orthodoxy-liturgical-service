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
	lang, err := LoadLang(g.DataDir, r.Lang)
	if err != nil {
		return plan, err
	}
	mode, err := ModeByName(r.Mode)
	if err != nil {
		return plan, err
	}
	ids := t.Refs()
	for _, items := range plan.Slots {
		for _, it := range items {
			ids = append(ids, it.Ref)
		}
	}
	texts, err := g.Texts.Load(ctx, lang.Code, ids)
	if err != nil {
		return plan, err
	}
	nodes, err := Expand(t, plan, texts, lang)
	if err != nil {
		return plan, err
	}
	return plan, Render(w, BuildView(nodes, plan, mode, lang))
}
