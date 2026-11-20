package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The parts of a festal (polyeleos, vigil, feast) matins that the Menaion
// gives outside the canon. Keys under me.<MM>.<DD>.matins:
//
//	k_sedalen.<K>.<N>, k_sedalen.<K>.glory, k_sedalen.<K>.theotokion
//	    sessional hymns after the 1st, 2nd (and 3rd) kathisma; K = p after
//	    the polyeleos
//	magnification   the величание
//	rubric          marker: the prokeimenon and Gospel of matins (Meta)
//	ps50.sticheron  the sticheron after Psalm 50
//	praises.on.<N>, praises.sticheron.<N>, praises.glory, praises.now
var (
	reFMStart     = regexp.MustCompile(`^на утрени`)
	reFMSedalen   = regexp.MustCompile(`^по (\S+?)-?[ймя]?,? (стихословии|кафисме|стихологии),? седал`)
	reFMPolySed   = regexp.MustCompile(`^по полиеле[ие]+,? седал`)
	reFMMagnif    = regexp.MustCompile(`^величание`)
	reFMPs50      = regexp.MustCompile(`^по (н-м|н) псалме`)
	reFMPraises   = regexp.MustCompile(`^на хвалитех`)
	reFMPraisesOn = regexp.MustCompile(`на ([дзи])[,.]`)
	reFMRubric    = regexp.MustCompile(`^(таже,? )?(степенна|прокимен)`)
	reFMEnd       = regexp.MustCompile(`^(славословие|на стиховне|на литургии|на 1-м часе|на а-м часе|месяца|в тойже день)`)
)

func parseMenaionFeastMatins(d *Doc, prefix, path string) []corpus.Unit {
	var units []corpus.Unit
	seen := map[string]bool{}
	add := func(key, kind, heading, body string, meta map[string]any) {
		body = CleanText(body)
		if body == "" || seen[key] {
			return
		}
		seen[key] = true
		m := map[string]any{"heading": CleanText(heading), "file": filepath.Base(path)}
		for k, v := range meta {
			m[k] = v
		}
		units = append(units, corpus.Unit{ID: prefix + ".matins." + key, Book: "menaion", Kind: kind, Tone: toneOf(heading),
			Podoben: podobenOf(heading), Lang: "cu", Body: body, SourceID: "orthlib.mineya", Meta: m})
	}
	phase := "" // pre (before the canon) | praises
	prevPlain := ""
	rubric, rubricParas := "", 0
	section, expect, heading := "", "", ""
	n := map[string]int{}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		red := hip.PlainString(p.RedPrefix())
		isHead := len([]rune(strings.TrimSpace(red))) > 3
		// The rubric with the prokeimenon and the Gospel is black text.
		// It may continue over the next paragraphs («прокимен, глас д:» / the
		// text / the verse / «всякое дыхание…, евангелие…»).
		hasGospel := strings.Contains(plain, "еиглие") || strings.Contains(plain, "евангелие")
		// The Theophany's matins has no heading «на утрени»: it begins with
		// the sessional hymn after the first kathisma.
		if phase == "" && isHead && reFMSedalen.MatchString(plain) && strings.Contains(plain, "а-й") &&
			strings.Contains(prevPlain, "по шестопсалмии") {
			phase = "pre"
		}
		prevPlain = plain
		// After Psalm 50 at the Nativity and the Theophany: «Слава»
		// «Всяческая днесь…» instead of «Молитвами апостол», «И ныне» the
		// same (or its own), then the sticheron.
		if phase == "pre" {
			switch {
			case strings.HasPrefix(plain, "всяческая днесь"):
				if !seen["ps50.glory"] {
					add("ps50.glory", "hymn", "", text, nil)
				} else {
					add("ps50.now", "hymn", "", text, nil)
				}
				continue
			case seen["ps50.glory"] && strings.HasPrefix(plain, "и ныне") && strings.Contains(plain, "тойже"):
				add("ps50.now", "hymn", "", "-", map[string]any{"copy": prefix + ".matins.ps50.glory"}) // the text: the copy
				continue
			case seen["ps50.glory"] && !seen["ps50.sticheron"] && strings.Contains(plain, "стихира") &&
				len(strings.Fields(plain)) <= 12:
				section, expect, heading = "ps50", "sticheron", text
				continue
			}
		}
		if phase == "pre" && rubric == "" && (reFMRubric.MatchString(plain) || strings.Contains(plain, "степенна")) && strings.Contains(plain, "прокимен") &&
			!strings.HasPrefix(plain, "на литургии") {
			if hasGospel {
				parseMatinsRubric(text, plain, add)
			} else {
				rubric, rubricParas = text, 1
			}
			continue
		}
		if rubric != "" && phase == "pre" {
			rubric += " " + text
			rubricParas++
			if hasGospel || rubricParas > 5 {
				parseMatinsRubric(rubric, strings.ToLower(hip.PlainString(rubric)), add)
				rubric = "done"
			}
			continue
		}
		if isHead {
			switch {
			case phase == "" && reFMStart.MatchString(plain):
				phase = "pre"
				continue
			case phase == "pre" && reMCanon.MatchString(plain):
				phase, section = "canon", ""
				continue
			case phase != "" && reFMPraises.MatchString(plain):
				phase, section, expect, heading = "praises", "praises", "sticheron", text
				if m := reFMPraisesOn.FindStringSubmatch(text); m != nil {
					if on := toneLetters[[]rune(m[1])[0]]; on > 0 {
						add(fmt.Sprintf("praises.on.%d", on), "marker", "", text, nil)
					}
				} else if w := strings.Fields(text); len(w) > 0 {
					// «на хвалитех стихиры, на д҃» — numerals in Unicode.
					for i := range w {
						if hip.PlainString(w[i]) == "на" && i+1 < len(w) {
							if on := numeralOfWord(strings.TrimRight(w[i+1], ",.:")); on >= 4 && on <= 8 {
								add(fmt.Sprintf("praises.on.%d", on), "marker", "", text, nil)
								break
							}
						}
					}
				}
				continue
			case phase == "praises" && reFMEnd.MatchString(plain):
				return units
			case phase == "pre" && reFMPolySed.MatchString(plain):
				section, expect, heading = "k_sedalen.p", "sedalen", text
				continue
			case phase == "pre" && reFMSedalen.MatchString(plain):
				w := strings.Fields(text)
				k := 0
				if len(w) > 1 {
					k = numeralOfWord(w[1])
				}
				if k < 1 || k > 3 {
					k = 1
				}
				section, expect, heading = fmt.Sprintf("k_sedalen.%d", k), "sedalen", text
				continue
			case phase == "pre" && reFMMagnif.MatchString(plain):
				section, expect, heading = "magnification", "magnification", text
				if _, inline := splitHeading(p, text, true); inline != "" {
					add("magnification", "hymn", text, inline, nil)
					expect = ""
				}
				continue
			case phase == "pre" && reFMPs50.MatchString(plain):
				section, expect, heading = "ps50", "sticheron", text
				continue
			case section == "":
				continue
			case reGVTwice.MatchString(plain):
				if section == "praises" && n[section] > 0 {
					last := prefix + fmt.Sprintf(".matins.praises.sticheron.%d", n[section])
					for _, u := range units {
						if u.ID == last {
							n[section]++
							add(fmt.Sprintf("praises.sticheron.%d", n[section]), "sticheron", heading, u.Body, map[string]any{"repeat_of": last})
						}
					}
				}
				continue
			case reGlory.MatchString(plain) && strings.Contains(plain[:min(len(plain), 30)], "и ныне"):
				expect, heading = "glory_now", text
			case reGlory.MatchString(plain):
				expect, heading = "glory", text
			case reAndNow.MatchString(plain) || reMTheot.MatchString(plain):
				expect, heading = "now", text
				if reMCross.MatchString(plain) && !strings.Contains(plain, "или") {
					expect = "cross_now"
				}
			default:
				if section == "praises" {
					expect, heading = "sticheron", text // «Ины стихиры…», «глас д:»
				}
				continue
			}
			if _, inline := splitHeading(p, text, true); inline == "" {
				continue
			} else {
				text = inline
			}
		}
		if phase != "pre" && phase != "praises" || section == "" || expect == "" {
			continue
		}
		switch {
		case section == "magnification":
			add("magnification", "hymn", heading, text, nil)
			expect = ""
		case section == "ps50":
			add("ps50.sticheron", "sticheron", heading, text, nil)
			expect = ""
		case strings.HasPrefix(section, "k_sedalen"):
			switch expect {
			case "sedalen":
				n[section]++
				add(fmt.Sprintf("%s.%d", section, n[section]), "sedalen", heading, text, nil)
			case "glory":
				add(section+".glory", "sedalen", heading, text, nil)
				expect = ""
			case "now", "glory_now":
				add(section+".theotokion", "theotokion", heading, text, nil)
				expect = ""
			case "cross_now":
				add(section+".cross_theotokion", "theotokion", heading, text, nil)
				expect = ""
			}
		case section == "praises":
			switch expect {
			case "sticheron":
				n[section]++
				add(fmt.Sprintf("praises.sticheron.%d", n[section]), "sticheron", heading, text, nil)
			case "glory":
				add("praises.glory", "sticheron", heading, text, nil)
				expect = ""
			case "now", "glory_now", "cross_now":
				kind := "sticheron"
				if strings.Contains(hip.PlainString(heading), "бгородичен") {
					kind = "theotokion"
				}
				add("praises.now", kind, heading, text, map[string]any{"glory_now": expect == "glory_now"})
				expect = ""
			}
		}
	}
	return units
}

// parseMatinsRubric: «Степенна, а-й антифон д-го гласа. Прокимен, глас д:
// … Стих: … Всякое дыхание: Евангелие от Иоанна, зачало лз.»
func parseMatinsRubric(text, plain string, add func(key, kind, heading, body string, meta map[string]any)) {
	_, pos := hip.Plain(text)
	meta := map[string]any{}
	if m := reLitProk.FindStringSubmatchIndex(plain); m != nil {
		meta["prok_tone"] = 4 // the prokeimena of matins are in tone 4 unless stated
		if m[2] >= 0 {
			meta["prok_tone"] = uniNumeral(text, pos, plain, m[2], m[3])
		}
		meta["prok"] = uni(text, pos, plain, m[4], m[5])
		meta["prok_verse"] = uni(text, pos, plain, m[6], m[7])
	}
	if m := reLitGospel.FindStringSubmatchIndex(plain); m != nil {
		meta["gospel_book"] = gospelByName[plain[m[2]:m[3]]]
		meta["gospel"] = uniNumeral(text, pos, plain, m[4], m[5])
		meta["gospel_half"] = m[6] >= 0
	}
	if len(meta) > 0 {
		add("rubric", "marker", "", text, meta)
	}
}
