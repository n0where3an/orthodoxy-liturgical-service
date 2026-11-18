package liturgy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// Katavasia is the seasonal katavasia of Sunday matins
// (data/calendar/katavasia.yaml): the irmoi of a feast's canon in its
// period, «Отверзу уста моя» otherwise.
type Katavasia struct {
	Default map[int]string    `yaml:"default"`
	Periods []katavasiaPeriod `yaml:"periods"`
}

type katavasiaPeriod struct {
	From, To string
	Canon    string // unit prefix of the canon: me.MM.DD.matins.canon.N
	Title    string
	from, to calendar.MonthDay
}

// katavasiaOdes are the odes of a canon (the 2nd is not sung).
var katavasiaOdes = []int{1, 3, 4, 5, 6, 7, 8, 9}

// LoadKatavasia reads data/calendar/katavasia.yaml.
func LoadKatavasia(dataDir string) (*Katavasia, error) {
	path := filepath.Join(dataDir, "calendar", "katavasia.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var k Katavasia
	if err := yaml.Unmarshal(b, &k); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, o := range katavasiaOdes {
		if k.Default[o] == "" {
			return nil, fmt.Errorf("%s: no default irmos of ode %d", path, o)
		}
	}
	for i := range k.Periods {
		p := &k.Periods[i]
		if p.from, err = parseMD(p.From); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if p.to, err = parseMD(p.To); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if !strings.HasPrefix(p.Canon, "me.") {
			return nil, fmt.Errorf("%s: canon %q", path, p.Canon)
		}
	}
	return &k, nil
}

// For returns the irmos to sing as katavasia after each ode (1, 3..9) on
// Menaion day md: the period's canon where the corpus has its irmos, else
// the default.
func (k *Katavasia) For(ctx context.Context, l MenaionLister, md calendar.MonthDay) (map[int]string, error) {
	out, _, err := k.ForSeason(ctx, l, md)
	return out, err
}

// ForSeason is For, and whether a seasonal period (not the default) applies.
func (k *Katavasia) ForSeason(ctx context.Context, l MenaionLister, md calendar.MonthDay) (map[int]string, bool, error) {
	out := map[int]string{}
	if k == nil {
		return out, false, nil
	}
	for o, id := range k.Default {
		out[o] = id
	}
	for _, p := range k.Periods {
		if mdLess(md, p.from) || mdLess(p.to, md) {
			continue
		}
		ids, err := l.ListIDs(ctx, p.Canon+".ode.")
		if err != nil {
			return nil, false, err
		}
		have := map[string]bool{}
		for _, id := range ids {
			have[id] = true
		}
		for _, o := range katavasiaOdes {
			if id := fmt.Sprintf("%s.ode.%d.irmos", p.Canon, o); have[id] {
				out[o] = id
			}
		}
		return out, true, nil
	}
	return out, false, nil
}
