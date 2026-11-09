package calendar

import "time"

// Period is the part of the movable (Triodion) cycle a day belongs to.
type Period int

const (
	// PeriodOrdinary: from the Monday after All Saints to the Saturday before
	// the Sunday of the Publican and Pharisee (Octoechos and Menaion only).
	PeriodOrdinary Period = iota
	// PeriodLentenTriodion: from the Sunday of the Publican and Pharisee
	// (Pascha−70) to Great Saturday (Pascha−1).
	PeriodLentenTriodion
	// PeriodPentecostarion: from Pascha to the Sunday of All Saints (Pascha+56).
	PeriodPentecostarion
)

func (p Period) String() string {
	switch p {
	case PeriodLentenTriodion:
		return "lenten-triodion"
	case PeriodPentecostarion:
		return "pentecostarion"
	default:
		return "ordinary"
	}
}

// Offsets from Pascha of the Triodion boundaries.
const (
	PublicanOffset  = -70 // Неделя о мытаре и фарисее
	AllSaintsOffset = 56  // Неделя всех святых
	PentecostOffset = 49
	ThomasOffset    = 7
)

// MonthDay is a day of the fixed (Menaion) cycle, in the Julian calendar.
type MonthDay struct {
	Month time.Month
	Day   int
}

// Service names a service of the daily cycle. The calendar package only uses
// it to say where a transferred commemoration is sung.
type Service string

// ServiceCompline is great or small compline.
const ServiceCompline Service = "compline"

// Transfer moves the service of a Menaion day to another day.
type Transfer struct {
	From    MonthDay // Menaion day whose service is moved here
	Service Service  // service at which it is sung
}

// DayContext is everything the rules engine needs to know about a liturgical
// day from the calendar alone. It is independent of language, display mode
// and parish options.
type DayContext struct {
	Date    JDate
	Weekday time.Weekday
	// Menaion is the day of the fixed cycle (Julian month and day).
	Menaion MonthDay

	// Pascha is the Pascha of Date's Julian year; PaschaOffset = Date − Pascha
	// (negative before Pascha).
	Pascha       JDate
	PaschaOffset int
	// Key is the key of boundaries (1..35) of Date's year.
	Key int

	// CyclePascha is the latest Pascha not after Date, which opened the
	// current movable cycle; CycleOffset = Date − CyclePascha (>= 0).
	CyclePascha JDate
	CycleOffset int

	Period     Period
	BrightWeek bool

	// Tone is the tone (глас) of the Octoechos, 1..8. It counts from Thomas
	// Sunday continuously through Great Lent; during Bright Week each day has
	// its own tone (Pascha 1, Monday 2 … Friday 6, Saturday 8). Whether the
	// Octoechos is actually sung is up to the rules.
	Tone int

	// PentecostWeek is the actual number of the week after Pentecost: Sunday
	// N or the weekdays preceding it (Monday after Pentecost is week 1, All
	// Saints is Sunday 1). Zero outside the span from Pentecost to the Sunday
	// of the Publican.
	PentecostWeek int

	// LectionaryWeek is the week index before any jump or otstupka (see
	// Lection for the numbering).
	LectionaryWeek int
	// Lections are the ordinary Apostle and Gospel readings at the Liturgy.
	Lections Lections

	// Transfers lists Menaion services moved to this day, e.g. St John
	// Cassian (29 February) is sung at compline on 28 February in common years.
	Transfers []Transfer
}

// NewDayContext computes the calendar context of the liturgical day d.
func NewDayContext(d JDate) DayContext {
	y, m, dd := d.Date()
	p := Pascha(y)
	off := d.Sub(p)

	ctx := DayContext{
		Date:         d,
		Weekday:      d.Weekday(),
		Menaion:      MonthDay{Month: m, Day: dd},
		Pascha:       p,
		PaschaOffset: off,
		Key:          KeyOfBoundaries(y),
	}

	cycleYear := y
	if off < 0 {
		cycleYear = y - 1
	}
	cyc := newCycleInfo(cycleYear)
	ctx.CyclePascha = cyc.pascha
	ctx.CycleOffset = d.Sub(cyc.pascha)

	inTriodion := off >= PublicanOffset && off < 0
	switch {
	case inTriodion:
		ctx.Period = PeriodLentenTriodion
	case off >= 0 && off <= AllSaintsOffset:
		ctx.Period = PeriodPentecostarion
	default:
		ctx.Period = PeriodOrdinary
	}
	ctx.BrightWeek = off >= 0 && off < 7

	ctx.Tone = toneOf(d, p, off)

	triodionWeek := 0
	if inTriodion {
		triodionWeek = 33 + lectionaryWeek(d, p.AddDays(PublicanOffset))
		ctx.LectionaryWeek = triodionWeek
	} else {
		ctx.LectionaryWeek = lectionaryWeek(d, cyc.pentecost)
		if ctx.LectionaryWeek > 0 {
			ctx.PentecostWeek = ctx.LectionaryWeek
		}
	}
	ctx.Lections = computeLections(d, cyc, inTriodion, triodionWeek)

	if m == time.February && dd == 28 && !IsLeap(y) {
		ctx.Transfers = append(ctx.Transfers, Transfer{
			From:    MonthDay{Month: time.February, Day: 29},
			Service: ServiceCompline,
		})
	}
	return ctx
}

// brightWeekTones are the tones of Bright Week by weekday (Sunday = Pascha).
var brightWeekTones = [7]int{1, 2, 3, 4, 5, 6, 8}

// Tone returns the Octoechos tone of day d (see DayContext.Tone).
func Tone(d JDate) int {
	p := Pascha(d.Year())
	return toneOf(d, p, d.Sub(p))
}

func toneOf(d, pascha JDate, off int) int {
	if off >= 0 && off < 7 {
		return brightWeekTones[d.Weekday()]
	}
	thomas := pascha.AddDays(ThomasOffset)
	if d.Before(thomas) {
		thomas = Pascha(d.Year() - 1).AddDays(ThomasOffset)
	}
	return floorDiv(d.Sub(thomas), 7)%8 + 1
}
