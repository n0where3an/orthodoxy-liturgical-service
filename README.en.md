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

Services: `vespers`, `compline` (small compline; great on its days), `midnight_office`, `matins` (joined to the
first hour unless `-standalone`), `first_hour`, `third_hour`, `sixth_hour`,
`ninth_hour`, `typika`, `liturgy` (St John Chrysostom; St Basil the Great on its days). Vespers and compline are dated
by the day they belong to: they are served the evening before. What is covered is in [Coverage](#coverage) below.

| Option | Meaning |
|---|---|
| `-date YYYY-MM-DD` | civil (Gregorian) date; `-julian` for old style |
| `-service` | `vespers`, `compline`, `midnight_office`, `matins`, `first_hour` … `ninth_hour`, `typika`, `liturgy` |
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

The margin names the book each text is taken from (Horologion, Menaion,
General Menaion, Octoechos, Hieratikon, Psalter, Irmologion, Apostle, Gospel),
where it changes; the short connecting formulas («Слава», «И ныне», «Аминь»,
«Господи, помилуй» thrice, exclamations, refrains, the verses before the
stichera) do not interrupt it. The common texts (the Trisagion, «Приидите»,
the Creed, «Господи, помилуй» 12 and 40) are marked as the book of the
service itself: the Hieratikon at the liturgy, the Horologion otherwise. On a
narrow screen it stands above the text.

## Sources and rules

The Irmologion's irmoi by tone (`ir.irmos.<tone>.<ode>.<n>`, 01glas–08glas)
complete the irmoi the Octoechos, the Menaion and the Triodion cite by their
first words («ирмос: Яко по суху:»): the same ode's irmos beginning so, of
the unit's tone first; abbreviations and variants («прпдбным» /
«преподобным») by the first words when they pick one irmos
(`Meta.completed_from`). About 200 are completed; ambiguous cues stay with a
warning. The 9th-ode refrains (`ir.pripev.*`) replace «Честнейшую» on the
feast, its apodosis, the synaxes after the Nativity and the Theophany («якоже
на самый праздник»; the Forerunner's canon with his own), and on the
Circumcision (01.01) and the Three Hierarchs (01.30) with their own.
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
Menaion day lacks, from the common service of the saint's rank: on any day the
troparion, kontakion and the sessional hymn after the 3rd ode (not where the
Menaion says «тропарь праздника» — a forefeast or afterfeast — nor for
additional services `me.MM.DDp`); with a sign (great vespers) also the common
paremias (ch. 39), the aposticha, the prokeimenon of matins, and the
prokeimenon, Apostle, alleluia, Gospel and communion verse of the liturgy
(ch. 40). The rank comes from the day's own reference («на литургии служба
преподобническая», «чтения три стительская», «евангелие стительское»), else
from its Apostle and Gospel, else from its title (with a sign the last
commemoration, the Russian saint added after the Greek ones; without, the
first); copies carry `copied_from` (and `rank_from_title` when guessed).

## Coverage

✅ supported · ⚠️ partly (see the note) · ❌ not yet (the service reports
«ещё не реализовано») · — not served that day.
Vespers and compline are listed under the day they belong to: they are served the
evening before. Typika: the service in place of the liturgy on days without one
(so far after the Royal Hours only).

### Ordinary year (Menaion + Octoechos)

| Day | Vespers | Compline ⁶ | Midnight<br>Office ⁷ | Matins | Hours | Typika | Liturgy |
|---|---|---|---|---|---|---|---|
| Weekday (Mon–Fri), saint without a sign | ✅ daily | ✅ small | ✅ daily | ✅ daily | ✅ | ❌ | ✅ |
| Weekday, saint with a sign (polyeleos, vigil) | ✅ great / vigil ¹ | ✅ / — before a vigil | ✅ / — before a vigil | ✅ festal ² | ✅ | ❌ | ✅ |
| Weekday in an afterfeast or on the apodosis | ✅ daily | ✅ the feast's kontakion | ✅ daily | ✅ ⁵ | ✅ | ❌ | ⚠️ ⁸ |
| Fixed twelve feast on a weekday | ✅ vigil | — | — | ✅ festal | ✅ | ❌ | ✅ |
| Saturday (any) | ✅ daily or vigil (Friday evening) | ✅ (Friday evening) | ✅ Saturday | ⚠️ ⁹ | ✅ | ❌ | ⚠️ ⁴ |
| Sunday, saint without a sign | ✅ vigil, without litiya | ✅ (where there is no vigil) | ✅ Sunday | ✅ ³ | ✅ ¹⁰ | ❌ | ✅ |
| Sunday, saint with a sign | ✅ vigil (litiya if the Menaion has one) | ✅ / — before a vigil | ✅ Sunday / — after a saint's vigil | ✅ ³ | ✅ ¹⁰ | ❌ | ✅ |
| Sunday in an afterfeast or on the apodosis | ✅ vigil | ✅ the feast's kontakion | ✅ Sunday | ✅ ³ | ✅ ¹⁰ | ❌ | ✅ |
| Sunday with its own readings (before/after the Exaltation, the Nativity — of the Fathers —, the Theophany) | ⚠️ not before the Exaltation (forefeast with the Renewal) | ✅ | ✅ Sunday | ⚠️ not before the Exaltation | ✅ ¹⁰ | ❌ | ⚠️ ¹¹ |
| Feast of the Lord on Sunday (Exaltation, Meeting, Transfiguration) | ✅ vigil | — | — | ✅ festal | ✅ | ❌ | ✅ |
| Feast of the Theotokos on Sunday | ✅ vigil | — | — | ✅ ³ | ✅ the resurrectional troparion, «Слава» the feast's; the resurrectional and the feast's kontakia alternately (the resurrectional at the 1st hour) | ❌ | ✅ |
| Nativity of Christ, Theophany and their eves (Royal Hours) | ✅ the eve: on a weekday with the liturgy, on Saturday and Sunday apart ¹³ | ✅ great, with the litiya ¹⁴ | — on the feast | ✅ ¹⁵ | ✅ Royal ¹³ | ✅ after the Royal Hours ¹³ | ✅ the feast ¹², the eve ¹³ |
| Saint with a sign whose readings the Menaion does not give (≈10 days) | ❌ | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ |

¹ Readings missing in the Menaion are taken from the General Menaion by the
saint's rank. ² Needs the matins Gospel in the Menaion (or «евангелие
святительское / преподобническое»). The canon by the Typikon, ch. 11 («аще ли
поется стаго полиелей: тогда не поется октоих»): the canon of the Theotokos
(the third canon of the tone's Sunday matins) with its irmos, then the
saint's; the Menaion's own canon of the Theotokos (a vigil, ch. 47) or of the
forefeast where it gives one; the katavasia after every ode, the irmos of
the last canon.
³ The Octoechos with the Menaion: eothinon Gospel (the cycle of eleven from
All Saints), evlogitaria, hypakoe, antiphons and prokeimenon of the tone; the
katavasia by season from `data/calendar/katavasia.yaml`.
⁴ By the Typikon, ch. 12 («Бог Господь»): «Апостоли, мученицы», the saint's
troparion and kontakion or «Помяни, Господи», «Со святыми упокой», «Яко
начатки»; the saint's prokeimenon, Apostle, alleluia, Gospel and communion
verse before the day's. On Dimitrievskaya parental Saturday (the last
Saturday before 26 October; a Saturday earlier when it falls on a great feast —
polyeleos or higher in the Typikon, great vespers in the Menaion), after the day's, for the departed: the
prokeimenon «Души их», the Apostle (1 Thess, зач. 270), the Gospel (John,
зач. 16), the alleluia verse and communion verse «Блажени, яже избрал». Not
yet: the Octoechos troparia at the beatitudes (in the corpus for tones 2 and
8 only), the Trinity parental Saturday (the Triodion; the Meatfare one: note ¹⁶), Saturday
in an afterfeast.
⁵ As the Menaion of the afterfeast day gives it: at «Бог Господь» the feast's
troparion twice, «Слава» the saint's, «И ныне» the feast's (thrice the
feast's without the saint's); the feast's sessional hymns; the feast's canon
with its irmos on 8 and the saint's on 4, no Octoechos; the katavasia after
every ode, by the season or the feast's irmoi; after the 3rd ode the saint's
kontakion and sessional hymn, after the 6th the feast's kontakion and ikos;
the exapostilarion, «Слава, и ныне» the feast's; the feast's aposticha;
«Честнейшую» is sung (ch. 20). Afterfeast days with the great doxology (the
Synaxis of the Forerunner on 7 January etc.: the Menaion has its praises):
no aposticha, the praises and the great doxology; the Menaion's kontakia in
its order (after the 3rd ode the last but one with its ikos and the
sessional hymn, after the 6th the last with its ikos); at the 9th ode the
feast's refrains where the Menaion says «якоже на самый праздник» (26
December, 7 January), else «Честнейшую». The Synaxis of the Theotokos (26
December) by the Menaion: the Nativity's service (its canons on 12, the
exapostilarion thrice), the day's sessional hymns and praises, after the
3rd ode the feast's kontakion, ikos and ipakoi, after the 6th «Иже прежде
денницы» (kontakion and ikos cut from the Menaion by the manifest). The
apodosis on a weekday: «вся служба праздника» without the polyeleos and the
Gospel — the feast's troparion thrice, its sessional hymns, Psalm 50 read,
the feast's canons, the katavasia by the Menaion (the first canon's irmoi;
on 12 September and 13 August «Крест начертав»), the sessional hymn (ipakoi)
after the 3rd ode, the feast's kontakion and ikos after the 6th, the 9th ode
with the feast's refrains, the exapostilarion twice, the feast's praises, the
great doxology, the troparion; the saint of the day is not sung. The
apodosis of the Entry (25 November), whose Menaion gives its own matins with
the saints, as an afterfeast day with the doxology. Not yet: the apodosis on
Saturday, afterfeast days with a polyeleos saint (9 September, 9 and 12
January, 4 February — Russian saints added to the day's service).
A few days still fail: 16 January (the source gives the service as one run
of text), days whose Menaion canon lacks the troparia of an ode, and vigils
without a matins Gospel (the corpus has no matins Gospels by rank).

⁸ Weekdays and Saturday of the afterfeast and the apodosis: the beatitudes from
the feast's canon (3rd ode) and the saint's (6th), or by the Menaion; at a
feast of the Lord the entrance «Спаси ны, Сыне Божий, [the feast's refrain]»;
the troparion of the feast, of the church and of the saint, «Слава» the
saint's kontakion, «И ныне» the feast's (Typikon ch. 52: no hymns of a church
of the Lord in an afterfeast of the Lord, of the Theotokos in hers); the
prokeimenon, alleluia and communion verse of the feast and of the saint, the
day's and the saint's Apostle and Gospel (on Saturday too the day's first:
«апостол и еиглие прежде субботы…, таже праздника»; no Saturday hymns — the
Octoechos is not sung); the feast's zadostoinik (in the afterfeast of the Nativity and the Theophany
«Любити убо нам», «Недоумеет всяк язык» with their refrains). Not yet:
afterfeast days with their own readings (the Saturdays after the Exaltation,
the Nativity, the Theophany).
⁹ Saturday with «Бог Господь» (Typikon ch. 12), a saint without a sign: the
saint's troparion twice, «Слава, и ныне» the theotokion; kathismata 16 and 17,
the small litany after each; after the 16th the Octoechos's martyrs' sessional
hymns — two, after the 17th two and the one for the departed; the saint's canon
with its irmos, then the Octoechos's of the martyrs (the canon for the departed
is sung at compline on Friday), the katavasia the irmos of the last canon; the
saint's exapostilarion, «Слава» the Octoechos's «Иже и мертвыми и живыми», «И
ныне» the theotokion; the Octoechos's stichera of the praises are sung at the
aposticha — the martyrs' three with «Исполнихомся заутра», «И буди светлость»,
«Слава» the saint's idiomelon or «Слава, и ныне» their theotokion. A saint
with a sign (ch. 11) has the saint's own order — of the doxology (kathismata
16 and 17, the Menaion's sessional hymns where it gives them, else the
Octoechos's martyrs'; the saint's stichera at the praises, the great
doxology, no aposticha) or of the polyeleos and vigil, as on a weekday; no
Saturday canon of the Octoechos: «канон бцды прешедшия недели утренний» (the
third canon of the tone's Sunday matins) with its irmos, then the saint's,
or the Menaion's own canon of the Theotokos where it gives one («прилагается
же и канон бцды», ch. 47); the saint's exapostilarion twice, «Слава, и ныне»
the theotokion. Not yet: the canon of a church of the Lord or the Theotokos
before the Theotokos's, Saturdays with «Аллилуиа» (a saint without a
troparion, ch. 13), parental Saturdays, Saturdays of a forefeast or
afterfeast; days whose Menaion lacks the matins Gospel (the vigils of 08.01,
08.02, 08.29, 11.14) or the sessional hymn after the 3rd ode (08.30) fail.
¹⁰ Typikon, ch. 52: the resurrectional troparion, «Слава» the saint's
(without one «Слава, и ныне»), «И ныне» the theotokion of the hour; the
resurrectional kontakion. A saint with a sign (doxology and higher, ch. 3–5):
the saint's kontakion at the 1st hour, at the others the resurrectional and the saint's
alternately. In a forefeast or afterfeast «Слава» the feast's, the kontakion
the feast's and the resurrectional alternately (with a great saint, the
feast's and the saint's). The Sunday of the Fathers: «Слава» theirs, their kontakion
and the forefeast's alternately («Рукописаннаго образа», in the forefeast
«Веселися, Вифлееме»); after the Nativity the troparia and kontakia of the
feast and of the saints (David, Joseph, James) alternately.
¹¹ By the rubrics of the Menaion (09.08, 09.15, 12.18, 12.26, 01.01, 01.07;
in the corpus `ms.<sunday>.*`): the Sunday's prokeimenon, Apostle, alleluia,
Gospel and communion verse — the tone's prokeimenon left out (except after
the Nativity: «прокимен гласа: и стых»); its readings before the ordinary
ones, these when the table of readings appoints them (the Fathers': «ряд
оставляется»), then the saint's. The Fathers: the resurrectional troparion
and theirs (in the forefeast also the forefeast's), «Слава» their kontakion,
«И ныне» the forefeast's. After the Nativity: the resurrectional troparion,
the feast's and the saints', «Слава» the saints' kontakion, «И ныне» the
feast's. In the afterfeast of the Nativity and the Theophany their
zadostoiniks («Любити убо нам», «Недоумеет всяк язык»). Not yet: the
beatitudes from the canon of the Fathers and of the saints (the Octoechos's
are sung with the canon of the saint of the day), the Sunday before the
Theophany on the Circumcision (1 January: St Basil's liturgy) and on the
eves, the Saturdays before and after these feasts.
¹² By the Menaion (12.25, 01.06): the liturgy of St John Chrysostom on the
feast itself when the eve is a weekday (St Basil's is served on the eve
then); the festal antiphons (Pss 110, 111, 109 and 113, 114, 117) with their
refrains and troparion, the entrance verse («Из чрева прежде денницы»,
«Благословен грядый»), the feast's troparion and kontakion, «Елицы во Христа
крестистеся» instead of the Trisagion, the prokeimenon, Apostle (Gal, зач.
209; Tit, зач. 302), alleluia, Gospel (Mt, зач. 3; зач. 6), the zadostoinik
«Величай, душе моя… Любити убо нам» / «…Недоумеет всяк язык», the communion
verse, the dismissal «Иже в вертепе родивыйся» / «Иже во Иордане креститися
изволивый». The feast on Sunday or Monday (the eve on Saturday or Sunday):
St Basil's liturgy on the feast itself.
¹³ The eve (the Menaion, 24 December, 5 January; 25 December, 6 January).
The Royal Hours on the eve, or, the feast falling on Sunday or Monday, on the
Friday before (no liturgy on that Friday): each hour's own psalms, «Слава»
the forefeast's troparion, «И ныне» the hour's theotokion, the stichera
(«Вифлееме, уготовися», «Днесь вод освящается естество») with the verses «Бог
от юга приидет» / «Сего ради помянух Тя», the canonarch's sticheron at the
9th hour, the prokeimenon, prophecy, Apostle, Gospel, the forefeast's
kontakion. After the 9th hour the typika (`-service typika`): the beatitudes
without troparia, the forefeast's kontakion; on the eve without the Creed,
ending with «Достойно есть» and the dismissal (the liturgy follows), on the
Friday with the Creed, «Буди имя Господне» and Psalm 33. The liturgy of a
weekday eve is vespers joined to St Basil's liturgy: «Господи, воззвах» with
four stichera twice on 8 and «Слава, и ныне», the entrance with the Gospel,
«Свете тихий», the prokeimenon of the day (on the Nativity's eve on Friday
the great «Кто Бог велий»), 8 / 13 readings, after the 3rd and the 6th the
reader's troparion with its verses («Тайно родился еси в вертепе», «Явился
еси в мире»), the choir singing its end; the small litany, the Trisagion,
the eve's prokeimenon, Apostle, alleluia, Gospel (not the ordinary ones), «О
Тебе радуется»; after the Nativity's liturgy its troparion and kontakion
before the candle. On the Theophany's eve, after the prayer behind the ambo,
the Great Blessing of Water by the Menaion (6 January): the troparia «Глас
Господень на водах», «Днесь вод освящается», «Яко человек», «Ко гласу
вопиющаго»; three readings from Isaiah, the prokeimenon «Господь
просвещение мое», the Apostle (1 Cor, зач. 143 from its middle), alleluia,
the Gospel (Mk, зач. 2); the great litany with the petitions for the water
(the priest secretly: «Господи Иисусе Христе, Единородный Сыне»); «Велий
еси, Господи» thrice and the prayer of blessing, «Сам и ныне, Владыко»
thrice; the prayer at the bowing of heads; the Cross plunged thrice with
the troparion «Во Иордане», the sprinkling, «Слава, и ныне» «Воспоим,
вернии»; then «Буди имя Господне». The texts are cut from the Menaion
(wb.*). On the feast itself the Typikon has no blessing of water. Not yet:
the polychronion at the 9th Royal Hour.
The eve on Saturday or Sunday (the Royal Hours on Friday): St John
Chrysostom's liturgy at its time — on Sunday the Sunday's of the Fathers or
before the Theophany (note ¹¹), on Saturday the day's, with the Apostle and
Gospel «прежде субботе пред Рождеством / Богоявлением, и потом ряд» and the
forefeast's troparion and kontakion; in the evening great vespers apart
(`-service vespers` with the date of the feast): «Блажен муж» (the whole
kathisma on Saturday evening, the first antiphon on Sunday evening),
«Господи, воззвах» on 8, the entrance, the prokeimenon of the day, the
readings with the troparia; no Trisagion — after the small litany the
prokeimenon, Apostle, alleluia, Gospel (the Nativity's: Gal, зач. 207, Mt,
зач. 53), the augmented and evening litanies, no aposticha; on the Theophany's eve,
after the exclamation, the Great Blessing of Water (the same order); the
dismissal; then the feast's troparion and kontakion. The Saturdays before the Nativity
and the Theophany read their own Apostle and Gospel before the ordinary
ones; on the weekdays and Saturdays of a forefeast the liturgy has the
forefeast's troparion and the saint's, «Слава» the saint's kontakion, «И
ныне» the forefeast's (ch. 52: «дне же не глаголется»).
¹⁴ Great compline by the Horologion: Pss 4, 6, 12, 24, 30, 90, «С нами Бог»
with its verses (Is 8–9), «День прешед», «Безплотное естество», the Creed,
«Пресвятая Владычице» and the other supplications, the troparia; Pss 50,
101, the prayer of Manasseh, the troparia «Помилуй нас, Господи»; Pss 69,
142, «Слава в вышних Богу». On the Nativity and the Theophany (the Menaion):
the beginning with the deacon, the feast's troparion instead of «Просвети
очи моя», its kontakion instead of «Помилуй нас, Господи»; after «Слава в
вышних» the litiya in the narthex: the stichera of the litiya, the litany
«Спаси, Боже, люди Твоя» (by the Hieratikon, as at vespers), the
aposticha, «Ныне отпущаеши», the feast's troparion thrice (on the Theophany
the blessing of the loaves), «Буди имя Господне», Psalm 33 — and matins. In
Great Lent, from the 2nd to the 6th week, Monday to Thursday evenings:
«Сподоби, Господи», the canon to the Theotokos of the Octoechos, «Господи
сил» with its verses, «Господи, аще не быхом», the prayers, St Ephrem's
prayer with the prostrations, «Нескверная», «И даждь нам», the forgiveness,
the litany «Помолимся о великом господине». Not yet: the first week (St
Andrew's canon), Holy Week, the Annunciation in Lent, a Menaion canon at
compline.

¹⁵ Matins of the Nativity and the Theophany by the Menaion, as festal
matins: «Бог Господь» and the feast's troparion thrice (on the Theophany its
own, not the Menaion's first: the Royal Hours' troparia precede it), two
kathismata with their sessional hymns (no third), the polyeleos and its
sessional hymn, the anabathmoi, prokeimenon, Gospel, after Psalm 50
«Всяческая днесь…» at «Слава… И ныне…» instead of «Молитвами апостол /
Богородицы», and the sticheron; two canons with the katavasia (both irmoi),
the ipakoi after the 3rd ode, the kontakion and ikos after the 6th, the 9th
ode with the Irmologion's refrains, the exapostilarion thrice, the praises,
the great doxology. The Menaion canon parser was fixed: the ipakoi is taken
as the sessional hymn after the 3rd ode, the 9th ode's refrains and rubrics
no longer become troparia (this also fixed 06.29, 08.15, 09.14, 01.01), the
Nativity's 7th ode lacking a heading after the ikos. The magnification with
the selected psalm comes from the Irmologion («Псалми избраннии») on every
feast it gives (the Nativity and the Entry of the Theotokos, the Exaltation,
the Nativity of Christ, the Theophany, the Meeting, the Annunciation, the
Nativity and the Beheading of the Forerunner, Ss Peter and Paul, the
Transfiguration, the Dormition): the clergy sing it, the choir repeats it,
and sings it after each verse of the psalm; «Слава, и ныне: Аллилуиа»
thrice. Saints with a sign whose Menaion has no magnification get the
Irmologion's common one of their rank (venerable, hierarch, martyr,
apostle, the bodiless powers), the saint's name in the vocative in place of
«имярек» («святителю отче Николае»), with the common selected psalm; the
rank is the one used for the General Menaion, the name comes from the day's
title. Several saints («отец наших», «мучеников»), women saints, princes of
no other rank, or a name not found: no magnification is supplied. The
Menaion's own magnifications get the common psalm of the rank; those of the
icons of the Theotokos, the selected psalm of her Nativity. Not yet: exact
pairing of the 9th-ode refrains with the second canon's troparia.
¹⁶ The liturgy of the preparatory weeks by the Lenten Triodion (orthlib,
`tp.<day>.*`). The Sundays of the Publican, the Prodigal, meatfare,
cheesefare: the typical antiphons, the beatitudes of the tone on 6 and the
Triodion's 6th ode on 4 (with its «Слава» and theotokion); the
resurrectional troparion and kontakion (and the temple's), «Слава» the
Triodion's kontakion, «И ныне» «Предстательство»; the Menaion's saint is not
sung on the Sundays of the Publican and the Prodigal («поем в пяток на
повечерии»), on cheesefare Sunday he is; the prokeimenon and alleluia of the
tone, on meatfare and cheesefare Sundays their own («Велий Господь наш»,
«Помолитеся»); the Sunday's Apostle and Gospel (the lectionary's: the
Triodion's); «Хвалите Господа». The meatfare Saturday of the departed: the
beatitudes on 8 from the 3rd and 6th odes of the Triodion's canon; «Глубиною
мудрости», «Слава» «Со святыми упокой», «И ныне» «Тебе и стену»; the
prokeimenon «Души их во благих водворятся», the day's readings and the
departed's (1 Thess, зач. 270; Jn, зач. 16), alleluia and communion verse
«Блажени, яже избрал». The cheesefare Saturday of the venerable fathers: the
beatitudes on 8 from the 3rd and 6th odes; «Боже отец наших», «Слава» their
kontakion; «Восхвалятся преподобнии», the day's readings and the fathers'
(Gal, зач. 213; Mt, зач. 43), alleluia «Насаждени в дому Господни»,
communion verse «Радуйтеся, праведнии». The other weekdays: the ordinary
liturgy of the Menaion and the Octoechos with the readings of weeks 34–36;
no liturgy on Wednesday and Friday of cheesefare week (unless a feast). The
Meeting in these weeks: the festal liturgy. Not yet: the Meeting's afterfeast
in these weeks (shortened by the Typikon).
¹⁷ The Saturdays and Sundays of Great Lent by the Lenten Triodion
(`tp.<day>.*`) and the Typikon (ch. 49): the 1st Saturday (St Theodore the
Recruit), the 2nd–4th (parental), the 5th (the Akathist), Sundays 1–5. The
liturgy: on Sundays St Basil's; the beatitudes of the tone and, where the
Triodion says so, an ode of the day's canon (Sunday 1: the 6th, the Cross:
the Cross's 3rd), on Saturdays the 3rd and 6th odes of the saint's canon (St
Theodore's, the Akathist's); the resurrectional troparion, the temple's and
the day's («Пречистому образу», St Gregory Palamas, «Спаси, Господи», St John
of the Ladder, St Mary of Egypt), the resurrectional kontakion, «Слава» the
Triodion's, «И ныне» «Предстательство»; on the Sunday of the Cross «Кресту
Твоему» instead of the Trisagion and «Знаменася»; prokeimena, alleluia and
communion verses by the Triodion (Sunday 1 «Благословен еси, Господи Боже
отец наших», 3 «Спаси, Господи, люди Твоя», 4 the tone's and «Восхвалятся
преподобнии»; Sunday 2 «вся служба стительская»: the hierarch's prokeimenon,
readings, alleluia, communion verse from the General Menaion); the day's
readings from the lectionary and the second ones — the saint's, the
departed's, the Theotokos's (by the rubrics' pericopes). Parental Saturdays:
«Апостоли, мученицы», «Помяни, Господи», «Слава» «Со святыми упокой», «И
ныне» «Тебе и стену»; the day's prokeimenon, alleluia, communion verse and
the departed's. The hours (also on the Sundays and Saturdays of the
preparatory weeks): the resurrectional troparion, «Слава» the day's, the
Triodion's kontakion; on Saturdays the day's troparion and kontakion, on
parental ones the departed's. The Sundays' vespers (Saturday evening, the
vigil): «Господи, воззвах» on 10 — the Octoechos's (resurrectional, of
Anatolius) and the Triodion's (4; the 2nd Sunday the hierarch's on 4, the
4th and 5th the saints' on 3), «Слава» the Triodion's, «И ныне» the
dogmatikon of the tone; «Господь воцарися»; the Octoechos's aposticha,
«Слава» the Triodion's, «И ныне» a theotokion in its tone (the Cross:
«Слава, и ныне» the Triodion's); «Богородице Дево» twice and the day's
troparion (the 2nd Sunday: thrice); the Menaion's saint at compline. St John
of the Ladder's troparion by the Triodion: «Пустынный житель». The Sundays'
matins: the Sunday order of the Octoechos with the Triodion for the
Menaion — the day's troparion at «Слава» of «Бог Господь»; «Непорочны»
instead of the polyeleos; after Psalm 50 «Покаяния отверзи ми двери», «На
спасения стези», «Помилуй мя, Боже», «Множества содеянных»; the canon:
resurrectional 4, the Theotokos's 2, the Triodion's 8 (Sunday 1:
resurrectional 4, Cross-and-Resurrection 2, Theotokos 2, Triodion 6); the
Triodion's katavasia on Sundays 1 and 3, else «Отверзу уста моя»; the
Triodion's sessional hymn after the 3rd ode (Sunday 2 with «Ныне время
делательное», 4 and 5 with the saint's kontakion and ikos), after the 6th
the Triodion's kontakion and ikos (1, 3), St Gregory's (2) or the Sunday's;
the eothinon exapostilarion, «Слава» the Triodion's, «И ныне» its
theotokion; the praises: the Octoechos 4 and the Triodion 4, «Слава» the
Triodion's idiomelon («паки тойже», 1 and 3) or the Gospel sticheron. The
1st Saturday's matins (St Theodore the Recruit): the Saturday order with
«Бог Господь», the Triodion for the Menaion — «Велия веры» twice, «Вся паче
смысла»; kathismata 16 and 17, after the 16th the Octoechos's martyrs'
sessional hymn, after the 17th the saint's; the temple's canon (the
Theotokos's) on 6 and the saint's on 8, the katavasia «Отверзу уста моя»;
after the 3rd ode his sessional hymn, after the 6th his kontakion and ikos;
his exapostilarion and its theotokion; his praises on 4 with «Слава» and «И
ныне»; his aposticha with «Праведник яко финикс», «Насаждени в дому
Господни», «И ныне» «Владычице, приими молитвы» (cited by its first words in
the Triodion, given in full by the General Menaion). Not yet: the matins of
the Akathist Saturday (the Akathist in its stations) and of the parental
Saturdays (matins with «Аллилуиа»), St Theodore's second canon, St Gregory's stichera at the praises (the Octoechos's are sung),
the
Saturdays' vespers (Friday evening: Lenten, with Friday's paremias, usually
with the Presanctified liturgy), the Octoechos troparia at the
beatitudes of the parental Saturdays (the verses alone), the Annunciation,
Lazarus Saturday, Palm Sunday.
¹⁸ The matins of the weekdays of Great Lent (Monday to Friday of the six
weeks) — with «Alleluia», by the Typikon (ch. 49, «thus let matins be … all
the holy Forty Days») and the Horologion: instead of «God is the Lord»,
Alleluia in the tone of the week with the verses «My soul waketh early unto
Thee» and the Trinity hymns of the tone (the first with the day's ending:
«at the intercessions of Thy bodiless powers», «of Thy Forerunner», «by the
power of Thy Cross», «of Thine apostles and St Nicholas»); three kathismata
by the Lenten rule of the Psalter (ch. 17; the 5th week its own); after the
1st the Octoechos's sessional hymns, after the 2nd and 3rd the Triodion's
(Joseph's and Theodore's), «Glory: the same»; the canon of the Menaion with
the triodes of the Triodion (`tp.w<week>_<day>.*`, no Octoechos), with
-typikon the Lenten biblical odes on 14; after the 3rd ode the saint's
sessional hymn, after the 6th his kontakion and ikos; the katavasia — the
irmos of the last triode, in the 3rd and 6th odes without a triode the
Menaion's irmos after the ode only; «Glory to Thee, our God, glory to Thee»
and «Let us bless the Father» as in the Triodion; after the 9th «It is truly
meet», the Trinity photagogikon of the tone thrice with the day's ending;
the praises and the doxology read; the aposticha: the Triodion's idiomelon
twice, the martyricon, the theotokion (on the 6th Friday «Glory» the
idiomelon «Having completed the soul-profiting», «Both now» the same);
«It is good», «Standing in the temple of Thy glory», «Lord, have mercy» 40,
«Christ our God, the Existing One», «O Heavenly King», the prayer of St
Ephrem — and the first hour. Not yet: the days of saints with a sign
(doxology, polyeleos) in Lent, the Wednesday and Friday of the Cross (the
Cross's canon), the 5th Thursday (the Great Canon), the Annunciation and its
afterfeast; the martyrs' sessional hymn after the 6th ode where the saint has
no kontakion; the readings at the kathismata; the prostrations.
⁶ By the Horologion: Pss 50, 69 and 101 (so in the Moscow 1980 Horologion),
«Слава в вышних», «Сподоби, Господи», the Creed, the Octoechos canon of
supplication to the Theotokos (Typikon ch. 9) with «Пресвятая Богородице,
спаси нас» and the sessional hymn after the 6th ode, «Достойно есть», the
Trisagion; the troparia by the weekday and the church (ch. 52); on Friday
evening «Апостоли, мученицы», «Помяни, Господи», «Яко начатки»; on Saturday
evening the kontakion of the tone; a saint with a sign — his kontakion; a
forefeast or afterfeast — the feast's kontakion. Not served before a vigil.
⁷ By the Horologion; St Basil's prayers from 22 September to Palm Sunday; the
litany «Помолимся о великом господине» after the dismissal. Not served after
an all-night vigil. The prayer to the Trinity «Всемогущая и животворящая» (on
Sunday) is given by its first words only, as in the Horologion.

### Triodion and Pentecostarion

| Period | Vespers | Compline | Midnight<br>Office | Matins | Hours | Typika | Liturgy |
|---|---|---|---|---|---|---|---|
| Preparatory weeks (Publican and Pharisee … Cheesefare) | ❌ | ❌ | ❌ | ❌ | ⚠️ Sundays and Saturdays ¹⁷ | ❌ | ⚠️ ¹⁶ |
| Great Lent, weekdays | ❌ | ⚠️ great ¹⁴ | ❌ | ⚠️ ¹⁸ | ❌ Lenten hours | ❌ | — (Presanctified ❌ Wed, Fri) |
| Great Lent, Saturdays and Sundays (St Basil's liturgy on Sundays) | ⚠️ Sundays ¹⁷ | ❌ | ❌ | ⚠️ Sundays, the 1st Saturday ¹⁷ | ✅ ¹⁷ | ❌ | ✅ ¹⁷ |
| Annunciation in Lent | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Palm Sunday, Holy Week | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Pascha and Bright Week (Paschal hours) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Pentecostarion (Thomas Sunday … All Saints), with Ascension and Pentecost | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

The Triodion is not imported yet; this is the largest remaining part.

### Still to do outside the calendar

| Item | Status |
|---|---|
| A Menaion canon at compline (forefeasts, a saint's service moved to compline) | ❌ |
| Irmologion: the irmoi by tone (to complete the irmoi cited by their first words); the 9th-ode refrains in the afterfeast and on the apodosis | ✅ |
| Psalter in the Triodion: Meatfare and Cheesefare weeks, Great Lent, its 5th week, Holy Week («Устав о Псалтири») | ⚠️ the kathismata of matins on the weekdays of Great Lent (the 5th week too) ¹⁸; not at the hours and vespers |
| Liturgy of St Basil | ⚠️ by the Hieratikon: its own secret prayers (of the catechumens, the faithful, the offertory, the anaphora, before the Lord's Prayer, of the bowing of heads, of thanksgiving, of the consumption), the words of institution «Даде святым Своим учеником…», the consecration «Хлеб убо сей…», «Чашу же сию…», «О Тебе радуется» (a feast's zadostoinik on the feast), St Basil's dismissal (typed in: the orthlib Hieratikon lacks it). Served on 1 January, on the Nativity and the Theophany on Sunday and Monday, and on their weekday eves (with vespers). Not yet: the Sundays of Great Lent, Great Thursday and Saturday (the Triodion) |
| Presanctified Liturgy | ❌ |
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
