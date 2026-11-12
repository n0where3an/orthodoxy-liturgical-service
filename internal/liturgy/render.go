package liturgy

import (
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
	for i := range nodes {
		n := &nodes[i]
		d := mode.Detail(n)
		if d == Omit {
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

var pageTmpl = template.Must(template.New("page").Funcs(template.FuncMap{
	"last": func(i, n int) bool { return i == n-1 },
}).Parse(`<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — {{.Plan.Civil}}</title>
<style>
:root { --ink: #1b1b1b; --cinnabar: #b3261e; --muted: #6b6b6b; --bg: #fffdf8; }
@media (prefers-color-scheme: dark) { :root { --ink: #ece7dc; --cinnabar: #ff7a6b; --muted: #a59f94; --bg: #1a1814; } }
body { background: var(--bg); color: var(--ink); margin: 0 auto; max-width: 46rem; padding: 1.5rem 1rem 4rem;
  font: 1.25rem/1.55 "Ponomar Unicode", "Monomakh Unicode", "Triodion Unicode", "Menaion Unicode", serif; }
header { font-family: system-ui, sans-serif; font-size: .85rem; color: var(--muted); margin-bottom: 1.5rem; }
h1 { color: var(--cinnabar); text-align: center; font-weight: normal; font-size: 2rem; margin: .5rem 0 1rem; }
.item { margin: .55rem 0; }
.item p { margin: 0 0 .35rem; }
.who, .repeat, .rubric { color: var(--cinnabar); }
.who { margin-right: .35em; }
.repeat { margin-left: .35em; }
.cue { color: var(--muted); }
</style>
</head>
<body>
<header>{{.Plan.Civil}} · {{.UI.julian}} {{.Plan.Date}} · {{.UI.tone}} {{.Plan.Day.Tone}} · {{.UI.mode}}: {{.Mode}} · {{.UI.rule}}: {{.Plan.Rule}}</header>
<h1>{{.Title}}</h1>
{{range .Items}}{{$it := .}}<div class="item {{.Class}}{{if .Cue}} cue{{end}}">
{{- range $i, $p := .Paras}}<p>
{{- if and (eq $i 0) $it.Who}}<span class="who">{{$it.Who}}</span>{{end}}{{$p}}
{{- if and $it.Repeat (last $i (len $it.Paras))}}<span class="repeat">{{$it.Repeat}}</span>{{end}}</p>{{end}}</div>
{{end}}</body>
</html>
`))
