package calendar

import "time"

// DefaultEveningStart is the conventional local time from which services
// belong to the next liturgical day (the day begins with the ninth hour and
// vespers). It is a default, not a rule of the Typikon: callers that know the
// actual service schedule or sunset should pass their own value.
const DefaultEveningStart = 16 * time.Hour

// LiturgicalDate returns the liturgical day that the civil moment t belongs
// to. The liturgical day begins in the evening: from eveningStart (time of day
// in t's location) onwards, t belongs to the next day. For example, the
// all-night vigil served on Saturday evening belongs to Sunday.
func LiturgicalDate(t time.Time, eveningStart time.Duration) JDate {
	d := FromTime(t)
	// Wall-clock time of day, so DST transitions do not shift the boundary.
	h, mi, s := t.Clock()
	tod := time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(s)*time.Second
	if tod >= eveningStart {
		return d.AddDays(1)
	}
	return d
}

// Eve returns the civil day on whose evening the services of liturgical day d
// begin (ninth hour, vespers, all-night vigil): the day before d.
func (d JDate) Eve() JDate { return d.AddDays(-1) }
