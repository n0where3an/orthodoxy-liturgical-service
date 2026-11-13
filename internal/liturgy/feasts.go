package liturgy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// Feast is a twelve feast with its afterfeast period (feast day..apodosis).
type Feast struct {
	ID        string `yaml:"id"`
	Title     string `yaml:"title"`
	Date      string `yaml:"date"`     // fixed: "MM-DD" (Julian)
	Apodosis  string `yaml:"apodosis"` // fixed: "MM-DD"
	Offset    *int   `yaml:"pascha_offset"`
	ApoOffset *int   `yaml:"apodosis_offset"`
	Troparion string `yaml:"troparion"` // unit key
	Kontakion string `yaml:"kontakion"` // unit key

	from, to calendar.MonthDay
}

// Feasts is the list of twelve feasts.
type Feasts struct{ list []*Feast }

// LoadFeasts reads data/calendar/feasts.yaml.
func LoadFeasts(dataDir string) (*Feasts, error) {
	path := filepath.Join(dataDir, "calendar", "feasts.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Feasts []*Feast `yaml:"feasts"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, f := range doc.Feasts {
		switch {
		case f.Date != "":
			if f.from, err = parseMD(f.Date); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", path, f.ID, err)
			}
			if f.to, err = parseMD(f.Apodosis); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", path, f.ID, err)
			}
			if mdLess(f.to, f.from) {
				return nil, fmt.Errorf("%s: %s: apodosis before the feast", path, f.ID)
			}
		case f.Offset != nil && f.ApoOffset != nil:
		default:
			return nil, fmt.Errorf("%s: %s: needs date/apodosis or pascha_offset/apodosis_offset", path, f.ID)
		}
		if f.Kontakion == "" || f.Troparion == "" {
			return nil, fmt.Errorf("%s: %s: troparion and kontakion are required", path, f.ID)
		}
	}
	return &Feasts{list: doc.Feasts}, nil
}

func parseMD(s string) (calendar.MonthDay, error) {
	var m, d int
	if _, err := fmt.Sscanf(s, "%d-%d", &m, &d); err != nil || m < 1 || m > 12 || d < 1 || d > 31 {
		return calendar.MonthDay{}, fmt.Errorf("bad month-day %q", s)
	}
	return calendar.MonthDay{Month: time.Month(m), Day: d}, nil
}

func mdLess(a, b calendar.MonthDay) bool {
	return a.Month < b.Month || (a.Month == b.Month && a.Day < b.Day)
}

func (f *Feast) contains(day calendar.DayContext) bool {
	if f.Offset != nil {
		return day.PaschaOffset >= *f.Offset && day.PaschaOffset <= *f.ApoOffset
	}
	return !mdLess(day.Menaion, f.from) && !mdLess(f.to, day.Menaion)
}

// IsFeastDay reports whether day is the feast itself (not the afterfeast).
func (f *Feast) IsFeastDay(day calendar.DayContext) bool {
	if f.Offset != nil {
		return day.PaschaOffset == *f.Offset
	}
	return day.Menaion == f.from
}

// At returns the twelve feast whose period contains day, or nil.
func (fs *Feasts) At(day calendar.DayContext) *Feast {
	if fs == nil {
		return nil
	}
	for _, f := range fs.list {
		if f.contains(day) {
			return f
		}
	}
	return nil
}

// Part returns the unit key of a part of the feast's service.
func (f *Feast) Part(part string) (string, error) {
	switch part {
	case "troparion":
		return f.Troparion, nil
	case "kontakion":
		return f.Kontakion, nil
	}
	return "", fmt.Errorf("unknown feast part %q", part)
}

func feastID(f *Feast) string {
	if f == nil {
		return ""
	}
	return f.ID
}
