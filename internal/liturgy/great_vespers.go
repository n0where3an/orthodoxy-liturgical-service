package liturgy

import (
	"fmt"
	"strconv"
	"strings"
)

// Selectors of great vespers (polyeleos, vigil, feast) from the Menaion's
// own service (keys me.<MM>.<DD>.gvespers.*):
//
//   - «Господи, воззвах» on 8 (or as the Menaion says: «на ѕ҃»): the Menaion
//     stichera, repeated from the first one when there are fewer;
//   - «Слава» of the Menaion; «И ныне»: the Menaion's hymn, or, where it only
//     cites it, the theotokion of the tone of «Слава» (the dogmatikon);
//   - the readings with «Премудрость», the title and «Вонмем»;
//   - the litiya and the aposticha (with the Menaion's own verses), «И ныне»
//     as above (the aposticha theotokion of the tone);
//   - the troparia: at a vigil the saint's troparion twice and «Богородице
//     Дево» once, at a feast the feast's troparion thrice; otherwise as at
//     daily vespers.

const greatLICDefault = 8

func greatVespersSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	switch sel.From {
	case "gv_lic":
		return gvLIC(f)
	case "gv_glory_now":
		return gvGloryNow(f, sel.Section)
	case "gv_paremias":
		return gvParemias(f)
	case "gv_litiya":
		return gvStichera(f, "litiya")
	case "gv_aposticha":
		return gvStichera(f, "aposticha")
	case "gv_troparia":
		return gvTroparia(f, sel.Part)
	}
	return nil, fmt.Errorf("unknown great vespers selector %q", sel.From)
}

func gvLIC(f Facts) ([]SlotItem, error) {
	men := mainMenaion(f, "gvespers.lic.sticheron")
	if len(men) == 0 {
		return nil, fmt.Errorf("no Menaion stichera at «Господи, воззвах» (great vespers) for %02d.%02d",
			int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	places := greatLICDefault
	if on := markerN(mainMenaion(f, "gvespers.lic.on")); on > 0 {
		places = on
	}
	var refs []string
	refs = append(refs, men...)
	for i := 0; len(refs) < places; i++ { // fewer stichera: repeated from the first
		refs = append(refs, men[i%len(men)])
	}
	return licWithVerses(f, refs), nil
}

// licWithVerses puts the verses of «Господи, воззвах» from the 21st (the
// first twenty are in the template) before the stichera: on 10 each of the
// ten has its verse, on 8 verses 21 and 22 come first.
func licWithVerses(f Facts, stichera []string) []SlotItem {
	if len(stichera) > 10 {
		stichera = stichera[:10]
	}
	var items []SlotItem
	first := 30 - len(stichera) + 1
	// The verses without stichera: by the practice (Facts.Psalmody).
	for v := 21; v < first; v++ {
		if it, ok := f.psalmVerse(fmt.Sprintf("ch.vespers.lic.verse.%d", v)); ok {
			items = append(items, it)
		}
	}
	for i, st := range stichera {
		items = append(items, reader(fmt.Sprintf("ch.vespers.lic.verse.%d", first+i)), SlotItem{Ref: st})
	}
	return items
}

// bogorTheotokion: the resurrectional theotokion of the tone (Menaion,
// «Богородичны воскресны на осмь гласов»): lic (dogmatikon) or aposticha.
func bogorTheotokion(tone int, part string) string {
	if tone < 1 || tone > 8 {
		return ""
	}
	return fmt.Sprintf("me.bogor.%d.%s", tone, part)
}

// gvGloryNow: «Слава, и ныне» of a section (lic, litiya, aposticha).
func gvGloryNow(f Facts, section string) ([]SlotItem, error) {
	k := "gvespers." + section + "."
	glory, now := firstMain(f, k+"glory"), firstMain(f, k+"now")
	part := "aposticha"
	if section == "lic" {
		part = "lic"
	}
	switch {
	case glory == "" && now != "":
		// «Слава, и ныне» — one hymn.
		return []SlotItem{reader("common.glory_now"), {Ref: now}}, nil
	case glory != "" && now != "":
		return []SlotItem{reader("common.glory"), {Ref: glory}, reader("common.and_now"), {Ref: now}}, nil
	case glory != "":
		th := bogorTheotokion(f.Tones[glory], part)
		if th == "" {
			return nil, fmt.Errorf("%s: unknown tone of «Слава» %s", section, glory)
		}
		return []SlotItem{reader("common.glory"), {Ref: glory}, reader("common.and_now"), {Ref: th}}, nil
	}
	// No «Слава»: the theotokion of the tone of the last sticheron.
	st := mainMenaion(f, k+"sticheron")
	if len(st) == 0 {
		return nil, nil
	}
	th := bogorTheotokion(f.Tones[st[len(st)-1]], part)
	if th == "" {
		return nil, fmt.Errorf("%s: no «Слава, и ныне» and unknown tone", section)
	}
	return []SlotItem{reader("common.glory_now"), {Ref: th}}, nil
}

func gvParemias(f Facts) ([]SlotItem, error) {
	texts := mainMenaion(f, "gvespers.paremia")
	if len(texts) == 0 {
		return nil, fmt.Errorf("no readings (paremias) for %02d.%02d", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	var items []SlotItem
	for _, id := range texts {
		n := id[strings.LastIndex(id, ".")+1:]
		items = append(items, SlotItem{Ref: "lit.wisdom", Performer: Litanist})
		if title := id + ".title"; containsID(f.Menaion, "gvespers.paremia."+n+".title", title) {
			items = append(items, reader(title))
		}
		items = append(items, SlotItem{Ref: "lit.let_us_attend", Performer: Litanist}, reader(id))
	}
	return items, nil
}

func containsID(m map[string][]string, kind, id string) bool {
	for _, x := range m[kind] {
		if x == id {
			return true
		}
	}
	return false
}

func gvStichera(f Facts, section string) ([]SlotItem, error) {
	k := "gvespers." + section + "."
	stichera := mainMenaion(f, k+"sticheron")
	if len(stichera) == 0 {
		return nil, fmt.Errorf("no Menaion stichera at the %s (great vespers)", section)
	}
	verses := map[string]string{}
	for _, id := range mainMenaion(f, k+"verse") {
		verses[id[strings.LastIndex(id, ".")+1:]] = id
	}
	var items []SlotItem
	for i, id := range stichera {
		if section == "aposticha" && i > 0 {
			if v := verses[strconv.Itoa(i+1)]; v != "" {
				items = append(items, reader(v))
			}
		}
		items = append(items, SlotItem{Ref: id})
	}
	gn, err := gvGloryNow(f, section)
	if err != nil {
		return nil, err
	}
	return append(items, gn...), nil
}

// gvTroparia: part "vigil" (at the blessing of loaves), "feast" (the
// feast's troparion thrice) or "" (as at daily vespers).
func gvTroparia(f Facts, part string) ([]SlotItem, error) {
	troparia := mainMenaion(f, "troparion")
	switch part {
	case "feast":
		if len(troparia) == 0 {
			return nil, fmt.Errorf("no troparion of the feast")
		}
		return []SlotItem{{Ref: troparia[0], Repeat: 3}}, nil
	case "theotokos":
		// A vigil without the blessing of loaves (Sunday): «Богородице Дево» thrice.
		return []SlotItem{{Ref: "ch.theotokos_rejoice", Repeat: 3}}, nil
	case "vigil":
		if len(troparia) == 0 {
			return nil, fmt.Errorf("no troparion of the saint")
		}
		return []SlotItem{{Ref: troparia[0], Repeat: 2}, reader("common.glory_now"), {Ref: "ch.theotokos_rejoice"}}, nil
	}
	return vespersTroparia(f)
}
