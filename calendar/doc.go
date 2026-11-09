// Package calendar is the calendar core of the liturgical generator: Julian
// dates (JDate, via Julian Day Numbers), the Alexandrian paschalion, the
// liturgical day with its evening start, and DayContext — the movable-cycle
// facts about a day that the rules engine works from (offset from Pascha,
// Triodion period, Octoechos tone, week after Pentecost, ordinary Apostle and
// Gospel readings, transfers such as St John Cassian on 28 February).
//
// All computations are done on Julian dates. Civil (Gregorian) dates are
// converted only at the boundary (FromCivil, FromTime, Civil, Time); the
// difference between the styles is derived from JDN, never hard-coded.
//
// Ordinary readings follow the Russian practice with the Lukan jump and the
// otstupka before the Sunday of the Publican, in the scheme of A. A. Andreev
// (О порядке чтения Евангелия за Божественной литургией в свете рекомендаций
// Поместного собора 1917–18 гг. // Вестник Екатеринбургской духовной
// семинарии, 2018) as implemented by the Ponomar project. Weekdays are
// computed by formula; Saturdays and Sundays use the generated table in
// lectionary_weekend.go. See ponomar_test.go for the cross-checks.
package calendar
