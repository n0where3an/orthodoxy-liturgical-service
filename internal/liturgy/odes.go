package liturgy

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

// The verses of the biblical odes sung with the troparia of the matins
// canon (Ирмологий, «песней стихи»; the Typikon, -typikon). A canon «на N»
// in an ode takes the last N−2 verses before «Слава» and «И ныне»; when the
// ode has fewer verses, the first places are sung without one (the irmos
// and one troparion: the 6th, 7th, 9th odes; on Sunday the troparion with
// «Слава, Господи, святому воскресению Твоему»).

// odeVerses lists the verses of the biblical odes of a set (weekday,
// sunday, lent) by ode, in the order of the book.
func odeVerses(ctx context.Context, l MenaionLister, set string) (map[int][]string, error) {
	ids, err := l.ListIDs(ctx, "ir.ode.")
	if err != nil {
		return nil, err
	}
	out := map[int][]string{}
	num := map[string]int{}
	for _, id := range ids {
		p := strings.Split(id, ".") // ir ode O set N
		if len(p) != 5 || p[3] != set {
			continue
		}
		o, err1 := strconv.Atoi(p[2])
		n, err2 := strconv.Atoi(p[4])
		if err1 != nil || err2 != nil {
			continue
		}
		out[o] = append(out[o], id)
		num[id] = n
	}
	for o := range out {
		sort.Slice(out[o], func(i, j int) bool { return num[out[o][i]] < num[out[o][j]] })
	}
	return out, nil
}

// withOdeVerses puts the verses of ode o before the places of a canon
// (items: the irmos, troparia, theotokia of all its canons in the ode).
func withOdeVerses(f Facts, set string, o int, items []SlotItem) []SlotItem {
	verses := f.OdeVerses[o]
	n := len(items)
	if len(verses) == 0 || n < 3 {
		return items
	}
	if set == "feast" {
		// «Аще влдчний праздник, или празднуемаго стаго есть, глаголем
		// ирмос дважды», then the verses with the troparia.
		first := items[0]
		first.Repeat = 2
		return append([]SlotItem{first}, versesBefore(verses, items[1:], "")...)
	}
	return versesBefore(verses, items, set)
}

// versesBefore puts the verses before all but the last two items («Слава»,
// «И ныне»), counted from the end; the first items without one when the ode
// has fewer verses (on Sunday the second with the refrain of the resurrection).
func versesBefore(verses []string, items []SlotItem, set string) []SlotItem {
	n := len(items)
	if n < 2 {
		return items
	}
	versed := min(n-2, len(verses))
	unversed := n - 2 - versed
	verses = verses[len(verses)-versed:]
	var out []SlotItem
	for i, it := range items {
		switch {
		case i < unversed:
			if set == "sunday" && i == 1 {
				out = append(out, reader("ir.sunday.refrain"))
			}
		case i < n-2:
			out = append(out, reader(verses[i-unversed]))
		case i == n-2:
			out = append(out, reader("common.glory"))
		default:
			out = append(out, reader("common.and_now"))
		}
		out = append(out, it)
	}
	return out
}
