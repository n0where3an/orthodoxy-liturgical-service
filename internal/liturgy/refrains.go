package liturgy

import "fmt"

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
}

// refrain returns the id of a refrain of the day's feast, "" if none.
func refrain(f Facts, canon int, key string) string {
	if f.Feast == nil {
		return ""
	}
	id := fmt.Sprintf("ir.pripev.%s.%d.%s", f.Feast.ID, canon, key)
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
		c := min(i+1, 2)
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
