// Package liturgy turns a day context into a resolved service: rules produce
// a plan (slot -> text unit references), the plan fills a template, the
// resulting tree is filtered by a display mode and rendered.
package liturgy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Performer of a node.
type Performer string

const (
	Priest Performer = "priest"
	Deacon Performer = "deacon"
	Reader Performer = "reader"
	Choir  Performer = "choir"
	People Performer = "people"
)

// Voice in which a node is said.
type Voice string

const (
	Aloud  Voice = "aloud"
	Quiet  Voice = "quiet"
	Secret Voice = "secret"
)

// Audience of a rubric.
type Audience string

const (
	AudienceAll    Audience = ""       // shown to everybody
	AudienceKliros Audience = "kliros" // instructions for readers and singers
	AudienceLaity  Audience = "laity"  // explanations for the faithful
)

// Node is a template node. Exactly one of Ref, Slot, Rubric or Children is set.
type Node struct {
	ID        string    `yaml:"id"`
	Ref       string    `yaml:"ref"`    // text unit key
	Slot      string    `yaml:"slot"`   // named slot filled by the plan
	Rubric    string    `yaml:"rubric"` // rubric key (see i18n)
	Audience  Audience  `yaml:"audience"`
	Performer Performer `yaml:"performer"`
	Voice     Voice     `yaml:"voice"`
	Title     string    `yaml:"title"`  // key of the prayer's or group's title
	During    string    `yaml:"during"` // id of the node this one is said during
	Repeat    int       `yaml:"repeat"`
	When      string    `yaml:"when"` // CEL over service params
	Heading   bool      `yaml:"heading"`
	Children  []Node    `yaml:"children"`
}

// Template is the unchanging order of a service with named slots.
type Template struct {
	Service string `yaml:"service"`
	Nodes   []Node `yaml:"nodes"`
}

// LoadTemplate reads data/templates/<service>.yaml.
func LoadTemplate(dataDir, service string) (*Template, error) {
	path := filepath.Join(dataDir, "templates", service+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Template
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &t, nil
}

func (t *Template) validate() error {
	ids := map[string]bool{}
	var walk func([]Node) error
	walk = func(ns []Node) error {
		for _, n := range ns {
			set := 0
			for _, s := range []string{n.Ref, n.Slot, n.Rubric} {
				if s != "" {
					set++
				}
			}
			if len(n.Children) > 0 {
				set++
			}
			if set != 1 {
				return fmt.Errorf("node %q: exactly one of ref, slot, rubric, children must be set", n.ID)
			}
			if n.ID != "" {
				if ids[n.ID] {
					return fmt.Errorf("duplicate node id %q", n.ID)
				}
				ids[n.ID] = true
			}
			if err := walk(n.Children); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(t.Nodes)
}

// Refs returns all fixed unit keys used by the template.
func (t *Template) Refs() []string {
	var out []string
	var walk func([]Node)
	walk = func(ns []Node) {
		for _, n := range ns {
			if n.Ref != "" {
				out = append(out, n.Ref)
			}
			walk(n.Children)
		}
	}
	walk(t.Nodes)
	return out
}
