package liturgy

import (
	"fmt"
	"strconv"
)

// Selectors of Sunday great vespers (Saturday evening) with a saint without
// a sign, after the Octoechos («В субботу на велицей вечерни») and the
// Typikon:
//
//   - «Господи, воззвах» on 10: the resurrectional stichera (3) and those of
//     Anatolius (4) of the tone, and 3 of the Menaion; where the Menaion has
//     none, the stichera of the Theotokos (Paul of Amorion);
//   - «Слава» of the Menaion, if any; «И ныне» the dogmatikon of the tone;
//   - the prokeimenon «Господь воцарися» with its three verses;
//   - the aposticha: the resurrectional sticheron and the alphabetical
//     ones with their verses; «Слава» of the Menaion, if any, and «И ныне»
//     the theotokion in its tone, else «Слава, и ныне» the Octoechos
//     theotokion.
//
// The day's tone is that of the Octoechos week starting this Sunday.

const sundayMenaionStichera = 3

func sundayVespersSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	switch sel.From {
	case "sv_lic":
		return svLIC(f)
	case "sv_glory_now":
		return svGloryNow(f)
	case "sv_prokeimenon_verses":
		p := "ch.vespers.prokeimenon.sun"
		return []SlotItem{reader(p + ".verse"), choir(p), reader(p + ".verse.2"), choir(p), reader(p + ".verse.3")}, nil
	case "sv_aposticha":
		return svAposticha(f)
	}
	return nil, fmt.Errorf("unknown Sunday vespers selector %q", sel.From)
}

func svLIC(f Facts) ([]SlotItem, error) {
	oct := f.Octoechos["gvespers.lic.sticheron"]
	if len(oct) != 7 {
		return nil, fmt.Errorf("Octoechos tone %d: %d Sunday stichera at «Господи, воззвах», want 7", f.Day.Tone, len(oct))
	}
	refs := append([]string{}, oct...)
	men := mainMenaion(f, "vespers.lic.sticheron")
	if len(men) == 0 {
		th := f.Octoechos["gvespers.lic.theotokos"]
		if len(th) != sundayMenaionStichera {
			return nil, fmt.Errorf("Octoechos tone %d: no stichera of the Theotokos", f.Day.Tone)
		}
		return licWithVerses(append(refs, th...)), nil
	}
	for i := 0; i < sundayMenaionStichera; i++ { // fewer: repeated from the first
		refs = append(refs, men[i%len(men)])
	}
	return licWithVerses(refs), nil
}

func svGloryNow(f Facts) ([]SlotItem, error) {
	dog := first(f.Octoechos, "gvespers.lic.dogmatikon")
	if dog == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no dogmatikon", f.Day.Tone)
	}
	glory := firstMain(f, "vespers.lic.glory")
	if glory == "" {
		glory = firstMain(f, "vespers.lic.glory_now") // a forefeast: «Слава» of it
	}
	if glory == "" {
		return []SlotItem{reader("common.glory_now"), {Ref: dog}}, nil
	}
	return []SlotItem{reader("common.glory"), {Ref: glory}, reader("common.and_now"), {Ref: dog}}, nil
}

func svAposticha(f Facts) ([]SlotItem, error) {
	st := f.Octoechos["gvespers.aposticha.sticheron"]
	if len(st) == 0 {
		return nil, fmt.Errorf("Octoechos tone %d: no Sunday aposticha", f.Day.Tone)
	}
	verses := map[string]string{}
	for _, id := range f.Octoechos["gvespers.aposticha.verse"] {
		verses[id[len(id)-1:]] = id
	}
	var items []SlotItem
	for i, id := range st {
		if v := verses[strconv.Itoa(i+1)]; i > 0 && v != "" {
			items = append(items, reader(v))
		}
		items = append(items, SlotItem{Ref: id})
	}
	if g := firstMain(f, "vespers.aposticha.glory"); g != "" {
		th := bogorTheotokion(f.Tones[g], "aposticha")
		if th == "" {
			return nil, fmt.Errorf("unknown tone of the Menaion's «Слава» %s", g)
		}
		return append(items, reader("common.glory"), SlotItem{Ref: g}, reader("common.and_now"), SlotItem{Ref: th}), nil
	}
	th := first(f.Octoechos, "gvespers.aposticha.theotokion")
	if th == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no aposticha theotokion", f.Day.Tone)
	}
	return append(items, reader("common.glory_now"), SlotItem{Ref: th}), nil
}
