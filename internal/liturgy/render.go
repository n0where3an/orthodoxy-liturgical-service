package liturgy

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// View is the data passed to the HTML template.
type View struct {
	Lang  string
	Title string
	Mode  string
	Plan  *Plan
	Items []ViewItem
	UI    map[string]string
}

// ViewItem is one rendered block (a text or a rubric).
type ViewItem struct {
	Lang   string // set when the text is in a fallback language
	Class  string
	Who    string   // performer label, cinnabar
	Paras  []string // paragraphs of the text
	Repeat string   // repetition rubric, cinnabar, after the last paragraph
	Cue    bool
}

// BuildView applies the mode to the expanded nodes.
func BuildView(nodes []RNode, plan *Plan, mode Mode, lang *Lang) View {
	v := View{Lang: lang.Code, Mode: mode.Name(), Plan: plan, UI: lang.UI}
	prevWho := Performer("")
	for i := 0; i < len(nodes); i++ {
		n := &nodes[i]
		d := mode.Detail(n)
		if d == Omit {
			continue
		}
		if d == Placeholder {
			// Consecutive secret prayers of one group collapse into one rubric.
			for i+1 < len(nodes) && nodes[i+1].Voice == Secret && nodes[i+1].Title == n.Title {
				i++
			}
			v.Items = append(v.Items, ViewItem{Class: "rubric placeholder", Paras: []string{lang.Placeholder(n.Title, n.DuringTitle)}})
			continue
		}
		switch n.Kind {
		case KindHeading:
			if v.Title == "" {
				v.Title = n.Body
			}
			continue
		case KindRubric:
			v.Items = append(v.Items, ViewItem{Class: "rubric " + string(n.Audience), Paras: []string{n.Body}})
			continue
		}
		it := ViewItem{Class: "text " + string(n.Performer)}
		if n.Lang != "" && n.Lang != lang.Code {
			it.Lang = n.Lang
			it.Class += " fallback"
		}
		// Name the performer when it changes, and always for clergy.
		if n.Performer != "" && (n.Performer != prevWho || n.Performer == Priest || n.Performer == Deacon) {
			it.Who = lang.Performers[n.Performer]
		}
		prevWho = n.Performer
		switch d {
		case Cue:
			it.Cue = true
			it.Paras = []string{n.Incipit + "…"}
		case Heading:
			it.Paras = []string{n.Title}
		default:
			it.Paras = strings.Split(n.Body, "\n")
		}
		if n.Repeat > 1 {
			it.Repeat = lang.RepeatLabel(n.Repeat)
		}
		v.Items = append(v.Items, it)
	}
	return v
}

// Render writes the HTML page.
func Render(w io.Writer, v View) error {
	return pageTmpl.Execute(w, v)
}

// ParallelView shows the same service in two languages, block by block.
type ParallelView struct {
	Left, Right View
	Rows        []ParallelRow
}

// ParallelRow is one block in both languages.
type ParallelRow struct{ Left, Right ViewItem }

// NewParallelView zips two views built from the same plan and mode.
func NewParallelView(left, right View) (ParallelView, error) {
	if len(left.Items) != len(right.Items) {
		return ParallelView{}, fmt.Errorf("parallel views differ: %d and %d blocks", len(left.Items), len(right.Items))
	}
	pv := ParallelView{Left: left, Right: right}
	for i := range left.Items {
		pv.Rows = append(pv.Rows, ParallelRow{left.Items[i], right.Items[i]})
	}
	return pv, nil
}

// RenderParallel writes the two-language HTML page.
func RenderParallel(w io.Writer, pv ParallelView) error {
	return parallelTmpl.Execute(w, pv)
}

const pageStyle = `
:root { --ink: #1b1b1b; --cinnabar: #b3261e; --muted: #6b6b6b; --bg: #fffdf8; --line: #e6dfd2; }
@media (prefers-color-scheme: dark) { :root { --ink: #ece7dc; --cinnabar: #ff7a6b; --muted: #a59f94; --bg: #1a1814; --line: #3a352d; } }
body { background: var(--bg); color: var(--ink); margin: 0 auto; max-width: 46rem; padding: 1.5rem 1rem 4rem;
  font: 1.25rem/1.55 "Ponomar Unicode", "Monomakh Unicode", "Triodion Unicode", "Menaion Unicode", serif; }
body.parallel { max-width: 92rem; }
header { font-family: system-ui, sans-serif; font-size: .85rem; color: var(--muted); margin-bottom: 1.5rem; }
h1 { color: var(--cinnabar); text-align: center; font-weight: normal; font-size: 2rem; margin: .5rem 0 1rem; }
.item { margin: .55rem 0; }
.item p { margin: 0 0 .35rem; }
.who, .repeat, .rubric { color: var(--cinnabar); }
.who { margin-right: .35em; }
.repeat { margin-left: .35em; }
.cue { color: var(--muted); }
.fallback { border-left: 3px solid var(--line); padding-left: .5rem; }
.fallback-note { font-family: system-ui, sans-serif; font-size: .7rem; color: var(--muted); }
:lang(en) { font-family: Georgia, "Times New Roman", serif; }
.row { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; border-top: 1px solid var(--line); }
@media (max-width: 48rem) { .row { grid-template-columns: 1fr; gap: .25rem; } }
`

// itemTmpl renders one block.
const itemTmpl = `{{define "item"}}<div class="item {{.Class}}{{if .Cue}} cue{{end}}"{{if .Lang}} lang="{{.Lang}}"{{end}}>
{{- $it := .}}{{range $i, $p := .Paras}}<p>
{{- if and (eq $i 0) $it.Who}}<span class="who">{{$it.Who}}</span>{{end}}{{$p}}
{{- if and $it.Repeat (last $i (len $it.Paras))}}<span class="repeat">{{$it.Repeat}}</span>{{end}}</p>{{end}}
{{- if .Lang}}<div class="fallback-note">{{.Lang}}</div>{{end}}</div>{{end}}`

var funcs = template.FuncMap{"last": func(i, n int) bool { return i == n-1 }}

var parallelTmpl = template.Must(template.New("parallel").Funcs(funcs).Parse(itemTmpl + `<!doctype html>
<html lang="{{.Left.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Left.Title}} / {{.Right.Title}} — {{.Left.Plan.Civil}}</title>
<style>` + pageStyle + `</style>
</head>
<body class="parallel">
<header>{{.Left.Plan.Civil}} · {{.Left.UI.julian}} {{.Left.Plan.Date}} · {{.Left.UI.tone}} {{.Left.Plan.Day.Tone}} · {{.Left.UI.mode}}: {{.Left.Mode}} · {{.Left.UI.rule}}: {{.Left.Plan.Rule}}</header>
<div class="row"><h1 lang="{{.Left.Lang}}">{{.Left.Title}}</h1><h1 lang="{{.Right.Lang}}">{{.Right.Title}}</h1></div>
{{range .Rows}}<div class="row"><div lang="{{$.Left.Lang}}">{{template "item" .Left}}</div><div lang="{{$.Right.Lang}}">{{template "item" .Right}}</div></div>
{{end}}</body>
</html>
`))

var pageTmpl = template.Must(template.New("page").Funcs(funcs).Parse(itemTmpl + `<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — {{.Plan.Civil}}</title>
<style>` + pageStyle + `</style>
</head>
<body>
<header>{{.Plan.Civil}} · {{.UI.julian}} {{.Plan.Date}} · {{.UI.tone}} {{.Plan.Day.Tone}} · {{.UI.mode}}: {{.Mode}} · {{.UI.rule}}: {{.Plan.Rule}}</header>
<h1>{{.Title}}</h1>
{{range .Items}}{{template "item" .}}
{{end}}</body>
</html>
`))
