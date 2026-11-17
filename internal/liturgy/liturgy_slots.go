package liturgy

import (
	"fmt"
	"time"
)

// Selectors of the Liturgy. Part says whose service it is:
//
//   - "" — a weekday: the hymns and readings of the day; a Menaion saint's
//     own prokeimenon, alleluia, readings and communion verse are added after
//     the day's (Typikon: «дне и святаго»);
//   - "saint" — a saint with a sign (polyeleos, vigil) on a weekday: the
//     same, with the saint's troparion and kontakion after the entrance and
//     the beatitudes from the canon;
//   - "feast" — a twelve feast: everything of the feast, the festal
//     antiphons (feasts of the Lord) or the beatitudes from the canon, the
//     entrance verse, the zadostoinik.

// The twelve feasts whose zadostoinik refrain is in the corpus
// (lit.zadostoinik.refrain.<feast>).
var zadostoinikRefrain = map[string]bool{"nativity_of_theotokos": true, "exaltation": true, "entry_of_theotokos": true,
	"meeting": true, "annunciation": true, "transfiguration": true, "dormition": true}

type dayLiturgy struct {
	troparia, kontakia []string
	prokTone, alTone   int
}

// Troparia, kontakia and the tones of the prokeimenon and alleluia of the
// day (Часослов, «Указ о тропарех»; Служебник, «Прокимны и аллилуиа дневнии»).
var weekdayLiturgy = map[time.Weekday]dayLiturgy{
	time.Monday:    {[]string{"ch.troparion.day.mon"}, []string{"ch.kontakion.day.mon"}, 4, 5},
	time.Tuesday:   {[]string{"ch.troparion.day.tue"}, []string{"ch.kontakion.day.tue"}, 7, 4},
	time.Wednesday: {[]string{"ch.troparion.day.wed"}, []string{"ch.kontakion.day.wed"}, 3, 8},
	time.Thursday:  {[]string{"ch.troparion.day.thu", "ch.troparion.day.thu.2"}, []string{"ch.kontakion.day.thu", "ch.kontakion.day.thu.2"}, 8, 1},
	time.Friday:    {[]string{"ch.troparion.day.wed"}, []string{"ch.kontakion.day.wed"}, 7, 1},
}

func choir(ref string) SlotItem    { return SlotItem{Ref: ref, Performer: Choir} }
func litanist(ref string) SlotItem { return SlotItem{Ref: ref, Performer: Litanist} }
func priest(ref string) SlotItem   { return SlotItem{Ref: ref, Performer: Priest} }

func liturgySlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	feast := sel.Part == "feast"
	switch sel.From {
	case "lit_antiphon":
		return litAntiphon(f, sel.Section, sel.Part)
	case "lit_entrance":
		if e := firstMain(f, "lit.entrance"); e != "" && feast {
			return []SlotItem{choir(e)}, nil
		}
		return []SlotItem{choir("ch.lit.come_let_us_worship.weekday")}, nil
	case "lit_troparia":
		return litTroparia(f, sel.Part)
	case "lit_trisagion":
		if t := firstMain(f, "lit.trisagion"); t != "" && feast {
			return []SlotItem{{Ref: t, Repeat: 3, Performer: Choir}, choir("common.glory_now"), choir(t)}, nil
		}
		return []SlotItem{{Ref: "common.trisagion", Repeat: 3, Performer: Choir}, choir("common.glory_now"),
			choir("common.trisagion.immortal"), choir("common.trisagion")}, nil
	case "lit_prokeimena":
		return litProkeimena(f, feast)
	case "lit_apostles":
		return litReadings(f, feast, "apostle")
	case "lit_alleluia":
		return litAlleluia(f, feast)
	case "lit_gospels":
		return litReadings(f, feast, "gospel")
	case "lit_kinonik":
		return litKinonik(f, feast)
	case "lit_truly_meet":
		z := firstMain(f, "lit.zadostoinik")
		if z == "" && feast && f.Feast != nil && zadostoinikRefrain[f.Feast.ID] {
			// Not named in the rubric: the irmos of the 9th ode of the first canon.
			z = firstMain(f, "matins.canon.1.ode.9.irmos")
		}
		if z != "" && feast {
			// The zadostoinik: the refrain of the feast, then the irmos.
			if f.Feast != nil && zadostoinikRefrain[f.Feast.ID] {
				return []SlotItem{choir("lit.zadostoinik.refrain." + f.Feast.ID), choir(z)}, nil
			}
			return []SlotItem{choir(z)}, nil
		}
		return []SlotItem{choir("ch.lit.it_is_truly_meet")}, nil
	}
	return nil, fmt.Errorf("unknown liturgy selector %q", sel.From)
}

func dayOf(f Facts) (dayLiturgy, error) {
	d, ok := weekdayLiturgy[f.Day.Weekday]
	if !ok {
		return d, fmt.Errorf("no weekday hymns of the Liturgy on %v", f.Day.Weekday)
	}
	return d, nil
}

// litAntiphon: the typical antiphons (Psalms 102, 145, the beatitudes) or
// the festal antiphons of the Menaion.
func litAntiphon(f Facts, n, part string) ([]SlotItem, error) {
	if part == "feast" {
		if festal := mainMenaion(f, "lit.antiphon."+n); len(festal) > 0 {
			var items []SlotItem
			for _, id := range festal {
				items = append(items, choir(id))
			}
			return items, nil
		}
	}
	switch n {
	case "1":
		return []SlotItem{choir("ch.typika.psalm_102"), choir("common.glory_now"), choir("ch.typika.psalm_102.refrain")}, nil
	case "2":
		return []SlotItem{choir("common.glory"), choir("ch.typika.psalm_145"), choir("common.and_now"), choir("ch.typika.only_begotten")}, nil
	}
	return litBeatitudes(f, part)
}

// litBeatitudes: «Во Царствии Твоем» and the beatitudes; with a canon of the
// Menaion — on 8: troparia after «Блажени милостивии»… and at «Слава»,
// the theotokion at «И ныне».
func litBeatitudes(f Facts, part string) ([]SlotItem, error) {
	var items []SlotItem
	verse := func(i int) SlotItem { return choir(fmt.Sprintf("ch.typika.beatitudes.%d", i)) }
	troparia := mainMenaion(f, "lit.beatitudes")
	if part == "" || len(troparia) == 0 {
		for i := 0; i <= 10; i++ {
			items = append(items, verse(i))
		}
		return append(items, choir("common.glory_now")), nil
	}
	for i := 0; i <= 4; i++ {
		items = append(items, verse(i))
	}
	k := 0
	next := func() SlotItem { t := troparia[k%len(troparia)]; k++; return choir(t) }
	for i := 5; i <= 10; i++ {
		items = append(items, verse(i), next())
	}
	items = append(items, choir("common.glory"), next())
	th := firstMain(f, "lit.beatitudes.theotokion")
	if th == "" {
		return append(items, choir("common.and_now"), next()), nil
	}
	return append(items, choir("common.and_now"), choir(th)), nil
}

// templeOfDay: the dedications whose hymns are those of the weekday itself
// (Monday: the bodiless powers; Wednesday, Friday: the Cross; Thursday:
// St Nicholas) — not sung twice.
var templeOfDay = map[time.Weekday][]string{
	time.Monday: {"archangel_michael"}, time.Wednesday: {"exaltation"}, time.Thursday: {"nicholas"},
	time.Friday: {"exaltation"},
}

// litTemple: the temple whose troparion and kontakion are added after the
// entrance, nil when none is given or its hymns are already the day's.
func litTemple(f Facts, troparia []string) *Temple {
	t := f.Temple
	if t == nil {
		return nil
	}
	for _, id := range troparia {
		if id == t.Troparion {
			return nil // the patronal feast itself
		}
	}
	for _, id := range templeOfDay[f.Day.Weekday] {
		if id == t.ID {
			return nil
		}
	}
	return t
}

// litTroparia: the troparia and kontakia after the entrance. With a temple
// (Типикон; «Богослужебные указания»):
//
//   - weekday, a church of the Lord: the temple's troparion, the day's, the
//     saint's; the temple's kontakion, the day's, the saint's; «Слава» «Со
//     святыми упокой», «И ныне» «Предстательство»;
//   - of the Theotokos: the day's troparion, the temple's, the saint's; the
//     day's kontakion, the saint's; «Слава» «Со святыми упокой», «И ныне»
//     the temple's kontakion;
//   - of a saint: the day's troparion, the temple's, the saint's; the day's
//     kontakion, the temple's, the saint's; «Слава» «Со святыми упокой»,
//     «И ныне» «Предстательство»;
//   - a saint with a sign: the Lord's temple's troparion first, else after
//     the saint's; «Слава» the saint's kontakion, «И ныне» the kontakion of
//     a church of the Theotokos, else «Предстательство» (the temple's
//     kontakion before «Слава» in a church of the Lord or of a saint);
//   - at a twelve feast only the feast's.
func litTroparia(f Facts, part string) ([]SlotItem, error) {
	troparia, kontakia := mainMenaion(f, "troparion"), mainMenaion(f, "kontakion")
	switch part {
	case "feast":
		if len(troparia) == 0 || len(kontakia) == 0 {
			return nil, fmt.Errorf("no troparion or kontakion of the feast")
		}
		return []SlotItem{choir(troparia[0]), choir("common.glory_now"), choir(kontakia[0])}, nil
	case "saint":
		if len(troparia) == 0 || len(kontakia) == 0 {
			return nil, fmt.Errorf("no troparion or kontakion of the saint")
		}
		t := litTemple(f, troparia)
		var items []SlotItem
		if t != nil && t.Kind == TempleLord {
			items = append(items, choir(t.Troparion))
		}
		for _, tr := range troparia {
			items = append(items, choir(tr))
		}
		now := "ch.kontakion.protection"
		if t != nil {
			switch t.Kind {
			case TempleTheotokos:
				items = append(items, choir(t.Troparion))
				now = t.Kontakion
			case TempleSaint:
				items = append(items, choir(t.Troparion), choir(t.Kontakion))
			case TempleLord:
				items = append(items, choir(t.Kontakion))
			}
		}
		return append(items, choir("common.glory"), choir(kontakia[0]), choir("common.and_now"), choir(now)), nil
	}
	d, err := dayOf(f)
	if err != nil {
		return nil, err
	}
	if len(troparia) == 0 || len(kontakia) == 0 {
		return nil, fmt.Errorf("no Menaion troparion or kontakion for %02d.%02d", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	t := litTemple(f, troparia)
	var items []SlotItem
	if t != nil && t.Kind == TempleLord {
		items = append(items, choir(t.Troparion))
	}
	for _, tr := range d.troparia {
		items = append(items, choir(tr))
	}
	if t != nil && t.Kind != TempleLord {
		items = append(items, choir(t.Troparion))
	}
	items = append(items, choir(troparia[0]))
	if t != nil && t.Kind == TempleLord {
		items = append(items, choir(t.Kontakion))
	}
	for _, k := range d.kontakia {
		items = append(items, choir(k))
	}
	if t != nil && t.Kind == TempleSaint {
		items = append(items, choir(t.Kontakion))
	}
	now := "ch.kontakion.protection"
	if t != nil && t.Kind == TempleTheotokos {
		now = t.Kontakion
	}
	return append(items, choir(kontakia[0]), choir("common.glory"), choir("ch.kontakion.with_the_saints"),
		choir("common.and_now"), choir(now)), nil
}

// prokeimenon sung with its verse: the reader announces the tone and says it,
// the choir sings it, the verse, the choir, the first half, the choir.
func prokeimenonBlock(tone int, prok, verse string) []SlotItem {
	return []SlotItem{reader(fmt.Sprintf("lit.prokeimenon.tone.%d", tone)), reader(prok), choir(prok),
		reader(verse), choir(prok), reader(prok), choir(prok)}
}

func litProkeimena(f Facts, feast bool) ([]SlotItem, error) {
	men := firstMain(f, "lit.prokeimenon")
	menVerse := firstMain(f, "lit.prokeimenon.verse")
	if feast {
		if men == "" || f.Tones[men] == 0 {
			return nil, fmt.Errorf("no prokeimenon of the feast")
		}
		return prokeimenonBlock(f.Tones[men], men, menVerse), nil
	}
	d, err := dayOf(f)
	if err != nil {
		return nil, err
	}
	day := WeekdayKey(f.Day.Weekday)
	items := prokeimenonBlock(d.prokTone, "sl.prokeimenon."+day, "sl.prokeimenon."+day+".verse")
	if men != "" && f.Tones[men] > 0 {
		// The saint's prokeimenon: once, without its verse.
		items = append(items, reader(fmt.Sprintf("lit.prokeimenon.tone.%d", f.Tones[men])), reader(men), choir(men))
	}
	return items, nil
}

// litReadings: the Apostle(s) or Gospel(s): the ordinary one and the
// saint's, or the feast's.
func litReadings(f Facts, feast bool, which string) ([]SlotItem, error) {
	type reading struct{ text, announce string }
	var list []reading
	men := firstMain(f, "lit."+which)
	if !feast {
		l, err := lection(f, which)
		if err != nil {
			return nil, err
		}
		key := map[string]string{"apostle": "ap", "gospel": "gos"}[which]
		ref := fmt.Sprintf("lect.%s.%d.%d", key, l.Week, int(l.Weekday))
		list = append(list, reading{ref, ref + ".announce"})
	}
	if men != "" {
		list = append(list, reading{men, men + ".announce"})
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no %s of the feast", which)
	}
	// Only the first reading is announced; the others follow it at once.
	var items []SlotItem
	first := list[0]
	if which == "apostle" {
		items = append(items, litanist("lit.wisdom"), reader(first.announce), litanist("lit.let_us_attend"), reader(first.text))
	} else {
		items = append(items, litanist(first.announce), choir("sl.lit.glory_to_thee_o_lord"), priest("lit.let_us_attend"), litanist(first.text))
	}
	for _, r := range list[1:] {
		if which == "apostle" {
			items = append(items, reader(r.text))
		} else {
			items = append(items, litanist(r.text))
		}
	}
	return items, nil
}

func litAlleluia(f Facts, feast bool) ([]SlotItem, error) {
	block := func(tone int, v1, v2 string) []SlotItem {
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", tone)), choir("common.alleluia.thrice"),
			reader(v1), choir("common.alleluia.thrice")}
		if v2 != "" {
			items = append(items, reader(v2), choir("common.alleluia.thrice"))
		}
		return items
	}
	men := firstMain(f, "lit.alleluia.verse")
	menV2 := ""
	if l := mainMenaion(f, "lit.alleluia.verse"); len(l) > 1 {
		menV2 = l[1]
	}
	if feast {
		if men == "" || f.Tones[men] == 0 {
			return nil, fmt.Errorf("no alleluia of the feast")
		}
		return block(f.Tones[men], men, menV2), nil
	}
	d, err := dayOf(f)
	if err != nil {
		return nil, err
	}
	day := WeekdayKey(f.Day.Weekday)
	items := block(d.alTone, "sl.alleluia."+day+".verse.1", "sl.alleluia."+day+".verse.2")
	if men != "" {
		items = append(items, reader(men), choir("common.alleluia.thrice")) // the saint's verse
	}
	return items, nil
}

func litKinonik(f Facts, feast bool) ([]SlotItem, error) {
	men := firstMain(f, "lit.kinonik")
	if feast {
		if men == "" {
			return nil, fmt.Errorf("no communion verse of the feast")
		}
		return []SlotItem{choir(men), choir("common.alleluia.thrice")}, nil
	}
	items := []SlotItem{choir("sl.kinonik." + WeekdayKey(f.Day.Weekday))}
	if men != "" && men != items[0].Ref {
		items = append(items, choir(men))
	}
	return append(items, choir("common.alleluia.thrice")), nil
}
