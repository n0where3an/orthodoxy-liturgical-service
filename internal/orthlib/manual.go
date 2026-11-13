package orthlib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// manualFile is a file of texts typed in HIP (data/texts/*.yaml).
type manualFile struct {
	// Lang of the texts (default "cu"); for "cu" texts are written in HIP.
	Lang   string `yaml:"lang"`
	Source struct {
		ID, Title, License, Note string
	} `yaml:"source"`
	// Further sources, referenced by units' "source".
	Sources []struct {
		ID, Title, License, Note string
	} `yaml:"sources"`
	Units []struct {
		Key    string `yaml:"key"`
		Book   string `yaml:"book"`
		Kind   string `yaml:"kind"`
		Tone   int    `yaml:"tone"`
		HIP    string `yaml:"hip"`
		Text   string `yaml:"text"`
		Source string `yaml:"source"`
	} `yaml:"units"`
}

// ImportManual reads the texts typed in for the project. Church Slavonic
// units are returned in units; texts in other languages are translations of
// existing units and are returned in translations.
func ImportManual(dir string) ([]corpus.Source, []corpus.Unit, []corpus.Unit, error) {
	var translations []corpus.Unit
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, nil, nil, err
	}
	var srcs []corpus.Source
	var units []corpus.Unit
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, nil, err
		}
		var m manualFile
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", f, err)
		}
		if m.Source.ID == "" {
			return nil, nil, nil, fmt.Errorf("%s: no source id", f)
		}
		lang := m.Lang
		if lang == "" {
			lang = "cu"
		}
		srcs = append(srcs, corpus.Source{ID: m.Source.ID, Title: m.Source.Title, License: m.Source.License, Note: m.Source.Note})
		known := map[string]bool{m.Source.ID: true}
		for _, s := range m.Sources {
			srcs = append(srcs, corpus.Source{ID: s.ID, Title: s.Title, License: s.License, Note: s.Note})
			known[s.ID] = true
		}
		for _, u := range m.Units {
			src := m.Source.ID
			if u.Source != "" {
				if !known[u.Source] {
					return nil, nil, nil, fmt.Errorf("%s: %s: unknown source %q", f, u.Key, u.Source)
				}
				src = u.Source
			}
			body := strings.TrimSpace(u.Text)
			if lang == "cu" {
				body = hip.ToUnicode(u.HIP)
			}
			if body == "" {
				return nil, nil, nil, fmt.Errorf("%s: %s: empty text", f, u.Key)
			}
			unit := corpus.Unit{ID: u.Key, Book: u.Book, Kind: u.Kind, Tone: u.Tone, Lang: lang, Body: body, SourceID: src}
			if lang == "cu" {
				units = append(units, unit)
			} else {
				translations = append(translations, unit)
			}
		}
	}
	return srcs, units, translations, nil
}
