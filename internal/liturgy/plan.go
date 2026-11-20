package liturgy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// Params are the request options that change the service (not its display).
type Params struct {
	// Priest: served with a priest; otherwise the reader's order
	// ("Молитвами святых отец наших…" instead of exclamations).
	Priest bool `json:"priest"`
	// Standalone: the hour is served on its own, with the usual beginning and
	// a dismissal. By default each hour is joined to its neighbours as in
	// parish practice: matins + 1st hour; 3rd + 6th hour before the liturgy;
	// 9th hour before vespers.
	Standalone bool `json:"standalone"`
	// Typikon: follow the Typikon instead of parish practice (as in the
	// «Богослужебные указания»), where the two differ.
	Typikon bool `json:"typikon"`
	// Deacon: served with a deacon (litanies and some exclamations are his).
	Deacon bool `json:"deacon"`
	// Temple: the dedication of the church (an id of data/calendar/temples.yaml
	// or a Menaion day me.MM.DD); "" when not given.
	Temple string `json:"temple,omitempty"`
	// Psalmody: how the psalm verses before the stichera are done where the
	// beginning is sung («Господи, воззвах» from «Положи, Господи»; the
	// praises when the doxology is sung): "short" (straight to the verse of
	// the first sticheron; the default), "sung" or "read".
	Psalmody string `json:"psalmody,omitempty"`
}

// PsalmodyOrDefault returns Psalmody, "short" when not given.
func (p Params) PsalmodyOrDefault() string {
	switch p.Psalmody {
	case "sung", "read", "short":
		return p.Psalmody
	}
	return "short"
}

// SlotItem references one text unit placed into a slot.
type SlotItem struct {
	Ref       string    `json:"ref"`
	Repeat    int       `json:"repeat,omitempty"`
	Performer Performer `json:"performer,omitempty"` // overrides the slot node's performer
}

// Plan is the resolved service: what goes where, without texts. It depends
// on the date, service and parameters, not on language or display mode, and
// can be cached by (date, service, params).
type Plan struct {
	Date    string `json:"date"`  // Julian, YYYY-MM-DD
	Civil   string `json:"civil"` // Gregorian
	Service string `json:"service"`
	Rule    string `json:"rule"`
	// Template: data/templates/<template>.yaml (default: the service).
	Template string                `json:"template,omitempty"`
	Params   Params                `json:"params"`
	Day      DaySummary            `json:"day"`
	Slots    map[string][]SlotItem `json:"slots"`
}

// DaySummary is the part of the day context recorded in a plan.
type DaySummary struct {
	Weekday      string `json:"weekday"`
	Period       string `json:"period"`
	PaschaOffset int    `json:"pascha_offset"`
	Tone         int    `json:"tone"`
	Feast        string `json:"feast,omitempty"`
}

// Planner builds plans.
type Planner struct {
	Rules     *Rules
	Feasts    *Feasts
	Temples   *Temples
	Katavasia *Katavasia
	Menaion   MenaionLister
}

// UnsupportedError reports a recognised case that is not implemented yet.
type UnsupportedError struct{ Rule, Reason string }

func (e *UnsupportedError) Error() string { return fmt.Sprintf("%s (rule %s)", e.Reason, e.Rule) }

// toneKinds: the Menaion units whose tones the selectors need (theotokia
// by the tone of «Слава», prokeimena).
var toneKinds = []string{"gvespers.lic.glory", "gvespers.lic.now", "gvespers.litiya.glory", "gvespers.aposticha.glory", "gvespers.lic.sticheron",
	"gvespers.litiya.sticheron", "gvespers.aposticha.sticheron", "lit.prokeimenon", "lit.alleluia.verse", "matins.prokeimenon",
	"vespers.lic.glory", "vespers.lic.glory_now", "vespers.aposticha.glory"}

// Plan resolves service on liturgical day d.
func (p *Planner) Plan(ctx context.Context, d calendar.JDate, service string, params Params) (*Plan, error) {
	day := calendar.NewDayContext(d)
	men, err := MenaionFacts(ctx, p.Menaion, day.Menaion)
	if err != nil {
		return nil, err
	}
	oct, err := OctoechosFacts(ctx, p.Menaion, day.Tone, day.Weekday)
	if err != nil {
		return nil, err
	}
	f := Facts{Day: day, Menaion: men, Octoechos: oct}
	toneIDs := append([]string{}, men["troparion"]...)
	for _, k := range toneKinds {
		toneIDs = append(toneIDs, men[k]...)
	}
	if tones, err := p.Menaion.UnitTones(ctx, toneIDs); err == nil {
		f.Tones = tones
	} else {
		return nil, err
	}
	if p.Feasts != nil {
		f.Feast = p.Feasts.At(day)
	}
	if ml, ok := p.Menaion.(metaLister); ok {
		ids := append(append([]string{}, men["troparion"]...), men["kontakion"]...)
		if f.Of, err = ml.UnitMeta(ctx, ids, "of"); err != nil {
			return nil, err
		}
	}
	if f.Feast != nil && f.Feast.Offset == nil && !f.Feast.IsFeastDay(day) {
		if f.FeastMenaion, err = MenaionFacts(ctx, p.Menaion, f.Feast.from); err != nil {
			return nil, err
		}
		var ids []string
		for _, k := range toneKinds {
			ids = append(ids, f.FeastMenaion[k]...)
		}
		tones, err := p.Menaion.UnitTones(ctx, ids)
		if err != nil {
			return nil, err
		}
		for id, t := range tones {
			f.Tones[id] = t
		}
	}
	if f.Special = specialSunday(day); f.Special == "" {
		f.Special = specialSaturday(day)
	}
	if f.Special != "" {
		ids, err := p.Menaion.ListIDs(ctx, "ms."+f.Special+".")
		if err != nil {
			return nil, err
		}
		f.SpecialUnits = map[string]string{}
		for _, id := range ids {
			f.SpecialUnits[strings.TrimPrefix(id, "ms."+f.Special+".")] = id
		}
		tones, err := p.Menaion.UnitTones(ctx, ids)
		if err != nil {
			return nil, err
		}
		for id, t := range tones {
			f.Tones[id] = t
		}
	}
	f.EveOf = eveOf(day)
	eveFeastDay := day
	if f.EveOf != "" {
		eveFeastDay = calendar.NewDayContext(d.AddDays(1))
	} else if service == "vespers" {
		// The vespers of the feast itself, served on its eve.
		f.EveOf = eveOf(calendar.NewDayContext(d.AddDays(-1)))
	}
	if f.EveOf != "" {
		prefix := "ev." + f.EveOf + "."
		ids, err := p.Menaion.ListIDs(ctx, prefix)
		if err != nil {
			return nil, err
		}
		f.Eve = map[string]string{}
		for _, id := range ids {
			f.Eve[strings.TrimPrefix(id, prefix)] = id
		}
		tones, err := p.Menaion.UnitTones(ctx, ids)
		if err != nil {
			return nil, err
		}
		for id, t := range tones {
			f.Tones[id] = t
		}
		if p.Feasts != nil {
			f.EveFeast = p.Feasts.At(eveFeastDay)
		}
	}
	if f.RoyalHours = royalHours(day); f.RoyalHours != "" {
		prefix := "rh." + f.RoyalHours + "."
		ids, err := p.Menaion.ListIDs(ctx, prefix)
		if err != nil {
			return nil, err
		}
		f.Royal = map[string]string{}
		for _, id := range ids {
			f.Royal[strings.TrimPrefix(id, prefix)] = id
		}
		tones, err := p.Menaion.UnitTones(ctx, ids)
		if err != nil {
			return nil, err
		}
		for id, t := range tones {
			f.Tones[id] = t
		}
	}
	if f.Temple, err = p.Temples.Find(params.Temple); err != nil {
		return nil, err
	}
	if f.Sign, err = p.daySign(ctx, day.Menaion); err != nil {
		return nil, err
	}
	if f.Day.Departed, err = p.departed(ctx, d); err != nil {
		return nil, err
	}
	f.Typikon = params.Typikon
	f.Psalmody = params.PsalmodyOrDefault()
	if service == "vespers" {
		// «Егда прилучится празднество… глаголем на утрени две кафисмы, а
		// третию рядовую на вечерни» (Устав о Псалтири).
		eve := calendar.NewDayContext(d.Eve())
		evm, err := MenaionFacts(ctx, p.Menaion, eve.Menaion)
		if err != nil {
			return nil, err
		}
		prefix := fmt.Sprintf("me.%02d.%02d.", int(eve.Menaion.Month), eve.Menaion.Day)
		for _, kind := range []string{"vespers.great", "gvespers.paremia"} {
			for _, id := range evm[kind] {
				if strings.HasPrefix(id, prefix) {
					f.EveFestal = true
				}
			}
		}
		if ef := p.Feasts.At(eve); ef != nil && ef.IsFeastDay(eve) {
			f.EveFestal = true
		}
	}
	if service == "matins" && day.Weekday != time.Sunday { // the canon of the Theotokos
		if f.SundayOctoechos, err = OctoechosFacts(ctx, p.Menaion, day.Tone, time.Sunday); err != nil {
			return nil, err
		}
	}
	if service == "matins" {
		f.OdeSet = "weekday"
		switch {
		case day.Weekday == time.Sunday:
			f.OdeSet = "sunday"
		case f.festalOdes():
			f.OdeSet = "feast"
		}
		if params.Typikon {
			set := f.OdeSet
			if set == "feast" {
				set = "sunday" // «песней стихи, поемии в недели и праздники»
			}
			if f.OdeVerses, err = odeVerses(ctx, p.Menaion, set); err != nil {
				return nil, err
			}
		}
		if f.Feast != nil && f.Feast.IsFeastDay(day) {
			ids, err := p.Menaion.ListIDs(ctx, "ir.pripev."+f.Feast.ID+".")
			if err != nil {
				return nil, err
			}
			if len(ids) > 0 {
				f.Refrains = map[string]bool{}
				for _, id := range ids {
					f.Refrains[id] = true
				}
			}
		}
	}
	afterfeast := f.Feast != nil && !f.Feast.IsFeastDay(day)
	if service == "matins" && (day.Weekday == time.Sunday || afterfeast) {
		// The seasonal katavasia: on Sunday, and in the afterfeast (sung
		// after every ode).
		if f.Katavasia, f.KatavasiaSeasonal, err = p.Katavasia.ForSeason(ctx, p.Menaion, day.Menaion); err != nil {
			return nil, err
		}
	}
	if service == "matins" && day.Weekday == time.Sunday {
		if f.Eothinon = day.Eothinon; f.Eothinon > 0 {
			ids := []string{fmt.Sprintf("oc.eothinon.%d.sticheron", f.Eothinon)}
			if tones, err := p.Menaion.UnitTones(ctx, ids); err == nil {
				for id, t := range tones {
					f.Tones[id] = t
				}
			}
		}
	}
	r, err := p.Rules.Match(service, f)
	if err != nil {
		return nil, err
	}
	if r.Unsupported != "" {
		return nil, &UnsupportedError{Rule: r.ID, Reason: r.Unsupported}
	}
	slots, err := r.Fill(f, service)
	if err != nil {
		return nil, fmt.Errorf("rule %s: %w", r.ID, err)
	}
	if err := p.Rules.FillSlotRules(service, f, slots); err != nil {
		return nil, err
	}
	// The liturgy of St Basil the Great: «О Тебе радуется» instead of
	// «Достойно есть» (a feast's zadostoinik stays).
	if len(slots["basil"]) > 0 {
		for i, it := range slots["truly_meet"] {
			if it.Ref == "ch.lit.it_is_truly_meet" {
				slots["truly_meet"][i].Ref = "ch.lit.all_of_creation"
			}
		}
	}
	y, m, dd := d.Date()
	return &Plan{
		Date:     fmt.Sprintf("%04d-%02d-%02d", y, int(m), dd),
		Civil:    d.CivilString(),
		Service:  service,
		Rule:     r.ID,
		Template: r.Template,
		Params:   params,
		Day: DaySummary{
			Weekday: weekdayName(day.Weekday), Period: day.Period.String(),
			PaschaOffset: day.PaschaOffset, Tone: day.Tone, Feast: feastID(f.Feast),
		},
		Slots: slots,
	}, nil
}

// daySign: the sign of the day in the Typikon's calendar (the highest of its
// commemorations); "" without the Typikon in the corpus.
func (p *Planner) daySign(ctx context.Context, md calendar.MonthDay) (string, error) {
	ml, ok := p.Menaion.(metaLister)
	if !ok {
		return "", nil
	}
	ids, err := p.Menaion.ListIDs(ctx, fmt.Sprintf("ty.%02d.%02d.sign.", int(md.Month), md.Day))
	if err != nil || len(ids) == 0 {
		return "", err
	}
	signs, err := ml.UnitMeta(ctx, ids, "sign")
	if err != nil {
		return "", err
	}
	sign := "none"
	for _, s := range signs {
		if signRank[s] > signRank[sign] {
			sign = s
		}
	}
	return sign, nil
}

// greatFeast: a twelve feast, a sign of the polyeleos or higher in the
// Typikon, or great vespers in the Menaion (feasts added after the Typikon,
// as «Всех скорбящих Радость», 24 October).
func (p *Planner) greatFeast(ctx context.Context, d calendar.JDate) (bool, error) {
	day := calendar.NewDayContext(d)
	if p.Feasts != nil {
		if fe := p.Feasts.At(day); fe != nil && fe.IsFeastDay(day) {
			return true, nil
		}
	}
	sign, err := p.daySign(ctx, day.Menaion)
	if err != nil || signRank[sign] >= signRank["polyeleos"] {
		return err == nil, err
	}
	ids, err := p.Menaion.ListIDs(ctx, fmt.Sprintf("me.%02d.%02d", int(day.Menaion.Month), day.Menaion.Day))
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if strings.HasSuffix(id, ".vespers.great") || strings.HasSuffix(id, ".gvespers.paremia.1") {
			return true, nil
		}
	}
	return false, nil
}

// departed: the parental Saturday on d. Dimitrievskaya, when it falls on a
// great feast, is moved a Saturday earlier (and again, while that one is a
// great feast too).
func (p *Planner) departed(ctx context.Context, d calendar.JDate) (string, error) {
	day := calendar.NewDayContext(d)
	if day.Weekday != time.Saturday || day.Departed != "" && day.Departed != calendar.DepartedDemetrius {
		return day.Departed, nil
	}
	for k := 0; k <= 3; k++ {
		c := d.AddDays(7 * k)
		if calendar.NewDayContext(c).Departed != calendar.DepartedDemetrius {
			continue
		}
		for i := 0; i < 4; i++ {
			great, err := p.greatFeast(ctx, c)
			if err != nil {
				return "", err
			}
			if !great {
				break
			}
			c = c.AddDays(-7)
		}
		if c == d {
			return calendar.DepartedDemetrius, nil
		}
		return "", nil
	}
	return "", nil
}
