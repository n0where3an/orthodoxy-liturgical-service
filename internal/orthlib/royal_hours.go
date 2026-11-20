package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The Royal Hours of the eves of the Nativity (12.24) and the Theophany
// (01.05), printed in full in the Menaion («последование часов, певаемых в
// навечерии ржства хрстова»). Units under rh.<feast>:
//
//	kontakion                      the forefeast's, at every hour
//	<H>.psalm.<1..3>               the hour's psalms (H: 1, 3, 6, 9)
//	<H>.troparion                  «Слава» — the forefeast's troparion
//	<H>.theotokion                 «И ныне» — the hour's theotokion
//	<H>.hymn.<N>, .verse.<N>,      the stichera and their verses, in order
//	<H>.canonarch.<N>              (one numbering); «Слава», «И ныне» and the
//	                               repetitions («паки тойже») are hymns that
//	                               copy their texts; the canonarch reads the
//	                               9th hour's sticheron before it is sung
//	<H>.prokeimenon, .prokeimenon.verse
//	<H>.reading.<1..3>, .reading.<N>.title  the prophecy, the Apostle and
//	                               the Gospel with their announcements
var royalHoursFiles = map[string]string{
	"nativity":  "mineya/dec/dec/Декабрь/24.hip",
	"theophany": "mineya/jan/jan/Январь/05.hip",
}

var (
	reRHStart     = regexp.MustCompile(`^последование часов`)
	reRHHour      = regexp.MustCompile(`^(?:таже,? )?час (третий|шестый|девятый)\.?$`)
	reRHPsalm     = regexp.MustCompile(`(?:^|\s)псалом \S+?\.?$`)
	reRHVerseOn   = regexp.MustCompile(`^таже,? стих[.,]|припев:$`) // «таже стих. глас д:», «вторый припев:» — the verse follows
	reRHGospel    = regexp.MustCompile(`^еиглие от (матфеа|марка|луки|иоанна)`)
	reRHEnd       = regexp.MustCompile(`^посем псалом рв|^посем псалом рв\.`)
	reRHProk      = regexp.MustCompile(`прокимен,? глас (\S+?)[.:]\s*(.+?)\s*стих:\s*(.+)$`)
	reRHVerse     = regexp.MustCompile(`^(?:посем |таже,? |таже стих[.,]? )?стих:\s*`)
	reRHReading   = regexp.MustCompile(`^\[(?:глава|зач|варух)[^\]]*\]\s*`)
	reRHTitle     = regexp.MustCompile(`чтение\.?$|^еиглие от \S+, зачало`)
	reBracketHead = regexp.MustCompile(`^\[[^\]]*\]\s*`)
	reRHRepeat    = regexp.MustCompile(`паки тойж?д?е|вторый лик,? тойже|^и ныне,? (паки )?тойже`)
)

// ImportRoyalHours parses the Royal Hours of both eves.
func ImportRoyalHours(root string) ([]corpus.Unit, []string) {
	var units []corpus.Unit
	var warnings []string
	for feast, file := range royalHoursFiles {
		d, err := LoadDoc(filepath.Join(root, file))
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		us, err := parseRoyalHours(d, feast)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", file, err))
		}
		units = append(units, us...)
	}
	return units, warnings
}

func parseRoyalHours(d *Doc, feast string) ([]corpus.Unit, error) {
	prefix := "rh." + feast + "."
	var units []corpus.Unit
	seen := map[string]bool{}
	add := func(key, kind, body string, meta map[string]any) string {
		id := prefix + key
		if seen[id] {
			return id
		}
		seen[id] = true
		units = append(units, corpus.Unit{ID: id, Book: "menaion", Kind: kind, Lang: "cu", Body: CleanText(body),
			SourceID: "orthlib.mineya", Meta: meta})
		return id
	}
	hour := 0
	expect := ""  // psalm | troparion | theotokion | kontakion | reading
	section := "" // psalms | hymns | readings
	nPsalm, nHymn, nRead := 0, 0, 0
	lastHymn := "" // the last sticheron, for its repetitions
	pendingTitle, pendingTitle0 := "", ""
	skip := false // the canonarch's proclamation and the polychronion (9th hour)
	verseNext, canonarch := false, false
	nextHour := map[int]int{1: 3, 3: 6, 6: 9}
	hk := func(k string) string { return fmt.Sprintf("%d.%s", hour, k) }
	marker := func(ref string) {
		nHymn++
		add(hk(fmt.Sprintf("hymn.%d", nHymn)), "marker", "", map[string]any{"copy": ref})
	}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.TrimSpace(strings.ToLower(hip.PlainString(text)))
		if plain == "" {
			continue
		}
		if hour == 0 {
			if reRHStart.MatchString(plain) {
				hour, section = 1, "psalms"
			}
			continue
		}
		if m := reRHHour.FindStringSubmatch(plain); m != nil {
			hour = map[string]int{"третий": 3, "шестый": 6, "девятый": 9}[m[1]]
			section, nPsalm, nHymn, nRead, lastHymn, skip = "psalms", 0, 0, 0, "", false
			continue
		}
		if hour == 9 && reRHEnd.MatchString(plain) {
			break // the typika
		}
		// The next hour without its heading (the Nativity's 6th): its psalms
		// after the readings.
		if section == "readings" && reRHPsalm.MatchString(plain) && len(strings.Fields(plain)) <= 10 && hour < 9 {
			hour = nextHour[hour]
			section, nPsalm, nHymn, nRead, lastHymn, skip = "psalms", 0, 0, 0, "", false
		}
		if expect != "" {
			switch expect {
			case "psalm":
				nPsalm++
				add(hk(fmt.Sprintf("psalm.%d", nPsalm)), "psalm", text, nil)
			case "troparion":
				add(hk("troparion"), "troparion", text, nil)
			case "theotokion":
				add(hk("theotokion"), "theotokion", text, nil)
				section = "hymns"
			case "kontakion":
				add("kontakion", "kontakion", text, nil)
			}
			expect = ""
			continue
		}
		switch {
		case reRHPsalm.MatchString(plain) && section == "psalms" && len(strings.Fields(plain)) <= 10:
			expect = "psalm"
			continue
		case strings.HasPrefix(plain, "слава, тропарь"):
			expect = "troparion"
			continue
		case strings.HasPrefix(plain, "и ныне, бгородичен"):
			expect = "theotokion"
			continue
		case strings.Contains(plain, "кондак, глас") && strings.HasSuffix(plain, ":") || strings.Contains(plain, "кондак, глас") && strings.HasSuffix(plain, "."):
			expect = "kontakion"
			continue
		}
		if section == "hymns" {
			if m := reRHProk.FindStringSubmatchIndex(plain); m != nil {
				_, pos := hip.Plain(text)
				tone := uniNumeral(text, pos, plain, m[2], m[3])
				add(hk("prokeimenon"), "prokeimenon", strings.TrimRight(uni(text, pos, plain, m[4], m[5]), ". "), map[string]any{"tone": tone})
				units[len(units)-1].Tone = tone
				add(hk("prokeimenon.verse"), "verse", uni(text, pos, plain, m[6], m[7]), nil)
				section = "readings"
				continue
			}
			// The canonarch reads the sticheron, the deacon the polychronion
			// (left out), then «оба лика вкупе поют предреченную стихиру».
			if strings.Contains(plain, "канонарх") {
				canonarch = true
				continue
			}
			if skip {
				if strings.Contains(plain, "оба лика вкупе") && lastHymn != "" {
					skip = false
					marker(lastHymn)
				}
				continue
			}
			if canonarch && len(strings.Fields(plain)) > 12 {
				canonarch, skip = false, true
				nHymn++
				lastHymn = add(hk(fmt.Sprintf("canonarch.%d", nHymn)), "sticheron", text, map[string]any{"canonarch": true})
				continue
			}
			words := len(strings.Fields(plain))
			if verseNext {
				verseNext = false
				nHymn++
				v := text
				if strings.HasPrefix(plain, "стих:") {
					v = strings.TrimSpace(v[strings.Index(v, ":")+1:])
				}
				add(hk(fmt.Sprintf("verse.%d", nHymn)), "verse", v, nil)
				continue
			}
			switch {
			case reRHVerseOn.MatchString(plain):
				verseNext = true
			case reRHVerse.MatchString(plain):
				nHymn++
				add(hk(fmt.Sprintf("verse.%d", nHymn)), "verse", strings.TrimSpace(text[strings.Index(text, ":")+1:]), nil)
			case reRHRepeat.MatchString(plain):
				if strings.HasPrefix(plain, "и ныне") {
					marker("common.and_now")
				}
				if lastHymn != "" {
					marker(lastHymn)
				}
				if strings.Contains(plain, "слава") && !strings.HasPrefix(plain, "и ныне") {
					marker("common.glory")
				}
			case strings.HasPrefix(plain, "дважды"):
				if lastHymn != "" {
					marker(lastHymn)
				}
			case strings.HasPrefix(plain, "слава, и ныне") && words <= 8:
				marker("common.glory_now")
			case strings.HasPrefix(plain, "слава") && words <= 8:
				marker("common.glory")
			case strings.HasPrefix(plain, "и ныне") && words <= 8:
				marker("common.and_now")
			case words > 12 && !strings.HasPrefix(plain, "таже") && !strings.HasPrefix(plain, "посем"):
				nHymn++
				lastHymn = add(hk(fmt.Sprintf("hymn.%d", nHymn)), "sticheron", text, nil)
			}
			continue
		}
		if section == "readings" {
			switch {
			case reRHReading.MatchString(plain) || pendingTitle != "" && len(strings.Fields(plain)) > 12:
				nRead++
				add(hk(fmt.Sprintf("reading.%d", nRead)), "reading", reBracketHead.ReplaceAllString(text, ""), nil)
				if m := reRHGospel.FindStringSubmatch(pendingTitle0); m != nil {
					// «еиглие от марка, зачало а.»: the Gospel's announcement.
					add(hk(fmt.Sprintf("reading.%d.title", nRead)), "announcement", "",
						map[string]any{"copy": fmt.Sprintf("lect.zach.%s.1.announce", gospelByName[m[1]])})
				} else if pendingTitle != "" {
					add(hk(fmt.Sprintf("reading.%d.title", nRead)), "announcement", pendingTitle, nil)
				}
				pendingTitle0 = ""
				pendingTitle = ""
			case reRHTitle.MatchString(plain):
				t := text
				if i := strings.Index(t, ":"); i >= 0 && strings.HasPrefix(plain, "иерей") {
					t = strings.TrimSpace(t[i+1:])
				}
				// «Таже, паремиа. Пророчества…», «Таже апсл, к галатом…».
				if strings.HasPrefix(plain, "таже") {
					if i := strings.Index(t, ". "); i >= 0 && i < len(t)/2 {
						t = strings.TrimSpace(t[i+2:])
					} else if i := strings.Index(t, ", "); i >= 0 {
						t = strings.TrimSpace(t[i+2:])
					}
					if r := []rune(t); len(r) > 0 {
						t = strings.ToUpper(string(r[0])) + string(r[1:])
					}
				}
				pendingTitle, pendingTitle0 = t, plain
			}
		}
	}
	for _, h := range []int{1, 3, 6, 9} {
		for _, k := range []string{"psalm.3", "troparion", "theotokion", "hymn.1", "prokeimenon", "reading.3", "reading.3.title"} {
			if !seen[fmt.Sprintf("%s%d.%s", prefix, h, k)] {
				return units, fmt.Errorf("royal hour %d: no %s", h, k)
			}
		}
	}
	if !seen[prefix+"kontakion"] {
		return units, fmt.Errorf("no kontakion")
	}
	return units, nil
}
