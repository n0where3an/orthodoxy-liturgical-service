package liturgy

import (
	"fmt"
)

// The liturgy of a Sunday with readings of its own (specialSunday; part
// "sunday_special"), by the rubrics of the Menaion:
//
//   - before the Exaltation (09.08: «прокимен, апсл, аллилуиа, и еиглие
//     прежде недели пред воздвижением, и ряд под зачало. таже стаго.
//     прокимен гласа оставляется»), before and after the Theophany (01.01,
//     01.07), after the Exaltation (09.15: «прокимен праздника»): the
//     Sunday's prokeimenon instead of the tone's, its readings before the
//     ordinary ones (when the lectionary appoints them) and the saint's;
//   - of the Fathers (12.18: «ряд же недели оставляется»): the resurrectional
//     troparion and the Fathers'; «Слава» the Fathers' kontakion, «И ныне» the
//     forefeast's (out of it «Слава, и ныне» the Fathers'); their
//     prokeimenon, readings, alleluia; not the saint of the day;
//   - after the Nativity (12.26): the resurrectional troparion, the feast's,
//     the saints' (David, Joseph, James); «Слава» their kontakion, «И ныне»
//     the feast's; the prokeimenon of the tone and theirs, their readings
//     and alleluia, then the ordinary readings;
//
// the communion verse «Хвалите Господа» and the Sunday's own («Радуйтеся,
// праведнии», «Явися благодать»), else the saint's; in an afterfeast the
// feast's zadostoinik.
func specialSundayLiturgySlot(sel SlotSelector, f Facts) ([]SlotItem, bool, error) {
	su := func(k string) string { return f.SpecialUnits[k] }
	noSaint := f.Special == "fathers" || f.Special == "after_nativity"
	switch sel.From {
	case "lit_troparia":
		items, err := specialSundayTroparia(f)
		return items, true, err
	case "lit_truly_meet":
		if fT, _ := feastHymns(f); f.Feast == nil || fT == "" {
			return []SlotItem{choir("ch.lit.it_is_truly_meet")}, true, nil
		}
		items, err := afterfeastZadostoinik(f)
		return items, true, err
	case "lit_prokeimena":
		own, verse := su("lit.prokeimenon"), su("lit.prokeimenon.verse")
		var items []SlotItem
		keepTone := f.Special == "after_nativity" || own == ""
		if own == "" {
			// «прокимен праздника» (after the Exaltation).
			if p := feastMain(f, "lit.prokeimenon"); p != "" && f.Tones[p] > 0 {
				own, verse, keepTone = p, feastMain(f, "lit.prokeimenon.verse"), false
			}
		}
		if keepTone {
			p, v := first(f.Octoechos, "lit.prokeimenon"), first(f.Octoechos, "lit.prokeimenon.verse")
			if p == "" {
				return nil, true, fmt.Errorf("Octoechos tone %d: no Sunday prokeimenon", f.Day.Tone)
			}
			items = prokeimenonBlock(f.Day.Tone, p, v)
		}
		if own != "" {
			if f.Tones[own] == 0 {
				return nil, true, fmt.Errorf("Sunday %s: no tone of its prokeimenon", f.Special)
			}
			if len(items) == 0 {
				items = prokeimenonBlock(f.Tones[own], own, verse)
			} else {
				items = append(items, prokeimenonOnce(f.Tones[own], own)...)
			}
		}
		if men := firstMain(f, "lit.prokeimenon"); men != "" && !noSaint && f.Tones[men] > 0 {
			items = append(items, prokeimenonOnce(f.Tones[men], men)...)
		}
		return items, true, nil
	case "lit_alleluia":
		v1, v2 := su("lit.alleluia.verse.1"), su("lit.alleluia.verse.2")
		if v1 == "" {
			return nil, false, nil // the tone's (after the Exaltation)
		}
		tone := f.Tones[v1]
		if tone == 0 {
			return nil, true, fmt.Errorf("Sunday %s: no tone of its alleluia", f.Special)
		}
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", tone)), choir("common.alleluia.thrice"),
			reader(v1), choir("common.alleluia.thrice")}
		if v2 != "" {
			items = append(items, reader(v2), choir("common.alleluia.thrice"))
		}
		if men := firstMain(f, "lit.alleluia.verse"); men != "" && !noSaint {
			items = append(items, reader(men), choir("common.alleluia.thrice"))
		}
		return items, true, nil
	case "lit_apostles", "lit_gospels":
		which := map[string]string{"lit_apostles": "apostle", "lit_gospels": "gospel"}[sel.From]
		items, err := specialSundayReadings(f, which, noSaint)
		return items, true, err
	case "lit_kinonik":
		items := []SlotItem{choir("sl.kinonik.sun")}
		if k := su("lit.kinonik"); k != "" {
			items = append(items, choir(k))
		} else if men := firstMain(f, "lit.kinonik"); men != "" && !noSaint {
			items = append(items, choir(men))
		}
		return append(items, choir("common.alleluia.thrice")), true, nil
	}
	return nil, false, nil
}

// specialSundayReadings: the Sunday's own reading, the ordinary one when the
// lectionary appoints it («и ряд под зачало»), the saint's; only the first
// is announced.
func specialSundayReadings(f Facts, which string, noSaint bool) ([]SlotItem, error) {
	var list []string
	if own := f.SpecialUnits["lit."+which]; own != "" {
		list = append(list, own)
	}
	if !f.Day.Lections.Omitted {
		l, err := lection(f, which)
		if err != nil {
			return nil, err
		}
		key := map[string]string{"apostle": "ap", "gospel": "gos"}[which]
		list = append(list, fmt.Sprintf("lect.%s.%d.%d", key, l.Week, int(l.Weekday)))
	}
	if men := firstMain(f, "lit."+which); men != "" && !noSaint {
		list = append(list, men)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("Sunday %s: no %s", f.Special, which)
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

// specialSundayTroparia: the troparia and kontakia after the entrance.
func specialSundayTroparia(f Facts) ([]SlotItem, error) {
	res, kon := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	if res == "" || kon == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion or kontakion", f.Day.Tone)
	}
	fT, fK := feastHymns(f)
	sp := func(k string) (string, error) {
		if id := f.SpecialUnits[k]; id != "" {
			return id, nil
		}
		return "", fmt.Errorf("Sunday %s: no %s in the corpus", f.Special, k)
	}
	switch f.Special {
	case "fathers":
		t, err := sp("troparion")
		if err != nil {
			return nil, err
		}
		if fK == "" {
			k, err := sp("kontakion")
			if err != nil {
				return nil, err
			}
			return []SlotItem{choir(res), choir(t), choir(kon), choir("common.glory_now"), choir(k)}, nil
		}
		k, err := sp("kontakion.forefeast")
		if err != nil {
			return nil, err
		}
		items := []SlotItem{choir(res), choir(t)}
		if fT != "" {
			items = append(items, choir(fT))
		}
		return append(items, choir(kon), choir("common.glory"), choir(k), choir("common.and_now"), choir(fK)), nil
	case "after_nativity":
		if f.Feast == nil {
			break
		}
		t, err := sp("troparion")
		if err != nil {
			return nil, err
		}
		k, err := sp("kontakion")
		if err != nil {
			return nil, err
		}
		return []SlotItem{choir(res), choir(f.Feast.Troparion), choir(t), choir(kon),
			choir("common.glory"), choir(k), choir("common.and_now"), choir(f.Feast.Kontakion)}, nil
	}
	if f.Feast != nil {
		return sundayAfterfeastTroparia(f)
	}
	if fK != "" {
		// A forefeast: the resurrectional troparion, the forefeast's (when
		// the Menaion gives it), the saint's; «Слава» the saint's kontakion,
		// «И ныне» the forefeast's.
		items := []SlotItem{choir(res)}
		if fT != "" {
			items = append(items, choir(fT))
		}
		if s := saintHymns(f, "troparion", fT); len(s) > 0 {
			items = append(items, choir(s[0]))
		}
		items = append(items, choir(kon))
		if s := saintHymns(f, "kontakion", fK); len(s) > 0 {
			return append(items, choir("common.glory"), choir(s[0]), choir("common.and_now"), choir(fK)), nil
		}
		return append(items, choir("common.glory_now"), choir(fK)), nil
	}
	return sundayTroparia(f)
}
