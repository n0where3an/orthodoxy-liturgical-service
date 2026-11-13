package liturgy

import (
	"fmt"
	"strings"
	"time"
)

// Selectors of daily vespers. They implement the Typikon's distribution of
// Octoechos and Menaion hymns for a day of the Menaion without a sign (and,
// as far as the Menaion supplies them, for its other daily-vespers days):
//
//   - «Господи, воззвах» on 6: the Menaion stichera (up to 6), completed by
//     the first Octoechos stichera of the day;
//   - «Слава»: the Menaion doxastikon if any; «И ныне»: the Menaion
//     theotokion, else the Octoechos one (on Tuesday and Thursday evenings a
//     cross-theotokion);
//   - aposticha: the Menaion's own, else the Octoechos with their verses;
//   - troparia: the troparion(s) of the day, «Слава, и ныне» and the
//     dismissal theotokion of the tone of the (last) troparion.

const licCount = 6

func vespersSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	switch sel.From {
	case "vespers_lic":
		return vespersLIC(f)
	case "vespers_glory_now":
		return vespersGloryNow(sel.Section, f)
	case "vespers_aposticha":
		return vespersAposticha(f)
	case "vespers_troparia":
		return vespersTroparia(f)
	}
	return nil, fmt.Errorf("unknown vespers selector %q", sel.From)
}

func vespersLIC(f Facts) ([]SlotItem, error) {
	men := f.Menaion["vespers.lic.sticheron"]
	if len(men) > licCount {
		men = men[:licCount]
	}
	// "Стихиры святых на 6 (на 4)": the Menaion fills that many places; its
	// stichera are repeated from the first one on (3 on 6: each twice; 4 on
	// 6: the first two twice). The Octoechos completes the six.
	var menItems []SlotItem
	places := len(men)
	if on := markerN(f.Menaion["vespers.lic.on"]); on > len(men) && len(men) > 0 {
		places = min(on, licCount)
	}
	extra := places - len(men)
	for i, id := range men {
		menItems = append(menItems, SlotItem{Ref: id})
		if i < extra {
			menItems = append(menItems, SlotItem{Ref: id})
		}
	}
	need := licCount - len(menItems)
	oct := f.Octoechos["vespers.lic.sticheron"]
	if len(oct) < need {
		return nil, fmt.Errorf("Octoechos tone %d, %v: %d stichera at «Господи, воззвах», need %d",
			f.Day.Tone, f.Day.Weekday, len(oct), need)
	}
	var items []SlotItem
	for _, id := range oct[:need] {
		items = append(items, SlotItem{Ref: id})
	}
	return append(items, menItems...), nil
}

// markerN reads N from a marker id "….on.N".
func markerN(ids []string) int {
	if len(ids) == 0 {
		return 0
	}
	var n int
	fmt.Sscanf(ids[0][strings.LastIndex(ids[0], ".")+1:], "%d", &n)
	return n
}

func first(m map[string][]string, kind string) string {
	if l := m[kind]; len(l) > 0 {
		return l[0]
	}
	return ""
}

func vespersGloryNow(section string, f Facts) ([]SlotItem, error) {
	if section != "lic" && section != "aposticha" {
		return nil, fmt.Errorf("unknown section %q", section)
	}
	k := "vespers." + section + "."
	doxology := func(ref string) SlotItem { return SlotItem{Ref: ref, Performer: Reader} }
	if g := first(f.Menaion, k+"glory_now"); g != "" {
		return []SlotItem{doxology("common.glory_now"), {Ref: g}}, nil
	}
	theotokion := first(f.Menaion, k+"theotokion")
	if theotokion == "" {
		theotokion = first(f.Octoechos, k+"theotokion")
	}
	if theotokion == "" {
		return nil, fmt.Errorf("no theotokion for the %s (tone %d, %v)", section, f.Day.Tone, f.Day.Weekday)
	}
	if g := first(f.Menaion, k+"glory"); g != "" {
		return []SlotItem{doxology("common.glory"), {Ref: g}, doxology("common.and_now"), {Ref: theotokion}}, nil
	}
	return []SlotItem{doxology("common.glory_now"), {Ref: theotokion}}, nil
}

func vespersAposticha(f Facts) ([]SlotItem, error) {
	src, k := f.Octoechos, "vespers.aposticha."
	if len(f.Menaion[k+"sticheron"]) > 0 {
		src = f.Menaion // the Menaion's own aposticha (afterfeasts, …)
	}
	stichera := src[k+"sticheron"]
	if len(stichera) == 0 {
		return nil, fmt.Errorf("no aposticha (tone %d, %v)", f.Day.Tone, f.Day.Weekday)
	}
	verses := map[int]string{}
	for _, id := range src[k+"verse"] {
		var n int
		fmt.Sscanf(id[len(id)-1:], "%d", &n)
		verses[n] = id
	}
	var items []SlotItem
	for i, id := range stichera {
		n := i + 1
		if n > 1 {
			v := verses[n]
			if v == "" {
				v = defaultApostichaVerse(f.Day.Weekday, n)
			}
			if v != "" {
				items = append(items, SlotItem{Ref: v, Performer: Reader})
			}
		}
		items = append(items, SlotItem{Ref: id})
	}
	return items, nil
}

// defaultApostichaVerse: «К Тебе возведох очи мои», «Помилуй нас, Господи»;
// on Friday evening (the vespers of Saturday) «Блажени, яже избрал»,
// «Души их во благих водворятся».
func defaultApostichaVerse(day time.Weekday, n int) string {
	if n < 2 || n > 3 {
		return ""
	}
	if day == time.Saturday {
		return fmt.Sprintf("oc.vespers.aposticha.sat.verse.%d", n)
	}
	return fmt.Sprintf("ch.vespers.aposticha.verse.%d", n)
}

func vespersTroparia(f Facts) ([]SlotItem, error) {
	troparia := f.Menaion["troparion"]
	day := WeekdayKey(f.Day.Weekday)
	theotokion := func(tropRef string) (SlotItem, error) {
		tone := f.Tones[tropRef]
		if tone == 0 {
			return SlotItem{}, fmt.Errorf("unknown tone of %s", tropRef)
		}
		return SlotItem{Ref: fmt.Sprintf("ch.theotokion.dismissal.%d.%s", tone, day)}, nil
	}
	doxology := func(ref string) SlotItem { return SlotItem{Ref: ref, Performer: Reader} }

	if f.Feast != nil && !f.Feast.IsFeastDay(f.Day) {
		// Afterfeast: the saint's troparion, «Слава, и ныне», the feast's.
		var items []SlotItem
		for _, t := range troparia {
			if t != f.Feast.Troparion {
				items = append(items, SlotItem{Ref: t}, doxology("common.glory_now"))
				break
			}
		}
		return append(items, SlotItem{Ref: f.Feast.Troparion}), nil
	}
	switch len(troparia) {
	case 0:
		return nil, fmt.Errorf("no Menaion troparion for %02d.%02d in the corpus", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	case 1:
		th, err := theotokion(troparia[0])
		if err != nil {
			return nil, err
		}
		return []SlotItem{{Ref: troparia[0]}, doxology("common.glory_now"), th}, nil
	default:
		th, err := theotokion(troparia[1])
		if err != nil {
			return nil, err
		}
		return []SlotItem{{Ref: troparia[0]}, doxology("common.glory"), {Ref: troparia[1]}, doxology("common.and_now"), th}, nil
	}
}
