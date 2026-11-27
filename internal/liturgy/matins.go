package liturgy

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Selectors of daily (weekday) matins: Octoechos and Menaion hymns put in
// the order of the Typikon for a day of the Menaion without a sign.
//
//   - troparia at «Бог Господь»: the saint's troparion twice, «Слава»
//     (the second saint's troparion), «И ныне» the dismissal theotokion;
//   - sessional hymns after each kathisma (Octoechos) with their verses;
//   - the canon: the first Octoechos canon with its irmos, the second (when
//     the Menaion has a single canon), the Menaion canon(s); katavasia —
//     the irmos of the last canon after the 3rd, 6th, 8th and 9th odes.
//     The troparia are given once each; the Typikon's counts («на 6», «на
//     4») by repeating them are not applied. A saint of the polyeleos and
//     higher: the canon of the Theotokos instead of the Octoechos's
//     (theotokosCanon);
//   - after the 3rd ode the Menaion sessional hymn, after the 6th the
//     kontakion and ikos, after the 9th the exapostilaria (Octoechos, then
//     the Menaion's, then the theotokion);
//   - aposticha of the Octoechos, «Слава» of the Menaion if any.

var odeOrder = []int{1, 3, 4, 5, 6, 7, 8, 9}

func matinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	switch sel.From {
	case "matins_troparia":
		return matinsTroparia(f)
	case "matins_sedalny":
		k, _ := strconv.Atoi(sel.Section)
		return matinsSedalny(f, k)
	case "matins_canon":
		return matinsCanon(f, sel.Section)
	case "matins_sedalen":
		return matinsMenaionSedalen(f)
	case "matins_kontakion":
		return matinsKontakion(f)
	case "matins_exapostilaria":
		return matinsExapostilaria(f)
	case "matins_aposticha":
		return matinsAposticha(f)
	case "sat_sedalny":
		return saturdaySedalny(f)
	case "sat_aposticha":
		return saturdayAposticha(f)
	}
	return nil, fmt.Errorf("unknown matins selector %q", sel.From)
}

func reader(ref string) SlotItem { return SlotItem{Ref: ref, Performer: Reader} }

// mainMenaion returns the ids of kind from the day's main Menaion service
// (not the additional ones, "04p").
func mainMenaion(f Facts, kind string) []string {
	if f.MainPrefix != "" { // a view of another book as the day's (triodionView)
		return f.Menaion[kind]
	}
	prefix := fmt.Sprintf("me.%02d.%02d.", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	var out []string
	for _, id := range f.Menaion[kind] {
		if strings.HasPrefix(id, prefix) {
			out = append(out, id)
		}
	}
	return out
}

func firstMain(f Facts, kind string) string {
	if l := mainMenaion(f, kind); len(l) > 0 {
		return l[0]
	}
	return ""
}

func matinsTroparia(f Facts) ([]SlotItem, error) {
	items, err := vespersTroparia(f)
	if err != nil {
		return nil, err
	}
	// «Бог Господь»: the first troparion is sung twice.
	items[0].Repeat = 2
	return items, nil
}

func matinsSedalny(f Facts, k int) ([]SlotItem, error) {
	p := fmt.Sprintf("matins.sedalen.%d", k)
	hymns := f.Octoechos[p]
	if len(hymns) == 0 {
		return nil, fmt.Errorf("no Octoechos sessional hymns after kathisma %d (tone %d, %v)", k, f.Day.Tone, f.Day.Weekday)
	}
	verses := map[string]string{}
	for _, id := range f.Octoechos[p+".verse"] {
		verses[id[strings.LastIndex(id, ".")+1:]] = id
	}
	var items []SlotItem
	for _, id := range hymns {
		if v := verses[id[strings.LastIndex(id, ".")+1:]]; v != "" {
			items = append(items, reader(v))
		}
		items = append(items, SlotItem{Ref: id})
	}
	if th := first(f.Octoechos, p+".theotokion"); th != "" {
		items = append(items, reader("common.glory_now"), SlotItem{Ref: th})
	}
	return items, nil
}

// matinsCanon gives the odes listed in section ("1,3", "4-6", "7,8", "9").
func matinsCanon(f Facts, section string) ([]SlotItem, error) {
	odes, err := parseOdes(section)
	if err != nil {
		return nil, err
	}
	if f.withTheotokosCanon() {
		return theotokosCanon(f, odes)
	}
	menCanons := 0
	for c := 1; c <= 3; c++ {
		if firstMain(f, fmt.Sprintf("matins.canon.%d.ode.1.troparion", c)) != "" {
			menCanons = c
		}
	}
	if menCanons == 0 {
		return nil, fmt.Errorf("no Menaion canon for %02d.%02d in the corpus", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	octCanons := 2
	if menCanons >= 2 {
		octCanons = 1 // «един канон Октоиха»: the first
	}
	// Saturday (Типикон, гл. 12): the saint's canon with its irmos, then
	// the Octoechos's of the martyrs (the departed one is sung at compline).
	sat := f.Day.Weekday == time.Saturday
	if sat {
		octCanons = 1
	}
	cross := f.Day.Weekday == time.Wednesday || f.Day.Weekday == time.Friday
	var items []SlotItem
	for _, o := range odes {
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		start := len(items)
		var lastIrmos string
		part := func(src map[string][]string, base string, withIrmos bool, main bool) error {
			get := func(kind string) []string {
				if main {
					return mainMenaion(f, kind)
				}
				return src[kind]
			}
			irmos := get(base + ".irmos")
			trop := get(base + ".troparion")
			if len(trop) == 0 {
				return fmt.Errorf("canon %s: no troparia", base)
			}
			if len(irmos) > 0 {
				lastIrmos = irmos[0]
				if withIrmos {
					items = append(items, SlotItem{Ref: irmos[0]})
				}
			}
			for _, t := range trop {
				items = append(items, SlotItem{Ref: t})
			}
			th := get(base + ".theotokion")
			if cross {
				if c := get(base + ".cross_theotokion"); len(c) > 0 {
					th = c
				}
			}
			if g := get(base + ".glory"); len(g) > 0 && !main { // the Saturday canon
				items = append(items, reader("common.glory"), SlotItem{Ref: g[0]})
				if len(th) > 0 {
					items = append(items, reader("common.and_now"))
				}
			}
			if len(th) > 0 {
				items = append(items, SlotItem{Ref: th[0]})
			}
			return nil
		}
		octoechos := func() error {
			for c := 1; c <= octCanons; c++ {
				if err := part(f.Octoechos, fmt.Sprintf("matins.canon.%d.ode.%d", c, o), c == 1 && !sat, false); err != nil {
					return fmt.Errorf("Octoechos tone %d, %v: %w", f.Day.Tone, f.Day.Weekday, err)
				}
			}
			return nil
		}
		if !sat {
			if err := octoechos(); err != nil {
				return nil, err
			}
		}
		for c := 1; c <= menCanons; c++ {
			if err := part(f.Menaion, fmt.Sprintf("matins.canon.%d.ode.%d", c, o), sat && c == 1, true); err != nil {
				return nil, fmt.Errorf("Menaion %02d.%02d: %w", int(f.Day.Menaion.Month), f.Day.Menaion.Day, err)
			}
		}
		if sat {
			if err := octoechos(); err != nil {
				return nil, err
			}
		}
		if len(f.OdeVerses) > 0 { // -typikon: the verses of the biblical ode
			items = append(items[:start], withOdeVerses(f, f.OdeSet, o, append([]SlotItem(nil), items[start:]...))...)
		}
		// The katavasia: after every ode «в недели и праздники» (doxology and
		// higher), after the 3rd, 6th, 8th and 9th otherwise (Ирмологий).
		if (f.OdeSet == "feast" || o == 3 || o == 6 || o == 8 || o == 9) && lastIrmos != "" {
			items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: lastIrmos})
		}
	}
	return items, nil
}

func parseOdes(s string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(s, ",") {
		a, b, rng := strings.Cut(strings.TrimSpace(part), "-")
		from, err := strconv.Atoi(a)
		if err != nil {
			return nil, fmt.Errorf("odes %q", s)
		}
		to := from
		if rng {
			if to, err = strconv.Atoi(b); err != nil {
				return nil, fmt.Errorf("odes %q", s)
			}
		}
		for _, o := range odeOrder {
			if o >= from && o <= to {
				out = append(out, o)
			}
		}
	}
	return out, nil
}

// theotokionFor picks the Menaion theotokion of a part, the
// cross-theotokion on Wednesday and Friday when there is one.
func theotokionFor(f Facts, base string) string {
	if f.Day.Weekday == time.Wednesday || f.Day.Weekday == time.Friday {
		if c := firstMain(f, base+".cross_theotokion"); c != "" {
			return c
		}
	}
	return firstMain(f, base+".theotokion")
}

func matinsMenaionSedalen(f Facts) ([]SlotItem, error) {
	hymns := mainMenaion(f, "matins.sedalen")
	if len(hymns) == 0 {
		return nil, fmt.Errorf("no Menaion sessional hymn after the 3rd ode for %02d.%02d", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	var items []SlotItem
	for _, h := range hymns {
		items = append(items, SlotItem{Ref: h})
	}
	th := theotokionFor(f, "matins.sedalen")
	if g := firstMain(f, "matins.sedalen.glory"); g != "" {
		items = append(items, reader("common.glory"), SlotItem{Ref: g})
		if th != "" {
			items = append(items, reader("common.and_now"), SlotItem{Ref: th})
		}
		return items, nil
	}
	if th != "" {
		items = append(items, reader("common.glory_now"), SlotItem{Ref: th})
	}
	return items, nil
}

func matinsKontakion(f Facts) ([]SlotItem, error) {
	k := firstMain(f, "kontakion")
	if k == "" {
		return nil, fmt.Errorf("no Menaion kontakion for %02d.%02d", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	items := []SlotItem{{Ref: k}}
	if ikos := firstMain(f, "matins.ikos"); ikos != "" {
		items = append(items, SlotItem{Ref: ikos})
	}
	return items, nil
}

// Exapostilaria of the week (Октоих, «ексапостиларии всея седмицы»).
var weekdayExapostilaria = map[time.Weekday][]string{
	time.Monday:    {"oc.exapostilarion.mon"},
	time.Tuesday:   {"oc.exapostilarion.tue"},
	time.Wednesday: {"oc.exapostilarion.wed"},
	time.Thursday:  {"oc.exapostilarion.thu", "oc.exapostilarion.thu.2"},
	time.Friday:    {"oc.exapostilarion.wed"},
}

var weekdayExapostilarionTheotokion = map[time.Weekday]string{
	time.Monday: "oc.exapostilarion.mon.theotokion", time.Tuesday: "oc.exapostilarion.mon.theotokion",
	time.Wednesday: "oc.exapostilarion.wed.theotokion", time.Thursday: "oc.exapostilarion.thu.theotokion",
	time.Friday: "oc.exapostilarion.wed.theotokion",
}

func matinsExapostilaria(f Facts) ([]SlotItem, error) {
	if f.Day.Weekday == time.Saturday {
		return saturdayExapostilaria(f), nil
	}
	oct := weekdayExapostilaria[f.Day.Weekday]
	if len(oct) == 0 {
		return nil, fmt.Errorf("no weekday exapostilarion on %v", f.Day.Weekday)
	}
	var items []SlotItem
	for _, id := range oct {
		items = append(items, SlotItem{Ref: id})
	}
	th := theotokionFor(f, "matins.exapostilarion")
	if th == "" {
		th = weekdayExapostilarionTheotokion[f.Day.Weekday]
	}
	if men := mainMenaion(f, "matins.exapostilarion"); len(men) > 0 {
		items = append(items, reader("common.glory"))
		for _, id := range men {
			items = append(items, SlotItem{Ref: id})
		}
		return append(items, reader("common.and_now"), SlotItem{Ref: th}), nil
	}
	return append(items, reader("common.glory_now"), SlotItem{Ref: th}), nil
}

// matinsAposticha: the Octoechos's stichera — the Menaion's when it gives
// its own (the feast's in an afterfeast) — with their verses; «Слава» (and
// «И ныне») of the Menaion.
func matinsAposticha(f Facts) ([]SlotItem, error) {
	k := "matins.aposticha."
	stichera := f.Octoechos[k+"sticheron"]
	verseIDs := f.Octoechos[k+"verse"]
	if men := mainMenaion(f, k+"sticheron"); len(men) > 0 {
		stichera, verseIDs = men, mainMenaion(f, k+"verse")
	}
	if len(stichera) == 0 {
		return nil, fmt.Errorf("no Octoechos aposticha at matins (tone %d, %v)", f.Day.Tone, f.Day.Weekday)
	}
	verses := map[string]string{}
	for _, id := range verseIDs {
		verses[id[strings.LastIndex(id, ".")+1:]] = id
	}
	var items []SlotItem
	for i, id := range stichera {
		n := strconv.Itoa(i + 1)
		if i > 0 {
			v := verses[n]
			if v == "" && i <= 2 {
				v = "ch.matins.aposticha.verse." + n
			}
			if v != "" {
				items = append(items, reader(v))
			}
		}
		items = append(items, SlotItem{Ref: id})
	}
	if g := firstMain(f, k+"glory_now"); g != "" {
		return append(items, reader("common.glory_now"), SlotItem{Ref: g}), nil
	}
	th := firstMain(f, k+"now") // «И ныне, праздника»
	if th == "" {
		th = theotokionFor(f, "matins.aposticha")
	}
	if th == "" {
		th = first(f.Octoechos, k+"theotokion")
	}
	if g := firstMain(f, "matins.aposticha.glory"); g != "" {
		items = append(items, reader("common.glory"), SlotItem{Ref: g})
		if th != "" {
			items = append(items, reader("common.and_now"), SlotItem{Ref: th})
		}
		return items, nil
	}
	if th == "" {
		return nil, fmt.Errorf("no theotokion for the aposticha at matins")
	}
	return append(items, reader("common.glory_now"), SlotItem{Ref: th}), nil
}

// withTheotokosCanon: the Octoechos's canons give way to the canon of the
// Theotokos (Типикон, гл. 11): a saint of the polyeleos and higher on any
// weekday («аще ли поется стаго полиелей: тогда не поется октоих: но поем
// канон бцде, или аще предпразднство… со ирмосом на 6: и стаго на 8. сице
// убо во всю седмицу»), and on Saturday also of the great doxology («в
// куюлибо субботу славословие прилучится… канон бцды прешедшия недели
// утренний»). On a weekday of the great doxology the Octoechos stays.
func (f Facts) withTheotokosCanon() bool {
	switch {
	case f.Day.Weekday == time.Sunday:
		return false
	case f.Day.Weekday == time.Saturday && f.redSign():
		return true
	}
	// By the Menaion's own service: a day whose sign of the polyeleos is of
	// a service not imported as the main one (an icon of the Theotokos,
	// 07.08, 10.22) keeps the Octoechos with its simple saint.
	return len(mainMenaion(f, "gvespers.paremia")) > 0 || len(mainMenaion(f, "vespers.great")) > 0
}

// theotokosCanon: the canon of the Theotokos of the past Sunday's matins
// (the Octoechos's third) with its irmos — the resurrectional canon's where
// it is «тойже» — then the saint's canon(s) of the Menaion without theirs.
// The Menaion's own canon of the Theotokos (marked canon.<C>.theotokos) or,
// in a forefeast, of the forefeast is sung instead, first, with its irmos.
// Katavasia after every ode, the irmos of the last canon. The troparia are
// given once each (as in matinsCanon). The canon of a church of the Lord or
// the Theotokos «прежде» is not supported yet.
func theotokosCanon(f Facts, odes []int) ([]SlotItem, error) {
	var men []string
	for c := 1; c <= 3; c++ {
		if firstMain(f, fmt.Sprintf("matins.canon.%d.ode.1.troparion", c)) != "" {
			men = append(men, fmt.Sprintf("matins.canon.%d", c))
		}
	}
	if len(men) == 0 {
		return nil, fmt.Errorf("no Menaion canon for %02d.%02d in the corpus", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	// The Menaion's own canon of the Theotokos (a vigil, гл. 47: «прилагается
	// же и канон бцды») or of the forefeast, first with its irmos, instead of
	// the Octoechos's.
	menTheotokos := len(men) > 1 && firstMain(f, men[0]+".theotokos") != "" || f.inFeastPeriod()
	var items []SlotItem
	for _, o := range odes {
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		start := len(items)
		lastIrmos := ""
		if !menTheotokos {
			b := fmt.Sprintf("matins.canon.3.ode.%d", o)
			trop := f.SundayOctoechos[b+".troparion"]
			if len(trop) == 0 {
				return nil, fmt.Errorf("Octoechos tone %d: no canon of the Theotokos of Sunday matins, ode %d", f.Day.Tone, o)
			}
			irmos := first(f.SundayOctoechos, b+".irmos")
			if irmos == "" {
				irmos = first(f.SundayOctoechos, fmt.Sprintf("matins.canon.1.ode.%d.irmos", o))
			}
			lastIrmos = irmos
			if irmos != "" {
				items = append(items, SlotItem{Ref: irmos})
			}
			for _, t := range trop {
				items = append(items, SlotItem{Ref: t})
			}
		}
		for i, c := range men {
			mb := fmt.Sprintf("%s.ode.%d", c, o)
			mt := mainMenaion(f, mb+".troparion")
			if len(mt) == 0 {
				return nil, fmt.Errorf("Menaion %02d.%02d: canon %s: no troparia", int(f.Day.Menaion.Month), f.Day.Menaion.Day, mb)
			}
			if ir := firstMain(f, mb+".irmos"); ir != "" {
				lastIrmos = ir
				if menTheotokos && i == 0 {
					items = append(items, SlotItem{Ref: ir})
				}
			}
			for _, t := range mt {
				items = append(items, SlotItem{Ref: t})
			}
			if th := firstMain(f, mb+".theotokion"); th != "" {
				items = append(items, SlotItem{Ref: th})
			}
		}
		if len(f.OdeVerses) > 0 { // -typikon: the verses of the biblical ode
			items = append(items[:start], withOdeVerses(f, f.OdeSet, o, append([]SlotItem(nil), items[start:]...))...)
		}
		if lastIrmos != "" {
			items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: lastIrmos})
		}
	}
	return items, nil
}
