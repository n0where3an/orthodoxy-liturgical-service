package liturgy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cel.dev/cel-go/cel"
	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Kind of a resolved node.
type Kind string

const (
	KindText    Kind = "text"
	KindRubric  Kind = "rubric"
	KindHeading Kind = "heading"
)

// RNode is a resolved node of the expanded service tree (flattened).
type RNode struct {
	ID        string
	Kind      Kind
	Performer Performer
	Voice     Voice
	Audience  Audience
	Title     string
	During    string
	UnitID    string
	Body      string
	Incipit   string
	Repeat    int
	// DuringTitle: the title key of the node this one is said during.
	DuringTitle string
	// Lang of the text (differs from the requested one on fallback).
	Lang string
}

// Lang holds the localised strings of one language. Strings in the YAML
// file are written in HIP for Church Slavonic and converted on load.
type Lang struct {
	Code string `yaml:"lang"`
	Name string `yaml:"name"`
	// Fallback: languages to take a text from when it is not translated.
	Fallback   []string             `yaml:"fallback"`
	HIP        bool                 `yaml:"hip"`
	Performers map[Performer]string `yaml:"performers"`
	Repeat     map[int]string       `yaml:"repeat"`
	Rubrics    map[string]string    `yaml:"rubrics"`
	UI         map[string]string    `yaml:"ui"`
	// Titles of prayers and parts (HIP, kept raw), and placeholder rubrics
	// with "TITLE" / "DURING" to substitute before conversion.
	Titles       map[string]string `yaml:"titles"`
	Placeholders map[string]string `yaml:"placeholders"`
}

// Placeholder builds the rubric standing for secret prayers.
func (l *Lang) Placeholder(title, during string) string {
	tpl, t := l.Placeholders["generic"], l.Titles[title]
	if d := l.Titles[during]; d != "" && l.Placeholders["during"] != "" {
		tpl = strings.ReplaceAll(l.Placeholders["during"], "DURING", d)
	}
	if t == "" {
		t = l.Titles["secret_prayers"]
	}
	s := strings.ReplaceAll(tpl, "TITLE", t)
	if l.HIP {
		return hip.ToUnicode(s)
	}
	return s
}

// LoadLang reads data/i18n/<code>.yaml.
func LoadLang(dataDir, code string) (*Lang, error) {
	path := filepath.Join(dataDir, "i18n", code+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var l Lang
	if err := yaml.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if l.HIP {
		conv := func(m map[string]string) {
			for k, v := range m {
				m[k] = hip.ToUnicode(v)
			}
		}
		conv(l.Rubrics)
		for k, v := range l.Performers {
			l.Performers[k] = hip.ToUnicode(v)
		}
		for k, v := range l.Repeat {
			l.Repeat[k] = hip.ToUnicode(v)
		}
	}
	return &l, nil
}

// RepeatLabel returns the rubric for a repetition count ("три́жды", "м҃").
func (l *Lang) RepeatLabel(n int) string {
	if s, ok := l.Repeat[n]; ok {
		return s
	}
	return "×" + strconv.Itoa(n)
}

func paramEnv() (*cel.Env, error) {
	return cel.NewEnv(cel.Variable("priest", cel.BoolType), cel.Variable("standalone", cel.BoolType),
		cel.Variable("typikon", cel.BoolType), cel.Variable("deacon", cel.BoolType))
}

// Expand fills the template with the plan's slots and the loaded texts.
func Expand(t *Template, plan *Plan, texts map[string]corpus.Text, lang *Lang) ([]RNode, error) {
	env, err := paramEnv()
	if err != nil {
		return nil, err
	}
	vars := map[string]any{"priest": plan.Params.Priest, "standalone": plan.Params.Standalone, "typikon": plan.Params.Typikon,
		"deacon": plan.Params.Deacon}
	var out []RNode
	missing := map[string]bool{}
	titles := map[string]string{}
	var index func([]Node)
	index = func(ns []Node) {
		for _, n := range ns {
			if n.ID != "" {
				titles[n.ID] = n.Title
			}
			index(n.Children)
		}
	}
	index(t.Nodes)
	performer := func(p Performer) Performer {
		if p == Litanist {
			if plan.Params.Deacon {
				return Deacon
			}
			return Priest
		}
		return p
	}

	text := func(n Node, ref string, repeat int, perf Performer) {
		tx, ok := texts[ref]
		if !ok {
			missing[ref] = true
			return
		}
		kind := KindText
		if n.Heading {
			kind = KindHeading
		}
		if repeat == 0 {
			repeat = n.Repeat
		}
		if perf == "" {
			perf = n.Performer
		}
		out = append(out, RNode{
			ID: n.ID, Kind: kind, Performer: performer(perf), Voice: n.Voice, Title: n.Title,
			During: n.During, DuringTitle: titles[n.During], UnitID: ref, Body: tx.Body, Incipit: tx.Incipit, Repeat: repeat,
			Lang: tx.Lang,
		})
	}

	var walk func([]Node) error
	walk = func(ns []Node) error {
		for _, n := range ns {
			if n.When != "" {
				ok, err := evalBool(env, n.When, vars)
				if err != nil {
					return fmt.Errorf("node %q: %w", n.ID, err)
				}
				if !ok {
					continue
				}
			}
			switch {
			case n.Ref != "":
				text(n, n.Ref, 0, "")
			case n.Slot != "":
				items, ok := plan.Slots[n.Slot]
				if !ok {
					return fmt.Errorf("plan has no slot %q", n.Slot)
				}
				if n.Index > 0 {
					if n.Index > len(items) {
						continue
					}
					items = items[n.Index-1 : n.Index]
				}
				for _, it := range items {
					text(n, it.Ref, it.Repeat, it.Performer)
				}
			case n.Rubric != "":
				s, ok := lang.Rubrics[n.Rubric]
				if !ok {
					return fmt.Errorf("no rubric %q for language %s", n.Rubric, lang.Code)
				}
				out = append(out, RNode{ID: n.ID, Kind: KindRubric, Audience: n.Audience, Body: s, Performer: performer(n.Performer)})
			default:
				if n.IfSlot != "" && len(plan.Slots[n.IfSlot]) == 0 {
					continue
				}
				if err := walk(n.Children); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(t.Nodes); err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		var ids []string
		for id := range missing {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return out, fmt.Errorf("texts missing in the corpus (%s): %v", lang.Code, ids)
	}
	return out, nil
}

func evalBool(env *cel.Env, expr string, vars map[string]any) (bool, error) {
	ast, iss := env.Compile(expr)
	if iss != nil && iss.Err() != nil {
		return false, iss.Err()
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, err
	}
	v, _, err := prg.Eval(vars)
	if err != nil {
		return false, err
	}
	b, ok := v.Value().(bool)
	if !ok {
		return false, fmt.Errorf("condition %q is not boolean", expr)
	}
	return b, nil
}
