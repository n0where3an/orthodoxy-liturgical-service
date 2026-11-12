package liturgy

import "fmt"

// Detail is how much of a node a display mode shows.
type Detail int

const (
	Omit        Detail = iota
	Heading            // only the title
	Cue                // incipit
	Placeholder        // a generated rubric standing in for the text
	Full
)

// Mode decides the detail of each node.
type Mode interface {
	Name() string
	Detail(n *RNode) Detail
}

// TableMode is a mode described by a table (it may come from data later).
type TableMode struct {
	ModeName      string
	Chant         Detail // hymns and readings of reader/choir, responses
	Exclamation   Detail // litanies and exclamations of clergy
	SecretPrayer  Detail
	KlirosRubric  Detail
	ExplainRubric Detail
}

func (m TableMode) Name() string { return m.ModeName }

func (m TableMode) Detail(n *RNode) Detail {
	switch n.Kind {
	case KindHeading:
		return Full
	case KindRubric:
		switch n.Audience {
		case AudienceKliros:
			return m.KlirosRubric
		case AudienceLaity:
			return m.ExplainRubric
		default:
			return Full
		}
	}
	if n.Voice == Secret {
		return m.SecretPrayer
	}
	if n.Performer == Priest || n.Performer == Deacon {
		return m.Exclamation
	}
	return m.Chant
}

// The two modes of the specification.
var (
	Kliros = TableMode{ModeName: "kliros", Chant: Full, Exclamation: Cue, SecretPrayer: Omit, KlirosRubric: Full, ExplainRubric: Omit}
	Laity  = TableMode{ModeName: "laity", Chant: Full, Exclamation: Full, SecretPrayer: Placeholder, KlirosRubric: Omit, ExplainRubric: Full}
)

// ModeByName returns a built-in mode.
func ModeByName(name string) (Mode, error) {
	switch name {
	case "kliros":
		return Kliros, nil
	case "laity", "":
		return Laity, nil
	}
	return nil, fmt.Errorf("unknown mode %q (kliros, laity)", name)
}
