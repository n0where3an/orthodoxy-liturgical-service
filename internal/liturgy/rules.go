package liturgy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cel.dev/cel-go/cel"
	"gopkg.in/yaml.v3"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// Rule fills slots of a service when its condition holds. Rules are data.
type Rule struct {
	ID          string                  `yaml:"id"`
	Service     string                  `yaml:"service"`
	Priority    int                     `yaml:"priority"`
	When        string                  `yaml:"when"`
	Unsupported string                  `yaml:"unsupported"` // the case is recognised but not implemented yet
	Slots       map[string]SlotSelector `yaml:"slots"`

	prg cel.Program
}

// SlotSelector says where the texts of a slot come from.
type SlotSelector struct {
	From string `yaml:"from"` // menaion | unit
	Kind string `yaml:"kind"` // troparion, kontakion, ...
	// Alternate: the position of this hour among 1, 3, 6, 9 (1..4). With n
	// texts available, hour k takes text ((k-1) mod n) + 1 (Typikon: troparia
	// and kontakia at the hours are said alternately).
	Alternate int      `yaml:"alternate"`
	Refs      []string `yaml:"refs"` // for from: unit
	Repeat    int      `yaml:"repeat"`
}

// Rules is a loaded, compiled rule set.
type Rules struct {
	list []*Rule
}

// dayEnv declares the variables rules may use.
func dayEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("weekday", cel.IntType), // 0 = Sunday
		cel.Variable("period", cel.StringType),
		cel.Variable("pascha_offset", cel.IntType),
		cel.Variable("bright_week", cel.BoolType),
		cel.Variable("tone", cel.IntType),
		cel.Variable("month", cel.IntType),
		cel.Variable("day", cel.IntType),
		cel.Variable("menaion_troparia", cel.IntType),
		cel.Variable("menaion_kontakia", cel.IntType),
	)
}

// LoadRules reads every YAML file in data/rules.
func LoadRules(dataDir string) (*Rules, error) {
	env, err := dayEnv()
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dataDir, "rules", "*.yaml"))
	if err != nil {
		return nil, err
	}
	rs := &Rules{}
	ids := map[string]bool{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var list []*Rule
		if err := yaml.Unmarshal(b, &list); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		for _, r := range list {
			if ids[r.ID] {
				return nil, fmt.Errorf("%s: duplicate rule id %q", f, r.ID)
			}
			ids[r.ID] = true
			when := r.When
			if strings.TrimSpace(when) == "" {
				when = "true"
			}
			ast, iss := env.Compile(when)
			if iss != nil && iss.Err() != nil {
				return nil, fmt.Errorf("%s: rule %s: %w", f, r.ID, iss.Err())
			}
			if ast.OutputType() != cel.BoolType {
				return nil, fmt.Errorf("%s: rule %s: condition is not boolean", f, r.ID)
			}
			if r.prg, err = env.Program(ast); err != nil {
				return nil, err
			}
			rs.list = append(rs.list, r)
		}
	}
	sort.SliceStable(rs.list, func(i, j int) bool { return rs.list[i].Priority > rs.list[j].Priority })
	return rs, nil
}

// Facts are the day context plus corpus availability, as seen by rules.
type Facts struct {
	Day     calendar.DayContext
	Menaion map[string][]string // kind -> unit ids of the day, in order
}

func (f Facts) vars() map[string]any {
	return map[string]any{
		"weekday":          int64(f.Day.Weekday),
		"period":           f.Day.Period.String(),
		"pascha_offset":    int64(f.Day.PaschaOffset),
		"bright_week":      f.Day.BrightWeek,
		"tone":             int64(f.Day.Tone),
		"month":            int64(f.Day.Menaion.Month),
		"day":              int64(f.Day.Menaion.Day),
		"menaion_troparia": int64(len(f.Menaion["troparion"])),
		"menaion_kontakia": int64(len(f.Menaion["kontakion"])),
	}
}

// Match returns the highest-priority rule of the service whose condition holds.
func (rs *Rules) Match(service string, f Facts) (*Rule, error) {
	vars := f.vars()
	for _, r := range rs.list {
		if r.Service != service {
			continue
		}
		out, _, err := r.prg.Eval(vars)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.ID, err)
		}
		if ok, _ := out.Value().(bool); ok {
			return r, nil
		}
	}
	return nil, fmt.Errorf("no rule for %s on %v", service, f.Day.Date)
}

// MenaionLister lists unit ids by prefix (implemented by corpus.Store).
type MenaionLister interface {
	ListIDs(ctx context.Context, prefix string) ([]string, error)
}

// MenaionFacts collects the Menaion troparia/kontakia of the day. Services of
// a day are files "DD" and "DDp" etc.; the main one comes first.
func MenaionFacts(ctx context.Context, l MenaionLister, md calendar.MonthDay) (map[string][]string, error) {
	prefix := fmt.Sprintf("me.%02d.%02d", int(md.Month), md.Day)
	ids, err := l.ListIDs(ctx, prefix)
	if err != nil {
		return nil, err
	}
	type item struct {
		service string
		kind    string
		n       int
		id      string
	}
	var items []item
	for _, id := range ids {
		// me.MM.DD[suffix].kind.N
		parts := strings.Split(id, ".")
		if len(parts) != 5 {
			continue
		}
		var n int
		fmt.Sscanf(parts[4], "%d", &n)
		items = append(items, item{parts[2], parts[3], n, id})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].service != items[j].service {
			return len(items[i].service) < len(items[j].service) ||
				(len(items[i].service) == len(items[j].service) && items[i].service < items[j].service)
		}
		return items[i].n < items[j].n
	})
	out := map[string][]string{}
	for _, it := range items {
		out[it.kind] = append(out[it.kind], it.id)
	}
	return out, nil
}

// Fill resolves the rule's slot selectors into slot items.
func (r *Rule) Fill(f Facts) (map[string][]SlotItem, error) {
	slots := map[string][]SlotItem{}
	for name, sel := range r.Slots {
		switch sel.From {
		case "menaion":
			avail := f.Menaion[sel.Kind]
			if len(avail) == 0 {
				return nil, fmt.Errorf("slot %s: no Menaion %s for %02d.%02d in the corpus",
					name, sel.Kind, int(f.Day.Menaion.Month), f.Day.Menaion.Day)
			}
			k := sel.Alternate
			if k < 1 {
				k = 1
			}
			slots[name] = []SlotItem{{Ref: avail[(k-1)%len(avail)], Repeat: sel.Repeat}}
		case "unit":
			for _, ref := range sel.Refs {
				slots[name] = append(slots[name], SlotItem{Ref: ref, Repeat: sel.Repeat})
			}
		default:
			return nil, fmt.Errorf("slot %s: unknown source %q", name, sel.From)
		}
	}
	return slots, nil
}

// weekdayName is used in error messages.
func weekdayName(w time.Weekday) string { return w.String() }
