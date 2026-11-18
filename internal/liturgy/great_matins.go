package liturgy

import (
	"fmt"
	"strconv"
	"time"
)

// Selectors of festal matins (a twelve feast, a saint with a sign): the
// Menaion's own hymns in the order of the Typikon.
//
//   - «Бог Господь»: the feast's troparion thrice; a saint's twice, «Слава,
//     и ныне», the resurrectional theotokion of its tone;
//   - after each kathisma and after the polyeleos the Menaion's sessional
//     hymns (else the Octoechos ones after the kathismata);
//   - the prokeimenon and the Gospel of matins; after Psalm 50 «Молитвами
//     апостол», «Молитвами Богородицы», «Помилуй мя, Боже» and the sticheron;
//   - the canon: at a feast the Menaion's canons only, with katavasia after
//     every ode; at a saint's as at daily matins;
//   - the exapostilaria of the Menaion; the stichera at the praises with
//     their verses; after the great doxology the troparion.

func greatMatinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	feast := sel.Part == "feast"
	switch sel.From {
	case "gm_troparia":
		return gmTroparia(f, feast)
	case "gm_sedalny":
		return gmSedalny(f, sel.Section)
	case "gm_magnification":
		if m := firstMain(f, "matins.magnification"); m != "" {
			return []SlotItem{{Ref: m, Repeat: 2, Performer: Clergy}, choir(m)}, nil
		}
		return nil, nil
	case "gm_prokeimenon":
		p, v := firstMain(f, "matins.prokeimenon"), firstMain(f, "matins.prokeimenon.verse")
		if p == "" || f.Tones[p] == 0 {
			return nil, fmt.Errorf("no prokeimenon of matins in the Menaion")
		}
		return []SlotItem{litanist(fmt.Sprintf("lit.prokeimenon.tone.%d", f.Tones[p])), litanist(p), choir(p),
			litanist(v), choir(p), litanist(p), choir(p)}, nil
	case "gm_gospel":
		g := firstMain(f, "matins.gospel")
		if g == "" {
			return nil, fmt.Errorf("no Gospel of matins in the Menaion")
		}
		return []SlotItem{litanist(g + ".announce"), choir("sl.lit.glory_to_thee_o_lord"), litanist("lit.let_us_attend"),
			priest(g), choir("sl.lit.glory_to_thee_o_lord")}, nil
	case "gm_ps50":
		items := []SlotItem{reader("ch.psalm.50"), choir("common.glory"), choir("ch.matins.ps50.glory"),
			choir("common.and_now"), choir("ch.matins.ps50.now"), choir("ch.matins.ps50.verse")}
		if s := firstMain(f, "matins.ps50.sticheron"); s != "" {
			items = append(items, choir(s))
		}
		return items, nil
	case "gm_canon":
		return festalCanon(f, sel.Section, feast)
	case "gm_exapostilaria":
		return gmExapostilaria(f, feast)
	case "gm_praises":
		return gmPraises(f)
	case "gm_sedalen":
		// The sessional hymn after the 3rd ode; some feasts have none there.
		if len(mainMenaion(f, "matins.sedalen")) == 0 {
			return nil, nil
		}
		return matinsMenaionSedalen(f)
	case "gm_end_troparia":
		t := mainMenaion(f, "troparion")
		if len(t) == 0 {
			return nil, fmt.Errorf("no troparion")
		}
		if feast {
			return []SlotItem{choir(t[0])}, nil
		}
		th, err := resurrectionalTheotokion(f, t[len(t)-1])
		if err != nil {
			return nil, err
		}
		return []SlotItem{choir(t[0]), choir("common.glory_now"), choir(th)}, nil
	}
	return nil, fmt.Errorf("unknown festal matins selector %q", sel.From)
}

func resurrectionalTheotokion(f Facts, troparion string) (string, error) {
	tone := f.Tones[troparion]
	if tone == 0 {
		return "", fmt.Errorf("unknown tone of %s", troparion)
	}
	return fmt.Sprintf("ch.theotokion.dismissal.%d.sun", tone), nil
}

func gmTroparia(f Facts, feast bool) ([]SlotItem, error) {
	t := mainMenaion(f, "troparion")
	if len(t) == 0 {
		return nil, fmt.Errorf("no Menaion troparion for %02d.%02d", int(f.Day.Menaion.Month), f.Day.Menaion.Day)
	}
	if feast {
		return []SlotItem{{Ref: t[0], Repeat: 3, Performer: Choir}}, nil
	}
	items := []SlotItem{{Ref: t[0], Repeat: 2, Performer: Choir}}
	if len(t) > 1 {
		items = append(items, choir("common.glory"), choir(t[1]), choir("common.and_now"))
	} else {
		items = append(items, choir("common.glory_now"))
	}
	th, err := resurrectionalTheotokion(f, t[len(t)-1])
	if err != nil {
		return nil, err
	}
	return append(items, choir(th)), nil
}

// gmSedalny: the Menaion's sessional hymns after kathisma k ("1".."3") or
// after the polyeleos ("p"); the Octoechos ones when the Menaion has none.
func gmSedalny(f Facts, k string) ([]SlotItem, error) {
	base := "matins.k_sedalen." + k
	hymns := mainMenaion(f, base)
	if len(hymns) == 0 {
		if k == "p" {
			return nil, nil
		}
		n, _ := strconv.Atoi(k)
		items, err := matinsSedalny(f, n)
		if err != nil {
			return nil, nil // no Octoechos sessional hymns (Sunday): the kathisma has none here
		}
		return items, nil
	}
	var items []SlotItem
	for _, h := range hymns {
		items = append(items, SlotItem{Ref: h})
	}
	th := theotokionFor(f, base)
	if g := firstMain(f, base+".glory"); g != "" {
		items = append(items, reader("common.glory"), SlotItem{Ref: g})
		if th != "" {
			items = append(items, reader("common.and_now"), SlotItem{Ref: th})
		}
	} else if th != "" {
		items = append(items, reader("common.glory_now"), SlotItem{Ref: th})
	}
	return items, nil
}

// festalCanon: at a feast the Menaion's canons, katavasia (the irmos of the
// last canon) after every ode; at a saint's, as at daily matins.
func festalCanon(f Facts, section string, feast bool) ([]SlotItem, error) {
	if !feast {
		return matinsCanon(f, section)
	}
	odes, err := parseOdes(section)
	if err != nil {
		return nil, err
	}
	cross := f.Day.Weekday == time.Wednesday || f.Day.Weekday == time.Friday
	var items []SlotItem
	for _, o := range odes {
		items = append(items, reader(fmt.Sprintf("ch.canon.ode.%d", o)))
		if o == 9 {
			if r, k, ok := odeNineRefrains(f, festalOdeCanons(f, o, cross)); ok {
				items = append(append(items, r...), reader("ch.canon.katavasia"))
				items = append(items, k...)
				continue
			}
		}
		last := ""
		for c := 1; c <= 3; c++ {
			b := fmt.Sprintf("matins.canon.%d.ode.%d", c, o)
			trop := mainMenaion(f, b+".troparion")
			if len(trop) == 0 {
				continue
			}
			if ir := firstMain(f, b+".irmos"); ir != "" {
				last = ir
				if c == 1 {
					items = append(items, SlotItem{Ref: ir})
				}
			}
			for _, t := range trop {
				items = append(items, SlotItem{Ref: t})
			}
			th := firstMain(f, b+".theotokion")
			if cross {
				if x := firstMain(f, b+".cross_theotokion"); x != "" {
					th = x
				}
			}
			if th != "" {
				items = append(items, SlotItem{Ref: th})
			}
		}
		if last == "" {
			return nil, fmt.Errorf("Menaion %02d.%02d: no canon at ode %d", int(f.Day.Menaion.Month), f.Day.Menaion.Day, o)
		}
		items = append(items, reader("ch.canon.katavasia"), SlotItem{Ref: last})
	}
	return items, nil
}

func gmExapostilaria(f Facts, feast bool) ([]SlotItem, error) {
	ex := mainMenaion(f, "matins.exapostilarion")
	if len(ex) == 0 {
		return matinsExapostilaria(f)
	}
	var items []SlotItem
	if feast && len(ex) == 1 {
		items = append(items, SlotItem{Ref: ex[0], Repeat: 3}) // the feast's: thrice
		return items, nil
	}
	items = append(items, SlotItem{Ref: ex[0]})
	th := theotokionFor(f, "matins.exapostilarion")
	if len(ex) > 1 {
		items = append(items, reader("common.glory"))
		for _, e := range ex[1:] {
			items = append(items, SlotItem{Ref: e})
		}
		if th != "" {
			items = append(items, reader("common.and_now"), SlotItem{Ref: th})
		}
		return items, nil
	}
	if th != "" {
		items = append(items, reader("common.glory_now"), SlotItem{Ref: th})
	}
	return items, nil
}

// The verses of the praises before the stichera, the last N of them.
var praisesVerses = []string{"ch.matins.praises.149.2", "ch.matins.praises.149.3", "ch.matins.praises.149.verse",
	"ch.matins.praises.150.1", "ch.matins.praises.150.2", "ch.matins.praises.150.3", "ch.matins.praises.150.4",
	"ch.matins.praises.150.5"}

func gmPraises(f Facts) ([]SlotItem, error) {
	st := mainMenaion(f, "matins.praises.sticheron")
	if len(st) == 0 {
		return nil, nil
	}
	places := markerN(mainMenaion(f, "matins.praises.on"))
	if places < 4 || places > 8 {
		places = 4
	}
	start := len(praisesVerses) - places
	verses := praisesVerses[start:]
	var items []SlotItem
	// The verses before the stichera that the reading of Ps 149 (to verse 8)
	// leaves out: «Сотворити в них суд», «Хвалите Бога во святых Его» —
	// read with the doxology read, else by the practice.
	for i := 2; i < start; i++ {
		if !f.praisesSung() {
			items = append(items, reader(praisesVerses[i]))
		} else if it, ok := f.psalmVerse(praisesVerses[i]); ok {
			items = append(items, it)
		}
	}
	for i, v := range verses {
		items = append(items, reader(v), choir(st[i%len(st)]))
	}
	g, now := firstMain(f, "matins.praises.glory"), firstMain(f, "matins.praises.now")
	switch {
	case g != "" && now != "":
		items = append(items, reader("common.glory"), choir(g), reader("common.and_now"), choir(now))
	case now != "":
		items = append(items, reader("common.glory_now"), choir(now))
	case g != "":
		items = append(items, reader("common.glory"), choir(g))
	}
	return items, nil
}

// festalOdeCanons: the Menaion's canons in an ode, each with its irmos.
func festalOdeCanons(f Facts, o int, cross bool) []odeCanon {
	var out []odeCanon
	for c := 1; c <= 3; c++ {
		b := fmt.Sprintf("matins.canon.%d.ode.%d", c, o)
		trop := mainMenaion(f, b+".troparion")
		if len(trop) == 0 {
			continue
		}
		cn := odeCanon{irmos: firstMain(f, b+".irmos"), troparia: append([]string(nil), trop...)}
		th := firstMain(f, b+".theotokion")
		if cross {
			if x := firstMain(f, b+".cross_theotokion"); x != "" {
				th = x
			}
		}
		if th != "" {
			cn.troparia = append(cn.troparia, th)
		}
		if cn.irmos == "" && len(out) > 0 {
			// A «canon» without its irmos is the rest of the one before.
			out[len(out)-1].troparia = append(out[len(out)-1].troparia, cn.troparia...)
			continue
		}
		out = append(out, cn)
	}
	return out
}
