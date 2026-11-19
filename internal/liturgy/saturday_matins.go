package liturgy

import (
	"fmt"
	"strings"
)

// Saturday matins with «Бог Господь» (Типикон, гл. 12, «о службе
// субботней»), for a saint of the Menaion with a troparion and without a
// sign. It follows weekday matins with these differences:
//
//   - kathismata 16 and 17, a small litany after each;
//   - after the 16th the martyrs' sessional hymns, the first two, «Слава, и
//     ныне» the theotokion; after the 17th the martyrs' two and the
//     departed one, the theotokion (saturdaySedalny, matins_sedalny 2);
//   - the canons: the saint's with its irmos, then the Octoechos's of the
//     martyrs, «покоин бо канон поется в пяток, на повечерии» (matinsCanon);
//   - after the 9th ode the saint's exapostilarion, «Слава» the Octoechos's
//     («Иже и мертвыми и живыми»), «И ныне» the theotokion;
//   - «на хвалитех стихиры не поем октоиха, но поем их на стиховне»: the
//     martyrs' three with «Исполнихомся заутра» and «И буди светлость», the
//     departed fourth left out; «Слава» the saint's, else «Слава, и ныне»
//     their theotokion (saturdayAposticha). The Saturday aposticha of the
//     departed is only sung on Saturdays of Lent and with «Аллилуиа».
//
// A saint with a sign (the great doxology, polyeleos, vigil) is served by
// the saint's own order, of a weekday (matins.saturday.doxology) or festal matins,
// on kathismata 16 and 17. The Octoechos's Saturday canons give way (гл. 11:
// «в куюлибо субботу славословие прилучится… поем канон бцды прешедшия
// недели утренний, со ирмосом на 6: и стаго на 8»; «аще ли поется стаго
// полиелей: тогда не поется октоих»): theotokosCanon. After the 9th
// ode «светилен стаго дважды. слава, и ныне, бгородичен». Where the
// Menaion gives no sessional hymns after a kathisma, the Octoechos's
// martyrs' are sung.

func saturdaySedalny(f Facts) ([]SlotItem, error) {
	const p = "matins.sedalen.1"
	hymns := f.Octoechos[p]
	if len(hymns) < 2 {
		return nil, fmt.Errorf("no Saturday sessional hymns after kathisma 16 (tone %d)", f.Day.Tone)
	}
	items := []SlotItem{{Ref: hymns[0]}}
	if v := first(f.Octoechos, p+".verse"); v != "" && strings.HasSuffix(v, ".2") {
		items = append(items, reader(v))
	}
	items = append(items, SlotItem{Ref: hymns[1]})
	if th := first(f.Octoechos, p+".theotokion"); th != "" {
		items = append(items, reader("common.glory_now"), SlotItem{Ref: th})
	}
	return items, nil
}

func saturdayExapostilaria(f Facts) []SlotItem {
	th := theotokionFor(f, "matins.exapostilarion")
	if th == "" {
		th = "oc.exapostilarion.sat.theotokion"
	}
	men := mainMenaion(f, "matins.exapostilarion")
	if len(men) > 0 && f.redSign() {
		// «Светилен стаго дважды. слава, и ныне, бгородичен» (гл. 11).
		items := []SlotItem{{Ref: men[0], Repeat: 2}}
		if len(men) > 1 {
			items[0].Repeat = 0
			for _, id := range men[1:] {
				items = append(items, SlotItem{Ref: id})
			}
		}
		return append(items, reader("common.glory_now"), SlotItem{Ref: th})
	}
	if len(men) == 0 {
		return []SlotItem{{Ref: "oc.exapostilarion.sat"}, reader("common.glory_now"), {Ref: th}}
	}
	var items []SlotItem
	for _, id := range men {
		items = append(items, SlotItem{Ref: id})
	}
	return append(items, reader("common.glory"), SlotItem{Ref: "oc.exapostilarion.sat"},
		reader("common.and_now"), SlotItem{Ref: th})
}

func saturdayAposticha(f Facts) ([]SlotItem, error) {
	stichera := f.Octoechos["matins.praises.sticheron"]
	if len(stichera) < 3 {
		return nil, fmt.Errorf("Octoechos tone %d: %d martyrs' stichera at the praises", f.Day.Tone, len(stichera))
	}
	items := []SlotItem{
		{Ref: stichera[0]},
		reader("ch.matins.aposticha.verse.2"), {Ref: stichera[1]},
		reader("ch.matins.aposticha.verse.3"), {Ref: stichera[2]},
	}
	th := first(f.Octoechos, "matins.praises.theotokion")
	if th == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no theotokion of the praises", f.Day.Tone)
	}
	if g := firstMain(f, "matins.aposticha.glory"); g != "" {
		return append(items, reader("common.glory"), SlotItem{Ref: g}, reader("common.and_now"), SlotItem{Ref: th}), nil
	}
	return append(items, reader("common.glory_now"), SlotItem{Ref: th}), nil
}
