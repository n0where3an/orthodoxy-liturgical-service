# orthodoxy-liturgical-service
Orthodox liturgical sequence generator

## Quick start

Requirements: Go, PostgreSQL, `unrar`, a local copy of orthlib.ru.

```sh
cp .env.example .env            # set DATABASE_URL, ORTHLIB_DIR
scripts/extract-orthlib.sh      # RAR archives -> var/orthlib/*.hip
go build -o var/liturgy ./cmd/liturgy
var/liturgy migrate
var/liturgy import -v           # Chasoslov/Sluzhebnik units + Menaion troparia/kontakia
var/liturgy plan   -date 2026-11-12                 # JSON plan
var/liturgy render -date 2026-11-12 -mode kliros -o ninth.html
var/liturgy serve                                   # http://127.0.0.1:8080/?date=2026-11-12&mode=laity
```

Dates are civil (Gregorian); add `-julian` for old style. Options: `-reader`
(reader's order, no priest), `-standalone` (end with a dismissal).
Church Slavonic text is Unicode; install the Ponomar Unicode or Monomakh fonts.

Layout: `calendar/` (Julian dates, paschalion, day context), `internal/hip`
(HIP → Unicode), `internal/orthlib` (import), `internal/corpus` (storage),
`internal/liturgy` (rules → plan → tree → mode → HTML), `data/` (templates,
rules, i18n, import manifest).

## License

GNU Affero General Public License v3.0 — see [LICENSE](LICENSE).

The table of Saturday/Sunday ordinary readings in `calendar/lectionary_weekend.go`
is generated from A. A. Andreev's tables as encoded in the
[Ponomar](https://github.com/typiconman/ponomar) project (GPLv3).
