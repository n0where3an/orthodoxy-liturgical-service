package liturgy

import (
	"fmt"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// The 9th ode at a twelve feast (Ирмологий, «Припевы на 9-й песни»):
// «Честнейшую» is not sung; the irmos of each canon twice (both choirs) with
// the refrain of the feast, each troparion with a refrain — the book's
// sequence when it gives several, otherwise the irmos's — «вместо Слава»,
// «вместо И ныне» where given; the katavasia the refrain with the irmos.

// odeCanon is one canon in an ode: its irmos and troparia (with the
// theotokion, if any).
type odeCanon struct {
	irmos    string
	troparia []string
	n        int // the canon's number in the Menaion (0: by its place)
}

// refrain returns the id of a refrain of the day's feast (or of the day's
// own set, RefrainSet: the Circumcision, the Three Hierarchs), "" if none;
// the third canon takes RefrainThird's (01.07: the Forerunner's).
func refrain(f Facts, canon int, key string) string {
	set := f.RefrainSet
	if set == "" && f.Feast != nil {
		set = f.Feast.ID
	}
	if canon >= 3 {
		if f.RefrainThird == "" {
			canon = 2
		} else {
			set, canon = f.RefrainThird, 1
		}
	}
	if set == "" {
		return ""
	}
	id := fmt.Sprintf("ir.pripev.%s.%d.%s", set, canon, key)
	if f.Refrains[id] {
		return id
	}
	return ""
}

// odeNineRefrains builds the 9th ode with the refrains; ok is false when
// the feast has none.
func odeNineRefrains(f Facts, canons []odeCanon) (items, katavasia []SlotItem, ok bool) {
	if len(f.Refrains) == 0 || refrain(f, 1, "irmos") == "" {
		return nil, nil, false
	}
	var gloryAt, nowAt int = -1, -1
	for i, cn := range canons {
		n := cn.n
		if n == 0 {
			n = i + 1
		}
		c := min(n, 2)
		if n >= 3 && f.RefrainThird != "" {
			c = 3 // the third canon (01.07: the Forerunner's): its own refrains (refrain)
		}
		ri := refrain(f, c, "irmos")
		if ri == "" {
			ri = refrain(f, 1, "irmos")
		}
		ri2 := refrain(f, c, "irmos2")
		if ri2 == "" {
			ri2 = ri
		}
		if cn.irmos != "" {
			items = append(items, choir(ri), SlotItem{Ref: cn.irmos}, choir(ri2), SlotItem{Ref: cn.irmos})
			if i < 2 {
				katavasia = append(katavasia, choir(ri), SlotItem{Ref: cn.irmos})
			}
		}
		var list []string
		for k := 1; refrain(f, c, fmt.Sprint(k)) != ""; k++ {
			list = append(list, refrain(f, c, fmt.Sprint(k)))
		}
		for j, t := range cn.troparia {
			r := ri
			if len(list) > 0 {
				r = list[j%len(list)]
			}
			if i == len(canons)-1 && len(cn.troparia) >= 2 {
				switch j {
				case len(cn.troparia) - 2:
					gloryAt = len(items)
				case len(cn.troparia) - 1:
					nowAt = len(items)
				}
			}
			items = append(items, choir(r), SlotItem{Ref: t})
		}
	}
	// «Вместо Слава», «вместо И ныне» of the last canon given.
	last := min(len(canons), 2)
	if g := refrain(f, last, "glory"); g != "" && gloryAt >= 0 {
		items[gloryAt] = choir(g)
	}
	if n := refrain(f, last, "now"); n != "" && nowAt >= 0 {
		items[nowAt] = choir(n)
	}
	return items, katavasia, true
}

// dayRefrains: the days whose Menaion has the 9th ode sung with refrains of
// their own («на ф-й песни честнейшую не поем, но поем припевы»): the
// Circumcision (01.01, in the Nativity's afterfeast), the Three Hierarchs
// (01.30); the synaxis of the Forerunner (01.07) the Theophany's and, at his
// canon, the Forerunner's («припевы праздника, и предтечи»). The
// Irmologion's sets (ir.pripev.<set>.*).
func dayRefrains(day calendar.DayContext) (set, third string) {
	switch {
	case day.Menaion.Month == time.January && day.Menaion.Day == 1:
		return "circumcision", ""
	case day.Menaion.Month == time.January && day.Menaion.Day == 30:
		return "three_hierarchs", ""
	case day.Menaion.Month == time.January && day.Menaion.Day == 7:
		return "theophany", "synaxis_of_forerunner"
	}
	return "", ""
}
