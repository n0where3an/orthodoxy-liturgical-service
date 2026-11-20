package liturgy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// royalHours: the feast whose Royal Hours are sung on the day — on its eve
// (24 December, 5 January) when that is a weekday; when the feast falls on
// Sunday or Monday «часы поем прежде в пяток» (the Menaion): the Friday
// before.
func royalHours(day calendar.DayContext) string {
	m, d, wd := day.Menaion.Month, day.Menaion.Day, day.Weekday
	weekday := wd >= time.Monday && wd <= time.Friday
	switch {
	case m == time.December && (d == 24 && weekday || (d == 22 || d == 23) && wd == time.Friday):
		return "nativity"
	case m == time.January && (d == 5 && weekday || (d == 3 || d == 4) && wd == time.Friday):
		return "theophany"
	}
	return ""
}

// royalHoursSlot fills the parts of a Royal Hour (rh.<feast>.<H>.*,
// orthlib.ImportRoyalHours): "troparion" (at «Слава»), "kontakion",
// "psalms", "hymns" (the stichera with their verses), "prokeimenon",
// "readings" (the prophecy, the Apostle, the Gospel).
func royalHoursSlot(sel SlotSelector, f Facts, service string) ([]SlotItem, error) {
	h := map[string]string{"first_hour": "1", "third_hour": "3", "sixth_hour": "6", "ninth_hour": "9"}[service]
	if f.Royal == nil || h == "" && sel.Part != "kontakion" {
		return nil, fmt.Errorf("no Royal Hours for %s", service)
	}
	u := func(k string) string { return f.Royal[h+"."+k] }
	need := func(k string) (string, error) {
		if id := u(k); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("Royal Hours: no %s.%s in the corpus", h, k)
	}
	switch sel.Part {
	case "troparion":
		t, err := need("troparion")
		return []SlotItem{reader(t)}, err
	case "kontakion":
		k := f.Royal["kontakion"]
		if k == "" {
			return nil, fmt.Errorf("Royal Hours: no kontakion")
		}
		return []SlotItem{reader(k)}, nil
	case "psalms":
		var items []SlotItem
		for n := 1; n <= 3; n++ {
			p, err := need(fmt.Sprintf("psalm.%d", n))
			if err != nil {
				return nil, err
			}
			items = append(items, reader(p))
		}
		return items, nil
	case "hymns":
		type hymn struct {
			n    int
			kind string
			id   string
		}
		var hymns []hymn
		for k, id := range f.Royal {
			parts := strings.Split(k, ".") // H.kind.N
			if len(parts) != 3 || parts[0] != h {
				continue
			}
			if n, err := strconv.Atoi(parts[2]); err == nil && (parts[1] == "hymn" || parts[1] == "verse" || parts[1] == "canonarch") {
				hymns = append(hymns, hymn{n, parts[1], id})
			}
		}
		sort.Slice(hymns, func(i, j int) bool { return hymns[i].n < hymns[j].n })
		if len(hymns) == 0 {
			return nil, fmt.Errorf("Royal Hours: no stichera of hour %s", h)
		}
		var items []SlotItem
		for _, x := range hymns {
			switch x.kind {
			case "verse":
				items = append(items, litanist(x.id))
			case "canonarch":
				items = append(items, reader(x.id))
			default:
				items = append(items, choir(x.id))
			}
		}
		return items, nil
	case "prokeimenon":
		p, err := need("prokeimenon")
		if err != nil {
			return nil, err
		}
		if f.Tones[p] == 0 {
			return nil, fmt.Errorf("Royal Hours: no tone of the prokeimenon of hour %s", h)
		}
		items := []SlotItem{litanist("lit.let_us_attend"), priest("lit.peace_be_unto_all"),
			reader("lit.response.and_with_thy_spirit"), litanist("lit.wisdom")}
		return append(items, prokeimenonBlock(f.Tones[p], p, u("prokeimenon.verse"))...), nil
	case "readings":
		var items []SlotItem
		for n := 1; n <= 3; n++ {
			r, err := need(fmt.Sprintf("reading.%d", n))
			if err != nil {
				return nil, err
			}
			title := u(fmt.Sprintf("reading.%d.title", n))
			if n < 3 {
				items = append(items, litanist("lit.wisdom"))
				if title != "" {
					items = append(items, reader(title))
				}
				items = append(items, litanist("lit.let_us_attend"), reader(r))
				continue
			}
			// The Gospel.
			items = append(items, priest("sl.lit.wisdom_aright_gospel"), priest("lit.peace_be_unto_all"),
				choir("lit.response.and_with_thy_spirit"))
			if title != "" {
				items = append(items, litanist(title))
			}
			items = append(items, choir("sl.lit.glory_to_thee_o_lord"), litanist("lit.let_us_attend"), litanist(r),
				choir("sl.lit.glory_to_thee_o_lord"))
		}
		return items, nil
	}
	return nil, fmt.Errorf("unknown Royal Hours part %q", sel.Part)
}
