package liturgy

import (
	"strings"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
)

// SourceBook names the book a text is taken from, for the margin of the
// rendered service: the unit's book, made precise by where the text comes
// from —
//
//   - a reading of the lectionary (the Menaion's too, which cites it by its
//     pericope): "apostle" or "gospel";
//   - a text the Menaion day lacks, taken from the General Menaion
//     (copied_from om.*): "general_menaion";
//   - without the book (texts of the tests): by the id's prefix.
func SourceBook(tx corpus.Text) string {
	from, _ := tx.Meta["copied_from"].(string)
	book := tx.Book
	if book == "" {
		book = bookOfID(tx.UnitID)
	}
	switch {
	case book == "lectionary" || strings.HasPrefix(from, "lect."):
		ref := tx.UnitID
		if from != "" {
			ref = from
		}
		if strings.HasPrefix(ref, "lect.ap.") || strings.HasPrefix(ref, "lect.zach.ap.") {
			return "apostle"
		}
		return "gospel"
	case strings.HasPrefix(from, "om."):
		return "general_menaion"
	}
	return book
}

// bookOfID: the book by the unit id's prefix.
func bookOfID(id string) string {
	for _, p := range []struct{ prefix, book string }{
		{"me.", "menaion"}, {"om.", "general_menaion"}, {"oc.", "octoechos"}, {"ch.", "chasoslov"},
		{"common.", "chasoslov"}, {"sl.", "sluzhebnik"}, {"lit.", "sluzhebnik"}, {"ps.", "psalter"},
		{"ir.", "irmologion"}, {"lect.", "lectionary"}, {"tc.", "triodion"}, {"tp.", "triodion"}, {"ty.", "typikon"},
	} {
		if strings.HasPrefix(id, p.prefix) {
			return p.book
		}
	}
	return ""
}

// marginBook: the book shown in the margin of service, "" for the short
// connecting formulas between the texts, which neither carry the book nor
// interrupt it: «Слава», «И ныне», «Аминь», «Господи, помилуй» (up to thrice —
// not the 12 or 40 times of the hours), exclamations, responses, refrains,
// the verses before the stichera, the announcements («Прокимен, глас…», «Ко
// римляном послания…»), the headings of the odes.
//
// The common texts (common.*: the Trisagion, «Царю Небесный», «Приидите,
// поклонимся», the Creed, «Господи, помилуй» 12 and 40…) are the same in
// every service; they are marked as the book of the service's ordinary —
// the Sluzhebnik at the liturgy, the Chasoslov otherwise — not as the book
// they happen to be cut from.
func marginBook(tx corpus.Text, repeat int, service string) string {
	switch tx.UnitID {
	case "common.glory", "common.and_now", "common.glory_now", "common.glory_now_short", "common.amen":
		return ""
	case "common.lord_have_mercy":
		if repeat <= 3 {
			return ""
		}
		return ordinaryBook(service)
	}
	if strings.HasPrefix(tx.UnitID, "ch.canon.") {
		return ""
	}
	switch tx.Kind {
	case "exclamation", "response", "doxology", "rubric", "refrain", "verse", "announcement", "hymn":
		return ""
	}
	if strings.HasPrefix(tx.UnitID, "common.") {
		return ordinaryBook(service)
	}
	return SourceBook(tx)
}

// ordinaryBook: the book of the fixed order of the service.
func ordinaryBook(service string) string {
	if service == "liturgy" {
		return "sluzhebnik"
	}
	return "chasoslov"
}
