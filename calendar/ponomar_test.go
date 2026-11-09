package calendar

// Cross-checks against the Ponomar project (https://github.com/typiconman/ponomar),
// which encodes A. A. Andreev's tables of ordinary readings for all 65
// combinations of (key of the year, key of the next year).
//
// These tests run only when PONOMAR_DIR points at the "Ponomar" directory of
// a Ponomar checkout (the one containing scripts/ and languages/):
//
//	PONOMAR_DIR=~/projects/ponomar/Ponomar go test ./calendar -run Ponomar
//
// To regenerate lectionary_weekend.go from the tables:
//
//	PONOMAR_DIR=... go test ./calendar -run TestPonomarGenerateWeekend -update

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "regenerate lectionary_weekend.go from Ponomar data")

type andreevRow struct {
	date    JDate
	ap, gos int // 0, 0 when no ordinary reading
}

type andreevTable struct {
	file string
	year int // year whose Pascha opens the cycle
	rows []andreevRow
}

func ponomarDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("PONOMAR_DIR")
	if dir == "" {
		t.Skip("PONOMAR_DIR not set")
	}
	return dir
}

var yearRe = regexp.MustCompile(`for the year (\d+)`)

func loadAndreevTables(t *testing.T, dir string) []andreevTable {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "scripts", "Perl", "data", "[0-9]*.tsv"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no Andreev tables under %s: %v", dir, err)
	}
	var tables []andreevTable
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		tab := andreevTable{file: filepath.Base(f)}
		sc := bufio.NewScanner(bytes.NewReader(data))
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			if strings.HasPrefix(line, ";;") {
				if m := yearRe.FindStringSubmatch(line); m != nil {
					tab.year, _ = strconv.Atoi(m[1])
				}
				continue
			}
			v := strings.Split(line, "\t")
			if len(v) < 5 {
				continue
			}
			if tab.year == 0 {
				t.Fatalf("%s: data before year header", f)
			}
			mon, _ := strconv.Atoi(v[0])
			day, _ := strconv.Atoi(v[1])
			y := tab.year
			if mon < 3 {
				y++
			}
			row := andreevRow{date: MustJDate(y, time.Month(mon), day)}
			if len(v) >= 7 && v[5] != "" && v[6] != "" {
				row.ap, _ = strconv.Atoi(v[5])
				row.gos, _ = strconv.Atoi(v[6])
			}
			tab.rows = append(tab.rows, row)
		}
		tables = append(tables, tab)
	}
	if len(tables) != 65 {
		t.Fatalf("expected 65 tables, got %d", len(tables))
	}
	return tables
}

// TestPonomarLections compares NewDayContext with every row of every table.
func TestPonomarLections(t *testing.T) {
	tables := loadAndreevTables(t, ponomarDir(t))
	pairs := map[[2]int]bool{}
	for _, tab := range tables {
		cyc := newCycleInfo(tab.year)
		pairs[[2]int{cyc.key, cyc.publicanWeek}] = true
		for _, r := range tab.rows {
			got := NewDayContext(r.date).Lections
			if r.ap == 0 {
				if !got.Omitted {
					t.Errorf("%s %v: want omitted, got %+v", tab.file, r.date, got)
				}
				continue
			}
			if got.Omitted || got.Apostle.Week != r.ap || got.Gospel.Week != r.gos {
				t.Errorf("%s %v (%v): got ap=%d gos=%d omitted=%v, want ap=%d gos=%d",
					tab.file, r.date, r.date.Weekday(), got.Apostle.Week, got.Gospel.Week, got.Omitted, r.ap, r.gos)
			}
		}
	}
	if len(pairs) != 65 {
		t.Errorf("tables cover %d (key, Publican week) shapes, want 65", len(pairs))
	}
}

// TestPonomarWeekdaysFollowFormula guards the claim that Andreev's weekday
// readings need no table at all.
func TestPonomarWeekdaysFollowFormula(t *testing.T) {
	tables := loadAndreevTables(t, ponomarDir(t))
	for _, tab := range tables {
		cyc := newCycleInfo(tab.year)
		for _, r := range tab.rows {
			wd := r.date.Weekday()
			if wd == time.Saturday || wd == time.Sunday {
				continue
			}
			wk := lectionaryWeek(r.date, cyc.pentecost)
			ap := apostleWeek(wk, cyc.publicanWeek)
			gos := gospelWeek(wk, cyc.jumpWeek, cyc.publicanWeek, false)
			if r.ap != ap || r.gos != gos {
				t.Errorf("%s %v: formula ap=%d gos=%d, table ap=%d gos=%d", tab.file, r.date, ap, gos, r.ap, r.gos)
			}
		}
	}
}

// TestPonomarGenerateWeekend regenerates lectionary_weekend.go with -update,
// and otherwise checks that the committed file is up to date.
func TestPonomarGenerateWeekend(t *testing.T) {
	tables := loadAndreevTables(t, ponomarDir(t))
	type entry struct {
		key  [2]int
		year int
		ovs  []weekendOverride
	}
	var entries []entry
	for _, tab := range tables {
		cyc := newCycleInfo(tab.year)
		e := entry{key: [2]int{cyc.key, cyc.publicanWeek}, year: tab.year}
		for _, r := range tab.rows {
			wd := r.date.Weekday()
			if wd != time.Saturday && wd != time.Sunday {
				continue
			}
			wk := lectionaryWeek(r.date, cyc.pentecost)
			ap := apostleWeek(wk, cyc.publicanWeek)
			gos := gospelWeek(wk, cyc.jumpWeek, cyc.publicanWeek, true)
			if r.ap == ap && r.gos == gos {
				continue
			}
			_, m, d := r.date.Date()
			e.ovs = append(e.ovs, weekendOverride{md: uint16(int(m)*100 + d), ap: int8(r.ap), gos: int8(r.gos)})
		}
		entries = append(entries, e)
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if a.key[0] != b.key[0] {
			return a.key[0] - b.key[0]
		}
		return a.key[1] - b.key[1]
	})

	var buf bytes.Buffer
	buf.WriteString(`// Code generated by TestPonomarGenerateWeekend; DO NOT EDIT.
//
// Source: A. A. Andreev's tables of ordinary readings as encoded in the Ponomar
// project (scripts/Perl/data/*.tsv). Only the Saturday and Sunday entries that
// differ from apostleWeek/gospelWeek are kept; weekdays follow the formula
// exactly.

package calendar

// weekendOverrides is keyed by {key of the cycle's year, number of the Sunday
// after Pentecost that is the Publican and Pharisee}. These two values fix the
// whole layout from Pentecost to the Publican; a (key, next key) pair does not,
// because whether the next year is leap also moves the Publican.
// Entries are {month*100+day (Julian), Apostle week, Gospel week}; 0, 0 means
// no ordinary reading on that day.
var weekendOverrides = map[[2]uint8][]weekendOverride{
`)
	total := 0
	for _, e := range entries {
		k1, _ := KeyLetter(e.key[0])
		k2, _ := KeyLetter(KeyOfBoundaries(e.year + 1))
		fmt.Fprintf(&buf, "\t{%d, %d}: { // key %s, Publican = Sunday %d after Pentecost; e.g. %d (%s→%s)\n",
			e.key[0], e.key[1], k1, e.key[1], e.year, k1, k2)
		for _, o := range e.ovs {
			fmt.Fprintf(&buf, "\t\t{%d, %d, %d},\n", o.md, o.ap, o.gos)
		}
		buf.WriteString("\t},\n")
		total += len(e.ovs)
	}
	buf.WriteString("}\n")
	src, err := format.Source(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d cycle shapes, %d weekend overrides", len(entries), total)

	const out = "lectionary_weekend.go"
	if *update {
		if err := os.WriteFile(out, src, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	cur, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cur, src) {
		t.Errorf("%s is out of date; rerun with -update", out)
	}
}

// TestPonomarTones compares Tone with the tones in Ponomar's Pentecostarion
// files (languages/xml/pentecostarion/NN.xml is day NN−1 after Pascha).
func TestPonomarTones(t *testing.T) {
	dir := ponomarDir(t)
	toneRe := regexp.MustCompile(`Tone="(\d)"`)
	files, _ := filepath.Glob(filepath.Join(dir, "languages", "xml", "pentecostarion", "*.xml"))
	checked := 0
	for _, f := range files {
		n, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(f), ".xml"))
		if err != nil {
			continue
		}
		data, _ := os.ReadFile(f)
		m := toneRe.FindSubmatch(data)
		if m == nil {
			continue
		}
		want, _ := strconv.Atoi(string(m[1]))
		for year := 1900; year <= 2100; year += 7 {
			d := Pascha(year).AddDays(n - 1)
			if !d.Before(Pascha(year + 1).AddDays(PublicanOffset)) {
				continue
			}
			if got := Tone(d); got != want {
				t.Errorf("Pascha %d + %d days: tone %d, Ponomar %d", year, n-1, got, want)
			}
		}
		checked++
	}
	if checked < 200 {
		t.Errorf("only %d Pentecostarion files with tones found", checked)
	}
}
