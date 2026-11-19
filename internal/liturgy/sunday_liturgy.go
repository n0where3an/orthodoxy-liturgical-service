package liturgy

import (
	"fmt"
	"strings"
)

// Selectors of the Sunday liturgy, after the Octoechos and «Богослужебные
// указания». Part "sunday": with a saint (with or without a sign);
// "sunday_feast": a twelve feast of the Theotokos; "sunday_afterfeast": an
// afterfeast or apodosis. The saint's or the feast's prokeimenon, alleluia,
// readings and communion verse follow those of the Sunday.
//
//   - the typical antiphons; the beatitudes: without the Menaion the
//     Octoechos on 6 (after «Блажени милостивии»), with a saint the
//     Octoechos on 4 and the saint's (the Menaion's beatitudes, else the
//     3rd ode of its canon) on 4; «Слава» the Trinity hymn and «И ныне» the
//     theotokion of the Octoechos;
//   - «Приидите… воскресый из мертвых»;
//   - the resurrectional troparion, the temple's, the saint's; the
//     resurrectional kontakion, the temple's (not in a church of the
//     Theotokos); «Слава» the saint's kontakion; «И ныне» the kontakion of a
//     church of the Theotokos, else «Предстательство» («Со святыми упокой»
//     is not sung on Sunday);
//   - the prokeimenon and alleluia of the tone (and the saint's if he has
//     them), the Sunday readings of the week, «Хвалите Господа с небес».
func sundayLiturgySlot(sel SlotSelector, f Facts) ([]SlotItem, bool, error) {
	if sel.Part == "sunday_special" {
		if items, ok, err := specialSundayLiturgySlot(sel, f); ok || err != nil {
			return items, ok, err
		}
	}
	oc := func(k string) string { return first(f.Octoechos, k) }
	switch sel.From {
	case "lit_antiphon":
		if sel.Section != "3" {
			return nil, false, nil
		}
		items, err := sundayBeatitudes(f)
		return items, true, err
	case "lit_entrance":
		return []SlotItem{choir("ch.lit.come_let_us_worship.sunday")}, true, nil
	case "lit_troparia":
		var items []SlotItem
		var err error
		switch sel.Part {
		case "sunday_feast":
			items, err = sundayFeastTroparia(f)
		case "sunday_afterfeast":
			items, err = sundayAfterfeastTroparia(f)
		default:
			items, err = sundayTroparia(f)
		}
		return items, true, err
	case "lit_trisagion":
		// On the apodosis the feast's hymn instead of the Trisagion («Кресту Твоему»).
		if sel.Part != "sunday_afterfeast" || f.Feast == nil || !f.Feast.IsApodosis(f.Day) {
			return nil, false, nil
		}
		if t := feastMain(f, "lit.trisagion"); t != "" {
			return []SlotItem{{Ref: t, Repeat: 3, Performer: Choir}, choir("common.glory_now"), choir(t)}, true, nil
		}
		return nil, false, nil
	case "lit_truly_meet":
		if sel.Part != "sunday_afterfeast" {
			return nil, false, nil
		}
		items, err := afterfeastZadostoinik(f)
		return items, true, err
	case "lit_prokeimena":
		p, v := oc("lit.prokeimenon"), oc("lit.prokeimenon.verse")
		if p == "" {
			return nil, true, fmt.Errorf("Octoechos tone %d: no Sunday prokeimenon", f.Day.Tone)
		}
		items := prokeimenonBlock(f.Day.Tone, p, v)
		if men := firstMain(f, "lit.prokeimenon"); men != "" && f.Tones[men] > 0 {
			items = append(items, reader(fmt.Sprintf("lit.prokeimenon.tone.%d", f.Tones[men])), reader(men), choir(men))
		}
		return items, true, nil
	case "lit_alleluia":
		// lit.alleluia.verse.1, .2 (one kind, numbered).
		vs := f.Octoechos["lit.alleluia.verse"]
		if len(vs) == 0 {
			return nil, true, fmt.Errorf("Octoechos tone %d: no Sunday alleluia", f.Day.Tone)
		}
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", f.Day.Tone)), choir("common.alleluia.thrice"),
			reader(vs[0]), choir("common.alleluia.thrice")}
		if len(vs) > 1 {
			items = append(items, reader(vs[1]), choir("common.alleluia.thrice"))
		}
		if men := firstMain(f, "lit.alleluia.verse"); men != "" {
			items = append(items, reader(men), choir("common.alleluia.thrice"))
		}
		return items, true, nil
	case "lit_kinonik":
		items := []SlotItem{choir("sl.kinonik.sun")}
		if men := firstMain(f, "lit.kinonik"); men != "" {
			items = append(items, choir(men))
		}
		return append(items, choir("common.alleluia.thrice")), true, nil
	}
	return nil, false, nil
}

// sundayFeastTroparia: a twelve feast of the Theotokos on Sunday — the
// resurrectional troparion and the feast's; «Слава» the resurrectional
// kontakion, «И ныне» the feast's (Типикон).
func sundayFeastTroparia(f Facts) ([]SlotItem, error) {
	res, kon := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	troparia, kontakia := mainMenaion(f, "troparion"), mainMenaion(f, "kontakion")
	if res == "" || kon == "" || len(troparia) == 0 || len(kontakia) == 0 {
		return nil, fmt.Errorf("no resurrectional or festal troparion or kontakion (tone %d)", f.Day.Tone)
	}
	return []SlotItem{choir(res), choir(troparia[0]), choir("common.glory"), choir(kon), choir("common.and_now"), choir(kontakia[0])}, nil
}

// sundayAfterfeastTroparia: Sunday in an afterfeast — the resurrectional
// troparion, the feast's, the saint's; the resurrectional kontakion; «Слава»
// the saint's kontakion, «И ныне» the feast's (without a saint: «Слава, и
// ныне» the feast's). The temple's are not sung in the afterfeast.
func sundayAfterfeastTroparia(f Facts) ([]SlotItem, error) {
	res, kon := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	if res == "" || kon == "" || f.Feast == nil {
		return nil, fmt.Errorf("no resurrectional troparion or kontakion (tone %d), or no feast", f.Day.Tone)
	}
	saint := func(kind, feastID string) string {
		for _, id := range mainMenaion(f, kind) {
			if id != feastID && !f.ofFeast(id) {
				return id
			}
		}
		return ""
	}
	items := []SlotItem{choir(res), choir(f.Feast.Troparion)}
	if t := saint("troparion", f.Feast.Troparion); t != "" {
		items = append(items, choir(t))
	}
	items = append(items, choir(kon))
	if k := saint("kontakion", f.Feast.Kontakion); k != "" {
		return append(items, choir("common.glory"), choir(k), choir("common.and_now"), choir(f.Feast.Kontakion)), nil
	}
	return append(items, choir("common.glory_now"), choir(f.Feast.Kontakion)), nil
}

var nativityTheophanyZadostoinik = map[string]string{"nativity": "lit.zadostoinik.nativity", "theophany": "lit.zadostoinik.theophany"}

// afterfeastZadostoinik: in the afterfeast, the zadostoinik of the feast
// (the day's own if its Menaion names one): the feast's refrain and the
// irmos of the 9th ode of its first canon.
func afterfeastZadostoinik(f Facts) ([]SlotItem, error) {
	if f.Feast == nil {
		return nil, fmt.Errorf("no feast")
	}
	// The Nativity and the Theophany: «Величай, душе моя, честнейшую…» (the
	// Irmologion's refrain) and the irmos of the 9th ode of the second
	// canon («Любити убо нам», «Недоумеет всяк язык»).
	if z, ok := nativityTheophanyZadostoinik[f.Feast.ID]; ok {
		return []SlotItem{choir("ir.pripev." + f.Feast.ID + ".1.irmos"), choir(z)}, nil
	}
	z := firstMain(f, "lit.zadostoinik")
	if z == "" {
		z = feastMain(f, "lit.zadostoinik")
	}
	if z == "" {
		z = feastMain(f, "matins.canon.1.ode.9.irmos")
	}
	if z == "" {
		return nil, fmt.Errorf("no zadostoinik of the feast %s", f.Feast.ID)
	}
	if zadostoinikRefrain[f.Feast.ID] {
		return []SlotItem{choir("lit.zadostoinik.refrain." + f.Feast.ID), choir(z)}, nil
	}
	return []SlotItem{choir(z)}, nil
}

func sundayBeatitudes(f Facts) ([]SlotItem, error) {
	oct := f.Octoechos["lit.beatitudes"]
	glory, now := first(f.Octoechos, "lit.beatitudes.glory"), first(f.Octoechos, "lit.beatitudes.now")
	if len(oct) < 6 || glory == "" || now == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no Sunday beatitudes", f.Day.Tone)
	}
	men := mainMenaion(f, "lit.beatitudes")
	if len(men) == 0 {
		men = mainMenaion(f, "matins.canon.1.ode.3.troparion")
	}
	verse := func(i int) SlotItem { return choir(fmt.Sprintf("ch.typika.beatitudes.%d", i)) }
	var troparia []string
	plain := 5 // verses 0..4 without troparia: on 6
	if len(men) == 0 {
		troparia = oct[:6]
	} else {
		plain = 3 // on 8: the Octoechos on 4, the saint's on 4
		troparia = append(troparia, oct[:4]...)
		for i := 0; i < 4; i++ {
			troparia = append(troparia, men[i%len(men)])
		}
	}
	var items []SlotItem
	for i := 0; i < plain; i++ {
		items = append(items, verse(i))
	}
	for i, t := range troparia {
		items = append(items, verse(plain+i), choir(t))
	}
	return append(items, choir("common.glory"), choir(glory), choir("common.and_now"), choir(now)), nil
}

func sundayTroparia(f Facts) ([]SlotItem, error) {
	res, kon := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	if res == "" || kon == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion or kontakion", f.Day.Tone)
	}
	troparia, kontakia := mainMenaion(f, "troparion"), mainMenaion(f, "kontakion")
	t := litTemple(f, troparia)
	items := []SlotItem{choir(res)}
	if t != nil {
		items = append(items, choir(t.Troparion))
	}
	if len(troparia) > 0 {
		items = append(items, choir(troparia[0]))
	}
	items = append(items, choir(kon))
	if t != nil && t.Kind != TempleTheotokos {
		items = append(items, choir(t.Kontakion))
	}
	now := "ch.kontakion.protection"
	if t != nil && t.Kind == TempleTheotokos {
		now = t.Kontakion
	}
	if len(kontakia) > 0 {
		return append(items, choir("common.glory"), choir(kontakia[0]), choir("common.and_now"), choir(now)), nil
	}
	return append(items, choir("common.glory_now"), choir(now)), nil
}

// feastMain: the first unit of a kind of the main service of the day of the
// feast (in its afterfeast).
func feastMain(f Facts, kind string) string {
	if f.Feast == nil {
		return ""
	}
	prefix := fmt.Sprintf("me.%02d.%02d.", int(f.Feast.from.Month), f.Feast.from.Day)
	for _, id := range f.FeastMenaion[kind] {
		if strings.HasPrefix(id, prefix) {
			return id
		}
	}
	return ""
}
