package calendar

import (
	"fmt"
	"time"
)

// JDate is a date in the Julian calendar (old style), stored as a Julian Day
// Number (JDN). All internal computations use JDate; civil (Gregorian) dates
// appear only at the input/output boundary (FromCivil, FromTime, Civil, Time).
//
// The zero value is JDN 0 (1 January 4713 BC, Julian) and is not a meaningful
// liturgical date; construct values with NewJDate, MustJDate, FromJDN or
// FromCivil.
type JDate struct {
	jdn int
}

// unixEpochJDN is the JDN of 1970-01-01 (Gregorian).
const unixEpochJDN = 2440588

// IsLeap reports whether year is a leap year in the Julian calendar.
func IsLeap(year int) bool { return mod(year, 4) == 0 }

// DaysIn returns the number of days in the given month of a Julian year.
func DaysIn(year int, m time.Month) int {
	switch m {
	case time.February:
		if IsLeap(year) {
			return 29
		}
		return 28
	case time.April, time.June, time.September, time.November:
		return 30
	default:
		return 31
	}
}

// NewJDate returns the Julian calendar date year-month-day, validating it.
func NewJDate(year int, month time.Month, day int) (JDate, error) {
	if month < time.January || month > time.December {
		return JDate{}, fmt.Errorf("calendar: invalid month %d", month)
	}
	if day < 1 || day > DaysIn(year, month) {
		return JDate{}, fmt.Errorf("calendar: invalid day %d for %d-%02d (Julian)", day, year, month)
	}
	return JDate{jdn: julianToJDN(year, int(month), day)}, nil
}

// MustJDate is like NewJDate but panics on an invalid date. Intended for
// constants and tests.
func MustJDate(year int, month time.Month, day int) JDate {
	d, err := NewJDate(year, month, day)
	if err != nil {
		panic(err)
	}
	return d
}

// FromJDN returns the date with the given Julian Day Number.
func FromJDN(jdn int) JDate { return JDate{jdn: jdn} }

// FromCivil converts a civil (proleptic Gregorian) date to a JDate.
func FromCivil(year int, month time.Month, day int) (JDate, error) {
	if month < time.January || month > time.December {
		return JDate{}, fmt.Errorf("calendar: invalid month %d", month)
	}
	if day < 1 || day > gregorianDaysIn(year, month) {
		return JDate{}, fmt.Errorf("calendar: invalid day %d for %d-%02d (Gregorian)", day, year, month)
	}
	return JDate{jdn: gregorianToJDN(year, int(month), day)}, nil
}

// FromTime converts the civil calendar date of t (in t's own location) to a
// JDate. The time of day is ignored; see LiturgicalDate for the evening
// transition.
func FromTime(t time.Time) JDate {
	y, m, d := t.Date()
	return JDate{jdn: gregorianToJDN(y, int(m), d)}
}

// JDN returns the Julian Day Number.
func (d JDate) JDN() int { return d.jdn }

// Date returns the Julian calendar year, month and day.
func (d JDate) Date() (year int, month time.Month, day int) {
	y, m, dd := jdnToJulian(d.jdn)
	return y, time.Month(m), dd
}

// Year returns the Julian calendar year.
func (d JDate) Year() int { y, _, _ := d.Date(); return y }

// Month returns the Julian calendar month.
func (d JDate) Month() time.Month { _, m, _ := d.Date(); return m }

// Day returns the Julian calendar day of month.
func (d JDate) Day() int { _, _, dd := d.Date(); return dd }

// Weekday returns the day of the week (identical in both calendars).
func (d JDate) Weekday() time.Weekday { return time.Weekday(mod(d.jdn+1, 7)) }

// Civil returns the civil (proleptic Gregorian) date of d.
func (d JDate) Civil() (year int, month time.Month, day int) {
	y, m, dd := jdnToGregorian(d.jdn)
	return y, time.Month(m), dd
}

// Time returns midnight of the civil date of d in loc (UTC if loc is nil).
func (d JDate) Time(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	y, m, dd := d.Civil()
	return time.Date(y, m, dd, 0, 0, 0, 0, loc)
}

// StyleDifference returns the difference between the styles on d: how many
// days the same month/day label occurs earlier in the civil (Gregorian)
// calendar. It changes on 1 March O.S. of a century year that is not a
// Gregorian leap year: 12 until 29 February 1900 O.S., 13 from 1 March 1900
// O.S. (14 March N.S.), 14 from 1 March 2100 O.S. (15 March N.S.). It is
// computed from JDN, never assumed; use Civil for actual conversion.
func (d JDate) StyleDifference() int {
	jy, jm, jd := jdnToJulian(d.jdn)
	// Number the Julian calendar fields as if they were Gregorian; the JDN
	// difference is the style difference.
	return d.jdn - gregorianToJDN(jy, jm, jd)
}

// AddDays returns d shifted by n days.
func (d JDate) AddDays(n int) JDate { return JDate{jdn: d.jdn + n} }

// Sub returns the number of days d - o.
func (d JDate) Sub(o JDate) int { return d.jdn - o.jdn }

// Before reports whether d is before o.
func (d JDate) Before(o JDate) bool { return d.jdn < o.jdn }

// After reports whether d is after o.
func (d JDate) After(o JDate) bool { return d.jdn > o.jdn }

// String formats d as "YYYY-MM-DD O.S." (Julian).
func (d JDate) String() string {
	y, m, dd := d.Date()
	return fmt.Sprintf("%04d-%02d-%02d O.S.", y, int(m), dd)
}

// CivilString formats the civil date of d as "YYYY-MM-DD".
func (d JDate) CivilString() string {
	y, m, dd := d.Civil()
	return fmt.Sprintf("%04d-%02d-%02d", y, int(m), dd)
}

// --- conversions (integer algorithms valid for all JDN >= 0) ---

func julianToJDN(y, m, d int) int {
	a := (14 - m) / 12
	yy := y + 4800 - a
	mm := m + 12*a - 3
	return d + (153*mm+2)/5 + 365*yy + floorDiv(yy, 4) - 32083
}

func gregorianToJDN(y, m, d int) int {
	a := (14 - m) / 12
	yy := y + 4800 - a
	mm := m + 12*a - 3
	return d + (153*mm+2)/5 + 365*yy + floorDiv(yy, 4) - floorDiv(yy, 100) + floorDiv(yy, 400) - 32045
}

func jdnToJulian(jdn int) (y, m, d int) {
	c := jdn + 32082
	dd := floorDiv(4*c+3, 1461)
	e := c - floorDiv(1461*dd, 4)
	mm := (5*e + 2) / 153
	d = e - (153*mm+2)/5 + 1
	m = mm + 3 - 12*(mm/10)
	y = dd - 4800 + mm/10
	return
}

func jdnToGregorian(jdn int) (y, m, d int) {
	a := jdn + 32044
	b := floorDiv(4*a+3, 146097)
	c := a - floorDiv(146097*b, 4)
	dd := (4*c + 3) / 1461
	e := c - 1461*dd/4
	mm := (5*e + 2) / 153
	d = e - (153*mm+2)/5 + 1
	m = mm + 3 - 12*(mm/10)
	y = 100*b + dd - 4800 + mm/10
	return
}

func gregorianDaysIn(year int, m time.Month) int {
	if m == time.February {
		if mod(year, 4) == 0 && (mod(year, 100) != 0 || mod(year, 400) == 0) {
			return 29
		}
		return 28
	}
	return DaysIn(year, m)
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func mod(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}

// ceilDiv returns ceil(a/b) for b > 0.
func ceilDiv(a, b int) int { return -floorDiv(-a, b) }
