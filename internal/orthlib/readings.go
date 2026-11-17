package orthlib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

// LectionarySource describes the ordinary readings.
var LectionarySource = corpus.Source{
	ID: "orthlib.lectionary", Title: "Апостол и Евангелие (рядовые чтения)",
	Edition: "orthlib.ru HIP; table of readings from the Ponomar project (lectionary.tsv)",
	URL:     "https://orthlib.ru",
	Note:    "Readings assembled by verse ranges; the liturgical beginnings (зачала) as marked in the source.",
}

// bookFiles maps the lectionary's book codes to orthlib files.
var bookFiles = map[string]string{
	"Acts": "apostol/apostol/05deyan.hip", "Jas": "apostol/apostol/06iak.hip",
	"I Pet": "apostol/apostol/07pet1.hip", "II Pet": "apostol/apostol/08pet2.hip",
	"I Jn": "apostol/apostol/09in1.hip", "II Jn": "apostol/apostol/10in2.hip",
	"III Jn": "apostol/apostol/11in3.hip", "Jude": "apostol/apostol/12iud.hip",
	"Rom": "apostol/apostol/13rim.hip", "I Cor": "apostol/apostol/14kor1.hip",
	"II Cor": "apostol/apostol/15kor2.hip", "Gal": "apostol/apostol/16gal.hip",
	"Eph": "apostol/apostol/17ef.hip", "Phil": "apostol/apostol/18flp.hip",
	"Col": "apostol/apostol/19kol.hip", "I Thess": "apostol/apostol/20sol1.hip",
	"II Thess": "apostol/apostol/21sol2.hip", "I Tim": "apostol/apostol/22tim1.hip",
	"II Tim": "apostol/apostol/23tim2.hip", "Tit": "apostol/apostol/24tit.hip",
	"Philem": "apostol/apostol/25flm.hip", "Philip": "apostol/apostol/18flp.hip", "Heb": "apostol/apostol/26evr.hip",
	"Mt": "evangelie/ev/Евангелие/1mf.hip", "Mk": "evangelie/ev/Евангелие/2mk.hip",
	"Lk": "evangelie/ev/Евангелие/3lk.hip", "Jn": "evangelie/ev/Евангелие/4in.hip",
}

// Reading announcements («К Римляном послания святаго апостола Павла
// чтение», «От Матфея святаго Евангелия чтение»), in HIP.
var announcements = map[string]string{
	"Acts":     "Дjья'нiй ст~ы'хъ а=п\\слъ чте'нiе.",
	"Jas":      "Собо'рнагw посла'нiя i=а'кwвля чте'нiе.",
	"I Pet":    "Собо'рнагw посла'нiя петро'ва чте'нiе.",
	"II Pet":   "Собо'рнагw посла'нiя петро'ва чте'нiе.",
	"I Jn":     "Собо'рнагw посла'нiя i=wа'ннова чте'нiе.",
	"II Jn":    "Собо'рнагw посла'нiя i=wа'ннова чте'нiе.",
	"III Jn":   "Собо'рнагw посла'нiя i=wа'ннова чте'нiе.",
	"Jude":     "Собо'рнагw посла'нiя i=у'динa чте'нiе.",
	"Rom":      "Къ ри'мляномъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"I Cor":    "Къ корi'нfяномъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"II Cor":   "Къ корi'нfяномъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Gal":      "Къ гала'тwмъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Eph":      "Къ _е=фесе'мъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Phil":     "Къ фiлипписi'мъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Philip":   "Къ фiлипписi'мъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Col":      "Къ коло'ссаемъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"I Thess":  "Къ солу'няномъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"II Thess": "Къ солу'няномъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"I Tim":    "Къ тiмоfе'ю посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"II Tim":   "Къ тiмоfе'ю посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Tit":      "Къ ти'ту посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Philem":   "Къ филимо'ну посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Heb":      "Къ _е=vре'емъ посла'нiя ст~а'гw а=п\\сла па'vла чте'нiе.",
	"Mt":       "W\\т матfе'а ст~а'гw _е=v\\гл~iа чте'нiе.",
	"Mk":       "W\\т ма'рка ст~а'гw _е=v\\гл~iа чте'нiе.",
	"Lk":       "W\\т луки` ст~а'гw _е=v\\гл~iа чте'нiе.",
	"Jn":       "W\\т i=wа'нна ст~а'гw _е=v\\гл~iа чте'нiе.",
}

// ImportReadings builds the ordinary (рядовые) Apostle and Gospel readings
// of the lectionary table (Ponomar, scripts/Perl/data/lectionary.tsv: effweek,
// weekday, epistle, pericope, gospel, pericope) from the orthlib Apostol and
// Gospel: units lect.ap.<week>.<weekday> and lect.gos.<week>.<weekday>, and
// their announcements (<key>.announce).
func ImportReadings(root, lectionaryTSV string) ([]corpus.Unit, []string, error) {
	f, err := os.Open(lectionaryTSV)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	books := map[string]*Book{}
	book := func(code string) (*Book, error) {
		if b, ok := books[code]; ok {
			return b, nil
		}
		file, ok := bookFiles[code]
		if !ok {
			return nil, fmt.Errorf("unknown book %q", code)
		}
		b, err := ParseBook(filepath.Join(root, file))
		if err != nil {
			return nil, err
		}
		books[code] = b
		return b, nil
	}
	var units []corpus.Unit
	var warnings []string
	add := func(prefix, ref, pericope, week, dow string) {
		if strings.TrimSpace(ref) == "" {
			return // no reading (e.g. Lenten weekdays)
		}
		code, ranges, err := ParseRef(ref)
		if err != nil {
			warnings = append(warnings, err.Error())
			return
		}
		b, err := book(code)
		if err != nil {
			warnings = append(warnings, err.Error())
			return
		}
		text, err := b.Reading(ranges)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s.%s.%s (%s): %v", prefix, week, dow, ref, err))
			return
		}
		key := fmt.Sprintf("%s.%s.%s", prefix, week, dow)
		zach := strings.Fields(pericope)
		meta := map[string]any{"ref": ref, "book": code}
		if len(zach) > 0 {
			if n, err := strconv.Atoi(zach[0]); err == nil {
				meta["zachalo"] = n
			}
		}
		units = append(units, corpus.Unit{ID: key, Book: "lectionary", Kind: "reading", Lang: "cu", Body: text,
			SourceID: "orthlib.lectionary", Meta: meta})
		units = append(units, corpus.Unit{ID: key + ".announce", Book: "lectionary", Kind: "announcement", Lang: "cu",
			Body: hipAnnouncement(code), SourceID: "orthlib.lectionary"})
	}
	// The rows run Monday..Saturday, Sunday for each week. The week column
	// has misprints (two runs of week 3 and 4 rows are numbered one week
	// too low), so the week is counted from the order of the rows.
	sc := bufio.NewScanner(f)
	week, prevPos := 0, -1
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, ";;") || strings.TrimSpace(line) == "" {
			continue
		}
		v := strings.Split(line, "\t")
		if len(v) < 6 {
			continue
		}
		col, err1 := strconv.Atoi(v[0])
		dow, err2 := strconv.Atoi(v[1])
		if err1 != nil || err2 != nil || dow < 0 || dow > 6 {
			warnings = append(warnings, fmt.Sprintf("lectionary: bad row %q", line))
			continue
		}
		pos := (dow + 6) % 7 // Monday 0 … Sunday 6
		switch {
		case prevPos < 0:
			week = col
		case pos <= prevPos:
			week++
		}
		prevPos = pos
		if col != week {
			warnings = append(warnings, fmt.Sprintf("lectionary: row %q counted as week %d", line, week))
		}
		w, d := strconv.Itoa(week), strconv.Itoa(dow)
		add("lect.ap", v[2], v[3], w, d)
		add("lect.gos", v[4], v[5], w, d)
	}
	return units, warnings, sc.Err()
}

func hipAnnouncement(code string) string {
	return hipToUnicode(announcements[code])
}

// apostolOrder: the books of the Apostol (one numbering of pericopes).
var apostolOrder = []string{"Acts", "Jas", "I Pet", "II Pet", "I Jn", "II Jn", "III Jn", "Jude", "Rom", "I Cor", "II Cor",
	"Gal", "Eph", "Phil", "Col", "I Thess", "II Thess", "I Tim", "II Tim", "Tit", "Philem", "Heb"}

// gospelKeys: the Gospels, each with its own numbering.
var gospelKeys = map[string]string{"Mt": "mt", "Mk": "mk", "Lk": "lk", "Jn": "jn"}

// ImportZachala makes every pericope of the Apostol and the Gospels a
// reading: lect.zach.ap.<N> (one numbering for the whole Apostol),
// lect.zach.<mt|mk|lk|jn>.<N>; «от полу» — <key>.half; with announcements
// (<key>.announce). The Menaion and the feasts cite readings this way.
func ImportZachala(root string) ([]corpus.Unit, []string, error) {
	var units []corpus.Unit
	var warnings []string
	seen := map[string]bool{}
	emit := func(code, key string, b *Book) {
		for _, z := range b.Zachala {
			id := fmt.Sprintf("lect.zach.%s.%d", key, z.N)
			if z.Half {
				id += ".half"
			}
			if seen[id] {
				continue
			}
			ranges, err := b.ZachaloRanges(z.N, z.Half)
			// The Gospel of the feasts of the Theotokos: Lk 10:38-42 and 11:27-28.
			if err == nil && code == "Lk" && z.N == 54 && !z.Half {
				ranges = append(ranges, VerseRange{FromChapter: 11, FromVerse: 27, ToChapter: 11, ToVerse: 28})
			}
			if err == nil {
				var text string
				if text, err = b.Reading(ranges); err == nil {
					seen[id] = true
					units = append(units,
						corpus.Unit{ID: id, Book: "lectionary", Kind: "reading", Lang: "cu", Body: text, SourceID: "orthlib.lectionary",
							Meta: map[string]any{"book": code, "zachalo": z.N, "half": z.Half,
								"ref": fmt.Sprintf("%s_%d:%d-%d:%d", code, ranges[0].FromChapter, ranges[0].FromVerse, ranges[0].ToChapter, ranges[0].ToVerse)}},
						corpus.Unit{ID: id + ".announce", Book: "lectionary", Kind: "announcement", Lang: "cu",
							Body: hipAnnouncement(code), SourceID: "orthlib.lectionary"})
					continue
				}
			}
			warnings = append(warnings, fmt.Sprintf("%s: %v", id, err))
		}
	}
	for _, code := range apostolOrder {
		b, err := ParseBook(filepath.Join(root, bookFiles[code]))
		if err != nil {
			return nil, nil, err
		}
		emit(code, "ap", b)
	}
	for code, key := range gospelKeys {
		b, err := ParseBook(filepath.Join(root, bookFiles[code]))
		if err != nil {
			return nil, nil, err
		}
		emit(code, key, b)
	}
	return units, warnings, nil
}
