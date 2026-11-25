package liturgy

import (
	"fmt"
	"strings"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// specialSunday: a Sunday with readings of its own (Минея): before and after
// the Exaltation, the Nativity («святых отец» before it) and the Theophany.
// The units of the Sunday are ms.<id>.* (orthlib.ImportSpecialDays).
func specialSunday(day calendar.DayContext) string {
	if day.Weekday != time.Sunday {
		return ""
	}
	m, d := day.Menaion.Month, day.Menaion.Day
	switch {
	case m == time.September && d >= 7 && d <= 13:
		return "before_exaltation"
	case m == time.September && d >= 15 && d <= 21:
		return "after_exaltation"
	case m == time.December && d >= 18 && d <= 24:
		return "fathers"
	case m == time.December && d >= 26:
		return "after_nativity"
	case m == time.January && d <= 5:
		return "before_theophany"
	case m == time.January && d >= 7 && d <= 13:
		return "after_theophany"
	}
	return ""
}

// specialSaturday: a Saturday with readings of its own before the Nativity
// (12.18–24) and the Theophany (01.01–05): ms.sat_before_<feast>.* —
// «апсл и еиглие прежде субботе пред…, и ряд под зачало, и потом стаго»
// (the Menaion, 18 December; 5 January).
func specialSaturday(day calendar.DayContext) string {
	if day.Weekday != time.Saturday {
		return ""
	}
	m, d := day.Menaion.Month, day.Menaion.Day
	switch {
	case m == time.December && d >= 18 && d <= 24:
		return "sat_before_nativity"
	case m == time.January && d <= 5:
		return "sat_before_theophany"
	}
	return ""
}

// feastHymns: the troparion and kontakion of the feast whose afterfeast
// (Feast) or forefeast (the Menaion's «предпразднства») the day is in.
func feastHymns(f Facts) (troparion, kontakion string) {
	if f.Feast != nil && !f.Feast.IsFeastDay(f.Day) {
		return f.Feast.Troparion, f.Feast.Kontakion
	}
	for _, id := range mainMenaion(f, "kontakion") {
		if strings.HasPrefix(f.Of[id], "предпразднств") {
			kontakion = id
			break
		}
	}
	// The eve (24 December, 5 January): the day's own hymns are the
	// forefeast's.
	if f.Feast == nil && eveOf(f.Day) != "" {
		if kontakion == "" {
			kontakion = firstMain(f, "kontakion")
		}
		return firstMain(f, "troparion"), kontakion
	}
	// The troparion of the forefeast, or the one «Слава, и ныне» (12.21);
	// some days give none (09.07).
	for _, id := range mainMenaion(f, "troparion") {
		if of := f.Of[id]; strings.HasPrefix(of, "предпразднств") || kontakion != "" && strings.HasPrefix(of, "и ныне") {
			troparion = id
			break
		}
	}
	return troparion, kontakion
}

// saintHymns: the Menaion's troparia or kontakia of the saints of the day,
// not those of the feast.
func saintHymns(f Facts, kind, feast string) []string {
	var out []string
	for _, id := range mainMenaion(f, kind) {
		if id != feast && !f.ofFeast(id) {
			out = append(out, id)
		}
	}
	return out
}

// greatSaint: a saint of the great doxology or higher (Типикон, гл. 3–5:
// the saint's kontakion at the hours).
func greatSaint(f Facts) bool {
	if f.Sign != "" {
		return signRank[f.Sign] >= signRank["doxology"]
	}
	return len(mainMenaion(f, "vespers.great")) > 0 || firstMain(f, "matins.great_doxology") != ""
}

// sundayHours: the hymns of the hours on Sunday (Типикон, гл. 52: «на
// часех, тропарь воскрсн: слава, стаго: и ныне, бгородичен часов. по оче
// наш, кондак воскрсн»). Part "troparion_first": the resurrectional
// troparion; "troparion": the one at «Слава» (none: «Слава, и ныне»);
// "kontakion". Where the Typikon says «пременяюще», the hymns alternate by
// the hour (1st, 3rd, 6th, 9th):
//
//   - a saint without a sign: «Слава» the saint's (two saints alternate),
//     the resurrectional kontakion;
//   - a saint of the doxology, polyeleos, vigil (гл. 3–5): the saint's kontakion at
//     the 1st hour, then the resurrectional and the saint's alternately;
//   - a forefeast or afterfeast (гл. 52, the Menaion: «тропарь воскрсн:
//     слава, праздника… кондак праздника, и воскрсн, пременяюще»), with a
//     great saint «праздника же и стаго пременяюще»;
//   - a feast of the Theotokos on Sunday (the Menaion, 11.21: «слава,
//     тропарь праздника… кондак воскрсн. такожде и на всех часех кондаки
//     пременяюще глаголем»);
//   - the Sunday of the Fathers («слава, оцев… кондак оцев, и
//     предпразднства, пременяюще»), after the Nativity («слава, праздника…
//     стых же тропарь и праздника, пременяюще… кондак праздника, и стых»).
func sundayHours(f Facts, part string, pos int) ([]SlotItem, error) {
	res, resK := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	if res == "" || resK == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion or kontakion", f.Day.Tone)
	}
	if part == "troparion_first" {
		return []SlotItem{reader(res)}, nil
	}
	fT, fK := feastHymns(f)
	sT, sK := saintHymns(f, "troparion", fT), saintHymns(f, "kontakion", fK)
	var troparia, kontakia []string
	switch {
	case f.Feast != nil && f.Feast.IsFeastDay(f.Day):
		// A feast of the Theotokos on Sunday: «Слава» the feast's troparion;
		// the resurrectional kontakion and the feast's, alternately.
		troparia, kontakia = []string{f.Feast.Troparion}, []string{resK, f.Feast.Kontakion}
	case f.Special == "fathers":
		k := f.SpecialUnits["kontakion"]
		if fK != "" {
			k = f.SpecialUnits["kontakion.forefeast"]
		}
		troparia, kontakia = []string{f.SpecialUnits["troparion"]}, []string{k, resK}
		if fK != "" {
			kontakia[1] = fK
		}
	case f.Special == "after_nativity" && fT != "" && fK != "":
		troparia = []string{fT, f.SpecialUnits["troparion"]}
		kontakia = []string{fK, f.SpecialUnits["kontakion"]}
	case fT != "" || fK != "":
		troparia, kontakia = []string{fT}, []string{fK, resK}
		if fT == "" {
			troparia = sT
		} else if greatSaint(f) && len(sT) > 0 {
			troparia = append(troparia, sT[0])
		}
		if greatSaint(f) && len(sK) > 0 {
			kontakia[1] = sK[0]
		}
	default:
		troparia, kontakia = sT, []string{resK}
		if greatSaint(f) && len(sK) > 0 {
			kontakia = []string{sK[0], resK}
		}
	}
	list := troparia
	if part == "kontakion" {
		list = kontakia
	}
	// A forefeast without its kontakion in the Menaion (02.01): the others.
	var given []string
	for _, id := range list {
		if id != "" {
			given = append(given, id)
		}
	}
	list = given
	if len(list) == 0 {
		return nil, nil
	}
	if pos < 1 {
		pos = 1
	}
	return []SlotItem{reader(list[(pos-1)%len(list)])}, nil
}
