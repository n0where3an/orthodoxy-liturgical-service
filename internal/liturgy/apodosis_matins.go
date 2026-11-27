package liturgy

import (
	"fmt"
)

// Selectors of weekday matins on the apodosis of a fixed twelve feast: «вся
// служба праздника» without the polyeleos and the Gospel, as the Menaion of
// each apodosis gives it (09.12, 11.25, 12.31, 01.14, 08.13, 08.23: «на бг
// гдсь тропарь праздника трижды. по кафисмах седальны праздника. псалом н.
// канон праздника… катавасиа… по г-й песни седален (ипакои)… по з-й песни
// кондак и икос праздника. на ф-й песни честнейшую не поем, но поем припевы
// праздника… светилен праздника… на хвалитех стихиры праздника на д… слава,
// и ныне, праздника. славословие великое. по тристом тропарь праздника»):
//
//   - «Бог Господь»: the feast's troparion thrice;
//   - the feast day's sessional hymns after the kathismata (after a third,
//     the one after the polyeleos);
//   - Psalm 50 read, without the sticheron;
//   - the feast day's canons, the 9th ode with the refrains of the feast;
//     the katavasia after each ode the Menaion's (apodosisKatavasia);
//   - after the 3rd ode the feast's sessional hymn (ipakoi), after the 6th
//     its kontakion and ikos;
//   - the exapostilarion twice («слава, и ныне, тойже»), the praises of the
//     feast with its «Слава, и ныне», the great doxology, the troparion.
//
// The saint of the day is not sung (the Menaion sets him to compline).
//
// Where the Menaion of the apodosis has its own matins (11.25: «тропарь
// праздника дважды, слава, стых… канон праздника со ирмосом на з, и стых оба
// канона на и. катавасиа: хрстос раждается… на ф-й песни честнейшую не поем,
// но поем припев праздника… на хвалитех стихиры на з… славословие великое»)
// the day's hymns are sung as in the afterfeast, in the same order
// (matins.apodosis.menaion, ap_canon with part menaion).

func apodosisMatinsSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	if sel.From == "ap_magnificat" {
		// «Честнейшую» — not where the 9th ode is sung with the refrains.
		if len(f.Refrains) > 0 {
			return nil, nil
		}
		return []SlotItem{choir("ch.matins.magnificat.1")}, nil
	}
	if f.Feast == nil || f.FeastMenaion == nil {
		return nil, fmt.Errorf("no feast service for the apodosis on %v", f.Day.Date)
	}
	fv := feastView(f)
	switch sel.From {
	case "ap_troparia":
		return []SlotItem{{Ref: f.Feast.Troparion, Repeat: 3, Performer: Choir}}, nil
	case "ap_sedalny":
		items, err := gmSedalny(fv, sel.Section)
		if err == nil && len(items) == 0 && sel.Section == "3" {
			return gmSedalny(fv, "p")
		}
		return items, err
	case "ap_ps50":
		return []SlotItem{reader("ch.psalm.50")}, nil
	case "ap_canon":
		if sel.Part == "menaion" {
			// The apodosis's own canons (11.25: the feast's and the saints'),
			// the katavasia by the season.
			return festalCanonKatavasia(f, sel.Section, func(o int) string { return smKatavasia(f, "afterfeast", o) })
		}
		return festalCanonKatavasia(fv, sel.Section, func(o int) string {
			c := apodosisKatavasia[f.Feast.ID]
			if c == "" {
				return firstMain(fv, fmt.Sprintf("matins.canon.1.ode.%d.irmos", o))
			}
			return fmt.Sprintf("%s.ode.%d.irmos", c, o)
		})
	case "ap_sedalen":
		switch sel.Part {
		case "menaion":
			// The day's own: with several kontakia in the Menaion, the last
			// but one (with the first ikos, when there are two) and the
			// sessional hymn after the 3rd ode (01.07: the feast's; 11.25: the
			// saints'), the last after the 6th (ap_kontakion).
			if ks := mainMenaion(f, "kontakion"); len(ks) > 1 {
				items := []SlotItem{reader(ks[len(ks)-2])}
				if ik := mainMenaion(f, "matins.ikos"); len(ik) > 1 {
					items = append(items, reader(ik[0]))
				}
				if len(mainMenaion(f, "matins.sedalen")) > 0 {
					sed, err := matinsMenaionSedalen(f)
					if err != nil {
						return nil, err
					}
					items = append(items, sed...)
				}
				return items, nil
			}
			return afSedalen(f)
		case "feast_kontakion":
			// 12.26: «по г-й песни, кондак праздника… и икос, и ипакои».
			items, err := afKontakion(f)
			if err != nil {
				return nil, err
			}
			ip, err := greatMatinsSlot(SlotSelector{From: "gm_sedalen"}, fv)
			return append(items, ip...), err
		}
		return greatMatinsSlot(SlotSelector{From: "gm_sedalen"}, fv)
	case "ap_kontakion":
		// After the 6th ode: the day's last kontakion with the last ikos
		// (01.07: the Forerunner's; 11.25: the feast's), or its only one
		// (part day, 12.26: «иже прежде денницы»), else the feast's.
		ks, ik := mainMenaion(f, "kontakion"), mainMenaion(f, "matins.ikos")
		if len(ks) > 1 || sel.Part == "day" && len(ks) == 1 {
			items := []SlotItem{reader(ks[len(ks)-1])}
			if len(ik) > 0 {
				items = append(items, reader(ik[len(ik)-1]))
			}
			return items, nil
		}
		return afKontakion(f)
	case "ap_exapostilaria":
		ex := mainMenaion(fv, "matins.exapostilarion")
		if len(ex) == 1 && sel.Part == "thrice" {
			return []SlotItem{{Ref: ex[0], Repeat: 3}}, nil
		}
		if len(ex) == 1 {
			return []SlotItem{{Ref: ex[0]}, reader("common.glory_now"), {Ref: ex[0]}}, nil
		}
		return gmExapostilaria(fv, true)
	case "ap_praises":
		return gmPraises(fv)
	case "ap_end_troparia":
		return []SlotItem{choir(f.Feast.Troparion)}, nil
	}
	return nil, fmt.Errorf("unknown apodosis matins selector %q", sel.From)
}

// apodosisKatavasia: the katavasia of the apodosis by the Menaion, when not
// the irmoi of the feast's first canon («катавасиа, ирмос перваго канона»,
// 12.31, 01.14; «тойже ирмос», 02.09; «преукрашенная», 08.23; the canon
// «на ви… последи ирмос тойже», 09.21): «катавасиа: крст начертав» on 09.12
// and 08.13.
var apodosisKatavasia = map[string]string{
	"nativity_of_theotokos": "me.09.14.matins.canon.1",
	"transfiguration":       "me.09.14.matins.canon.1",
}

// feastView: the facts with the feast day's Menaion for the day's.
func feastView(f Facts) Facts {
	v := f
	v.Menaion = f.FeastMenaion
	v.Day.Menaion = f.Feast.from
	return v
}
