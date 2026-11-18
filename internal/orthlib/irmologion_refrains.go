package orthlib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The refrains of the 9th ode at the feasts of the Lord and of the
// Theotokos (Ирмологий, «Припевы на 9-й песни», 24pripev.hip): «Честнейшую»
// is not sung, each troparion is preceded by a refrain of the feast. Keys
// ir.pripev.<feast>.<canon>.<key>:
//
//	irmos      the refrain sung before the irmos (and at the katavasia)
//	irmos2     the second lik's refrain to the irmos, when it differs
//	1, 2, …    the refrains of the troparia, in order (when the book gives
//	           several; otherwise the irmos's refrain is sung with each)
//	glory, now «вместо Слава», «вместо И ныне»
//
// <canon> is 1 or 2 (the second canon of the feast, or of the saint after
// the Forerunner's and the hierarchs' canon).
var refrainFeasts = []struct{ heading, id string }{
	{"ржству бцды", "nativity_of_theotokos"},
	{"на воздвижение", "exaltation"},
	{"введению престыя бцды", "entry_of_theotokos"},
	{"на ржство хрстово", "nativity"},
	{"обрезанию хрста", "circumcision"},
	{"на бгоявление", "theophany"},
	{"к тропарем прдтечевым", "synaxis_of_forerunner"},
	{"трием стителем", "three_hierarchs"},
	{"на сретение", "meeting"},
	{"на блговещение", "annunciation"},
	{"на вознесение", "ascension"},
	{"на преображение", "transfiguration"},
	{"на успение", "dormition"},
}

var (
	reRefrainRubric = regexp.MustCompile(`^(на ф-й песни|посем|таже|и на кийждо|и прочих|прочих|прочыя|и ко второму|катавасиа|ведомо|и поклон|конец|к бгородичну|егда же достигнем|и ирмос|подобне|и к прочым|у трипеснца|\[|припевы на ф-й|к каноном же стых|и вторый лик)`)
	reRefrainGlory  = regexp.MustCompile(`^вместо(,| же,)? слава:?`)
	reRefrainNow    = regexp.MustCompile(`^вместо(,| же,)? и ныне`)
	reSecondCanon   = regexp.MustCompile(`втораго канона припев|ко второму канону|канона великаго василиа`)
)

// ImportOdeNineRefrains parses the refrains of the 9th ode.
func ImportOdeNineRefrains(root string) ([]corpus.Unit, error) {
	path := filepath.Join(root, "irmologij", "irmologij", "24pripev.hip")
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	var units []corpus.Unit
	feast, canon, n := "", 0, 0
	key := "" // the next refrain's role: irmos | glory | now | "" (troparia)
	alternative := false
	seen := map[string]bool{}
	add := func(k, text string) {
		id := fmt.Sprintf("ir.pripev.%s.%d.%s", feast, canon, k)
		if seen[id] || strings.TrimSpace(text) == "" {
			return
		}
		seen[id] = true
		units = append(units, corpus.Unit{ID: id, Book: "irmologion", Kind: "refrain", Lang: "cu",
			Body: strings.TrimSpace(text), SourceID: "orthlib.irmologion", Meta: map[string]any{"file": filepath.Base(path), "feast": feast}})
	}
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := strings.ToLower(hip.PlainString(text))
		if plain == "" {
			continue
		}
		heading := false
		for _, f := range refrainFeasts {
			if strings.HasPrefix(plain, f.heading) {
				feast, canon, n, key, heading = f.id, 1, 0, "irmos", true
				break
			}
		}
		if heading || feast == "" {
			continue
		}
		if reSecondCanon.MatchString(plain) {
			canon, n, key = 2, 0, "irmos"
			if strings.Contains(plain, "ко второму канону") {
				// «И ко второму канону припевают тойже припев.»
				for _, u := range units {
					if u.ID == fmt.Sprintf("ir.pripev.%s.1.irmos", feast) {
						add("irmos", u.Body)
					}
				}
			}
		}
		switch {
		case strings.HasPrefix(plain, "вторый же лик поет припев"):
			key = "irmos2" // «Вторый же лик поет припев: …» to the irmos
			continue
		case strings.HasPrefix(plain, "ведомо да будет, яко два"):
			// «…два сущыя припевы в начале припеваем ко ирмосом»: the first
			// refrain of the troparia is the second lik's to the irmos.
			units = renumberIrmos2(units, feast, canon)
			continue
		case reRefrainGlory.MatchString(plain), reRefrainNow.MatchString(plain):
			k := "glory"
			if reRefrainNow.MatchString(plain) {
				k = "now"
			}
			// «Вместо же, И ныне, поем припев Богородице: Величай…» — inline or next.
			if i := strings.LastIndex(plain, ":"); i >= 0 && len(strings.Fields(plain[i+1:])) >= 3 {
				_, pos := hip.Plain(text)
				add(k, uni(text, pos, plain, i+1, len(plain)))
			} else {
				key = k
			}
			continue
		case reRefrainRubric.MatchString(plain):
			continue
		case strings.HasSuffix(plain, ":") && len(strings.Fields(plain)) <= 6:
			continue // the cue of an irmos («таин еси бцде рай:»)
		}
		if alternative { // the refrain after «или:» is an alternative
			alternative = false
			continue
		}
		if strings.HasSuffix(plain, "или:") {
			alternative = true
			_, pos := hip.Plain(text)
			text = strings.TrimSpace(uni(text, pos, plain, 0, strings.LastIndex(plain, "или:")))
			text = strings.TrimRight(text, " ,.") + "."
		}
		switch key {
		case "irmos", "irmos2", "glory", "now":
			add(key, text)
			key = ""
		default:
			n++
			add(fmt.Sprint(n), text)
		}
	}
	got := map[string]bool{}
	for _, u := range units {
		got[u.Meta["feast"].(string)] = true
	}
	var missing []string
	for _, f := range refrainFeasts {
		if !got[f.id] {
			missing = append(missing, f.id)
		}
	}
	if len(missing) > 0 {
		return units, fmt.Errorf("%s: no refrains for %v", filepath.Base(path), missing)
	}
	return units, nil
}

// renumberIrmos2 turns refrain 1 of a canon into irmos2 and shifts the rest.
func renumberIrmos2(units []corpus.Unit, feast string, canon int) []corpus.Unit {
	prefix := fmt.Sprintf("ir.pripev.%s.%d.", feast, canon)
	for i := range units {
		if !strings.HasPrefix(units[i].ID, prefix) {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(strings.TrimPrefix(units[i].ID, prefix), "%d", &n); err != nil {
			continue
		}
		if n == 1 {
			units[i].ID = prefix + "irmos2"
		} else {
			units[i].ID = fmt.Sprintf("%s%d", prefix, n-1)
		}
	}
	return units
}
