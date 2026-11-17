package liturgy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// TempleKind: to whom the church is dedicated.
type TempleKind string

const (
	TempleLord      TempleKind = "lord"      // a feast of the Lord
	TempleTheotokos TempleKind = "theotokos" // the Theotokos
	TempleSaint     TempleKind = "saint"     // a saint (or the bodiless powers)
)

// Temple is the dedication of the church where the service is held: its
// troparion and kontakion (after the entrance at the liturgy) and the day
// of its feast.
type Temple struct {
	ID         string     `yaml:"id" json:"id"`
	Title      string     `yaml:"title" json:"title"`
	Kind       TempleKind `yaml:"kind" json:"kind"`
	Date       string     `yaml:"date" json:"date"` // Julian MM-DD of the patronal feast
	Troparion  string     `yaml:"troparion" json:"troparion"`
	Kontakion  string     `yaml:"kontakion" json:"kontakion"`
	month, day int
}

// Temples is the list of dedications to choose from (data/calendar/temples.yaml).
type Temples struct{ List []*Temple }

// LoadTemples reads data/calendar/temples.yaml.
func LoadTemples(dataDir string) (*Temples, error) {
	path := filepath.Join(dataDir, "calendar", "temples.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Temples []*Temple `yaml:"temples"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	seen := map[string]bool{}
	for _, t := range doc.Temples {
		if seen[t.ID] {
			return nil, fmt.Errorf("%s: duplicate temple %q", path, t.ID)
		}
		seen[t.ID] = true
		if err := t.check(); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, t.ID, err)
		}
	}
	return &Temples{List: doc.Temples}, nil
}

func (t *Temple) check() error {
	switch t.Kind {
	case TempleLord, TempleTheotokos, TempleSaint:
	default:
		return fmt.Errorf("kind %q (lord, theotokos, saint)", t.Kind)
	}
	if t.Troparion == "" || t.Kontakion == "" {
		return fmt.Errorf("troparion and kontakion are required")
	}
	md, err := parseMD(t.Date)
	if err != nil {
		return err
	}
	t.month, t.day = int(md.Month), md.Day
	return nil
}

var reMenaionTemple = regexp.MustCompile(`^me\.(\d\d)\.(\d\d)(p?)$`)

// Find resolves a -temple value: an id of the list, or a Menaion day
// me.MM.DD[p] (a saint's church: the day's first troparion and kontakion).
// "" means no dedication given.
func (ts *Temples) Find(id string) (*Temple, error) {
	if id == "" {
		return nil, nil
	}
	if ts != nil {
		for _, t := range ts.List {
			if t.ID == id {
				return t, nil
			}
		}
	}
	if m := reMenaionTemple.FindStringSubmatch(id); m != nil {
		t := &Temple{ID: id, Title: id, Kind: TempleSaint, Date: m[1] + "-" + m[2],
			Troparion: id + ".troparion.1", Kontakion: id + ".kontakion.1"}
		if err := t.check(); err != nil {
			return nil, fmt.Errorf("temple %q: %w", id, err)
		}
		return t, nil
	}
	return nil, fmt.Errorf("unknown temple %q (see data/calendar/temples.yaml, or me.MM.DD)", id)
}

// IsFeastDay: the patronal feast falls on Menaion day (month, day).
func (t *Temple) IsFeastDay(month, day int) bool {
	return t != nil && t.month == month && t.day == day
}
