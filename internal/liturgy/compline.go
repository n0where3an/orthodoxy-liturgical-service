package liturgy

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Small compline (Часослов, «Повечерие малое»), served the evening before
// the liturgical day — like vespers it is planned for the day it begins:
// «в неделю на повечерии» is the compline of Monday.
//
//   - the canon: «канон молебный престей бцде, во осмогласнике» (Типикон,
//     гл. 9) — the Octoechos's of the day (oc.<tone>.<day>.compline), read,
//     «Пресвятая Богородице, спаси нас» before each troparion; after the 6th
//     ode «Господи, помилуй» thrice, «Слава, и ныне», the sessional hymn;
//   - after «Достойно есть» and the Trisagion the troparia of the day
//     (Типикон, гл. 52), below.

func complineSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	switch sel.From {
	case "compline_canon":
		return complineCanon(f)
	case "compline_troparia":
		return complineTroparia(f), nil
	}
	return nil, fmt.Errorf("unknown compline selector %q", sel.From)
}

func complineCanon(f Facts) ([]SlotItem, error) {
	return octoechosReadCanon(f, "compline", "sl.vespers.most_holy_theotokos_save_us")
}

// octoechosReadCanon: a canon of the Octoechos read at compline or at the
// midnight office (oc.<tone>.<day>.<sub>.canon), the refrain before each
// troparion; after an ode with a sessional hymn «Господи, помилуй» thrice and
// the hymn — at «Слава, и ныне», or followed by «Слава, и ныне» and its
// theotokion.
func octoechosReadCanon(f Facts, sub, refrainRef string) ([]SlotItem, error) {
	p := sub + ".canon.ode."
	if len(f.Octoechos[p+"1.troparion"]) == 0 {
		return nil, fmt.Errorf("Octoechos tone %d, %v: no canon (%s)", f.Day.Tone, f.Day.Weekday, sub)
	}
	refrain := reader(refrainRef)
	sedalny := map[string]string{} // ode → id
	for _, id := range f.Octoechos[sub+".sedalen"] {
		sedalny[id[strings.LastIndex(id, ".")+1:]] = id
	}
	var items []SlotItem
	for _, o := range odeOrder {
		b := fmt.Sprintf("%s%d.", p, o)
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		if ir := first(f.Octoechos, b+"irmos"); ir != "" {
			items = append(items, reader(ir))
		}
		for _, t := range f.Octoechos[b+"troparion"] {
			items = append(items, refrain, reader(t))
		}
		if g := first(f.Octoechos, b+"glory"); g != "" {
			items = append(items, reader("common.glory"), reader(g))
		}
		if th := first(f.Octoechos, b+"theotokion"); th != "" {
			items = append(items, reader("common.and_now"), reader(th))
		}
		if s := sedalny[strconv.Itoa(o)]; s != "" {
			items = append(items, SlotItem{Ref: "common.lord_have_mercy", Repeat: 3, Performer: Reader})
			if th := first(f.Octoechos, fmt.Sprintf("%s.sedalen.%d.theotokion", sub, o)); th != "" {
				items = append(items, reader(s), reader("common.glory_now"), reader(th))
			} else {
				items = append(items, reader("common.glory_now"), reader(s))
			}
		}
	}
	return items, nil
}

// complineTroparia: after the Trisagion (Типикон, гл. 52):
//
//   - in a forefeast or afterfeast the feast's kontakion; with a saint whose
//     great doxology is sung, his kontakion, «Слава, и ныне» the feast's;
//   - a saint with a sign (the doxology and higher): his kontakion only;
//   - Saturday evening (no vigil): the kontakion of the tone;
//   - otherwise by the day: the troparion of a church of the Lord or of the
//     Theotokos, the day's, the troparion of a saint's church; «Боже отец
//     наших», «Иже во всем мире»; «Слава» «Со святыми упокой», «И ныне»
//     «Молитвами, Господи». The day's hymn stands for a church of the same
//     dedication. On Tuesday and Thursday evenings («Спаси, Господи») no
//     church of the Lord; on Friday evening «Апостоли, мученицы», «Помяни,
//     Господи» (not with a church of the Lord or of the Theotokos, nor when
//     Saturday's saint has a troparion), «Слава» «Со святыми упокой», «И
//     ныне» «Яко начатки».
func complineTroparia(f Facts) []SlotItem {
	feastK := ""
	if f.Feast != nil && !f.Feast.IsFeastDay(f.Day) {
		feastK = f.Feast.Kontakion
	} else {
		for _, k := range mainMenaion(f, "kontakion") {
			if f.ofFeast(k) { // «предпразднства»
				feastK = k
				break
			}
		}
	}
	saintK := saintHymn(f, "kontakion")
	sign := f.greatDoxology()
	switch {
	case feastK != "" && sign && saintK != "":
		return []SlotItem{reader(saintK), reader("common.glory_now"), reader(feastK)}
	case feastK != "":
		return []SlotItem{reader(feastK)}
	case sign && saintK != "":
		return []SlotItem{reader(saintK)}
	case f.Day.Weekday == time.Sunday:
		if k := first(f.Octoechos, "kontakion"); k != "" {
			return []SlotItem{reader(k)}
		}
	}

	t := f.Temple
	is := func(kinds ...TempleKind) bool {
		if t == nil {
			return false
		}
		for _, k := range kinds {
			if t.Kind == k {
				return true
			}
		}
		return false
	}
	isID := func(ids ...string) bool {
		if t == nil {
			return false
		}
		for _, id := range ids {
			if t.ID == id {
				return true
			}
		}
		return false
	}
	var items []SlotItem
	add := func(refs ...string) {
		for _, r := range refs {
			items = append(items, reader(r))
		}
	}
	switch f.Day.Weekday {
	case time.Monday, time.Tuesday:
		if is(TempleLord, TempleTheotokos) {
			add(t.Troparion)
		}
		add(weekdayLiturgy[f.Day.Weekday].troparia...)
		// The bodiless powers on Monday, the Forerunner on Tuesday: the day's.
		if is(TempleSaint) && !isID("archangel_michael", "nativity_of_forerunner") {
			add(t.Troparion)
		}
	case time.Wednesday, time.Friday:
		add("ch.troparion.day.wed") // «Спаси, Господи»: no church of the Lord
		if is(TempleTheotokos, TempleSaint) {
			add(t.Troparion)
		}
	case time.Thursday:
		if is(TempleLord, TempleTheotokos) || isID("archangel_michael", "nativity_of_forerunner") {
			add(t.Troparion)
		}
		add("ch.troparion.day.thu", "ch.troparion.day.thu.2")
		// Not the apostles' (the day's) nor St Nicholas's.
		if is(TempleSaint) && !isID("archangel_michael", "nativity_of_forerunner", "peter_and_paul", "nicholas") {
			add(t.Troparion)
		}
	case time.Saturday:
		lordOrTheotokos := is(TempleLord, TempleTheotokos)
		if lordOrTheotokos {
			add(t.Troparion)
		}
		add("ch.troparion.day.sat")
		if !lordOrTheotokos && len(mainMenaion(f, "troparion")) == 0 {
			add("ch.troparion.day.sat.2") // «Помяни, Господи»
		}
		add("common.glory", "ch.kontakion.with_the_saints", "common.and_now", "ch.kontakion.day.sat")
		return items
	}
	add("ch.compline.troparion.fathers", "ch.compline.troparion.martyrs",
		"common.glory", "ch.kontakion.with_the_saints", "common.and_now", "ch.compline.theotokion")
	return items
}
