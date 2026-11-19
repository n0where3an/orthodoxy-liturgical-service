package liturgy

import (
	"fmt"
	"strconv"
)

// Selectors of weekday matins in the afterfeast of a fixed twelve feast (not
// its apodosis), as the Menaion of each afterfeast day gives it («на утрени,
// по а-й стихологии седален праздника… канон праздника со ирмосом на 8, и
// святаго на 4… по 3-й песни кондак святаго, седален… по 6-й песни кондак и
// икос праздника… светилен праздника… на стиховне стихиры праздника»):
//
//   - «Бог Господь»: the feast's troparion twice, «Слава» the saint's, «И
//     ныне» the feast's; the feast's thrice when the saint has none;
//   - the sessional hymns after the kathismata: the Menaion's (the feast's),
//     else the feast day's, else the Octoechos's;
//   - the canon: the Menaion's canons only, the feast's with its irmos on 8,
//     the saint's on 4 (the Octoechos is not sung); the katavasia after every
//     ode, by the season or the irmoi of the feast's canon;
//   - after the 3rd ode the saint's kontakion and the sessional hymn; after
//     the 6th the feast's kontakion and ikos;
//   - the exapostilaria of the day, «Слава, и ныне» the feast's;
//   - the aposticha: the Menaion's stichera when it gives them (matins_aposticha).
//
// «Честнейшую» is sung (Типикон, гл. 20: «всегда поется чстнейшую… в
// предпразднствах, и попразднствах»).

func afterfeastMatinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	if f.Feast == nil {
		return nil, fmt.Errorf("no feast in the afterfeast on %v", f.Day.Date)
	}
	switch sel.From {
	case "af_troparia":
		return afTroparia(f), nil
	case "af_sedalny":
		k, _ := strconv.Atoi(sel.Section)
		return afSedalny(f, k)
	case "af_canon":
		return afCanon(f, sel.Section)
	case "af_sedalen":
		return afSedalen(f)
	case "af_kontakion":
		return afKontakion(f)
	case "af_exapostilaria":
		return afExapostilaria(f)
	}
	return nil, fmt.Errorf("unknown afterfeast matins selector %q", sel.From)
}

func afTroparia(f Facts) []SlotItem {
	feast := f.Feast.Troparion
	if t := saintHymn(f, "troparion"); t != "" {
		return []SlotItem{{Ref: feast, Repeat: 2, Performer: Choir}, reader("common.glory"), choir(t),
			reader("common.and_now"), choir(feast)}
	}
	return []SlotItem{{Ref: feast, Repeat: 3, Performer: Choir}}
}

// afSedalny: the sessional hymns after kathisma k; one hymn is sung again
// at «Слава, и ныне» («слава, и ныне, тойже»).
func afSedalny(f Facts, k int) ([]SlotItem, error) {
	kind := fmt.Sprintf("matins.k_sedalen.%d", k)
	hymns := mainMenaion(f, kind)
	if len(hymns) == 0 {
		if h := feastMain(f, kind); h != "" {
			hymns = []string{h}
		}
	}
	if len(hymns) == 0 {
		return matinsSedalny(f, k)
	}
	var items []SlotItem
	for _, h := range hymns {
		items = append(items, SlotItem{Ref: h})
	}
	if th := theotokionFor(f, kind); th != "" {
		return append(items, reader("common.glory_now"), SlotItem{Ref: th}), nil
	}
	return append(items, reader("common.glory_now"), SlotItem{Ref: hymns[len(hymns)-1]}), nil
}

// afCanon: the Menaion's canons — the first (the feast's) with its irmos
// on 8, the others on 4 — and the katavasia after every ode.
func afCanon(f Facts, section string) ([]SlotItem, error) {
	odes, err := parseOdes(section)
	if err != nil {
		return nil, err
	}
	var canons []string
	for c := 1; c <= 3; c++ {
		b := fmt.Sprintf("matins.canon.%d", c)
		if len(mainMenaion(f, b+".ode.1.troparion")) > 0 || len(mainMenaion(f, b+".ode.9.troparion")) > 0 {
			canons = append(canons, b)
		}
	}
	if len(canons) == 0 {
		return nil, fmt.Errorf("no Menaion canon for %02d.%02d in the corpus", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	var items []SlotItem
	for _, o := range odes {
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		start := len(items)
		for i, c := range canons {
			b := fmt.Sprintf("%s.ode.%d", c, o)
			trop := mainMenaion(f, b+".troparion")
			if len(trop) == 0 {
				continue
			}
			irmos, on := "", 4
			if i == 0 {
				irmos, on = firstMain(f, b+".irmos"), 8
			}
			items = append(items, odeOn(irmos, trop, theotokionFor(f, b), on)...)
		}
		if len(f.OdeVerses) > 0 { // -typikon: the verses of the biblical ode
			items = append(items[:start], withOdeVerses(f, f.OdeSet, o, append([]SlotItem(nil), items[start:]...))...)
		}
		k := smKatavasia(f, "afterfeast", o)
		if k == "" {
			k = firstMain(f, fmt.Sprintf("%s.ode.%d.irmos", canons[0], o))
		}
		if k != "" {
			items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: k})
		}
	}
	return items, nil
}

// afSedalen: after the 3rd ode the saint's kontakion, then the sessional
// hymn(s) of the day.
func afSedalen(f Facts) ([]SlotItem, error) {
	var items []SlotItem
	if k := saintHymn(f, "kontakion"); k != "" {
		items = append(items, reader(k))
	}
	if len(mainMenaion(f, "matins.sedalen")) == 0 {
		if len(items) == 0 {
			return nil, fmt.Errorf("no saint's kontakion nor sessional hymn after the 3rd ode for %02d.%02d",
				int(f.Day.Menaion.Month), f.Day.Menaion.Day)
		}
		return items, nil
	}
	sed, err := matinsMenaionSedalen(f)
	if err != nil {
		return nil, err
	}
	return append(items, sed...), nil
}

// afKontakion: after the 6th ode the feast's kontakion and ikos.
func afKontakion(f Facts) ([]SlotItem, error) {
	items := []SlotItem{reader(f.Feast.Kontakion)}
	ikos := feastMain(f, "matins.ikos")
	if ikos == "" {
		ikos = firstMain(f, "matins.ikos")
	}
	if ikos != "" {
		items = append(items, reader(ikos))
	}
	return items, nil
}

// afExapostilaria: the day's exapostilaria; «Слава, и ныне» the feast's —
// the day's last when it has more than one, else the feast day's (or the
// same again: «слава, и ныне, тойже»).
func afExapostilaria(f Facts) ([]SlotItem, error) {
	ex := mainMenaion(f, "matins.exapostilarion")
	feast := feastMain(f, "matins.exapostilarion")
	if len(ex) == 0 {
		if feast == "" {
			return nil, fmt.Errorf("no exapostilarion of the day nor of the feast")
		}
		return []SlotItem{{Ref: feast}, reader("common.glory_now"), {Ref: feast}}, nil
	}
	var items []SlotItem
	if len(ex) > 1 {
		for _, x := range ex[:len(ex)-1] {
			items = append(items, SlotItem{Ref: x})
		}
		return append(items, reader("common.glory_now"), SlotItem{Ref: ex[len(ex)-1]}), nil
	}
	items = append(items, SlotItem{Ref: ex[0]}, reader("common.glory_now"))
	if feast == "" {
		feast = ex[0]
	}
	return append(items, SlotItem{Ref: feast}), nil
}
