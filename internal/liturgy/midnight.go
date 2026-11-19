package liturgy

import "fmt"

// The midnight office (Часослов, «Полунощница»): daily (Monday..Friday:
// kathisma 17), Saturday (kathisma 9) and Sunday (the Trinity canon of the
// tone, oc.<tone>.sun.midnight). It is the service of the morning of its day.
// The Sunday's canon is read with «Пресвятая Троице, Боже наш, слава Тебе»;
// after the Trisagion the hypakoe of the tone.

func midnightSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	switch sel.From {
	case "midnight_canon":
		return octoechosReadCanon(f, "midnight", "oc.midnight.refrain")
	case "midnight_hypakoe":
		if h := first(f.Octoechos, "matins.hypakoe"); h != "" {
			return []SlotItem{reader(h)}, nil
		}
		return nil, fmt.Errorf("Octoechos tone %d: no hypakoe", f.Day.Tone)
	}
	return nil, fmt.Errorf("unknown midnight office selector %q", sel.From)
}
