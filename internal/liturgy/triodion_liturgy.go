package liturgy

import (
	"fmt"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// triodionDay: a day of the preparatory weeks with a service of its own in
// the Triodion (orthlib.ImportTriodionPrep; units tp.<id>.*), "" otherwise.
func triodionDay(day calendar.DayContext) string {
	switch day.PaschaOffset {
	case -70:
		return "publican"
	case -63:
		return "prodigal"
	case -57:
		return "sat_meatfare"
	case -56:
		return "meatfare"
	case -50:
		return "sat_cheesefare"
	case -49:
		return "cheesefare"
	}
	return ""
}

// The liturgy of the Sundays and Saturdays of the preparatory weeks (part
// "triodion"), by the Triodion's rubrics:
//
//   - the Sundays of the Publican, the Prodigal, meatfare, cheesefare: the
//     typical antiphons; the beatitudes of the tone on 6 and the Triodion's
//     6th ode on 4 («блаженна осмогласника, на з: и триоди песнь з-я, на д»),
//     its «Слава» and theotokion; the resurrectional troparion and kontakion
//     (with the temple's), «Слава» the Triodion's kontakion, «И ныне» «Предстательство»
//     (or the Theotokos temple's). The saint of the Menaion is sung at
//     compline on Friday on the Sundays of the Publican and the Prodigal
//     («последование прилучившагося стаго… поем в пяток на повечерии»);
//     on cheesefare Sunday his troparion and kontakion are sung. The
//     prokeimenon and alleluia of the tone, or the Sunday's own (meatfare:
//     «Велий Господь наш», cheesefare: «Помолитеся»); the readings of the
//     day (the lectionary: the Triodion's); «Хвалите Господа»;
//   - the meatfare Saturday of the departed («на литургии изобразительная:
//     и от канона триоди песнь г, и з. прокимен… душы их во блгих
//     водворятся… апсл… другий за упокой, к солуняном, зачало со,
//     аллилуиа… блжени, яже избрал»): the beatitudes on 8 from the 3rd and
//     the 6th odes; «Глубиною мудрости», «Слава» «Со святыми упокой», «И
//     ныне» «Тебе и стену»; the readings of the day and of the departed
//     (1 Thess, зач. 270; John, зач. 16); «Блажени, яже избрал»;
//   - the cheesefare Saturday of the venerable fathers: the beatitudes on 8
//     from the 3rd and the 6th odes; «Боже отец наших», «Слава» their
//     kontakion; «Восхвалятся преподобнии», the readings of the day and of
//     the fathers (Gal, зач. 213; Matthew, зач. 43), alleluia «Насаждени в
//     дому Господни», the communion verse «Радуйтеся, праведнии».
func triodionLiturgySlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	tu := func(k string) string { return f.Triodion[k] }
	sunday := f.Day.Weekday == time.Sunday
	switch sel.From {
	case "lit_antiphon":
		if sel.Section != "3" {
			return litAntiphon(f, sel.Section, "sunday")
		}
		return triodionBeatitudes(f)
	case "lit_entrance":
		if sunday {
			return liturgySlot(SlotSelector{From: "lit_entrance", Part: "sunday"}, f)
		}
		return []SlotItem{choir("ch.lit.come_let_us_worship.weekday")}, nil
	case "lit_troparia":
		return triodionTroparia(f)
	case "lit_prokeimena":
		if p := tu("lit.prokeimenon"); p != "" && f.Tones[p] > 0 {
			return prokeimenonBlock(f.Tones[p], p, tu("lit.prokeimenon.verse")), nil
		}
		if f.TriodionDay == "sat_meatfare" { // «прокимен апсла, глас ѕ: душы их»
			return prokeimenonBlock(6, "sl.prokeimenon.sat.dead", tu("lit.prokeimenon.verse")), nil
		}
		// The tone's alone: the saint of the day is not sung at the liturgy.
		p, v := first(f.Octoechos, "lit.prokeimenon"), first(f.Octoechos, "lit.prokeimenon.verse")
		if p == "" {
			return nil, fmt.Errorf("Octoechos tone %d: no Sunday prokeimenon", f.Day.Tone)
		}
		return prokeimenonBlock(f.Day.Tone, p, v), nil
	case "lit_alleluia":
		if v1 := tu("lit.alleluia.verse.1"); v1 != "" && f.Tones[v1] > 0 {
			items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", f.Tones[v1])), choir("common.alleluia.thrice"),
				reader(v1), choir("common.alleluia.thrice")}
			if v2 := tu("lit.alleluia.verse.2"); v2 != "" {
				items = append(items, reader(v2), choir("common.alleluia.thrice"))
			}
			return items, nil
		}
		if f.TriodionDay == "sat_meatfare" { // «аллилуиа, глас ѕ: блжени, яже избрал»
			return []SlotItem{reader("lit.alleluia.tone.6"), choir("common.alleluia.thrice"),
				reader("sl.alleluia.sat.dead.verse"), choir("common.alleluia.thrice")}, nil
		}
		vs := f.Octoechos["lit.alleluia.verse"]
		v1, v2 := "", ""
		if len(vs) > 0 {
			v1 = vs[0]
		}
		if len(vs) > 1 {
			v2 = vs[1]
		}
		if v1 == "" {
			return nil, fmt.Errorf("Octoechos tone %d: no Sunday alleluia", f.Day.Tone)
		}
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", f.Day.Tone)), choir("common.alleluia.thrice"),
			reader(v1), choir("common.alleluia.thrice")}
		if v2 != "" {
			items = append(items, reader(v2), choir("common.alleluia.thrice"))
		}
		return items, nil
	case "lit_apostles", "lit_gospels":
		which := map[string]string{"lit_apostles": "apostle", "lit_gospels": "gospel"}[sel.From]
		return triodionReadings(f, which)
	case "lit_kinonik":
		k := map[string]string{"sat_meatfare": "sl.kinonik.sat.dead", "sat_cheesefare": "sl.kinonik.sat"}[f.TriodionDay]
		if k == "" {
			k = "sl.kinonik.sun"
		}
		return []SlotItem{choir(k), choir("common.alleluia.thrice")}, nil
	}
	return liturgySlot(SlotSelector{From: sel.From, Section: sel.Section}, f)
}

// triodionBeatitudes: on Sunday the tone's on 6 and the Triodion's 6th ode
// on 4; on Saturday the Triodion's 3rd and 6th odes on 8; «Слава» the ode's
// own (the 6th), «И ныне» its theotokion.
func triodionBeatitudes(f Facts) ([]SlotItem, error) {
	ode := func(n int) []string {
		var out []string
		for k := 1; ; k++ {
			id := f.Triodion[fmt.Sprintf("canon.ode.%d.troparion.%d", n, k)]
			if id == "" {
				return out
			}
			out = append(out, id)
		}
	}
	verse := func(i int) SlotItem { return choir(fmt.Sprintf("ch.typika.beatitudes.%d", i)) }
	glory, now := f.Triodion["canon.ode.6.glory"], f.Triodion["canon.ode.6.theotokion"]
	var troparia []string
	plain := 3 // on 8
	if f.Day.Weekday == time.Sunday {
		oct := f.Octoechos["lit.beatitudes"]
		if len(oct) < 4 {
			return nil, fmt.Errorf("Octoechos tone %d: no Sunday beatitudes", f.Day.Tone)
		}
		t6 := ode(6)
		if len(t6) == 0 {
			return nil, fmt.Errorf("Triodion %s: no 6th ode for the beatitudes", f.TriodionDay)
		}
		// The tone's on 6, the Triodion's on 4 (its troparia repeated to fill).
		troparia = append(troparia, oct[:min(4, len(oct))]...)
		for i := 0; i < 4; i++ {
			troparia = append(troparia, t6[i%len(t6)])
		}
		if glory == "" {
			glory = first(f.Octoechos, "lit.beatitudes.glory")
		}
		if now == "" {
			now = first(f.Octoechos, "lit.beatitudes.now")
		}
	} else {
		all := append(ode(3), ode(6)...)
		if len(all) == 0 {
			return nil, fmt.Errorf("Triodion %s: no 3rd and 6th odes for the beatitudes", f.TriodionDay)
		}
		for i := 0; i < 8; i++ {
			troparia = append(troparia, all[i%len(all)])
		}
	}
	var items []SlotItem
	for i := 0; i < plain; i++ {
		items = append(items, verse(i))
	}
	for i, t := range troparia {
		items = append(items, verse(plain+i), choir(t))
	}
	if glory != "" {
		items = append(items, choir("common.glory"), choir(glory))
	} else {
		items = append(items, choir("common.glory"))
	}
	if now != "" {
		return append(items, choir("common.and_now"), choir(now)), nil
	}
	return append(items, choir("common.and_now")), nil
}

// triodionTroparia: the hymns after the entrance (see triodionLiturgySlot).
func triodionTroparia(f Facts) ([]SlotItem, error) {
	tu := func(k string) string { return f.Triodion[k] }
	k := tu("kontakion")
	if k == "" {
		return nil, fmt.Errorf("Triodion %s: no kontakion", f.TriodionDay)
	}
	switch f.TriodionDay {
	case "sat_meatfare":
		t, th := tu("troparion"), tu("theotokion")
		if t == "" || th == "" {
			return nil, fmt.Errorf("Triodion %s: no troparion or theotokion", f.TriodionDay)
		}
		return []SlotItem{choir(t), choir("common.glory"), choir(k), choir("common.and_now"), choir(th)}, nil
	case "sat_cheesefare":
		t := tu("troparion")
		if t == "" {
			return nil, fmt.Errorf("Triodion %s: no troparion", f.TriodionDay)
		}
		items := []SlotItem{choir(t)}
		tm := litTemple(f, nil)
		if tm != nil {
			items = append(items, choir(tm.Troparion))
			if tm.Kind != TempleTheotokos {
				items = append(items, choir(tm.Kontakion))
			}
		}
		now := "ch.kontakion.protection"
		if tm != nil && tm.Kind == TempleTheotokos {
			now = tm.Kontakion
		}
		return append(items, choir("common.glory"), choir(k), choir("common.and_now"), choir(now)), nil
	}
	res, resK := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	if res == "" || resK == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion or kontakion", f.Day.Tone)
	}
	var saintT, saintK []string
	if f.TriodionDay == "cheesefare" { // the Menaion is sung with the Triodion
		saintT, saintK = mainMenaion(f, "troparion"), mainMenaion(f, "kontakion")
	}
	tm := litTemple(f, saintT)
	items := []SlotItem{choir(res)}
	if tm != nil {
		items = append(items, choir(tm.Troparion))
	}
	if len(saintT) > 0 {
		items = append(items, choir(saintT[0]))
	}
	items = append(items, choir(resK))
	if tm != nil && tm.Kind != TempleTheotokos {
		items = append(items, choir(tm.Kontakion))
	}
	if len(saintK) > 0 {
		items = append(items, choir(saintK[0]))
	}
	now := "ch.kontakion.protection"
	if tm != nil && tm.Kind == TempleTheotokos {
		now = tm.Kontakion
	}
	return append(items, choir("common.glory"), choir(k), choir("common.and_now"), choir(now)), nil
}

// triodionReadings: the day's (the lectionary's: on these days the
// Triodion's), then the departed's or the fathers'.
func triodionReadings(f Facts, which string) ([]SlotItem, error) {
	l, err := lection(f, which)
	if err != nil {
		return nil, err
	}
	key := map[string]string{"apostle": "ap", "gospel": "gos"}[which]
	list := []string{fmt.Sprintf("lect.%s.%d.%d", key, l.Week, int(l.Weekday))}
	extra := map[string]map[string]string{
		"sat_meatfare":   {"apostle": departedApostle, "gospel": departedGospel},
		"sat_cheesefare": {"apostle": "lect.zach.ap.213", "gospel": "lect.zach.mt.43"},
	}[f.TriodionDay][which]
	if extra != "" {
		list = append(list, extra)
	}
	var items []SlotItem
	if which == "apostle" {
		items = append(items, litanist("lit.wisdom"), reader(list[0]+".announce"), litanist("lit.let_us_attend"), reader(list[0]))
		for _, r := range list[1:] {
			items = append(items, reader(r))
		}
		return items, nil
	}
	items = append(items, litanist(list[0]+".announce"), choir("sl.lit.glory_to_thee_o_lord"), priest("lit.let_us_attend"), litanist(list[0]))
	for _, r := range list[1:] {
		items = append(items, litanist(r))
	}
	return items, nil
}
