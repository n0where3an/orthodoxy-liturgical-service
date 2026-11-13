package orthlib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

// Manifest lists sources and the fixed text units cut out of orthlib files by
// anchors.
type Manifest struct {
	Sources []struct {
		ID      string `yaml:"id"`
		Title   string `yaml:"title"`
		Edition string `yaml:"edition"`
		URL     string `yaml:"url"`
		License string `yaml:"license"`
		Note    string `yaml:"note"`
	} `yaml:"sources"`
	Units []ManifestUnit `yaml:"units"`
}

// ManifestUnit is one text unit cut from a file.
type ManifestUnit struct {
	Key    string `yaml:"key"`
	Book   string `yaml:"book"`
	Kind   string `yaml:"kind"`
	Tone   int    `yaml:"tone"`
	Source string `yaml:"source"`
	File   string `yaml:"file"`
	After  string `yaml:"after"`
	From   string `yaml:"from"`
	To     string `yaml:"to"`
	Suffix string `yaml:"suffix"` // replaces trailing punctuation
	// DropBrackets removes editorial insertions in square brackets
	// ("[аще в монастыре: о святей обители сей,]").
	DropBrackets bool           `yaml:"drop_brackets"`
	Meta         map[string]any `yaml:"meta"`
}

var reBrackets = regexp.MustCompile(`\s*\[[^\]]*\]`)

// LoadManifest reads a manifest YAML file.
func LoadManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &m, nil
}

// SourceRecords converts manifest sources to corpus records.
func (m *Manifest) SourceRecords() []corpus.Source {
	out := make([]corpus.Source, len(m.Sources))
	for i, s := range m.Sources {
		out[i] = corpus.Source{ID: s.ID, Title: s.Title, Edition: s.Edition, URL: s.URL, License: s.License, Note: s.Note}
	}
	return out
}

// Extract cuts all manifest units from the extracted orthlib tree at root.
func (m *Manifest) Extract(root string) ([]corpus.Unit, error) {
	docs := map[string]*Doc{}
	var units []corpus.Unit
	var errs []error
	seen := map[string]bool{}
	for _, mu := range m.Units {
		if seen[mu.Key] {
			errs = append(errs, fmt.Errorf("duplicate key %s", mu.Key))
			continue
		}
		seen[mu.Key] = true
		d, ok := docs[mu.File]
		if !ok {
			var err error
			d, err = LoadDoc(filepath.Join(root, mu.File))
			if err != nil {
				return nil, err
			}
			docs[mu.File] = d
		}
		body, err := d.Slice(mu.After, mu.From, mu.To)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", mu.Key, err))
			continue
		}
		if mu.DropBrackets {
			body = CleanText(reBrackets.ReplaceAllString(body, ""))
		}
		if mu.Suffix != "" {
			body = strings.TrimRight(body, ",.:;") + mu.Suffix
		}
		// A unit cut from the middle of a sentence still starts a text.
		if r, n := utf8.DecodeRuneInString(body); n > 0 {
			body = string(unicode.ToUpper(r)) + body[n:]
		}
		units = append(units, corpus.Unit{
			ID: mu.Key, Book: mu.Book, Kind: mu.Kind, Tone: mu.Tone, Meta: mu.Meta,
			Lang: "cu", Body: body, SourceID: mu.Source,
		})
	}
	if len(errs) > 0 {
		return units, fmt.Errorf("%d manifest errors, first: %w", len(errs), errs[0])
	}
	return units, nil
}
