package calendar

import (
	"testing"
	"time"
)

const (
	minYear = 1900
	maxYear = 2100
)

// civilViaTime converts a Julian date to the civil date independently of the
// package's own Gregorian formulas, using Go's proleptic Gregorian time.
func civilViaTime(d JDate) (int, time.Month, int) {
	t := time.Unix(int64(d.JDN()-unixEpochJDN)*86400, 0).UTC()
	return t.Date()
}

func TestJDNKnownValues(t *testing.T) {
	tests := []struct {
		name string
		got  func() (JDate, error)
		jdn  int
	}{
		{"last Julian day in Rome", func() (JDate, error) { return NewJDate(1582, time.October, 4) }, 2299160},
		{"first Gregorian day", func() (JDate, error) { return FromCivil(1582, time.October, 15) }, 2299161},
		{"J2000 epoch", func() (JDate, error) { return FromCivil(2000, time.January, 1) }, 2451545},
		{"Unix epoch", func() (JDate, error) { return FromCivil(1970, time.January, 1) }, unixEpochJDN},
	}
	for _, tt := range tests {
		d, err := tt.got()
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if d.JDN() != tt.jdn {
			t.Errorf("%s: JDN %d, want %d", tt.name, d.JDN(), tt.jdn)
		}
	}
}

func TestJDateRoundTrip(t *testing.T) {
	start := MustJDate(minYear-1, time.January, 1)
	end := MustJDate(maxYear+1, time.December, 31)
	prevY, prevM, prevD := start.AddDays(-1).Date()
	for d := start; !d.After(end); d = d.AddDays(1) {
		y, m, dd := d.Date()
		back, err := NewJDate(y, m, dd)
		if err != nil || back != d {
			t.Fatalf("Julian round trip failed for %v: %v", d, err)
		}
		cy, cm, cd := d.Civil()
		if wy, wm, wd := civilViaTime(d); cy != wy || cm != wm || cd != wd {
			t.Fatalf("%v: civil %d-%d-%d, time package says %d-%d-%d", d, cy, cm, cd, wy, wm, wd)
		}
		if c, err := FromCivil(cy, cm, cd); err != nil || c != d {
			t.Fatalf("civil round trip failed for %v: %v", d, err)
		}
		if d.Weekday() != d.Time(nil).Weekday() {
			t.Fatalf("%v: weekday %v, time package %v", d, d.Weekday(), d.Time(nil).Weekday())
		}
		// Consecutive JDNs are consecutive Julian dates.
		if dd != 1 && (y != prevY || m != prevM || dd != prevD+1) {
			t.Fatalf("%v does not follow %d-%d-%d", d, prevY, prevM, prevD)
		}
		prevY, prevM, prevD = y, m, dd
	}
}

func TestLeapYears(t *testing.T) {
	tests := []struct {
		year         int
		julianLeap   bool
		gregorianFeb int
	}{
		{1900, true, 28}, // Julian leap, not Gregorian
		{1999, false, 28},
		{2000, true, 29},
		{2024, true, 29},
		{2026, false, 28},
		{2100, true, 28}, // Julian leap, not Gregorian
	}
	for _, tt := range tests {
		if IsLeap(tt.year) != tt.julianLeap {
			t.Errorf("IsLeap(%d) = %v", tt.year, !tt.julianLeap)
		}
		if got := gregorianDaysIn(tt.year, time.February); got != tt.gregorianFeb {
			t.Errorf("Gregorian February %d has %d days, want %d", tt.year, got, tt.gregorianFeb)
		}
		_, err := NewJDate(tt.year, time.February, 29)
		if (err == nil) != tt.julianLeap {
			t.Errorf("NewJDate(%d-02-29) error = %v", tt.year, err)
		}
	}
	if _, err := FromCivil(1900, time.February, 29); err == nil {
		t.Error("FromCivil(1900-02-29) accepted")
	}
}

func TestStyleDifference(t *testing.T) {
	tests := []struct {
		julian JDate
		civil  string
		diff   int
	}{
		{MustJDate(1900, time.February, 28), "1900-03-12", 12},
		{MustJDate(1900, time.February, 29), "1900-03-13", 12},
		{MustJDate(1900, time.March, 1), "1900-03-14", 13},
		{MustJDate(2026, time.March, 30), "2026-04-12", 13},
		{MustJDate(2100, time.February, 28), "2100-03-13", 13},
		{MustJDate(2100, time.February, 29), "2100-03-14", 13},
		{MustJDate(2100, time.March, 1), "2100-03-15", 14},
		{MustJDate(2100, time.December, 25), "2101-01-08", 14},
	}
	for _, tt := range tests {
		if got := tt.julian.CivilString(); got != tt.civil {
			t.Errorf("%v: civil %s, want %s", tt.julian, got, tt.civil)
		}
		if got := tt.julian.StyleDifference(); got != tt.diff {
			t.Errorf("%v: style difference %d, want %d", tt.julian, got, tt.diff)
		}
	}
}

// One year for each of the 35 keys of boundaries. Julian dates and letters
// are from the Ponomar paschalion baseline (scripts/Perl/data/paschalion_baseline.tsv).
var keyTable = []struct {
	key    int
	letter string
	year   int
	month  time.Month
	day    int
}{
	{1, "А", 2010, time.March, 22},
	{2, "Б", 1942, time.March, 23},
	{3, "В", 1980, time.March, 24},
	{4, "Г", 1991, time.March, 25},
	{5, "Д", 2007, time.March, 26},
	{6, "Е", 1950, time.March, 27},
	{7, "Ж", 1966, time.March, 28},
	{8, "Ѕ", 1999, time.March, 29},
	{9, "З", 2015, time.March, 30},
	{10, "И", 1947, time.March, 31},
	{11, "І", 1963, time.April, 1},
	{12, "К", 1990, time.April, 2},
	{13, "Л", 1944, time.April, 3},
	{14, "М", 1955, time.April, 4},
	{15, "Н", 1971, time.April, 5},
	{16, "О", 1987, time.April, 6},
	{17, "П", 1941, time.April, 7},
	{18, "Р", 1946, time.April, 8},
	{19, "С", 1979, time.April, 9},
	{20, "Т", 1995, time.April, 10},
	{21, "Ꙋ", 1949, time.April, 11},
	{22, "Ф", 1943, time.April, 12},
	{23, "Х", 1970, time.April, 13},
	{24, "Ѿ", 1997, time.April, 14},
	{25, "Ц", 2019, time.April, 15},
	{26, "Ч", 1951, time.April, 16},
	{27, "Ш", 1967, time.April, 17},
	{28, "Щ", 1994, time.April, 18},
	{29, "Ъ", 1948, time.April, 19},
	{30, "Ы", 1959, time.April, 20},
	{31, "Ь", 1975, time.April, 21},
	{32, "Ѣ", 2002, time.April, 22},
	{33, "Ю", 1945, time.April, 23},
	{34, "Ѫ", 2051, time.April, 24},
	{35, "Ѧ", 1983, time.April, 25},
}

func TestPascha35Keys(t *testing.T) {
	for _, tt := range keyTable {
		t.Run(tt.letter, func(t *testing.T) {
			p := Pascha(tt.year)
			want := MustJDate(tt.year, tt.month, tt.day)
			if p != want {
				t.Fatalf("Pascha(%d) = %v, want %v", tt.year, p, want)
			}
			if p.Weekday() != time.Sunday {
				t.Errorf("Pascha(%d) is a %v", tt.year, p.Weekday())
			}
			// All sample years lie in 1900-03-01..2100-02-28 O.S., where the
			// style difference is 13 days.
			wy, wm, wd := time.Date(tt.year, tt.month, tt.day+13, 0, 0, 0, 0, time.UTC).Date()
			if cy, cm, cd := p.Civil(); cy != wy || cm != wm || cd != wd {
				t.Errorf("civil Pascha %d: %d-%02d-%02d, want %d-%02d-%02d", tt.year, cy, cm, cd, wy, wm, wd)
			}
			if k := KeyOfBoundaries(tt.year); k != tt.key {
				t.Errorf("KeyOfBoundaries(%d) = %d, want %d", tt.year, k, tt.key)
			}
			if l, _ := KeyLetter(tt.key); l != tt.letter {
				t.Errorf("KeyLetter(%d) = %s, want %s", tt.key, l, tt.letter)
			}

			// Boundaries of the Triodion periods for this key.
			periods := []struct {
				off  int
				want Period
			}{
				{PublicanOffset - 1, PeriodOrdinary},
				{PublicanOffset, PeriodLentenTriodion},
				{-1, PeriodLentenTriodion},
				{0, PeriodPentecostarion},
				{AllSaintsOffset, PeriodPentecostarion},
				{AllSaintsOffset + 1, PeriodOrdinary},
			}
			for _, pp := range periods {
				ctx := NewDayContext(p.AddDays(pp.off))
				if ctx.Period != pp.want {
					t.Errorf("Pascha%+d: period %v, want %v", pp.off, ctx.Period, pp.want)
				}
				if ctx.PaschaOffset != pp.off {
					t.Errorf("Pascha%+d: offset %d", pp.off, ctx.PaschaOffset)
				}
			}
		})
	}
}

func TestPaschaExample2026(t *testing.T) {
	p := Pascha(2026)
	if p != MustJDate(2026, time.March, 30) {
		t.Fatalf("Pascha(2026) = %v, want 2026-03-30 O.S.", p)
	}
	if got := p.CivilString(); got != "2026-04-12" {
		t.Fatalf("civil Pascha 2026 = %s, want 2026-04-12", got)
	}
}

func TestPaschaRange(t *testing.T) {
	keys := map[int]bool{}
	for y := minYear; y <= maxYear; y++ {
		p := Pascha(y)
		if p.Weekday() != time.Sunday {
			t.Errorf("Pascha(%d) = %v is a %v", y, p, p.Weekday())
		}
		k := KeyOfBoundaries(y)
		if k < 1 || k > 35 {
			t.Errorf("key(%d) = %d", y, k)
		}
		keys[k] = true
		// Pascha is after the Julian equinox date (21 March) and the 14th day
		// of the paschal moon: the Gregorian date is never before 4 April in
		// this range.
		if _, m, d := p.Civil(); m < time.April || (m == time.April && d < 3) {
			t.Errorf("civil Pascha %d too early: %s", y, p.CivilString())
		}
	}
	if len(keys) != 35 {
		t.Errorf("%d distinct keys in %d..%d, want 35", len(keys), minYear, maxYear)
	}
}

// In a leap year every Triodion boundary before 1 March falls one month-day
// later than in a common year with the same key.
func TestTriodionLeapYears(t *testing.T) {
	type md struct {
		m time.Month
		d int
	}
	byKey := map[int]map[bool]md{}
	for y := minYear; y <= maxYear; y++ {
		pub := Pascha(y).AddDays(PublicanOffset)
		_, m, d := pub.Date()
		k := KeyOfBoundaries(y)
		if byKey[k] == nil {
			byKey[k] = map[bool]md{}
		}
		byKey[k][IsLeap(y)] = md{m, d}
		if m > time.February {
			t.Errorf("%d: Publican after February: %v", y, pub)
		}
	}
	pairs := 0
	for k, v := range byKey {
		common, ok1 := v[false]
		leap, ok2 := v[true]
		if !ok1 || !ok2 {
			continue
		}
		pairs++
		want := MustJDate(2001, common.m, common.d).AddDays(1) // 2001 is common
		if want.Month() != leap.m || want.Day() != leap.d {
			t.Errorf("key %d: Publican %v in common years, %v in leap years", k, common, leap)
		}
	}
	if pairs < 25 {
		t.Errorf("only %d keys seen in both leap and common years", pairs)
	}
}

func TestLiturgicalDate(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*3600)
	tests := []struct {
		civil time.Time
		want  JDate
	}{
		// Saturday morning → Saturday; Saturday vigil → Sunday.
		{time.Date(2026, time.November, 14, 9, 0, 0, 0, moscow), MustJDate(2026, time.November, 1)},
		{time.Date(2026, time.November, 14, 15, 59, 0, 0, moscow), MustJDate(2026, time.November, 1)},
		{time.Date(2026, time.November, 14, 16, 0, 0, 0, moscow), MustJDate(2026, time.November, 2)},
		{time.Date(2026, time.November, 14, 23, 30, 0, 0, moscow), MustJDate(2026, time.November, 2)},
		// Paschal vigil on Great Saturday night belongs to Pascha.
		{time.Date(2026, time.April, 11, 23, 30, 0, 0, moscow), Pascha(2026)},
		// Year boundary, and the eve of a Julian 29 February.
		{time.Date(2027, time.January, 13, 18, 0, 0, 0, moscow), MustJDate(2027, time.January, 1)},
		{time.Date(2028, time.March, 12, 18, 0, 0, 0, moscow), MustJDate(2028, time.February, 29)},
	}
	for _, tt := range tests {
		if got := LiturgicalDate(tt.civil, DefaultEveningStart); got != tt.want {
			t.Errorf("LiturgicalDate(%v) = %v, want %v", tt.civil, got, tt.want)
		}
	}
	if d := MustJDate(2026, time.November, 2); d.Eve() != MustJDate(2026, time.November, 1) {
		t.Errorf("Eve wrong")
	}
}

func TestCassianTransfer(t *testing.T) {
	tests := []struct {
		date     JDate
		transfer bool
	}{
		{MustJDate(2026, time.February, 28), true},
		{MustJDate(2027, time.February, 28), true},
		{MustJDate(2028, time.February, 28), false},
		{MustJDate(2028, time.February, 29), false},
		{MustJDate(1900, time.February, 28), false}, // 1900 is a Julian leap year
		{MustJDate(2100, time.February, 28), false}, // so is 2100
		{MustJDate(2101, time.February, 28), true},
		{MustJDate(2026, time.March, 1), false},
	}
	for _, tt := range tests {
		ctx := NewDayContext(tt.date)
		has := len(ctx.Transfers) == 1 &&
			ctx.Transfers[0].From == MonthDay{time.February, 29} &&
			ctx.Transfers[0].Service == ServiceCompline
		if has != tt.transfer || (!tt.transfer && len(ctx.Transfers) != 0) {
			t.Errorf("%v: transfers %+v, want transfer=%v", tt.date, ctx.Transfers, tt.transfer)
		}
	}
	if ctx := NewDayContext(MustJDate(2028, time.February, 29)); ctx.Menaion != (MonthDay{time.February, 29}) {
		t.Errorf("Menaion day of 2028-02-29: %+v", ctx.Menaion)
	}
}

func TestDayContext2026(t *testing.T) {
	p := Pascha(2026) // 30 March O.S.
	tests := []struct {
		name     string
		date     JDate
		period   Period
		tone     int
		pentWeek int
		lectWeek int
		bright   bool
	}{
		{"Pascha", p, PeriodPentecostarion, 1, 0, -7, true},
		{"Bright Monday", p.AddDays(1), PeriodPentecostarion, 2, 0, -6, true},
		{"Bright Saturday", p.AddDays(6), PeriodPentecostarion, 8, 0, -6, true},
		{"Thomas Sunday", p.AddDays(7), PeriodPentecostarion, 1, 0, -6, false},
		{"Pentecost", p.AddDays(49), PeriodPentecostarion, 7, 0, 0, false},
		{"Monday of the Holy Spirit", p.AddDays(50), PeriodPentecostarion, 7, 1, 1, false},
		{"All Saints", p.AddDays(56), PeriodPentecostarion, 8, 1, 1, false},
		{"Monday after All Saints", p.AddDays(57), PeriodOrdinary, 8, 2, 2, false},
		{"2nd Sunday after Pentecost", p.AddDays(63), PeriodOrdinary, 1, 2, 2, false},
		{"Publican 2027", Pascha(2027).AddDays(PublicanOffset), PeriodLentenTriodion, 0, 0, 33, false},
		{"Cheesefare Sunday 2027", Pascha(2027).AddDays(-49), PeriodLentenTriodion, 0, 0, 36, false},
		{"Clean Monday 2027", Pascha(2027).AddDays(-48), PeriodLentenTriodion, 0, 0, 37, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewDayContext(tt.date)
			if ctx.Period != tt.period {
				t.Errorf("period %v, want %v", ctx.Period, tt.period)
			}
			if tt.tone != 0 && ctx.Tone != tt.tone {
				t.Errorf("tone %d, want %d", ctx.Tone, tt.tone)
			}
			if ctx.PentecostWeek != tt.pentWeek {
				t.Errorf("PentecostWeek %d, want %d", ctx.PentecostWeek, tt.pentWeek)
			}
			if ctx.LectionaryWeek != tt.lectWeek {
				t.Errorf("LectionaryWeek %d, want %d", ctx.LectionaryWeek, tt.lectWeek)
			}
			if ctx.BrightWeek != tt.bright {
				t.Errorf("BrightWeek %v", ctx.BrightWeek)
			}
		})
	}
}

// The tone advances by one each Sunday from Thomas Sunday, continuously
// through Great Lent up to Palm Sunday, and restarts at tone 1 on Thomas.
func TestToneContinuity(t *testing.T) {
	for y := minYear; y <= maxYear; y++ {
		thomas := Pascha(y).AddDays(ThomasOffset)
		end := Pascha(y + 1).AddDays(-7) // Palm Sunday
		want := 1
		for d := thomas; !d.After(end); d = d.AddDays(7) {
			if got := Tone(d); got != want {
				t.Fatalf("%v (Sunday %d after Thomas %d): tone %d, want %d", d, d.Sub(thomas)/7, y, got, want)
			}
			for i := 1; i < 7; i++ { // weekdays keep Sunday's tone
				if got := Tone(d.AddDays(i)); got != want && d.AddDays(i).Before(Pascha(y+1)) {
					t.Fatalf("%v: tone %d, want %d", d.AddDays(i), got, want)
				}
			}
			want = want%8 + 1
		}
		if Tone(Pascha(y+1).AddDays(ThomasOffset)) != 1 {
			t.Fatalf("Thomas %d not tone 1", y+1)
		}
	}
	// All Saints is always tone 8.
	for y := minYear; y <= maxYear; y++ {
		if got := Tone(Pascha(y).AddDays(AllSaintsOffset)); got != 8 {
			t.Errorf("All Saints %d: tone %d", y, got)
		}
	}
}

func TestCycleShapeRange(t *testing.T) {
	for y := minYear - 1; y <= maxYear; y++ {
		cyc := newCycleInfo(y)
		if cyc.jumpWeek < 14 || cyc.jumpWeek > 19 {
			t.Errorf("%d: Sunday after the Exaltation is Sunday %d after Pentecost", y, cyc.jumpWeek)
		}
		if cyc.publicanWeek < 33 || cyc.publicanWeek > 38 {
			t.Errorf("%d: Publican is Sunday %d after Pentecost", y, cyc.publicanWeek)
		}
		if _, ok := weekendOverrides[[2]uint8{uint8(cyc.key), uint8(cyc.publicanWeek)}]; !ok {
			t.Errorf("%d: no Andreev table for key %d, Publican week %d", y, cyc.key, cyc.publicanWeek)
		}
	}
}

// Structural properties of the ordinary readings, for every cycle in range.
func TestLectionsRange(t *testing.T) {
	for y := minYear; y < maxYear; y++ {
		cyc := newCycleInfo(y)
		sep15 := MustJDate(y, time.September, 15)
		sundayAfterExaltation := sep15.AddDays(mod(7-int(sep15.Weekday()), 7))
		publican := Pascha(y + 1).AddDays(PublicanOffset)

		// Lukan jump: Monday after the Sunday after the Exaltation reads Gospel
		// week 18, while the Apostle keeps the actual week.
		jm := NewDayContext(sundayAfterExaltation.AddDays(1))
		if jm.Lections.Gospel.Week != 18 || jm.Lections.Apostle.Week != jm.PentecostWeek {
			t.Errorf("%d: Monday after Sunday after Exaltation %v: %+v (week %d)",
				y, jm.Date, jm.Lections, jm.PentecostWeek)
		}
		// Before the Publican: weekdays read week 33, the Sunday reads Zacchaeus (32).
		for i := 1; i <= 5; i++ {
			l := NewDayContext(publican.AddDays(-i)).Lections
			if l.Apostle.Week != 33 || l.Gospel.Week != 33 {
				t.Errorf("%d: %v before Publican: %+v", y, publican.AddDays(-i), l)
			}
		}
		// (unless it is the Theophany itself, which has its own readings)
		if l := NewDayContext(publican.AddDays(-7)).Lections; !l.Omitted && (l.Apostle.Week != 32 || l.Gospel.Week != 32) {
			t.Errorf("%d: Sunday before Publican %v: %+v", y, publican.AddDays(-7), l)
		}
		// Every day of the cycle has sane readings.
		for d := cyc.pascha; d.Before(publican); d = d.AddDays(1) {
			ctx := NewDayContext(d)
			l := ctx.Lections
			if ctx.Period == PeriodLentenTriodion {
				t.Fatalf("%v: unexpected Triodion", d)
			}
			if l.Omitted {
				if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
					t.Errorf("%v: weekday without ordinary readings", d)
				}
				continue
			}
			if l.Apostle.Weekday != d.Weekday() || l.Gospel.Weekday != d.Weekday() {
				t.Errorf("%v: weekday mismatch %+v", d, l)
			}
			if ctx.LectionaryWeek <= 0 {
				if l.Apostle.Week != ctx.LectionaryWeek || l.Gospel.Week != ctx.LectionaryWeek || l.Adjusted {
					t.Errorf("%v: Pentecostarion readings shifted: %+v", d, l)
				}
				continue
			}
			for _, w := range []int{l.Apostle.Week, l.Gospel.Week} {
				if w < 1 || w > 33 {
					t.Errorf("%v: week %d out of 1..33 (%+v)", d, w, l)
				}
			}
		}
	}
}

func TestPeriodString(t *testing.T) {
	for p, s := range map[Period]string{
		PeriodOrdinary: "ordinary", PeriodLentenTriodion: "lenten-triodion", PeriodPentecostarion: "pentecostarion",
	} {
		if p.String() != s {
			t.Errorf("%d.String() = %q", p, p.String())
		}
	}
}

func TestInvalidDates(t *testing.T) {
	bad := [][3]int{{2026, 2, 29}, {2026, 13, 1}, {2026, 0, 1}, {2026, 4, 31}, {2026, 1, 0}}
	for _, b := range bad {
		if _, err := NewJDate(b[0], time.Month(b[1]), b[2]); err == nil {
			t.Errorf("NewJDate(%v) accepted", b)
		}
	}
	if _, err := KeyLetter(0); err == nil {
		t.Error("KeyLetter(0) accepted")
	}
	if _, err := KeyLetter(36); err == nil {
		t.Error("KeyLetter(36) accepted")
	}
}
