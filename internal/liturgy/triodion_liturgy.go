package liturgy

import (
	"fmt"
	"time"

	"github.com/n0where3an/orthodoxy-liturgical-service/calendar"
)

// triodionDays: the Sundays and Saturdays of the Lenten Triodion with a
// service of their own (orthlib.ImportTriodionPrep; units tp.<id>.*), by
// their offset from Pascha.
var triodionDays = map[int]string{
	-70: "publican", -63: "prodigal", -57: "sat_meatfare", -56: "meatfare", -50: "sat_cheesefare", -49: "cheesefare",
	-43: "sat_lent1", -42: "lent1", -36: "sat_lent2", -35: "lent2", -29: "sat_lent3", -28: "lent3",
	-22: "sat_lent4", -21: "lent4", -15: "sat_akathist", -14: "lent5",
}

// triodionDay: the day's id in triodionDays, or a weekday of Great Lent
// (lentWeekday: w<week>_<day>, orthlib.ImportTriodionLent), "" otherwise.
func triodionDay(day calendar.DayContext) string {
	if d := triodionDays[day.PaschaOffset]; d != "" {
		return d
	}
	return lentWeekday(day.PaschaOffset)
}

// triodionProfile: what a day's liturgy takes from where (the Triodion's
// rubrics «на литургии…» and the Typikon, гл. 49).
type triodionProfile struct {
	// beatitudes: "tone+6" (the tone's on 6, the Triodion's 6th ode on 4),
	// "tone+3" (its 3rd ode on 4), "3+6" (the 3rd and 6th odes on 8),
	// "tone" (the tone's alone), "plain" (the verses alone).
	beatitudes string
	// troparion, kontakion: the day's (a tp.<id>. suffix, else a unit id);
	// saintKontakion: sung before «Слава» (lent2: St Gregory's).
	troparion, kontakion, saintKontakion string
	// departed: a parental Saturday («прокимен дне и за упокой…»).
	departed bool
	// prokeimenon: "own" (the Triodion's), "tone" (the Sunday's of the tone),
	// "tone+own" (the tone's, then the Triodion's once), "tone+hierarch".
	prokeimenon string
	ownProk     string // the own prokeimenon's unit when the rubric does not parse
	ownTone     int
	// menaion: the saint of the Menaion is sung (cheesefare Sunday).
	menaion bool
	// trisagion: sung instead of the Trisagion; kinonik: the communion verses.
	trisagion string
	kinonik   []string
	// hierarch: «и стому вся служба стительская» (lent2: St Gregory Palamas,
	// the General Menaion's readings, alleluia and communion verse).
	hierarch bool
}

var triodionProfiles = map[string]triodionProfile{
	"publican":       {beatitudes: "tone+6", kontakion: "kontakion", prokeimenon: "tone", kinonik: []string{"sl.kinonik.sun"}},
	"prodigal":       {beatitudes: "tone+6", kontakion: "kontakion", prokeimenon: "tone", kinonik: []string{"sl.kinonik.sun"}},
	"meatfare":       {beatitudes: "tone+6", kontakion: "kontakion", prokeimenon: "own", kinonik: []string{"sl.kinonik.sun"}},
	"cheesefare":     {beatitudes: "tone+6", kontakion: "kontakion", prokeimenon: "own", menaion: true, kinonik: []string{"sl.kinonik.sun"}},
	"sat_meatfare":   {beatitudes: "3+6", troparion: "troparion", kontakion: "kontakion", prokeimenon: "own", ownProk: "sl.prokeimenon.sat.dead", ownTone: 6, kinonik: []string{"sl.kinonik.sat.dead"}},
	"sat_cheesefare": {beatitudes: "3+6", troparion: "troparion", kontakion: "kontakion", prokeimenon: "own", kinonik: []string{"sl.kinonik.sat"}},
	// Great Lent: the Saturdays (St Theodore, the three parental ones, the
	// Akathist) and the Sundays (the liturgy of St Basil the Great).
	"sat_lent1":    {beatitudes: "3+6", troparion: "me.02.17.troparion.1", kontakion: "kontakion", prokeimenon: "own", kinonik: []string{"sl.kinonik.tue"}},
	"sat_lent2":    {beatitudes: "plain", departed: true},
	"sat_lent3":    {beatitudes: "plain", departed: true},
	"sat_lent4":    {beatitudes: "plain", departed: true},
	"sat_akathist": {beatitudes: "3+6", troparion: "troparion", kontakion: "me.03.25.kontakion.1", prokeimenon: "own", kinonik: []string{"sl.kinonik.wed"}},
	"lent1":        {beatitudes: "tone+6", troparion: "troparion", kontakion: "kontakion", prokeimenon: "own", kinonik: []string{"sl.kinonik.sun", "sl.kinonik.sat"}},
	"lent2": {beatitudes: "tone", troparion: "me.11.14p.troparion.1", kontakion: "kontakion", saintKontakion: "me.11.14p.kontakion.1",
		prokeimenon: "tone+hierarch", hierarch: true, kinonik: []string{"sl.kinonik.sun", "sl.kinonik.tue"}},
	"lent3": {beatitudes: "tone+3", troparion: "troparion", kontakion: "kontakion", prokeimenon: "own",
		trisagion: "me.09.14.lit.trisagion", kinonik: []string{"me.09.14.lit.kinonik"}},
	// St John of the Ladder: the Triodion's troparion «Пустынный житель».
	"lent4": {beatitudes: "tone", troparion: "troparion", kontakion: "kontakion", prokeimenon: "tone+own",
		ownProk: "tp.sat_cheesefare.lit.prokeimenon", ownTone: 7, kinonik: []string{"sl.kinonik.sun", "sl.kinonik.tue"}},
	"lent5": {beatitudes: "tone", troparion: "om.nun.troparion.1", kontakion: "kontakion", prokeimenon: "tone", kinonik: []string{"sl.kinonik.sun"}},
}

// The liturgy of the Sundays and Saturdays of the Lenten Triodion (part
// "triodion"), by the Triodion's rubrics and the Typikon (гл. 49):
//
//   - the Sundays of the preparatory weeks: the typical antiphons; the
//     beatitudes of the tone on 6 and the Triodion's 6th ode on 4; the
//     resurrectional troparion and kontakion (and the temple's), «Слава» the
//     Triodion's kontakion, «И ныне» «Предстательство»; the Menaion's saint
//     only on cheesefare Sunday («поем в пяток на повечерии»); the tone's
//     prokeimenon and alleluia or the Sunday's own; the lectionary's readings
//     (the Triodion's); «Хвалите Господа»;
//   - the Sundays of Great Lent (St Basil's liturgy): the resurrectional
//     troparion, the temple's and the day's («Пречистому образу»; St Gregory
//     Palamas; «Спаси, Господи, люди Твоя»; St John of the Ladder; St Mary of
//     Egypt), the resurrectional kontakion, «Слава» the Triodion's, «И ныне»
//     «Предстательство»; the beatitudes «от канона дне песнь ѕ» (1st), «от
//     канона креста песнь г» (3rd), else the tone's; the 1st Sunday's
//     «Благословен еси, Господи Боже отец наших», the 3rd's «Спаси, Господи»
//     and «Кресту Твоему» for the Trisagion, the 4th's prokeimenon of the
//     tone and «Восхвалятся преподобнии»; the 2nd's «вся служба
//     стительская» (the hierarch's readings of the General Menaion), the
//     4th's and 5th's saints' readings after the day's;
//   - the meatfare and the parental Saturdays of Lent: «Апостоли, мученицы»,
//     «Помяни, Господи», «Слава» «Со святыми упокой», «И ныне» «Тебе и
//     стену» (meatfare: «Глубиною мудрости»); the day's prokeimenon and the
//     departed's «Души их во благих водворятся», the day's readings and the
//     departed's, alleluia, «Радуйтеся, праведнии» and «Блажени, яже избрал»;
//   - the Saturdays of the cheesefare fathers, of St Theodore, of the
//     Akathist: their troparion and kontakion, the beatitudes from the 3rd
//     and 6th odes of their canon, their prokeimenon, alleluia and communion
//     verse, the day's readings and theirs.
func triodionLiturgySlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	pr, ok := triodionProfiles[f.TriodionDay]
	if !ok {
		return nil, fmt.Errorf("no Triodion liturgy for %q", f.TriodionDay)
	}
	tu := func(k string) string { return f.Triodion[k] }
	sunday := f.Day.Weekday == time.Sunday
	switch sel.From {
	case "lit_antiphon":
		if sel.Section != "3" {
			return litAntiphon(f, sel.Section, "sunday")
		}
		return triodionBeatitudes(f, pr)
	case "lit_entrance":
		if sunday {
			return liturgySlot(SlotSelector{From: "lit_entrance", Part: "sunday"}, f)
		}
		return []SlotItem{choir("ch.lit.come_let_us_worship.weekday")}, nil
	case "lit_troparia":
		return triodionTroparia(f, pr)
	case "lit_trisagion":
		if pr.trisagion != "" {
			return []SlotItem{{Ref: pr.trisagion, Repeat: 3, Performer: Choir}, choir("common.glory_now"), choir(pr.trisagion)}, nil
		}
		return liturgySlot(SlotSelector{From: "lit_trisagion"}, f)
	case "lit_prokeimena":
		return triodionProkeimena(f, pr)
	case "lit_alleluia":
		return triodionAlleluia(f, pr)
	case "lit_apostles", "lit_gospels":
		which := map[string]string{"lit_apostles": "apostle", "lit_gospels": "gospel"}[sel.From]
		return triodionReadings(f, pr, which)
	case "lit_kinonik":
		kin := pr.kinonik
		if pr.departed {
			kin = []string{"sl.kinonik.sat", "sl.kinonik.sat.dead"}
		}
		if len(kin) == 0 {
			if k := tu("lit.kinonik"); k != "" {
				kin = []string{k}
			}
		}
		var items []SlotItem
		for _, k := range kin {
			items = append(items, choir(k))
		}
		return append(items, choir("common.alleluia.thrice")), nil
	}
	return liturgySlot(SlotSelector{From: sel.From, Section: sel.Section}, f)
}

// unit: a tp.<id>. suffix of the day, or a unit id as it is.
func (pr triodionProfile) unit(f Facts, k string) string {
	if k == "" {
		return ""
	}
	if id := f.Triodion[k]; id != "" {
		return id
	}
	if isUnitID(k) {
		return k
	}
	return ""
}

// isUnitID: a full unit id (with a book prefix), not a tp.<id>. suffix.
func isUnitID(k string) bool {
	for _, p := range []string{"me.", "om.", "tp.", "ch.", "sl.", "oc."} {
		if len(k) > len(p) && k[:len(p)] == p {
			return true
		}
	}
	return false
}

// triodionBeatitudes: see triodionProfile.beatitudes; «Слава» and «И ныне»
// the ode's own glory and theotokion (else the tone's).
func triodionBeatitudes(f Facts, pr triodionProfile) ([]SlotItem, error) {
	ode := func(n int) []string {
		var out []string
		for k := 1; ; k++ {
			id := f.Triodion[fmt.Sprintf("canon.ode.%d.troparion.%d", n, k)]
			if id == "" {
				return out
			}
			out = append(out, id)
		}
	}
	verse := func(i int) SlotItem { return choir(fmt.Sprintf("ch.typika.beatitudes.%d", i)) }
	if pr.beatitudes == "plain" {
		var items []SlotItem
		for i := 0; i <= 10; i++ {
			items = append(items, verse(i))
		}
		return append(items, choir("common.glory_now")), nil
	}
	if pr.beatitudes == "tone" {
		return sundayBeatitudes(f)
	}
	n := 6
	if pr.beatitudes == "tone+3" {
		n = 3
	}
	glory, now := f.Triodion[fmt.Sprintf("canon.ode.%d.glory", n)], f.Triodion[fmt.Sprintf("canon.ode.%d.theotokion", n)]
	var troparia []string
	if pr.beatitudes == "3+6" {
		glory, now = f.Triodion["canon.ode.6.glory"], f.Triodion["canon.ode.6.theotokion"]
		all := append(ode(3), ode(6)...)
		if len(all) == 0 {
			return nil, fmt.Errorf("Triodion %s: no 3rd and 6th odes for the beatitudes", f.TriodionDay)
		}
		for i := 0; i < 8; i++ {
			troparia = append(troparia, all[i%len(all)])
		}
	} else {
		oct := f.Octoechos["lit.beatitudes"]
		if len(oct) < 4 {
			return nil, fmt.Errorf("Octoechos tone %d: no Sunday beatitudes", f.Day.Tone)
		}
		own := ode(n)
		if len(own) == 0 {
			return nil, fmt.Errorf("Triodion %s: no ode %d for the beatitudes", f.TriodionDay, n)
		}
		// The tone's on 6, the Triodion's on 4 (its troparia repeated to fill).
		troparia = append(troparia, oct[:4]...)
		for i := 0; i < 4; i++ {
			troparia = append(troparia, own[i%len(own)])
		}
		if glory == "" {
			glory = first(f.Octoechos, "lit.beatitudes.glory")
		}
		if now == "" {
			now = first(f.Octoechos, "lit.beatitudes.now")
		}
	}
	var items []SlotItem
	for i := 0; i < 3; i++ { // on 8
		items = append(items, verse(i))
	}
	for i, t := range troparia {
		items = append(items, verse(3+i), choir(t))
	}
	items = append(items, choir("common.glory"))
	if glory != "" {
		items = append(items, choir(glory))
	}
	items = append(items, choir("common.and_now"))
	if now != "" {
		items = append(items, choir(now))
	}
	return items, nil
}

// triodionTroparia: the hymns after the entrance (see triodionLiturgySlot).
func triodionTroparia(f Facts, pr triodionProfile) ([]SlotItem, error) {
	if pr.departed {
		th := "tp.sat_meatfare.theotokion" // «Тебе и стену»
		return []SlotItem{choir("ch.troparion.day.sat"), choir("ch.troparion.day.sat.2"), choir("common.glory"),
			choir("ch.kontakion.with_the_saints"), choir("common.and_now"), choir(th)}, nil
	}
	k := pr.unit(f, pr.kontakion)
	if k == "" {
		return nil, fmt.Errorf("Triodion %s: no kontakion", f.TriodionDay)
	}
	t := pr.unit(f, pr.troparion)
	if pr.troparion != "" && t == "" {
		return nil, fmt.Errorf("Triodion %s: no troparion", f.TriodionDay)
	}
	tm := litTemple(f, nil)
	now := "ch.kontakion.protection"
	if tm != nil && tm.Kind == TempleTheotokos {
		now = tm.Kontakion
	}
	if f.Day.Weekday != time.Sunday {
		switch f.TriodionDay {
		case "sat_meatfare": // «Глубиною мудрости… Со святыми упокой… Тебе и стену»
			return []SlotItem{choir(t), choir("common.glory"), choir(k), choir("common.and_now"), choir(f.Triodion["theotokion"])}, nil
		case "sat_akathist": // «Повеленное тайно», «Взбранной Воеводе»
			return []SlotItem{choir(t), choir("common.glory_now"), choir(k)}, nil
		}
		items := []SlotItem{choir(t)}
		if tm != nil {
			items = append(items, choir(tm.Troparion))
			if tm.Kind != TempleTheotokos {
				items = append(items, choir(tm.Kontakion))
			}
		}
		return append(items, choir("common.glory"), choir(k), choir("common.and_now"), choir(now)), nil
	}
	res, resK := first(f.Octoechos, "troparion"), first(f.Octoechos, "kontakion")
	if res == "" || resK == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion or kontakion", f.Day.Tone)
	}
	var saintT, saintK []string
	if pr.menaion { // the Menaion is sung with the Triodion
		saintT, saintK = mainMenaion(f, "troparion"), mainMenaion(f, "kontakion")
	}
	items := []SlotItem{choir(res)}
	if tm != nil {
		items = append(items, choir(tm.Troparion))
	}
	if t != "" {
		items = append(items, choir(t))
	}
	if len(saintT) > 0 {
		items = append(items, choir(saintT[0]))
	}
	items = append(items, choir(resK))
	if tm != nil && tm.Kind != TempleTheotokos {
		items = append(items, choir(tm.Kontakion))
	}
	if pr.saintKontakion != "" {
		items = append(items, choir(pr.saintKontakion))
	}
	if len(saintK) > 0 {
		items = append(items, choir(saintK[0]))
	}
	return append(items, choir("common.glory"), choir(k), choir("common.and_now"), choir(now)), nil
}

// toneProkeimenon: the Sunday's prokeimenon of the tone.
func toneProkeimenon(f Facts) ([]SlotItem, error) {
	p, v := first(f.Octoechos, "lit.prokeimenon"), first(f.Octoechos, "lit.prokeimenon.verse")
	if p == "" {
		return nil, fmt.Errorf("Octoechos tone %d: no Sunday prokeimenon", f.Day.Tone)
	}
	return prokeimenonBlock(f.Day.Tone, p, v), nil
}

func triodionProkeimena(f Facts, pr triodionProfile) ([]SlotItem, error) {
	own, verse, tone := f.Triodion["lit.prokeimenon"], f.Triodion["lit.prokeimenon.verse"], 0
	if own != "" {
		tone = f.Tones[own]
	}
	if pr.ownProk != "" && (own == "" || tone == 0) {
		own, tone = pr.ownProk, pr.ownTone
	}
	if pr.departed {
		d, err := dayOf(f)
		if err != nil {
			return nil, err
		}
		return append(prokeimenonBlock(d.prokTone, "sl.prokeimenon.sat", "sl.prokeimenon.sat.verse"),
			prokeimenonOnce(6, "sl.prokeimenon.sat.dead")...), nil
	}
	switch pr.prokeimenon {
	case "own":
		if own == "" || tone == 0 {
			return nil, fmt.Errorf("Triodion %s: no prokeimenon", f.TriodionDay)
		}
		return prokeimenonBlock(tone, own, verse), nil
	case "tone+own":
		items, err := toneProkeimenon(f)
		if err != nil || own == "" || tone == 0 {
			return items, err
		}
		return append(items, prokeimenonOnce(tone, own)...), nil
	case "tone+hierarch":
		items, err := toneProkeimenon(f)
		if err != nil {
			return nil, err
		}
		if h := "om.hierarch.lit.prokeimenon"; f.Tones[h] > 0 {
			items = append(items, prokeimenonOnce(f.Tones[h], h)...)
		} else {
			items = append(items, prokeimenonOnce(1, h)...)
		}
		return items, nil
	}
	return toneProkeimenon(f)
}

func triodionAlleluia(f Facts, pr triodionProfile) ([]SlotItem, error) {
	block := func(tone int, v1, v2 string) []SlotItem {
		items := []SlotItem{reader(fmt.Sprintf("lit.alleluia.tone.%d", tone)), choir("common.alleluia.thrice"),
			reader(v1), choir("common.alleluia.thrice")}
		if v2 != "" {
			items = append(items, reader(v2), choir("common.alleluia.thrice"))
		}
		return items
	}
	if pr.departed { // the day's, then the departed's verse
		d, err := dayOf(f)
		if err != nil {
			return nil, err
		}
		return append(block(d.alTone, "sl.alleluia.sat.verse.1", "sl.alleluia.sat.verse.2"),
			reader("sl.alleluia.sat.dead.verse"), choir("common.alleluia.thrice")), nil
	}
	if v1 := f.Triodion["lit.alleluia.verse.1"]; v1 != "" && f.Tones[v1] > 0 {
		return block(f.Tones[v1], v1, f.Triodion["lit.alleluia.verse.2"]), nil
	}
	if f.TriodionDay == "sat_meatfare" { // «аллилуиа, глас ѕ: блжени, яже избрал»
		return block(6, "sl.alleluia.sat.dead.verse", ""), nil
	}
	vs := f.Octoechos["lit.alleluia.verse"]
	if len(vs) == 0 {
		return nil, fmt.Errorf("Octoechos tone %d: no Sunday alleluia", f.Day.Tone)
	}
	v2 := ""
	if len(vs) > 1 {
		v2 = vs[1]
	}
	items := block(f.Day.Tone, vs[0], v2)
	if pr.hierarch { // the hierarch's verse
		items = append(items, reader("om.hierarch.lit.alleluia.verse.1"), choir("common.alleluia.thrice"))
	}
	if f.TriodionDay == "lent4" { // «аллилуиа, гласа, и прпдбнаго»
		items = append(items, reader("tp.sat_cheesefare.lit.alleluia.verse.1"), choir("common.alleluia.thrice"))
	}
	return items, nil
}

// triodionReadings: the day's (the lectionary's: the Triodion's), then the
// saint's, the departed's, the Theotokos's (tp.<id>.lit.<which>.2, copies of
// the lectionary's by the rubric's pericope; the departed's by default, the
// hierarch's of the General Menaion on the 2nd Sunday).
func triodionReadings(f Facts, pr triodionProfile, which string) ([]SlotItem, error) {
	l, err := lection(f, which)
	if err != nil {
		return nil, err
	}
	key := map[string]string{"apostle": "ap", "gospel": "gos"}[which]
	list := []string{fmt.Sprintf("lect.%s.%d.%d", key, l.Week, int(l.Weekday))}
	for n := 2; ; n++ {
		id := f.Triodion[fmt.Sprintf("lit.%s.%d", which, n)]
		if id == "" {
			break
		}
		list = append(list, id)
	}
	if len(list) == 1 {
		switch {
		case pr.departed || f.TriodionDay == "sat_meatfare":
			list = append(list, map[string]string{"apostle": departedApostle, "gospel": departedGospel}[which])
		case pr.hierarch:
			list = append(list, "om.hierarch.lit."+which)
		}
	}
	var items []SlotItem
	if which == "apostle" {
		items = append(items, litanist("lit.wisdom"), reader(list[0]+".announce"), litanist("lit.let_us_attend"), reader(list[0]))
		for _, r := range list[1:] {
			items = append(items, reader(r))
		}
		return items, nil
	}
	items = append(items, litanist(list[0]+".announce"), choir("sl.lit.glory_to_thee_o_lord"), priest("lit.let_us_attend"), litanist(list[0]))
	for _, r := range list[1:] {
		items = append(items, litanist(r))
	}
	return items, nil
}

// triodionHours: the hymns of the hours on the Triodion's Sundays and
// Saturdays (Типикон, гл. 49: «на часех, тропарь воскрсн. по тристом:
// кондак триоди»; the meatfare Saturday: «на часех глаголем тропарь и кондак
// за упокой»). Sunday: the resurrectional troparion, «Слава» the day's
// (Great Lent: «Пречистому образу», St Gregory, the Cross's, St John's, St
// Mary's; cheesefare: the Menaion's saint; else «Слава, и ныне»), the
// Triodion's kontakion. Saturday: the day's troparion at «Слава» and its
// kontakion; a parental Saturday «Апостоли, мученицы», «Слава» «Помяни,
// Господи», «Со святыми упокой».
func triodionHours(f Facts, part string) ([]SlotItem, error) {
	pr, ok := triodionProfiles[f.TriodionDay]
	if !ok {
		return nil, fmt.Errorf("no Triodion hours for %q", f.TriodionDay)
	}
	sunday := f.Day.Weekday == time.Sunday
	switch part {
	case "troparion_first":
		switch {
		case sunday:
			res := first(f.Octoechos, "troparion")
			if res == "" {
				return nil, fmt.Errorf("Octoechos tone %d: no resurrectional troparion", f.Day.Tone)
			}
			return []SlotItem{reader(res)}, nil
		case pr.departed:
			return []SlotItem{reader("ch.troparion.day.sat")}, nil
		}
		return nil, nil
	case "troparion":
		switch {
		case pr.departed:
			return []SlotItem{reader("ch.troparion.day.sat.2")}, nil
		case pr.menaion:
			if t := mainMenaion(f, "troparion"); len(t) > 0 {
				return []SlotItem{reader(t[0])}, nil
			}
			return nil, nil
		}
		if t := pr.unit(f, pr.troparion); t != "" {
			return []SlotItem{reader(t)}, nil
		}
		return nil, nil
	case "kontakion":
		if pr.departed {
			return []SlotItem{reader("ch.kontakion.with_the_saints")}, nil
		}
		k := pr.unit(f, pr.kontakion)
		if k == "" {
			return nil, fmt.Errorf("Triodion %s: no kontakion", f.TriodionDay)
		}
		return []SlotItem{reader(k)}, nil
	}
	return nil, fmt.Errorf("unknown Triodion hours part %q", part)
}

// triodionVespersSlot: the great vespers of a Sunday of Great Lent
// (Saturday evening, the vigil; the Triodion: «гдси воззвах… поем стихиры
// воскрсны г, и анатолиевы г, и триоди подобны д… слава, триоди… и ныне,
// бгородичен а-й гласа… на стиховне осмогласника. слава… и ныне,
// бгородичен в тойже глас… тропарь: бцде дво… дважды, и праздника
// единожды»): «Господи, воззвах» on 10 — the Octoechos's (the
// resurrectional 3, Anatolius's to fill) and the Triodion's (3 or 4);
// «Слава» the Triodion's, «И ныне» the dogmatikon of the tone; the
// aposticha of the Octoechos, «Слава» the Triodion's, «И ныне» the theotokion
// in its tone (the Cross: «Слава, и ныне» the Triodion's); «Богородице Дево»
// twice and the day's troparion (the 2nd Sunday: thrice). The Menaion's
// saint is sung at compline.
func triodionVespersSlot(sel SlotSelector, f Facts) ([]SlotItem, error) {
	tu := func(k string) string { return f.Triodion[k] }
	switch sel.From {
	case "tv_lic":
		oct := f.Octoechos["gvespers.lic.sticheron"]
		var own []string
		for n := 1; tu(fmt.Sprintf("gvespers.lic.sticheron.%d", n)) != ""; n++ {
			own = append(own, tu(fmt.Sprintf("gvespers.lic.sticheron.%d", n)))
		}
		if len(own) == 0 {
			return nil, fmt.Errorf("Triodion %s: no stichera at «Господи, воззвах»", f.TriodionDay)
		}
		mine := 4 // the Triodion's on 4 (fewer: repeated from the first)
		if len(own) == 3 && len(oct) >= 7 && f.TriodionDay != "lent2" {
			mine = 3 // «воскрсны г, и анатолиевы д, и минеи г»; the 2nd Sunday: «стаго на д»
		}
		if len(oct) < 10-mine {
			return nil, fmt.Errorf("Octoechos tone %d: %d Sunday stichera at «Господи, воззвах»", f.Day.Tone, len(oct))
		}
		refs := append([]string{}, oct[:10-mine]...)
		for i := 0; i < mine; i++ {
			refs = append(refs, own[i%len(own)])
		}
		return licWithVerses(f, refs), nil
	case "tv_glory_now":
		dog := first(f.Octoechos, "gvespers.lic.dogmatikon")
		g := tu("gvespers.lic.glory")
		if dog == "" || g == "" {
			return nil, fmt.Errorf("Triodion %s: no «Слава» or dogmatikon at «Господи, воззвах»", f.TriodionDay)
		}
		return []SlotItem{reader("common.glory"), {Ref: g}, reader("common.and_now"), {Ref: dog}}, nil
	case "tv_aposticha":
		items, err := svAposticha(Facts{Day: f.Day, Octoechos: f.Octoechos, Tones: f.Tones}) // the Octoechos's
		if err != nil {
			return nil, err
		}
		// Drop the Octoechos's «Слава, и ныне» and its theotokion.
		items = items[:len(items)-2]
		if gn := tu("gvespers.aposticha.glory_now"); gn != "" {
			return append(items, reader("common.glory_now"), SlotItem{Ref: gn}), nil
		}
		g := tu("gvespers.aposticha.glory")
		if g == "" {
			return nil, fmt.Errorf("Triodion %s: no «Слава» of the aposticha", f.TriodionDay)
		}
		th := bogorTheotokion(f.Tones[g], "aposticha")
		if th == "" {
			return nil, fmt.Errorf("Triodion %s: no theotokion in tone %d", f.TriodionDay, f.Tones[g])
		}
		return append(items, reader("common.glory"), SlotItem{Ref: g}, reader("common.and_now"), SlotItem{Ref: th}), nil
	case "tv_troparia":
		pr := triodionProfiles[f.TriodionDay]
		t := pr.unit(f, pr.troparion)
		if t == "" || f.TriodionDay == "lent2" { // «тропарь, бцде дво: трижды»
			return []SlotItem{{Ref: "ch.theotokos_rejoice", Repeat: 3}}, nil
		}
		return []SlotItem{{Ref: "ch.theotokos_rejoice", Repeat: 2}, {Ref: t}}, nil
	}
	return nil, fmt.Errorf("unknown Triodion vespers selector %q", sel.From)
}
