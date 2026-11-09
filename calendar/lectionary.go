package calendar

import "time"

// Lection identifies an ordinary (рядовое) reading of the Apostle or Gospel by
// its place in the lectionary table: the reading appointed for Weekday of
// lectionary week Week. The texts and pericope (зачало) numbers are data and
// live in the corpus; this package only computes the key.
//
// Week numbering (the weekdays of week N precede Sunday N):
//
//	-7          Pascha (Sunday only)
//	-6 .. 0     weeks of the Pentecostarion; Sunday 0 is Pentecost
//	 1 .. 33    weeks after Pentecost; Sunday 1 is All Saints,
//	            Sunday 32 is Zacchaeus
//	33 .. 43    Lenten Triodion: Sunday 33 is the Publican and Pharisee,
//	            Sunday 34 the Prodigal Son, 35 Meatfare, 36 Cheesefare,
//	            37.. the Sundays of Great Lent
type Lection struct {
	Week    int
	Weekday time.Weekday
}

// Lections are the ordinary readings of a day at the Liturgy.
type Lections struct {
	Apostle Lection
	Gospel  Lection
	// Omitted is true when no ordinary reading is appointed for this day
	// (Saturdays and Sundays around the Exaltation, the Nativity and the
	// Theophany that have their own readings). Apostle and Gospel are zero then.
	Omitted bool
	// Adjusted is true when the readings differ from the plain week count
	// (Lukan jump, otstupka/prestupka, rearranged Saturday/Sunday readings).
	Adjusted bool
}

// lectionaryWeek returns the week index of d counted from the Sunday base:
// Sunday base+7n is week n; Monday..Saturday after Sunday base+7(n-1) are week n.
func lectionaryWeek(d, base JDate) int {
	if d.Weekday() == time.Sunday {
		return floorDiv(d.Sub(base), 7)
	}
	return ceilDiv(d.Sub(base), 7)
}

// cycleInfo describes the movable cycle that runs from one Pascha to the
// Sunday of the Publican and Pharisee before the next Pascha.
type cycleInfo struct {
	pascha    JDate // Pascha opening the cycle
	pentecost JDate
	// jumpWeek (J) is the number of the Sunday after the Exaltation counted
	// after Pentecost (14..19). The Gospel switches to Luke (week 18) on the
	// following Monday.
	jumpWeek int
	// publicanWeek (W) is the number, counted after Pentecost, of the Sunday of
	// the Publican and Pharisee that closes the cycle (33..38 for real years).
	publicanWeek int
	key          int
}

func newCycleInfo(paschaYear int) cycleInfo {
	p := Pascha(paschaYear)
	pent := p.AddDays(49)
	// First Sunday strictly after the Exaltation (14 September).
	sep15 := MustJDate(paschaYear, time.September, 15)
	sundayAfter := sep15.AddDays(mod(7-int(sep15.Weekday()), 7))
	publican := Pascha(paschaYear + 1).AddDays(-70)
	return cycleInfo{
		pascha:       p,
		pentecost:    pent,
		jumpWeek:     sundayAfter.Sub(pent) / 7,
		publicanWeek: publican.Sub(pent) / 7,
		key:          KeyOfBoundaries(paschaYear),
	}
}

// computeLections returns the ordinary readings of d. cyc is the cycle d
// belongs to; inTriodion reports whether d lies between the Sunday of the
// Publican and Great Saturday, in which case triodionWeek is its week index.
func computeLections(d JDate, cyc cycleInfo, inTriodion bool, triodionWeek int) Lections {
	wd := d.Weekday()
	if inTriodion {
		l := Lection{Week: triodionWeek, Weekday: wd}
		return Lections{Apostle: l, Gospel: l}
	}
	wk := lectionaryWeek(d, cyc.pentecost)
	if wk <= 0 {
		l := Lection{Week: wk, Weekday: wd}
		return Lections{Apostle: l, Gospel: l}
	}
	weekend := wd == time.Saturday || wd == time.Sunday
	ap := apostleWeek(wk, cyc.publicanWeek)
	gos := gospelWeek(wk, cyc.jumpWeek, cyc.publicanWeek, weekend)
	res := Lections{
		Apostle: Lection{Week: ap, Weekday: wd},
		Gospel:  Lection{Week: gos, Weekday: wd},
	}
	if weekend {
		if o, ok := lookupWeekendOverride(cyc.key, cyc.publicanWeek, d); ok {
			if o.ap == 0 {
				return Lections{Omitted: true, Adjusted: true}
			}
			res.Apostle.Week = int(o.ap)
			res.Gospel.Week = int(o.gos)
		}
	}
	res.Adjusted = res.Apostle.Week != wk || res.Gospel.Week != wk
	return res
}

// apostleWeek maps the actual week after Pentecost to the Apostle lectionary
// week. The Apostle keeps its own count, without the Lukan jump, up to week
// 33. Weeks beyond 33 before the Publican are filled by the otstupka: the
// last k = W−33 weeks repeat weeks 34−k..33, so that the week before the
// Publican always reads week 33.
func apostleWeek(wk, publicanWeek int) int {
	if publicanWeek < 33 {
		// Prestupka: unreachable for real years (W is 33..38); kept total so
		// that the last week before the Publican still reads week 33.
		if wk == publicanWeek {
			return 33
		}
		return wk
	}
	if wk <= 33 {
		return wk
	}
	return 33 - (publicanWeek - wk)
}

// gospelWeek maps the actual week after Pentecost to the Gospel lectionary
// week, applying the Lukan jump (Лукина ступка) and the otstupka before the
// Publican. The scheme is the one of A. A. Andreev implemented in Ponomar
// (scripts/Perl/kahuna.pl); it reproduces Andreev's tables for every weekday.
//
// Before the jump (weeks up to J, the Sunday after the Exaltation) Matthew is
// read by the actual week. If Matthew runs out (J = 18, 19), the weekday
// readings of the last pre-jump week are those of week 17 and the extra weeks
// before it borrow weeks 30, 31.
//
// After the jump the sequence is: weeks 18..29 (Luke), then the Matthew weeks
// skipped by the jump (J+1..16; week 17 is not restored), then weeks 30..33.
// If the cycle has more weeks than that, the otstupka appends weeks 34−k..33.
//
// weekend selects the base value for Saturdays and Sundays before the jump;
// the remaining Saturday/Sunday rearrangements come from weekendOverrides.
func gospelWeek(wk, jumpWeek, publicanWeek int, weekend bool) int {
	if wk <= jumpWeek {
		switch {
		case wk < 17:
			return wk
		case weekend, wk == jumpWeek:
			return 17
		default:
			return 31 - (jumpWeek - 1 - wk)
		}
	}
	seq := make([]int, 0, 24)
	for w := 18; w <= 29; w++ {
		seq = append(seq, w)
	}
	for w := jumpWeek + 1; w <= 16; w++ {
		seq = append(seq, w)
	}
	for w := 30; w <= 33; w++ {
		seq = append(seq, w)
	}
	n := publicanWeek - jumpWeek
	if k := n - len(seq); k > 0 {
		for w := 34 - k; w <= 33; w++ {
			seq = append(seq, w)
		}
	} else if k < 0 {
		// Prestupka: unreachable for real years; keep the end of the sequence.
		seq = seq[-k:]
	}
	return seq[wk-jumpWeek-1]
}

// weekendOverride is one Saturday/Sunday entry of Andreev's tables that
// differs from the formula in apostleWeek/gospelWeek.
type weekendOverride struct {
	md      uint16 // month*100 + day (Julian)
	ap, gos int8   // 0, 0: no ordinary reading
}

func lookupWeekendOverride(key, publicanWeek int, d JDate) (weekendOverride, bool) {
	_, m, dd := d.Date()
	md := uint16(int(m)*100 + dd)
	for _, o := range weekendOverrides[[2]uint8{uint8(key), uint8(publicanWeek)}] {
		if o.md == md {
			return o, true
		}
	}
	return weekendOverride{}, false
}
