package liturgy

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The weekday matins of Great Lent, with «Аллилуиа» (Типикон, гл. 49, the
// Monday of the 1st week, «сице да бывает утреня … во всю святую
// четыредесятницу»; the Chasoslov):
//
//   - instead of «Бог Господь» Alleluia in the tone of the week with the
//     verses of Isaiah 26, then the Trinity hymns of the tone (the first with
//     the day's ending: «предстательствы безплотных Твоих…»);
//   - three kathismata (the Lenten Psalter: rules matins.kathisma_*.lent);
//     after the 1st the Octoechos's sessional hymns, after the 2nd and 3rd
//     the Triodion's, each «Слава, тойже», «И ныне» its theotokion;
//   - «осмогласник не поется»: the canon of the Menaion with its irmos and
//     the triodes of the Triodion in their odes, the biblical odes read
//     (-typikon) on 14; after an ode with a triode its irmos (the katavasia),
//     after the 3rd and 6th without one the Menaion's irmos; after the 3rd ode
//     the Menaion's sessional hymn once, after the 6th its kontakion;
//   - after the 9th «Достойно есть», the photagogikon of the tone thrice;
//   - the praises read, the doxology read; the aposticha: the Triodion's
//     idiomelon twice («Исполнихомся заутра», «И буди светлость»), the
//     martyricon, «Слава, и ныне» the theotokion;
//   - «Благо есть», the Trisagion, «В храме стояще славы Твоея», «Господи,
//     помилуй» 40, «Честнейшую», «Сый благословен», «Царю Небесный», the
//     prayer of St Ephrem; the first hour.

// reLentWeekday: the Lenten weekdays' Triodion ids (w<week>_<day>).
var reLentWeekday = regexp.MustCompile(`^w[1-6]_(mon|tue|wed|thu|fri)$`)

// lentWeekday: the id of a weekday of the six weeks of Great Lent (Clean
// Monday is Pascha − 48), "" otherwise.
func lentWeekday(offset int) string {
	d := offset + 48
	if d < 0 || d >= 42 || d%7 > 4 {
		return ""
	}
	return fmt.Sprintf("w%d_%s", d/7+1, []string{"mon", "tue", "wed", "thu", "fri"}[d%7])
}

var weekdayIDs = map[time.Weekday]string{time.Monday: "mon", time.Tuesday: "tue", time.Wednesday: "wed",
	time.Thursday: "thu", time.Friday: "fri"}

func lentenMatinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	if !reLentWeekday.MatchString(f.TriodionDay) || len(f.Triodion) == 0 {
		return nil, fmt.Errorf("no Lenten Triodion for %q", f.TriodionDay)
	}
	dow, tone := weekdayIDs[f.Day.Weekday], f.Day.Tone
	tp := func(k string) string { return f.Triodion[k] }
	switch sel.From {
	case "lm_alleluia":
		items := []SlotItem{litanist(fmt.Sprintf("lit.alleluia.tone.%d", tone))}
		for v := 1; v <= 4; v++ {
			items = append(items, litanist(fmt.Sprintf("ch.matins.alleluia.verse.%d", v)), choir("common.alleluia.thrice"))
		}
		return items, nil
	case "lm_troparia":
		return []SlotItem{choir(fmt.Sprintf("oc.%d.trinity.1.%s", tone, dow)),
			reader("common.glory"), choir(fmt.Sprintf("oc.%d.trinity.glory", tone)),
			reader("common.and_now"), choir(fmt.Sprintf("oc.%d.trinity.now", tone))}, nil
	case "lm_sedalny":
		if sel.Section == "1" {
			return matinsSedalny(f, 1)
		}
		k := "matins.sedalen.k" + sel.Section
		var items []SlotItem
		for n := 1; tp(fmt.Sprintf("%s.%d", k, n)) != ""; n++ {
			items = append(items, choir(tp(fmt.Sprintf("%s.%d", k, n))))
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("%s: no sessional hymn after the kathisma %s", f.TriodionDay, sel.Section)
		}
		if g := tp(k + ".glory"); g != "" {
			items = append(items, reader("common.glory"), choir(g))
			if th := tp(k + ".theotokion"); th != "" {
				items = append(items, reader("common.and_now"), choir(th))
			}
		} else if th := tp(k + ".theotokion"); th != "" {
			items = append(items, reader("common.glory_now"), choir(th))
		}
		return items, nil
	case "lm_canon":
		return lentenCanon(f, sel.Section)
	case "lm_kontakion":
		// «Аще же имать святый кондак, глаголем его зде».
		k := firstMain(f, "kontakion")
		if k == "" {
			return nil, nil
		}
		items := []SlotItem{reader(k)}
		if ik := firstMain(f, "matins.ikos"); ik != "" {
			items = append(items, reader(ik))
		}
		return items, nil
	case "lm_exapostilaria":
		return []SlotItem{{Ref: fmt.Sprintf("oc.%d.photagogikon.%s", tone, dow), Performer: Choir, Repeat: 3}}, nil
	case "lm_aposticha":
		id, mart, th := tp("matins.aposticha.idiomelon"), tp("matins.aposticha.martyricon"), tp("matins.aposticha.theotokion")
		if id == "" || mart == "" || th == "" {
			return nil, fmt.Errorf("%s: incomplete aposticha in the Triodion", f.TriodionDay)
		}
		items := []SlotItem{choir(id), reader("ch.matins.aposticha.verse.2"), choir(id),
			reader("ch.matins.aposticha.verse.3"), choir(mart)}
		if g := tp("matins.aposticha.glory"); g != "" { // «Слава» an idiomelon, «И ныне» the same
			return append(items, reader("common.glory"), choir(g), reader("common.and_now"), choir(th)), nil
		}
		return append(items, reader("common.glory_now"), choir(th)), nil
	case "lm_end_troparia":
		return []SlotItem{choir("ch.matins.in_the_temple")}, nil
	}
	return nil, fmt.Errorf("unknown Lenten matins selector %q", sel.From)
}

// lentPlace: a place of a triode in the book's order.
type lentPlace struct {
	n    int
	kind string // troparion | triadikon | theotokion | refrain
	id   string
}

// triode: the places of the Triodion's canon c in ode o
// (canon.<c>.ode.<o>.<n>.<kind>).
func triode(f Facts, c, o int) []lentPlace {
	prefix := fmt.Sprintf("canon.%d.ode.%d.", c, o)
	var out []lentPlace
	for k, id := range f.Triodion {
		ns, kind, ok := strings.Cut(strings.TrimPrefix(k, prefix), ".")
		if !strings.HasPrefix(k, prefix) || !ok {
			continue
		}
		n, err := strconv.Atoi(ns)
		if err != nil {
			continue
		}
		out = append(out, lentPlace{n, kind, id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].n < out[j].n })
	return out
}

// lentenCanon: the odes of section ("1-3", "4-6", "7-8", "9"), the 2nd
// among them when the Triodion has it (Tuesday).
func lentenCanon(f Facts, section string) ([]SlotItem, error) {
	var odes []int
	for _, part := range strings.Split(section, ",") {
		a, b, rng := strings.Cut(strings.TrimSpace(part), "-")
		from, err := strconv.Atoi(a)
		if err != nil {
			return nil, fmt.Errorf("odes %q", section)
		}
		to := from
		if rng {
			if to, err = strconv.Atoi(b); err != nil {
				return nil, fmt.Errorf("odes %q", section)
			}
		}
		for o := from; o <= to; o++ {
			odes = append(odes, o)
		}
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
	var items []SlotItem
	for _, o := range odes {
		var tri [][]lentPlace
		for c := 1; c <= 3; c++ {
			if p := triode(f, c, o); len(p) > 0 {
				tri = append(tri, p)
			}
		}
		var men []SlotItem
		menIrmos := ""
		if o != 2 {
			for c := 1; c <= menCanons; c++ {
				base := fmt.Sprintf("matins.canon.%d.ode.%d", c, o)
				if c == 1 {
					// «Поются же ирмосы минеи по каноне, токмо по третией и
					// шестой песнех, занеже у тех песней первее ирмос не поется».
					menIrmos = firstMain(f, base+".irmos")
					if menIrmos != "" && !((o == 3 || o == 6) && len(triode(f, 1, o)) == 0) {
						men = append(men, SlotItem{Ref: menIrmos})
					}
				}
				for _, t := range mainMenaion(f, base+".troparion") {
					men = append(men, SlotItem{Ref: t})
				}
				if c == menCanons { // «бгородичен же оставляется» but of the last
					if th := theotokionFor(f, base); th != "" {
						men = append(men, SlotItem{Ref: th})
					}
				}
			}
		}
		if len(men) == 0 && len(tri) == 0 {
			continue
		}
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		if len(tri) == 0 { // the Menaion's canon alone
			if len(f.OdeVerses) > 0 {
				men = withOdeVerses(f, "lent", o, men)
			} else if len(men) >= 3 {
				n := len(men)
				men = append(men[:n-2:n-2], reader("common.glory"), men[n-2], reader("common.and_now"), men[n-1])
			}
			items = append(items, men...)
			if (o == 3 || o == 6) && menIrmos != "" {
				items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: menIrmos})
			}
			continue
		}
		// The triodes: the last's «Слава» (its triadikon) and «И ныне» (its
		// theotokion) close the ode's places; its further troparia follow
		// with «Слава Тебе, Боже наш, слава Тебе»; then its irmos.
		lastC := len(tri) - 1
		last := tri[lastC]
		cut := len(last) // the first of the closing places
		for i, p := range last {
			if p.kind == "triadikon" || p.kind == "theotokion" {
				cut = i
				if i > 0 && last[i-1].kind == "refrain" { // «Благословим Отца…» before the triadikon
					cut = i - 1
				}
				break
			}
		}
		head := men
		if len(men) == 0 { // the 2nd ode: the triode's irmos first
			if irm := tpIrmos(f, 1, o); irm != "" {
				head = append(head, SlotItem{Ref: irm})
			}
		}
		for i, places := range tri {
			end := len(places)
			if i == lastC {
				end = cut
			}
			for _, p := range places[:end] {
				it := SlotItem{Ref: p.id}
				if p.kind == "refrain" {
					it.Performer = Reader
				}
				head = append(head, it)
			}
		}
		if verses := f.OdeVerses[o]; len(verses) > 0 {
			// The verses before the last places, counted from the end
			// (versesBefore: the last two get «Слава», «И ныне»).
			n := min(len(head), len(verses))
			unversed := len(head) - n
			vs := verses[len(verses)-n:]
			var out []SlotItem
			for i, it := range head {
				if i >= unversed && it.Performer != Reader {
					out = append(out, reader(vs[i-unversed]))
				}
				out = append(out, it)
			}
			head = out
		}
		items = append(items, head...)
		for i := cut; i < len(last); i++ {
			p := last[i]
			switch {
			case p.kind == "triadikon" && (i == 0 || last[i-1].kind != "refrain"):
				items = append(items, reader("common.glory"))
			case p.kind == "theotokion":
				items = append(items, reader("common.and_now"))
			}
			it := SlotItem{Ref: p.id}
			if p.kind == "refrain" {
				it.Performer = Reader
			}
			items = append(items, it)
		}
		kat := f.Triodion[fmt.Sprintf("canon.%d.ode.%d.katavasia", lastC+1, o)]
		if kat == "" {
			kat = tpIrmos(f, lastC+1, o)
		}
		if kat != "" {
			items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: kat})
		}
	}
	return items, nil
}

// tpIrmos: the irmos of the triode c in ode o (the full text printed after
// the ode when the beginning gives its first words only).
func tpIrmos(f Facts, c, o int) string {
	if k := f.Triodion[fmt.Sprintf("canon.%d.ode.%d.katavasia", c, o)]; k != "" {
		return k
	}
	return f.Triodion[fmt.Sprintf("canon.%d.ode.%d.irmos", c, o)]
}
