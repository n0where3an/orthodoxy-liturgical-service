# orthodoxy-liturgical-service
Orthodox liturgical sequence generator

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

Services: `liturgy` (St John Chrysostom: weekdays Mon–Fri of the ordinary
period, saints with a sign, fixed twelve feasts), `matins` (daily Mon–Fri; festal: saints with a sign and fixed twelve
feasts; joined to the first hour unless
`-standalone`), `vespers` (daily; Sunday great vespers at the Saturday-evening vigil with a saint
without a sign; great vespers and all-night vigil of Menaion saints with readings and of the
fixed twelve feasts), `first_hour`, `third_hour`, `sixth_hour`, `ninth_hour`.
Dates are civil (Gregorian); add `-julian` for old style. Options: `-reader`
(reader's order, no priest), `-standalone` (the hour on its own: usual
beginning and dismissal; by default hours are joined as in parish practice),
`-typikon` (Typikon instead of parish practice where they differ), `-deacon`
(litanies by a deacon).
`-temple <id>`: the dedication of the church where the service is held — an id
listed by `liturgy temples` (feasts of the Lord, of the Theotokos, patron saints;
`data/calendar/temples.yaml`) or a Menaion day `me.MM.DD`. At the liturgy the
temple's troparion and kontakion are sung after the entrance in the Typikon's
order for a church of the Lord, of the Theotokos or of a saint; rules can test
`temple`, `temple_kind` and `temple_feast` (the patronal feast is today).
In the web server: `&temple=<id>`.
Languages: `-lang cu|en`, `-lang2 en` for parallel output. Texts not yet
translated fall back to Church Slavonic (marked on the page).
Church Slavonic text is Unicode; install the Ponomar Unicode or Monomakh fonts.

The General Menaion (`orthlib.obschminea`, keys `om.<rank>.*`) supplies what a
Menaion day with a sign leaves to it: the common paremias (ch. 39), the
prokeimenon of matins, and the prokeimenon, Apostle, alleluia, Gospel and
communion verse of the liturgy (ch. 40). The rank comes from the day's own
reference («на литургии служба преподобническая», «чтения три стительская»,
«евангелие стительское»), else from its Apostle and Gospel, else from its title;
copies carry `copied_from` (and `rank_from_title` when guessed).

Layout: `calendar/` (Julian dates, paschalion, day context), `internal/hip`
(HIP → Unicode), `internal/orthlib` (import), `internal/corpus` (storage),
`internal/liturgy` (rules → plan → tree → mode → HTML), `data/` (templates,
rules, i18n, import manifest).

## License

GNU Affero General Public License v3.0 — see [LICENSE](LICENSE).

The table of Saturday/Sunday ordinary readings in `calendar/lectionary_weekend.go`
is generated from A. A. Andreev's tables as encoded in the
[Ponomar](https://github.com/typiconman/ponomar) project (GPLv3); the table of
ordinary readings (`lectionary.tsv`) is read from a Ponomar checkout at import.
