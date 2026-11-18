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

// --- Sunday great vespers with a saint with a sign, a feast of the
// Theotokos, or in an afterfeast (part "saint", "feast", "afterfeast") ---
//
// After the Typikon (and the Menaion's «аще случится в неделю»):
//
//   - «Господи, воззвах» on 10: a saint with a sign — the Octoechos 6 (3
//     resurrectional, 3 of Anatolius) and the saint's 4; a feast of the
//     Theotokos or an afterfeast — the Octoechos 4 and the Menaion's 6 (the
//     feast's, then the saint's);
//   - «Слава» of the Menaion; «И ныне»: at a feast the feast's, in the
//     afterfeast of a feast of the Lord the feast's, otherwise the
//     dogmatikon of the tone;
//   - the aposticha of the Octoechos with their verses; «Слава» of the
//     Menaion; «И ныне»: the feast's (feast, afterfeast), else the
//     theotokion in the tone of «Слава»;
//   - the paremias and the litiya when the Menaion has them.

// theotokosFeasts: the twelve feasts of the Theotokos.
var theotokosFeasts = map[string]bool{"nativity_of_theotokos": true, "entry_of_theotokos": true, "annunciation": true, "dormition": true}

// menVespers: the Menaion's section of the day: great vespers if it has one,
// else daily vespers.
func menVespers(f Facts) string {
	if len(mainMenaion(f, "gvespers.lic.sticheron")) > 0 {
		return "gvespers."
	}
	return "vespers."
}

// apodosisFacts: on the apodosis whose Menaion day gives no stichera («вся
// служба праздника») the hymns are those of the feast's own day.
func apodosisFacts(f Facts) Facts {
	if f.Feast == nil || !f.Feast.IsApodosis(f.Day) || len(mainMenaion(f, "vespers.lic.sticheron")) > 0 ||
		len(mainMenaion(f, "gvespers.lic.sticheron")) > 0 || f.FeastMenaion == nil {
		return f
	}
	g := f
	g.Menaion = f.FeastMenaion
	g.Day.Menaion = f.Feast.from
	return g
}

func sundayCombinedSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	stichera := sel.From == "sv_combined_lic" || sel.From == "sv_combined_glory_now" || sel.From == "sv_combined_aposticha"
	if sel.Part == "afterfeast" && stichera {
		if g := apodosisFacts(f); g.Day.Menaion != f.Day.Menaion {
			f = g
			sel.Part = "feast" // the feast's «Слава, и ныне»
		}
	}
	switch sel.From {
	case "sv_combined_lic":
		return svCombinedLIC(f, sel.Part)
	case "sv_combined_glory_now":
		return svCombinedGloryNow(f, sel.Part)
	case "sv_combined_aposticha":
		return svCombinedAposticha(f, sel.Part)
	case "sv_combined_troparia":
		return svCombinedTroparia(f, sel.Part)
	case "sv_combined_paremias": // when the Menaion has them
		if len(mainMenaion(f, "gvespers.paremia")) == 0 {
			return nil, nil
		}
		return gvParemias(f)
	case "sv_combined_litiya", "sv_combined_loaves": // at a vigil of the Menaion
		if len(mainMenaion(f, "gvespers.litiya.sticheron")) == 0 {
			return nil, nil
		}
		if sel.From == "sv_combined_loaves" {
			return []SlotItem{{Ref: "sl.vespers.loaves_prayer"}}, nil
		}
		return gvStichera(f, "litiya")
	}
	return nil, fmt.Errorf("unknown Sunday vespers selector %q", sel.From)
}

func svCombinedLIC(f Facts, part string) ([]SlotItem, error) {
	oct := f.Octoechos["gvespers.lic.sticheron"]
	if len(oct) != 7 {
		return nil, fmt.Errorf("Octoechos tone %d: %d Sunday stichera at «Господи, воззвах», want 7", f.Day.Tone, len(oct))
	}
	men := mainMenaion(f, menVespers(f)+"lic.sticheron")
	if len(men) == 0 {
		return nil, fmt.Errorf("no Menaion stichera at «Господи, воззвах» for %02d.%02d", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	nOct, nMen := 4, 6
	if part == "saint" {
		nOct, nMen = 6, 4
	}
	refs := append([]string{}, oct[:nOct]...)
	for i := 0; i < nMen; i++ { // fewer: repeated from the first
		refs = append(refs, men[i%len(men)])
	}
	return licWithVerses(refs), nil
}

// menGloryNow: «Слава» and «И ныне» of a section of the Menaion's vespers
// (either may be ""), or one hymn for both (glory_now).
func menGloryNow(f Facts, section string) (glory, now, gloryNow string) {
	k := menVespers(f) + section + "."
	glory = firstMain(f, k+"glory")
	now = firstMain(f, k+"now")
	if now == "" {
		now = firstMain(f, k+"theotokion")
	}
	gloryNow = firstMain(f, k+"glory_now")
	return
}

func svCombinedGloryNow(f Facts, part string) ([]SlotItem, error) {
	if part == "feast" {
		return gvGloryNow(f, "lic")
	}
	dog := first(f.Octoechos, "gvespers.lic.dogmatikon")
	if dog == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no dogmatikon", f.Day.Tone)
	}
	glory, now, gloryNow := menGloryNow(f, "lic")
	lordAfterfeast := part == "afterfeast" && f.Feast != nil && !theotokosFeasts[f.Feast.ID]
	if !lordAfterfeast {
		now = dog
	}
	switch {
	case glory != "" && now != "":
		return []SlotItem{reader("common.glory"), {Ref: glory}, reader("common.and_now"), {Ref: now}}, nil
	case gloryNow != "" && lordAfterfeast:
		return []SlotItem{reader("common.glory_now"), {Ref: gloryNow}}, nil
	case gloryNow != "":
		// «Слава, и ныне» of the Menaion is the feast's: «Слава» of it, «И ныне» the dogmatikon.
		return []SlotItem{reader("common.glory"), {Ref: gloryNow}, reader("common.and_now"), {Ref: dog}}, nil
	case now != "":
		return []SlotItem{reader("common.glory_now"), {Ref: now}}, nil
	}
	return []SlotItem{reader("common.glory_now"), {Ref: dog}}, nil
}

func svCombinedAposticha(f Facts, part string) ([]SlotItem, error) {
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
	if part == "feast" {
		gn, err := gvGloryNow(f, "aposticha")
		return append(items, gn...), err
	}
	glory, now, gloryNow := menGloryNow(f, "aposticha")
	if part == "saint" {
		now = "" // the theotokion in the tone of «Слава»
	}
	switch {
	case glory != "" && now != "":
		return append(items, reader("common.glory"), SlotItem{Ref: glory}, reader("common.and_now"), SlotItem{Ref: now}), nil
	case glory != "":
		th := bogorTheotokion(f.Tones[glory], "aposticha")
		if th == "" {
			return nil, fmt.Errorf("unknown tone of the Menaion's «Слава» %s", glory)
		}
		return append(items, reader("common.glory"), SlotItem{Ref: glory}, reader("common.and_now"), SlotItem{Ref: th}), nil
	case gloryNow != "":
		return append(items, reader("common.glory_now"), SlotItem{Ref: gloryNow}), nil
	}
	th := first(f.Octoechos, "gvespers.aposticha.theotokion")
	if th == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no aposticha theotokion", f.Day.Tone)
	}
	return append(items, reader("common.glory_now"), SlotItem{Ref: th}), nil
}

// svCombinedTroparia: at the blessing of loaves «Богородице Дево» twice and
// the saint's troparion once (at a feast of the Theotokos the feast's
// troparion thrice); without the litiya «Богородице Дево» thrice.
func svCombinedTroparia(f Facts, part string) ([]SlotItem, error) {
	troparia := mainMenaion(f, "troparion")
	if len(mainMenaion(f, "gvespers.litiya.sticheron")) == 0 || len(troparia) == 0 {
		return []SlotItem{{Ref: "ch.theotokos_rejoice", Repeat: 3}}, nil
	}
	if part == "feast" {
		return []SlotItem{{Ref: troparia[0], Repeat: 3}}, nil
	}
	saint := ""
	for _, t := range troparia {
		if !f.ofFeast(t) && (f.Feast == nil || t != f.Feast.Troparion) {
			saint = t
			break
		}
	}
	if saint == "" {
		saint = troparia[0]
	}
	return []SlotItem{{Ref: "ch.theotokos_rejoice", Repeat: 2}, {Ref: saint}}, nil
}
