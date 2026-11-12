package liturgy

import (
	"context"
	"fmt"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// Params are the request options that change the service (not its display).
type Params struct {
	// Priest: served with a priest; otherwise the reader's order
	// ("Молитвами святых отец наших…" instead of exclamations).
	Priest bool `json:"priest"`
	// Standalone: the hour is not followed by the next service, so it ends
	// with a dismissal. In parish practice the ninth hour leads into vespers.
	Standalone bool `json:"standalone"`
}

// SlotItem references one text unit placed into a slot.
type SlotItem struct {
	Ref    string `json:"ref"`
	Repeat int    `json:"repeat,omitempty"`
}

// Plan is the resolved service: what goes where, without texts. It depends
// on the date, service and parameters, not on language or display mode, and
// can be cached by (date, service, params).
type Plan struct {
	Date    string                `json:"date"`  // Julian, YYYY-MM-DD
	Civil   string                `json:"civil"` // Gregorian
	Service string                `json:"service"`
	Rule    string                `json:"rule"`
	Params  Params                `json:"params"`
	Day     DaySummary            `json:"day"`
	Slots   map[string][]SlotItem `json:"slots"`
}

// DaySummary is the part of the day context recorded in a plan.
type DaySummary struct {
	Weekday      string `json:"weekday"`
	Period       string `json:"period"`
	PaschaOffset int    `json:"pascha_offset"`
	Tone         int    `json:"tone"`
}

// Planner builds plans.
type Planner struct {
	Rules   *Rules
	Menaion MenaionLister
}

// UnsupportedError reports a recognised case that is not implemented yet.
type UnsupportedError struct{ Rule, Reason string }

func (e *UnsupportedError) Error() string { return fmt.Sprintf("%s (rule %s)", e.Reason, e.Rule) }

// Plan resolves service on liturgical day d.
func (p *Planner) Plan(ctx context.Context, d calendar.JDate, service string, params Params) (*Plan, error) {
	day := calendar.NewDayContext(d)
	men, err := MenaionFacts(ctx, p.Menaion, day.Menaion)
	if err != nil {
		return nil, err
	}
	f := Facts{Day: day, Menaion: men}
	r, err := p.Rules.Match(service, f)
	if err != nil {
		return nil, err
	}
	if r.Unsupported != "" {
		return nil, &UnsupportedError{Rule: r.ID, Reason: r.Unsupported}
	}
	slots, err := r.Fill(f)
	if err != nil {
		return nil, fmt.Errorf("rule %s: %w", r.ID, err)
	}
	y, m, dd := d.Date()
	return &Plan{
		Date:    fmt.Sprintf("%04d-%02d-%02d", y, int(m), dd),
		Civil:   d.CivilString(),
		Service: service,
		Rule:    r.ID,
		Params:  params,
		Day: DaySummary{
			Weekday: weekdayName(day.Weekday), Period: day.Period.String(),
			PaschaOffset: day.PaschaOffset, Tone: day.Tone,
		},
		Slots: slots,
	}, nil
}
