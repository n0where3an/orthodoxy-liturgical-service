package liturgy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The matins of the Sundays of Great Lent (Saturday night's vigil), by the
// Triodion and the Typikon (гл. 49): the Sunday order of the Octoechos
// (sundayMatinsSlot, part "saint") with the Triodion in the place of the
// saint's service:
//
//   - «Бог Господь»: the resurrectional troparion twice, «Слава» the day's
//     (Great Lent: «Пречистому образу», St Gregory, «Спаси, Господи», St
//     John, St Mary), «И ныне» the resurrectional theotokion of its tone;
//   - «Непорочны» (kathisma 17) instead of the polyeleos;
//   - after Psalm 50 «Покаяния отверзи ми двери», «На спасения стези»,
//     «Помилуй мя, Боже», «Множества содеянных мною лютых» (instead of
//     «Воскрес Иисус»);
//   - the canon: the resurrectional on 4, of the Theotokos on 2, the
//     Triodion's on 8 (the 1st Sunday: the resurrectional on 4, of the
//     Cross and Resurrection on 2, of the Theotokos on 2, the Triodion's on
//     6); the katavasia the Triodion's on the 1st and 3rd Sundays («Моря
//     чермную пучину», the Cross's), else «Отверзу уста моя»;
//   - after the 3rd ode the Triodion's sessional hymn (the 2nd Sunday its
//     kontakion «Ныне время делательное» before it, the 4th and 5th the
//     saint's kontakion and ikos), after the 6th the Triodion's kontakion and
//     ikos (the 1st and 3rd), else the Sunday's (the 2nd: St Gregory's);
//   - the exapostilarion of the eothinon, «Слава» the Triodion's, «И ныне» its
//     theotokion; the praises of the Octoechos on 4 and the Triodion's on 4,
//     «Слава» the Triodion's idiomelon («паки тойже»: the 1st and 3rd) else
//     the Gospel sticheron, «И ныне» «Преблагословенна еси».
var triodionMatins = map[string]struct {
	canon     string // sm_canon part
	after3    string // the kontakion before the sessional hymn after the 3rd ode: "", "triodion", "saint"
	after6    string // "triodion" (the Triodion's kontakion and ikos), "sunday", or a unit id
	katavasia bool   // the Triodion's own katavasia
}{
	"lent1": {canon: "lent1", after6: "triodion", katavasia: true},
	"lent2": {canon: "saint", after3: "triodion", after6: "me.11.14p.kontakion.1"},
	"lent3": {canon: "saint", after6: "triodion", katavasia: true},
	"lent4": {canon: "saint", after3: "saint", after6: "sunday"},
	"lent5": {canon: "saint", after3: "saint", after6: "sunday"},
}

// triodionView: the facts with the day's Triodion units as its Menaion
// (mainMenaion reads them under their kinds: matins.canon.1.ode.N.troparion,
// matins.sedalen, kontakion…), and the Triodion's katavasia.
func triodionView(f Facts) Facts {
	v := f
	v.MainPrefix = "tp." + f.TriodionDay + "."
	type item struct {
		kind string
		n    int
		id   string
	}
	var items []item
	glorySrc := ""
	for k, id := range f.Triodion {
		parts := strings.Split(k, ".")
		switch {
		case strings.HasPrefix(k, "canon.ode."):
			// canon.ode.N.troparion.K | canon.ode.N.glory | canon.ode.N.theotokion | canon.ode.N.katavasia
			ode := parts[2]
			base := "matins.canon.1.ode." + ode
			switch parts[3] {
			case "troparion":
				n, _ := strconv.Atoi(parts[4])
				items = append(items, item{base + ".troparion", n, id})
			case "glory": // «Слава, троичен»: after the troparia
				items = append(items, item{base + ".troparion", 1000, id})
			case "theotokion":
				items = append(items, item{base + ".theotokion", 0, id})
			}
		case strings.HasPrefix(k, "matins.") || k == "kontakion" || k == "ikos" || k == "troparion":
			if k == "matins.praises.glory" {
				glorySrc = id
			}
			kind, n := splitKindN(parts)
			if k == "ikos" {
				kind = "matins.ikos"
			}
			items = append(items, item{kind, n, id})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].kind != items[j].kind {
			return items[i].kind < items[j].kind
		}
		return items[i].n < items[j].n
	})
	v.Menaion = map[string][]string{}
	for _, it := range items {
		v.Menaion[it.kind] = append(v.Menaion[it.kind], it.id)
	}
	// «Слава, паки тойже»: the idiomelon sung at «Слава» is not among the four.
	if glorySrc != "" {
		if st := v.Menaion["matins.praises.sticheron"]; len(st) > 1 {
			v.Menaion["matins.praises.sticheron"] = st[:len(st)-1]
		}
	}
	// The day's troparion and the saint's kontakion where the Triodion cites
	// them (the Menaion's, the General Menaion's).
	pr := triodionProfiles[f.TriodionDay]
	if t := pr.unit(f, pr.troparion); t != "" {
		v.Menaion["troparion"] = []string{t}
	}
	if tm, ok := triodionMatins[f.TriodionDay]; ok && tm.katavasia {
		k := map[int]string{}
		for o := 1; o <= 9; o++ {
			if id := f.Triodion[fmt.Sprintf("canon.ode.%d.katavasia", o)]; id != "" {
				k[o] = id
			}
		}
		if len(k) > 0 {
			v.Katavasia, v.KatavasiaSeasonal = k, true
		}
	}
	return v
}

// triodionMatinsSlot: part "triodion" of the Sunday matins selectors.
func triodionMatinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	tm, ok := triodionMatins[f.TriodionDay]
	if !ok {
		return nil, fmt.Errorf("no Triodion matins for %q", f.TriodionDay)
	}
	v := triodionView(f)
	at := func(s string) SlotSelector { s2 := sel; s2.Part = s; return s2 }
	switch sel.From {
	case "sm_ps50":
		return []SlotItem{reader("ch.psalm.50"), choir("common.glory"), choir("tp.ps50.glory"),
			choir("common.and_now"), choir("tp.ps50.now"), choir("ch.matins.ps50.verse"), choir("tp.ps50.sticheron")}, nil
	case "sm_canon":
		return sundayMatinsSlot(at(tm.canon), v)
	case "sm_sedalen": // after the 3rd ode
		var items []SlotItem
		switch tm.after3 {
		case "triodion":
			if k := firstMain(v, "kontakion"); k != "" {
				items = append(items, reader(k))
			}
		case "saint":
			if k := firstMain(v, "kontakion"); k != "" {
				items = append(items, reader(k))
				if ik := firstMain(v, "matins.ikos"); ik != "" {
					items = append(items, reader(ik))
				}
			}
		}
		if len(mainMenaion(v, "matins.sedalen")) > 0 {
			sed, err := matinsMenaionSedalen(v)
			if err != nil {
				return nil, err
			}
			items = append(items, sed...)
		}
		return items, nil
	case "sm_kontakion": // after the 6th ode
		switch tm.after6 {
		case "triodion":
			items := []SlotItem{reader(firstMain(v, "kontakion"))}
			if ik := firstMain(v, "matins.ikos"); ik != "" {
				items = append(items, reader(ik))
			}
			return items, nil
		case "sunday":
			return []SlotItem{reader(first(f.Octoechos, "kontakion")), reader(first(f.Octoechos, "matins.ikos"))}, nil
		}
		return []SlotItem{reader(tm.after6)}, nil
	case "sm_praises":
		items, err := sundayMatinsSlot(at("saint"), v)
		if err != nil {
			return nil, err
		}
		if g := f.Triodion["matins.praises.glory"]; g != "" && len(items) >= 4 {
			items[len(items)-3] = choir(g) // «Слава» the Triodion's idiomelon
		}
		return items, nil
	case "sm_troparia", "sm_exapostilaria":
		return sundayMatinsSlot(at("saint"), v)
	}
	return sundayMatinsSlot(at("sunday"), f)
}
