package orthlib

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// ImportResurrectionalTheotokia reads «Богородичны воскресны на осмь
// гласов» at the end of the Menaion (mineya/bogor/bogor.hip): for each tone
// the theotokion at «Господи, воззвах» (the dogmatikon) and the one at the
// aposticha, sung at great vespers when the Menaion only cites them.
// Keys: me.bogor.<tone>.lic, me.bogor.<tone>.aposticha.
func ImportResurrectionalTheotokia(root string) ([]corpus.Unit, error) {
	path := filepath.Join(root, "mineya", "bogor", "bogor.hip")
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	var units []corpus.Unit
	tone, part := 0, ""
	for _, p := range d.Paras {
		text := CleanText(pageRef.ReplaceAllString(p.Text(), ""))
		plain := hip.PlainString(text)
		switch {
		case strings.HasPrefix(plain, "глас ") && strings.Contains(plain, "бгородичен"):
			if tone == 8 {
				return units, nil
			}
			tone, part = tone+1, "lic"
			continue
		case strings.HasPrefix(plain, "на стиховне бгородичен"):
			part = "aposticha"
			continue
		case tone == 0 || part == "":
			continue
		}
		units = append(units, corpus.Unit{
			ID: fmt.Sprintf("me.bogor.%d.%s", tone, part), Book: "menaion", Kind: "theotokion", Tone: tone,
			Lang: "cu", Body: text, SourceID: "orthlib.mineya",
		})
		part = ""
	}
	if len(units) != 16 {
		return units, fmt.Errorf("%s: %d theotokia, want 16", filepath.Base(path), len(units))
	}
	return units, nil
}
