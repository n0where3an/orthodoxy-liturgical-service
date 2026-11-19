package liturgy

import "fmt"

// The liturgy on a weekday (Monday..Saturday) of the afterfeast of a fixed
// twelve feast and on its apodosis (Типикон, гл. 52; the month calendar;
// «Богослужебные указания»). Part "afterfeast":
//
//   - the typical antiphons; the beatitudes of the day's Menaion («блаженна
//     праздника, песнь 3-я… и святаго, песнь 6-я»), else the 3rd ode of its
//     first canon (the feast's) and the 6th of the second (the saint's);
//   - at a feast of the Lord «Приидите… спаси ны, Сыне Божий, [the feast's
//     refrain], поющия Ти» («сице поем и до отдания праздника»);
//   - the troparia below;
//   - the prokeimenon, alleluia and communion verse of the feast and of the
//     saint; the Apostle and Gospel of the day and of the saint; the
//     zadostoinik of the feast. The Octoechos is not sung: the day's hymns
//     are not said («дне же не глаголется, зане октоих не поется»).

// Feasts of the Lord with the entrance refrain (lit.entrance.afterfeast.<id>).
var afterfeastEntrance = map[string]bool{"nativity": true, "theophany": true, "meeting": true,
	"transfiguration": true, "exaltation": true}

func afterfeastLiturgySlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	if f.Feast == nil {
		return nil, fmt.Errorf("no feast in the afterfeast")
	}
	switch sel.From {
	case "lit_antiphon":
		if sel.Section == "3" {
			return afterfeastBeatitudes(f)
		}
		return litAntiphon(f, sel.Section, "")
	case "lit_entrance":
		if afterfeastEntrance[f.Feast.ID] {
			return []SlotItem{choir("lit.entrance.afterfeast." + f.Feast.ID)}, nil
		}
		return []SlotItem{choir("ch.lit.come_let_us_worship.weekday")}, nil
	case "lit_troparia":
		return afterfeastLitTroparia(f), nil
	case "lit_prokeimena":
		p := feastMain(f, "lit.prokeimenon")
		if p == "" || f.Tones[p] == 0 {
			return nil, fmt.Errorf("no prokeimenon of the feast %s", f.Feast.ID)
		}
		items := prokeimenonBlock(f.Tones[p], p, feastMain(f, "lit.prokeimenon.verse"))
		if men := firstMain(f, "lit.prokeimenon"); men != "" && f.Tones[men] > 0 {
			items = append(items, prokeimenonOnce(f.Tones[men], men)...) // the saint's
		}
		return items, nil
	case "lit_alleluia":
		vs := feastAll(f, "lit.alleluia.verse")
		if len(vs) == 0 || f.Tones[vs[0]] == 0 {
			return nil, fmt.Errorf("no alleluia of the feast %s", f.Feast.ID)
		}
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", f.Tones[vs[0]])), choir("common.alleluia.thrice")}
		for _, v := range vs {
			items = append(items, reader(v), choir("common.alleluia.thrice"))
		}
		if men := firstMain(f, "lit.alleluia.verse"); men != "" {
			items = append(items, reader(men), choir("common.alleluia.thrice")) // the saint's
		}
		return items, nil
	case "lit_apostles":
		return litReadings(f, false, "apostle")
	case "lit_gospels":
		return litReadings(f, false, "gospel")
	case "lit_kinonik":
		k := feastMain(f, "lit.kinonik")
		if k == "" {
			return nil, fmt.Errorf("no communion verse of the feast %s", f.Feast.ID)
		}
		items := []SlotItem{choir(k)}
		if men := firstMain(f, "lit.kinonik"); men != "" && men != k {
			items = append(items, choir(men))
		}
		return append(items, choir("common.alleluia.thrice")), nil
	case "lit_truly_meet":
		return afterfeastZadostoinik(f)
	case "lit_trisagion":
		return liturgySlot(SlotSelector{From: "lit_trisagion"}, f)
	}
	return nil, fmt.Errorf("unknown afterfeast liturgy selector %q", sel.From)
}

// feastAll: all the ids of kind in the feast day's Menaion service.
func feastAll(f Facts, kind string) []string {
	var out []string
	if first := feastMain(f, kind); first != "" {
		prefix := first[:len(fmt.Sprintf("me.%02d.%02d.", int(f.Feast.from.Month), f.Feast.from.Day))]
		for _, id := range f.FeastMenaion[kind] {
			if len(id) > len(prefix) && id[:len(prefix)] == prefix {
				out = append(out, id)
			}
		}
	}
	return out
}

// afterfeastBeatitudes: the Menaion's beatitudes of the day, else the 3rd
// ode of the feast's canon and the 6th of the saint's, on 8.
func afterfeastBeatitudes(f Facts) ([]SlotItem, error) {
	if len(mainMenaion(f, "lit.beatitudes")) > 0 {
		return litBeatitudes(f, "afterfeast")
	}
	troparia := mainMenaion(f, "matins.canon.1.ode.3.troparion")
	troparia = append(troparia, mainMenaion(f, "matins.canon.2.ode.6.troparion")...)
	if len(troparia) == 0 {
		return litBeatitudes(f, "")
	}
	var items []SlotItem
	verse := func(i int) SlotItem { return choir(fmt.Sprintf("ch.typika.beatitudes.%d", i)) }
	for i := 0; i <= 4; i++ {
		items = append(items, verse(i))
	}
	k := 0
	next := func() SlotItem { t := troparia[k%len(troparia)]; k++; return choir(t) }
	for i := 5; i <= 10; i++ {
		items = append(items, verse(i), next())
	}
	return append(items, choir("common.glory"), next(), choir("common.and_now"), next()), nil
}

// afterfeastLitTroparia (Типикон, гл. 52): in the afterfeast of a feast of
// the Lord the feast's troparion, the troparion of a church of the Theotokos
// or of a saint, the saint's; the kontakion of a church of the Theotokos,
// «Слава» the saint's, «И ныне» the feast's (not the hymns of a church of the
// Lord). Of a feast of the Theotokos: the troparion of a church of the Lord,
// the feast's, the saint's; the kontakion of a church of the Lord, «Слава»
// the saint's, «И ныне» the feast's (not those of a church of the
// Theotokos). Without the saint's kontakion «Слава, и ныне» the feast's.
func afterfeastLitTroparia(f Facts) []SlotItem {
	feastT, feastK := f.Feast.Troparion, f.Feast.Kontakion
	saintT, saintK := saintHymn(f, "troparion"), saintHymn(f, "kontakion")
	t := f.Temple
	if t != nil && (t.Troparion == feastT || f.Feast.IsFeastDay(f.Day)) {
		t = nil // the patronal feast itself
	}
	lordFeast := afterfeastEntrance[f.Feast.ID]
	var items []SlotItem
	var templeK string
	switch {
	case t == nil:
	case lordFeast && t.Kind == TempleTheotokos:
		templeK = t.Kontakion
	case lordFeast && t.Kind == TempleSaint:
	case !lordFeast && t.Kind == TempleLord:
		items = append(items, choir(t.Troparion))
		templeK = t.Kontakion
		t = nil
	default:
		t = nil // a church of the Lord at a feast of the Lord, of the Theotokos at hers
	}
	items = append(items, choir(feastT))
	if t != nil {
		items = append(items, choir(t.Troparion))
	}
	if saintT != "" {
		items = append(items, choir(saintT))
	}
	if templeK != "" {
		items = append(items, choir(templeK))
	}
	if saintK != "" {
		return append(items, choir("common.glory"), choir(saintK), choir("common.and_now"), choir(feastK))
	}
	return append(items, choir("common.glory_now"), choir(feastK))
}
