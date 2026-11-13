package orthlib

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// manualFile is a file of texts typed in HIP (data/texts/*.yaml).
type manualFile struct {
	Source struct {
		ID, Title, License, Note string
	} `yaml:"source"`
	Units []struct {
		Key  string `yaml:"key"`
		Book string `yaml:"book"`
		Kind string `yaml:"kind"`
		Tone int    `yaml:"tone"`
		HIP  string `yaml:"hip"`
	} `yaml:"units"`
}

// ImportManual reads the texts typed in for the project.
func ImportManual(dir string) ([]corpus.Source, []corpus.Unit, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, nil, err
	}
	var srcs []corpus.Source
	var units []corpus.Unit
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, err
		}
		var m manualFile
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", f, err)
		}
		if m.Source.ID == "" {
			return nil, nil, fmt.Errorf("%s: no source id", f)
		}
		srcs = append(srcs, corpus.Source{ID: m.Source.ID, Title: m.Source.Title, License: m.Source.License, Note: m.Source.Note})
		for _, u := range m.Units {
			units = append(units, corpus.Unit{
				ID: u.Key, Book: u.Book, Kind: u.Kind, Tone: u.Tone,
				Lang: "cu", Body: hip.ToUnicode(u.HIP), SourceID: m.Source.ID,
			})
		}
	}
	return srcs, units, nil
}
