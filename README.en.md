# orthodoxy-liturgical-service
Orthodox liturgical sequence generator (Russian Typikon).

[Русская версия (основная)](README.ru.md) · [Contents](README.md)

## Quick start

Requirements: Go, PostgreSQL, `unrar`, a local copy of orthlib.ru.

```sh
cp .env.example .env            # set DATABASE_URL, ORTHLIB_DIR, PONOMAR_DIR
scripts/extract-orthlib.sh      # RAR archives -> var/orthlib/*.hip
go build -o var/liturgy ./cmd/liturgy
var/liturgy migrate
scripts/extract-english.sh      # Coverdale PDF -> var/english/coverdale.txt (optional)
var/liturgy import -v           # Slavonic corpus (readings if PONOMAR_DIR); English if ENGLISH_MD_DIR / coverdale.txt
var/liturgy plan   -date 2026-11-12 -service first_hour      # JSON plan
var/liturgy render -date 2026-11-12 -service third_hour -mode kliros -o terce.html
var/liturgy serve                                   # http://127.0.0.1:8080/?date=2026-11-12&service=sixth_hour&mode=laity
```

## Services and options

Services: `vespers`, `matins` (joined to the first hour unless `-standalone`),
`first_hour`, `third_hour`, `sixth_hour`, `ninth_hour`, `liturgy` (St John
Chrysostom). What is covered is in [Coverage](#coverage) below.

| Option | Meaning |
|---|---|
| `-date YYYY-MM-DD` | civil (Gregorian) date; `-julian` for old style |
| `-service` | `vespers`, `matins`, `first_hour` … `ninth_hour`, `liturgy` |
| `-reader` | reader's order, without a priest |
| `-deacon` | litanies by a deacon |
| `-standalone` | the hour on its own (usual beginning, dismissal); by default hours are joined as in parish practice |
| `-typikon` | the Typikon instead of parish practice where they differ: the canon of matins sung with the verses of the biblical odes; on summer Sundays the 17th kathisma instead of the polyeleos |
| `-psalmody short\|sung\|read` | the psalm verses before the stichera where the beginning is sung: at «Господи, воззвах» from «Положи, Господи, хранение», at the praises when the doxology is sung — `short` (default: straight to the verse of the first sticheron), `sung` or `read`. With the doxology read the praises are read |
| `-temple <id>` | dedication of the church: an id from `liturgy temples` (`data/calendar/temples.yaml`) or a Menaion day `me.MM.DD`; the temple's troparion and kontakion at the liturgy |
| `-lang cu\|en`, `-lang2 en` | language, parallel second language (untranslated texts fall back to Slavonic) |
| `-mode laity\|kliros` | display mode (`render`, `serve`) |

The web server takes the same options as query parameters
(`?date=…&service=…&mode=…&temple=…&psalmody=…`).

Church Slavonic text is Unicode; install the Ponomar Unicode or Monomakh fonts.

## Sources and rules

The Irmologion (`orthlib.irmologion`) gives the biblical odes in three sets —
weekdays, Sundays and feasts, Great Lent — as `ir.ode.<ode>.<set>.<n>` with
the places «на 14 / 8 / 6 / 4» marked, and the refrains of the 9th ode at the
feasts (`ir.pripev.<feast>.*`: on the feast day instead of «Честнейшую», each
troparion with its refrain). The set follows the sign of the saint in the
Typikon: black (six-fold) and none — the weekday odes, katavasia after the
3rd, 6th, 8th and 9th odes, the doxology read; red (the great doxology sung,
polyeleos, vigil), Sundays, feasts, fore- and afterfeasts — the festal odes,
katavasia after every ode. The signs come from the Typikon's calendar
(ch. 48, `ty.<MM>.<DD>.sign.<N>` per commemoration): `<(:.>` black —
six-fold; the same sign in red — doxology; red `+` — polyeleos; `\+/` — vigil of a saint (a middle feast, ch. 47); `(+)` — vigil; none; the day takes its highest sign. The Menaion's
«славословие великое» (`me.<day>.matins.great_doxology`) counts only in a
fore- or afterfeast (no sign of its own) or without the Typikon. On a red day at weekday matins
«Свят Господь Бог наш» after the 9th ode, the praises with the saint's
stichera, the great doxology sung, no aposticha. The kathismata follow the «Устав о
Псалтири» of the Following Psalter (summer and winter order; on a feast day of
the winter order two kathismata at matins and the third at the vespers that
evening).

The General Menaion (`orthlib.obschminea`, keys `om.<rank>.*`) supplies what a
Menaion day with a sign leaves to it: the common paremias (ch. 39), the
prokeimenon of matins, and the prokeimenon, Apostle, alleluia, Gospel and
communion verse of the liturgy (ch. 40). The rank comes from the day's own
reference («на литургии служба преподобническая», «чтения три стительская»,
«евангелие стительское»), else from its Apostle and Gospel, else from its title;
copies carry `copied_from` (and `rank_from_title` when guessed).

## Coverage

✅ supported · ⚠️ partly (see the note) · ❌ not yet (the service reports
«ещё не реализовано») · — not served that day.
Vespers is listed under the day it belongs to: it is served the evening before.

### Ordinary year (Menaion + Octoechos)

| Day | Vespers | Matins | Hours | Liturgy |
|---|---|---|---|---|
| Weekday (Mon–Fri), saint without a sign | ✅ daily | ✅ daily | ✅ | ✅ |
| Weekday, saint with a sign (polyeleos, vigil) | ✅ great / vigil ¹ | ✅ festal ² | ✅ | ✅ |
| Weekday in an afterfeast or on the apodosis | ✅ daily | ❌ | ✅ | ❌ |
| Fixed twelve feast on a weekday | ✅ vigil | ✅ festal | ✅ | ✅ |
| Saturday (any) | ✅ daily or vigil (Friday evening) | ❌ | ✅ | ❌ |
| Sunday, saint without a sign | ✅ vigil, without litiya | ✅ ³ | ❌ | ✅ |
| Sunday, saint with a sign | ✅ vigil (litiya if the Menaion has one) | ✅ ³ | ❌ | ✅ |
| Sunday in an afterfeast or on the apodosis | ✅ vigil | ✅ ³ | ❌ | ✅ |
| Sunday with its own readings (before/after the Nativity, the Theophany, before the Exaltation) | ⚠️ not before the Exaltation (forefeast with the Renewal) | ⚠️ not before the Exaltation | ❌ | ❌ |
| Feast of the Lord on Sunday (Exaltation, Meeting, Transfiguration) | ✅ vigil | ✅ festal | ✅ | ✅ |
| Feast of the Theotokos on Sunday | ✅ vigil | ✅ ³ | ⚠️ no resurrectional troparion | ✅ |
| Nativity of Christ, Theophany and their eves (Royal Hours) | ❌ | ❌ | ❌ | ❌ |
| Saint with a sign whose readings the Menaion does not give (≈10 days) | ❌ | ❌ | ✅ | ❌ |

¹ Readings missing in the Menaion are taken from the General Menaion by the
saint's rank. ² Needs the matins Gospel in the Menaion (or «евангелие
святительское / преподобническое»).
³ The Octoechos with the Menaion: eothinon Gospel (the cycle of eleven from
All Saints), evlogitaria, hypakoe, antiphons and prokeimenon of the tone; the
katavasia by season from `data/calendar/katavasia.yaml`.
A few days still fail where the Menaion import misses a troparion or a canon.

### Triodion and Pentecostarion

| Period | Vespers | Matins | Hours | Liturgy |
|---|---|---|---|---|
| Preparatory weeks (Publican and Pharisee … Cheesefare) | ❌ | ❌ | ❌ | ❌ |
| Great Lent, weekdays | ❌ | ❌ | ❌ Lenten hours | — (Presanctified ❌ Wed, Fri) |
| Great Lent, Saturdays and Sundays (St Basil's liturgy on Sundays) | ❌ | ❌ | ❌ | ❌ |
| Annunciation in Lent | ❌ | ❌ | ❌ | ❌ |
| Palm Sunday, Holy Week | ❌ | ❌ | ❌ | ❌ |
| Pascha and Bright Week (Paschal hours) | ❌ | ❌ | ❌ | ❌ |
| Pentecostarion (Thomas Sunday … All Saints), with Ascension and Pentecost | ❌ | ❌ | ❌ | ❌ |

The Triodion is not imported yet; this is the largest remaining part.

### Still to do outside the calendar

| Item | Status |
|---|---|
| Compline, Midnight Office (kathisma 17, on Saturday 9), Typika without the liturgy | ❌ |
| Irmologion: the irmoi by tone (to complete the irmoi cited by their first words); the 9th-ode refrains in the afterfeast and on the apodosis | ❌ |
| Psalter in the Triodion: Meatfare and Cheesefare weeks, Great Lent, its 5th week, Holy Week («Устав о Псалтири») | ❌ |
| Liturgy of St Basil, Presanctified Liturgy | ❌ |
| Patronal feast (`temple_feast`): vigil «аще храм» | ❌ (the flag is available to the rules) |
| Litiya with the temple's stichera at the Sunday vigil (`-typikon`) | ❌ |
| Dismissal of an afterfeast of the Lord on Sunday («Воскресый из мертвых» + the feast's) | ⚠️ only «Воскресый из мертвых» |
| Choice of Church Slavonic fonts | ❌ |
| English texts | ⚠️ fixed texts and the Menaion where translated; the rest falls back to Slavonic |

## Layout

`calendar/` (Julian dates, paschalion, day context), `internal/hip`
(HIP → Unicode), `internal/orthlib` (import), `internal/corpus` (storage),
`internal/liturgy` (rules → plan → tree → mode → HTML), `data/` (templates,
rules, i18n, import manifest).

## License

GNU Affero General Public License v3.0 — see [LICENSE](LICENSE).

The table of Saturday/Sunday ordinary readings in `calendar/lectionary_weekend.go`
is generated from A. A. Andreev's tables as encoded in the
[Ponomar](https://github.com/typiconman/ponomar) project (GPLv3); the table of
ordinary readings (`lectionary.tsv`) is read from a Ponomar checkout at import.
