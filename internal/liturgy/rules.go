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
//
// A service rule (no Slot) is chosen once per service: the highest-priority
// one whose condition holds. A slot rule (Slot set) fills one slot that the
// service rule left open, independently of which service rule was chosen —
// for texts that depend on the season rather than on the day's rank (e.g.
// the kontakion after the first hour, «Христос воскресе» before the hours).
type Rule struct {
	ID          string                  `yaml:"id"`
	Slot        string                  `yaml:"slot"`     // slot rule: the slot it fills
	With        SlotSelector            `yaml:"fill"`     // slot rule: how to fill it
	Service     string                  `yaml:"service"`  // one service, or
	Services    []string                `yaml:"services"` // several (e.g. all the hours)
	Priority    int                     `yaml:"priority"`
	When        string                  `yaml:"when"`
	Unsupported string                  `yaml:"unsupported"` // the case is recognised but not implemented yet
	Slots       map[string]SlotSelector `yaml:"slots"`

	prg cel.Program
}

// SlotSelector says where the texts of a slot come from.
type SlotSelector struct {
	From string `yaml:"from"` // menaion | unit | feast | feast_and_menaion | none
	Kind string `yaml:"kind"` // troparion, kontakion, ...
	// AlternateByHour: troparia and kontakia at the hours are said
	// alternately (Typikon). With n texts available, hour k (1st, 3rd, 6th,
	// 9th = 1..4) takes text ((k-1) mod n) + 1.
	AlternateByHour bool     `yaml:"alternate_by_hour"`
	Refs            []string `yaml:"refs"` // for from: unit
	Part            string   `yaml:"part"` // for from: feast(_and_menaion) — troparion | kontakion
	Repeat          int      `yaml:"repeat"`
}

// Rules is a loaded, compiled rule set.
type Rules struct {
	list      []*Rule // service rules, by priority
	slotRules []*Rule // slot rules, by priority
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
		cel.Variable("feast", cel.StringType), // twelve-feast id from the feast to its apodosis, or ""
		cel.Variable("feast_day", cel.BoolType),
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
			if r.Slot != "" {
				rs.slotRules = append(rs.slotRules, r)
			} else {
				rs.list = append(rs.list, r)
			}
		}
	}
	byPriority := func(l []*Rule) func(i, j int) bool {
		return func(i, j int) bool { return l[i].Priority > l[j].Priority }
	}
	sort.SliceStable(rs.list, byPriority(rs.list))
	sort.SliceStable(rs.slotRules, byPriority(rs.slotRules))
	return rs, nil
}

// Facts are the day context plus corpus availability, as seen by rules.
type Facts struct {
	Day     calendar.DayContext
	Menaion map[string][]string // kind -> unit ids of the day, in order
	Feast   *Feast              // twelve feast whose period (feast..apodosis) contains the day
}

func (f Facts) vars() map[string]any {
	feast, feastDay := "", false
	if f.Feast != nil {
		feast, feastDay = f.Feast.ID, f.Feast.IsFeastDay(f.Day)
	}
	return map[string]any{
		"feast":            feast,
		"feast_day":        feastDay,
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
		if !r.appliesTo(service) {
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

func (r *Rule) appliesTo(service string) bool {
	if r.Service == service {
		return true
	}
	for _, s := range r.Services {
		if s == service {
			return true
		}
	}
	return false
}

// HourPosition is the place of an hour in the order of the hours (1st, 3rd,
// 6th, 9th = 1..4); 0 for other services.
func HourPosition(service string) int {
	return map[string]int{"first_hour": 1, "third_hour": 2, "sixth_hour": 3, "ninth_hour": 4}[service]
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

// FillSlotRules fills the slots of service not set yet from slot rules.
func (rs *Rules) FillSlotRules(service string, f Facts, slots map[string][]SlotItem) error {
	vars := f.vars()
	for _, r := range rs.slotRules {
		if !r.appliesTo(service) {
			continue
		}
		if _, done := slots[r.Slot]; done {
			continue
		}
		out, _, err := r.prg.Eval(vars)
		if err != nil {
			return fmt.Errorf("rule %s: %w", r.ID, err)
		}
		if ok, _ := out.Value().(bool); !ok {
			continue
		}
		items, err := r.With.resolve(r.Slot, f, service)
		if err != nil {
			return fmt.Errorf("rule %s: %w", r.ID, err)
		}
		slots[r.Slot] = items
	}
	return nil
}

// Fill resolves the rule's slot selectors into slot items for service.
func (r *Rule) Fill(f Facts, service string) (map[string][]SlotItem, error) {
	slots := map[string][]SlotItem{}
	for name, sel := range r.Slots {
		items, err := sel.resolve(name, f, service)
		if err != nil {
			return nil, err
		}
		slots[name] = items
	}
	return slots, nil
}

func (sel SlotSelector) resolve(name string, f Facts, service string) ([]SlotItem, error) {
	items := []SlotItem{}
	{
		switch sel.From {
		case "menaion":
			avail := f.Menaion[sel.Kind]
			if len(avail) == 0 {
				return nil, fmt.Errorf("slot %s: no Menaion %s for %02d.%02d in the corpus",
					name, sel.Kind, int(f.Day.Menaion.Month), f.Day.Menaion.Day)
			}
			k := 1
			if sel.AlternateByHour && HourPosition(service) > 0 {
				k = HourPosition(service)
			}
			items = append(items, SlotItem{Ref: avail[(k-1)%len(avail)], Repeat: sel.Repeat})
		case "unit":
			for _, ref := range sel.Refs {
				items = append(items, SlotItem{Ref: ref, Repeat: sel.Repeat})
			}
		case "feast", "feast_and_menaion":
			if f.Feast == nil {
				return nil, fmt.Errorf("slot %s: no twelve feast on %v", name, f.Day.Date)
			}
			ref, err := f.Feast.Part(sel.Part)
			if err != nil {
				return nil, fmt.Errorf("slot %s: %w", name, err)
			}
			if sel.From == "feast" {
				items = append(items, SlotItem{Ref: ref, Repeat: sel.Repeat})
				break
			}
			// Afterfeast: the feast's hymn alternates with the saint's.
			avail := []string{ref}
			for _, id := range f.Menaion[sel.Kind] {
				if id != ref {
					avail = append(avail, id)
				}
			}
			k := 1
			if sel.AlternateByHour && HourPosition(service) > 0 {
				k = HourPosition(service)
			}
			items = append(items, SlotItem{Ref: avail[(k-1)%len(avail)], Repeat: sel.Repeat})
		case "none":
		default:
			return nil, fmt.Errorf("slot %s: unknown source %q", name, sel.From)
		}
	}
	return items, nil
}

// weekdayName is used in error messages.
func weekdayName(w time.Weekday) string { return w.String() }
