package liturgy

import (
	"fmt"
	"strconv"
	"strings"
)

// Selectors of Sunday matins (the all-night vigil after Sunday vespers),
// after the Octoechos («В неделю на утрени») and the Typikon. Part:
//
//   - "sunday": a saint without a sign (or no Menaion service);
//   - "saint": a saint with a sign (polyeleos, vigil);
//   - "feast": a twelve feast of the Theotokos;
//   - "afterfeast": the afterfeast or apodosis of a twelve feast.
//
// «Бог Господь»: the resurrectional troparion twice, «Слава» the saint's,
// «И ныне» the resurrectional theotokion of its tone (at a feast and in the
// afterfeast «И ныне» the feast's). The sessional hymns, the evlogitaria
// «Ангельский собор», the hypakoe, the antiphons and the prokeimenon of the
// tone, the Gospel of the eleven (eothinon), «Воскресение Христово видевше»,
// Psalm 50 with «Воскрес Иисус от гроба». The canon: resurrectional on 4,
// of the Cross and Resurrection on 3, of the Theotokos on 3 and the
// Menaion's on 4; with a saint with a sign the Octoechos 4 + 2 and the
// saint's on 8; at a feast and in the afterfeast the Octoechos 4 and the
// Menaion's on 10; the katavasia after every ode. After the 3rd ode the
// saint's kontakion, ikos and sessional hymn, after the 6th the Sunday
// kontakion and ikos (at a feast the other way round). The exapostilarion
// of the eothinon, «Слава» the Menaion's, «И ныне» its theotokion; the
// praises on 8 (the Octoechos 4 and the Menaion 4 when it has stichera),
// «Слава» the Gospel sticheron, «И ныне» «Преблагословенна еси»; after the
// doxology the resurrectional troparion only.

// eothinonGospels: the book and zachalo of the eleven matins Gospels.
var eothinonGospels = [12]struct {
	book string
	n    int
}{{}, {"mt", 116}, {"mk", 70}, {"mk", 71}, {"lk", 112}, {"lk", 113}, {"lk", 114}, {"jn", 63}, {"jn", 64}, {"jn", 65},
	{"jn", 66}, {"jn", 67}}

func sundayMatinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	part := sel.Part
	if part == "" {
		part = "sunday"
	}
	oc := func(k string) string { return first(f.Octoechos, "matins."+k) }
	switch sel.From {
	case "sm_troparia":
		return smTroparia(f, part)
	case "sm_sedalny":
		return smSedalny(f, sel.Section)
	case "sm_evlogitaria":
		v := "oc.sun.evlogitaria.verse"
		var items []SlotItem
		for i := 1; i <= 4; i++ {
			items = append(items, reader(v), choir(fmt.Sprintf("oc.sun.evlogitaria.%d", i)))
		}
		return append(items, reader("common.glory"), choir("oc.sun.evlogitaria.glory"), reader("common.and_now"),
			choir("oc.sun.evlogitaria.now"), SlotItem{Ref: "common.alleluia_glory_to_thee", Repeat: 3, Performer: Choir}), nil
	case "sm_hypakoe":
		if h := oc("hypakoe"); h != "" {
			return []SlotItem{reader(h)}, nil
		}
		return nil, fmt.Errorf("Octoechos tone %d: no hypakoe", f.Day.Tone)
	case "sm_stepenna":
		return smStepenna(f)
	case "sm_prokeimenon":
		p, v := oc("prokeimenon"), oc("prokeimenon.verse")
		if p == "" {
			return nil, fmt.Errorf("Octoechos tone %d: no matins prokeimenon", f.Day.Tone)
		}
		items := []SlotItem{litanist(fmt.Sprintf("lit.prokeimenon.tone.%d", f.Day.Tone)), litanist(p), choir(p)}
		if v != "" {
			items = append(items, litanist(v), choir(p))
		}
		return append(items, litanist(p), choir(p)), nil
	case "sm_gospel":
		if f.Eothinon < 1 || f.Eothinon > 11 {
			return nil, fmt.Errorf("no eothinon on %v", f.Day.Date)
		}
		e := eothinonGospels[f.Eothinon]
		g := fmt.Sprintf("lect.zach.%s.%d", e.book, e.n)
		return []SlotItem{litanist(g + ".announce"), choir("sl.lit.glory_to_thee_o_lord"), litanist("lit.let_us_attend"),
			priest(g), choir("sl.lit.glory_to_thee_o_lord")}, nil
	case "sm_after_gospel":
		return []SlotItem{choir("oc.1.sun.matins.resurrection_beheld")}, nil
	case "sm_ps50":
		return []SlotItem{reader("ch.psalm.50"), choir("common.glory"), choir("ch.matins.ps50.glory"),
			choir("common.and_now"), choir("ch.matins.ps50.now"), choir("ch.matins.ps50.verse"),
			choir("oc.1.sun.matins.ps50.sticheron")}, nil
	case "sm_canon":
		return smCanon(f, sel.Section, part)
	case "sm_sedalen": // after the 3rd ode
		return smAfterOde(f, part, 3)
	case "sm_kontakion": // after the 6th ode
		return smAfterOde(f, part, 6)
	case "sm_exapostilaria":
		return smExapostilaria(f, part)
	case "sm_praises":
		return smPraises(f, part)
	case "sm_end_troparion":
		if f.Day.Tone%2 == 1 {
			return []SlotItem{choir("oc.1.sun.matins.end_troparion")}, nil // «Днесь спасение»
		}
		return []SlotItem{choir("oc.4.sun.matins.end_troparion")}, nil // «Воскрес из гроба»
	}
	return nil, fmt.Errorf("unknown Sunday matins selector %q", sel.From)
}

// saintHymn: the first Menaion hymn of a kind that is the saint's (not the
// feast's), "" if none.
func saintHymn(f Facts, kind string) string {
	for _, id := range mainMenaion(f, kind) {
		if !f.ofFeast(id) && (f.Feast == nil || (id != f.Feast.Troparion && id != f.Feast.Kontakion)) {
			return id
		}
	}
	return ""
}

func smTroparia(f Facts, part string) ([]SlotItem, error) {
	res := first(f.Octoechos, "troparion")
	if res == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion", f.Day.Tone)
	}
	items := []SlotItem{{Ref: res, Repeat: 2, Performer: Choir}}
	switch part {
	case "feast":
		t := mainMenaion(f, "troparion")
		if len(t) == 0 {
			return nil, fmt.Errorf("no troparion of the feast")
		}
		return append(items, reader("common.glory_now"), choir(t[0])), nil
	case "afterfeast":
		if f.Feast == nil {
			return nil, fmt.Errorf("no feast")
		}
		if t := saintHymn(f, "troparion"); t != "" {
			return append(items, reader("common.glory"), choir(t), reader("common.and_now"), choir(f.Feast.Troparion)), nil
		}
		return append(items, reader("common.glory_now"), choir(f.Feast.Troparion)), nil
	}
	t := saintHymn(f, "troparion")
	if t == "" {
		return append(items, reader("common.glory_now"), choir(fmt.Sprintf("ch.theotokion.dismissal.%d.sun", f.Day.Tone))), nil
	}
	th, err := resurrectionalTheotokion(f, t) // the resurrectional theotokion of the saint's tone
	if err != nil {
		return nil, err
	}
	return append(items, reader("common.glory"), choir(t), reader("common.and_now"), choir(th)), nil
}

func smSedalny(f Facts, k string) ([]SlotItem, error) {
	base := "matins.sedalen." + k
	hymns := f.Octoechos[base]
	if len(hymns) == 0 {
		return nil, fmt.Errorf("Octoechos tone %d: no Sunday sessional hymns after kathisma %s", f.Day.Tone, k)
	}
	verses := map[string]string{}
	for _, id := range f.Octoechos[base+".verse"] {
		verses[id[strings.LastIndex(id, ".")+1:]] = id
	}
	var items []SlotItem
	for i, id := range hymns {
		if v := verses[strconv.Itoa(i+1)]; v != "" {
			items = append(items, reader(v))
		}
		items = append(items, choir(id))
	}
	if th := first(f.Octoechos, base+".theotokion"); th != "" {
		items = append(items, reader("common.glory_now"), choir(th))
	}
	return items, nil
}

func smStepenna(f Facts) ([]SlotItem, error) {
	var items []SlotItem
	for a := 1; a <= 4; a++ {
		verses := f.Octoechos[fmt.Sprintf("matins.stepenna.%d", a)]
		if len(verses) == 0 {
			break
		}
		for _, v := range verses {
			items = append(items, choir(v))
		}
		if g := first(f.Octoechos, fmt.Sprintf("matins.stepenna.%d.glory", a)); g != "" {
			items = append(items, choir("common.glory"), choir(g), choir("common.and_now"), choir(g))
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("Octoechos tone %d: no antiphons", f.Day.Tone)
	}
	return items, nil
}

// canonPart is a canon sung «на N» in an ode: its irmos (first canon only),
// troparia (repeated to fill N) and theotokion.
type canonPart struct {
	prefix string // …canon.<C> (Octoechos: matins.canon.<C>, Menaion: me.MM.DD.matins.canon.<C>)
	on     int
	irmos  bool
	oct    bool
}

// odeOn fills «на N» places with an ode of a canon.
func odeOn(irmos string, troparia []string, theotokion string, n int) []SlotItem {
	var items []SlotItem
	if irmos != "" {
		items = append(items, SlotItem{Ref: irmos})
	}
	need := n - len(items)
	if theotokion != "" {
		need--
	}
	for i := 0; i < need && len(troparia) > 0; i++ {
		items = append(items, SlotItem{Ref: troparia[i%len(troparia)]})
	}
	if theotokion != "" {
		items = append(items, SlotItem{Ref: theotokion})
	}
	return items
}

func smCanon(f Facts, section, part string) ([]SlotItem, error) {
	odes, err := parseOdes(section)
	if err != nil {
		return nil, err
	}
	// The Menaion's canons of the main service.
	var men []string
	for c := 1; c <= 3; c++ {
		if len(mainMenaion(f, fmt.Sprintf("matins.canon.%d.ode.1.troparion", c))) > 0 ||
			len(mainMenaion(f, fmt.Sprintf("matins.canon.%d.ode.9.troparion", c))) > 0 {
			men = append(men, fmt.Sprintf("matins.canon.%d", c))
		}
	}
	parts := []canonPart{{prefix: "matins.canon.1", on: 4, irmos: true, oct: true}}
	split := func(total int) {
		switch len(men) {
		case 0:
		case 1:
			parts = append(parts, canonPart{prefix: men[0], on: total})
		default:
			first := total - total/2 + 1 // 10: 6 + 4; 8: 5 + 3 → the feast's/first canon has more
			if total == 8 {
				first = 4
			}
			parts = append(parts, canonPart{prefix: men[0], on: first}, canonPart{prefix: men[1], on: total - first})
		}
	}
	switch part {
	case "saint":
		parts = append(parts, canonPart{prefix: "matins.canon.3", on: 2, oct: true})
		split(8)
	case "feast", "afterfeast":
		split(10)
	default:
		parts = append(parts, canonPart{prefix: "matins.canon.2", on: 3, oct: true}, canonPart{prefix: "matins.canon.3", on: 3, oct: true})
		if len(men) > 0 {
			parts = append(parts, canonPart{prefix: men[0], on: 4})
		}
	}
	var items []SlotItem
	for _, o := range odes {
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		start := len(items)
		if o == 9 && part == "feast" {
			// The resurrectional canon, then the feast's with its refrains.
			if r, k, ok := odeNineRefrains(f, festalOdeCanons(f, o, false)); ok {
				b := "matins.canon.1.ode.9"
				items = append(items, odeOn(first(f.Octoechos, b+".irmos"), f.Octoechos[b+".troparion"],
					first(f.Octoechos, b+".theotokion"), 4)...)
				items = append(append(items, r...), reader("ch.canon.katavasia"))
				items = append(items, k...)
				continue
			}
		}
		for _, p := range parts {
			b := fmt.Sprintf("%s.ode.%d", p.prefix, o)
			var trop []string
			irmos, theot := "", ""
			if p.oct {
				trop = f.Octoechos[b+".troparion"]
				theot = first(f.Octoechos, b+".theotokion")
				if p.irmos {
					irmos = first(f.Octoechos, b+".irmos")
				}
			} else {
				trop = mainMenaion(f, b+".troparion")
				theot = firstMain(f, b+".theotokion")
			}
			if len(trop) == 0 {
				continue
			}
			items = append(items, odeOn(irmos, trop, theot, p.on)...)
		}
		if len(f.OdeVerses) > 0 { // -typikon: the verses of the biblical ode
			if o != 9 || f.Refrains == nil {
				items = append(items[:start], withOdeVerses(f, "sunday", o, append([]SlotItem(nil), items[start:]...))...)
			}
		}
		k := smKatavasia(f, part, o)
		if k == "" {
			return nil, fmt.Errorf("no katavasia of ode %d", o)
		}
		items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: k})
	}
	return items, nil
}

// smKatavasia: the seasonal katavasia (data/calendar/katavasia.yaml); out
// of the listed periods, at a feast and in its afterfeast the irmos of the
// feast's first canon, otherwise «Отверзу уста моя».
func smKatavasia(f Facts, part string, ode int) string {
	if f.KatavasiaSeasonal {
		return f.Katavasia[ode]
	}
	switch part {
	case "feast":
		if k := firstMain(f, fmt.Sprintf("matins.canon.1.ode.%d.irmos", ode)); k != "" {
			return k
		}
	case "afterfeast":
		if k := feastMain(f, fmt.Sprintf("matins.canon.1.ode.%d.irmos", ode)); k != "" {
			return k
		}
	}
	return f.Katavasia[ode]
}

// smAfterOde: the kontakion and ikos (and after the 3rd ode the sessional
// hymn of the Menaion): of the saint (feast) after the 3rd, the Sunday's
// after the 6th; at a twelve feast the other way round.
func smAfterOde(f Facts, part string, ode int) ([]SlotItem, error) {
	sunday := []SlotItem{reader(first(f.Octoechos, "kontakion")), reader(first(f.Octoechos, "matins.ikos"))}
	if sunday[0].Ref == "" || sunday[1].Ref == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no Sunday kontakion or ikos", f.Day.Tone)
	}
	var men []SlotItem
	kon := saintHymn(f, "kontakion")
	if part == "feast" {
		kon = firstMain(f, "kontakion")
	}
	if kon != "" {
		men = append(men, reader(kon))
		if ikos := firstMain(f, "matins.ikos"); ikos != "" {
			men = append(men, reader(ikos))
		}
	}
	sedalen := func() []SlotItem {
		if part == "saint" || part == "feast" {
			if len(mainMenaion(f, "matins.sedalen")) == 0 {
				return nil
			}
			items, _ := matinsMenaionSedalen(f)
			return items
		}
		items, err := matinsMenaionSedalen(f)
		if err != nil {
			return nil
		}
		return items
	}
	switch {
	case ode == 3 && part == "feast":
		return append(sunday, sedalen()...), nil
	case ode == 3:
		return append(men, sedalen()...), nil
	case part == "feast":
		return men, nil
	}
	return sunday, nil
}

func smExapostilaria(f Facts, part string) ([]SlotItem, error) {
	if f.Eothinon < 1 {
		return nil, fmt.Errorf("no eothinon on %v", f.Day.Date)
	}
	e := fmt.Sprintf("oc.eothinon.%d.", f.Eothinon)
	items := []SlotItem{choir(e + "exapostilarion")}
	ex := mainMenaion(f, "matins.exapostilarion")
	if len(ex) == 0 {
		return append(items, reader("common.glory_now"), choir(e+"theotokion")), nil
	}
	items = append(items, reader("common.glory"))
	for _, x := range ex[:len(ex)-1] {
		items = append(items, choir(x))
	}
	last := ex[len(ex)-1]
	switch part {
	case "feast", "afterfeast":
		// «Слава» … «И ныне» the feast's (the Menaion's last exapostilarion).
		if len(ex) == 1 {
			return append(items, choir(last), reader("common.and_now"), choir(last)), nil
		}
		return append(items, reader("common.and_now"), choir(last)), nil
	}
	items = append(items, choir(last))
	th := theotokionFor(f, "matins.exapostilarion")
	if th == "" {
		th = e + "theotokion"
	}
	return append(items, reader("common.and_now"), choir(th)), nil
}

func smPraises(f Facts, part string) ([]SlotItem, error) {
	oct := f.Octoechos["matins.praises.sticheron"]
	if len(oct) < 8 || f.Eothinon < 1 {
		return nil, fmt.Errorf("Octoechos tone %d: %d praises stichera (eothinon %d)", f.Day.Tone, len(oct), f.Eothinon)
	}
	st := oct[:8]
	if men := mainMenaion(f, "matins.praises.sticheron"); len(men) > 0 && part != "sunday" {
		st = append([]string{}, oct[:4]...)
		for i := 0; i < 4; i++ {
			st = append(st, men[i%len(men)])
		}
	}
	verses := praisesVerses[len(praisesVerses)-8:]
	var items []SlotItem
	for i, v := range verses {
		items = append(items, reader(v), choir(st[i]))
	}
	now := "oc.sun.praises.now"
	if part == "feast" {
		if n := firstMain(f, "matins.praises.now"); n != "" {
			now = n
		}
	}
	return append(items, reader("common.glory"), choir(fmt.Sprintf("oc.eothinon.%d.sticheron", f.Eothinon)),
		reader("common.and_now"), choir(now)), nil
}
