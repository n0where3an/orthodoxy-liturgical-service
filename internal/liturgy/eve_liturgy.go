package liturgy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// eveOf: the feast whose eve the day is (24 December, 5 January).
func eveOf(day calendar.DayContext) string {
	switch {
	case day.Menaion.Month == time.December && day.Menaion.Day == 24:
		return "nativity"
	case day.Menaion.Month == time.January && day.Menaion.Day == 5:
		return "theophany"
	}
	return ""
}

// The vespers of the eve joined to the liturgy of St Basil the Great (the
// Menaion, 25 December, 6 January; ev.<feast>.*, orthlib.ImportEveVespers):
// «Господи, воззвах» with the four stichera, each twice, on 8; the entrance
// with the Gospel, «Свете тихий», the prokeimenon of the day (in the
// Nativity's eve on Friday the great «Кто Бог велий»); the readings, the
// reader's troparion with its verses after the 3rd and the 6th, the choir
// singing its end; after the small litany the Trisagion, and the liturgy
// with the eve's prokeimenon, Apostle, alleluia, Gospel and communion verse
// («дне же ряд… не чтется»). After the Nativity's liturgy its troparion
// and kontakion are sung before the candle.
func eveSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	u := func(k string) string { return f.Eve[k] }
	switch sel.Part {
	case "lord_i_call":
		var st []string
		for n := 1; ; n++ {
			id := u(fmt.Sprintf("lic.%d", n))
			if id == "" {
				break
			}
			st = append(st, id, id) // «дважды»
		}
		if len(st) < 8 {
			return nil, fmt.Errorf("eve vespers: %d stichera at «Господи, воззвах»", len(st))
		}
		var items []SlotItem
		for i, id := range st[:8] {
			items = append(items, reader(fmt.Sprintf("ch.vespers.lic.verse.%d", 23+i)), choir(id))
		}
		g := u("lic.glory_now")
		if g == "" {
			return nil, fmt.Errorf("eve vespers: no «Слава, и ныне» at «Господи, воззвах»")
		}
		return append(items, choir("common.glory_now"), choir(g)), nil
	case "vespers_lic", "lic_glory_now":
		// At the vespers apart: the stichera, «Слава, и ныне» in its own slot.
		items, err := eveSlot(SlotSelector{Part: "lord_i_call"}, f)
		if err != nil {
			return nil, err
		}
		if sel.Part == "lic_glory_now" {
			return items[len(items)-2:], nil
		}
		return items[:len(items)-2], nil
	case "vespers_gospel":
		// «По паремиах ектениа малая, и по возгласе прокимен…» (the
		// Nativity's own; the Theophany's «яко предписася», the liturgy's).
		pre := "wv.lit."
		if u(pre+"prokeimenon") == "" {
			pre = "lit."
		}
		var items []SlotItem
		p := u(pre + "prokeimenon")
		if p == "" || f.Tones[p] == 0 {
			return nil, fmt.Errorf("eve vespers: no prokeimenon before the Apostle")
		}
		items = append(items, litanist("lit.let_us_attend"), priest("lit.peace_be_unto_all"), reader("lit.response.and_with_thy_spirit"),
			litanist("lit.wisdom"))
		items = append(items, prokeimenonBlock(f.Tones[p], p, u(pre+"prokeimenon.verse"))...)
		if a := u(pre + "apostle"); a != "" {
			items = append(items, litanist("lit.wisdom"), reader(a+".announce"), litanist("lit.let_us_attend"), reader(a))
		}
		if v1 := u(pre + "alleluia.verse.1"); v1 != "" && f.Tones[v1] > 0 {
			items = append(items, priest("lit.peace_be_unto_all"), reader("lit.response.and_with_thy_spirit"),
				reader(fmt.Sprintf("lit.alleluia.tone.%d", f.Tones[v1])), choir("common.alleluia.thrice"), reader(v1),
				choir("common.alleluia.thrice"))
			if v2 := u(pre + "alleluia.verse.2"); v2 != "" {
				items = append(items, reader(v2), choir("common.alleluia.thrice"))
			}
		}
		g := u(pre + "gospel")
		if g == "" {
			return nil, fmt.Errorf("eve vespers: no Gospel")
		}
		return append(items, priest("sl.lit.wisdom_aright_gospel"), priest("lit.peace_be_unto_all"),
			choir("lit.response.and_with_thy_spirit"), litanist(g+".announce"), choir("sl.lit.glory_to_thee_o_lord"),
			litanist("lit.let_us_attend"), litanist(g), choir("sl.lit.glory_to_thee_o_lord")), nil
	case "prokeimenon":
		// The evening's: of the day the vespers belong to, the feast's.
		if f.EveOf == "nativity" && f.Day.Weekday == time.Friday {
			p := "ch.vespers.prokeimenon.great.who"
			items := []SlotItem{reader("lit.prokeimenon.tone.7"), reader(p), choir(p)}
			for v := 1; v <= 3; v++ {
				items = append(items, reader(fmt.Sprintf("%s.verse.%d", p, v)), choir(p))
			}
			return append(items, reader(p), choir(p)), nil
		}
		day := WeekdayKey((f.Day.Weekday + 1) % 7)
		p := "ch.vespers.prokeimenon." + day
		return []SlotItem{reader(p), choir(p), reader(p + ".verse"), choir(p), reader(p), choir(p)}, nil
	case "readings":
		type item struct {
			n    int
			kind string
			id   string
		}
		var list []item
		for k, id := range f.Eve {
			parts := strings.Split(k, ".")
			if len(parts) != 2 {
				continue
			}
			if n, err := strconv.Atoi(parts[1]); err == nil && parts[0] != "lic" {
				list = append(list, item{n, parts[0], id})
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].n < list[j].n })
		var items []SlotItem
		for _, x := range list {
			switch x.kind {
			case "reading":
				items = append(items, litanist("lit.wisdom"))
				if t := u(fmt.Sprintf("title.%d", x.n)); t != "" {
					items = append(items, reader(t))
				}
				items = append(items, litanist("lit.let_us_attend"), reader(x.id))
			case "troparion", "verse":
				items = append(items, reader(x.id))
			case "refrain", "mark":
				items = append(items, choir(x.id))
			}
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("eve vespers: no readings")
		}
		return items, nil
	case "lit_prokeimena":
		p := u("lit.prokeimenon")
		if p == "" || f.Tones[p] == 0 {
			return nil, fmt.Errorf("eve liturgy: no prokeimenon")
		}
		return prokeimenonBlock(f.Tones[p], p, u("lit.prokeimenon.verse")), nil
	case "lit_apostles":
		a := u("lit.apostle")
		if a == "" {
			return nil, fmt.Errorf("eve liturgy: no Apostle")
		}
		return []SlotItem{litanist("lit.wisdom"), reader(a + ".announce"), litanist("lit.let_us_attend"), reader(a)}, nil
	case "lit_alleluia":
		v1 := u("lit.alleluia.verse.1")
		if v1 == "" || f.Tones[v1] == 0 {
			return nil, fmt.Errorf("eve liturgy: no alleluia")
		}
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", f.Tones[v1])), choir("common.alleluia.thrice"),
			reader(v1), choir("common.alleluia.thrice")}
		if v2 := u("lit.alleluia.verse.2"); v2 != "" {
			items = append(items, reader(v2), choir("common.alleluia.thrice"))
		}
		return items, nil
	case "lit_gospels":
		g := u("lit.gospel")
		if g == "" {
			return nil, fmt.Errorf("eve liturgy: no Gospel")
		}
		return []SlotItem{litanist(g + ".announce"), choir("sl.lit.glory_to_thee_o_lord"), priest("lit.let_us_attend"), litanist(g)}, nil
	case "kinonik":
		k := u("lit.kinonik")
		if k == "" {
			k = "sl.kinonik.sun" // «Хвалите Господа с небес»
		}
		return []SlotItem{choir(k), choir("common.alleluia.thrice")}, nil
	case "after":
		if f.EveOf != "nativity" || f.EveFeast == nil {
			return nil, nil
		}
		return []SlotItem{choir(f.EveFeast.Troparion), choir("common.glory_now"), choir(f.EveFeast.Kontakion)}, nil
	}
	return nil, fmt.Errorf("unknown eve part %q", sel.Part)
}
