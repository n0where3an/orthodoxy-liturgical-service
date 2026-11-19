package orthlib

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// The afterfeast of the Transfiguration (8 August): matins with the feast's
// sessional hymns, its canon and the saint's, the feast's aposticha.
func TestAfterfeastMatinsImport(t *testing.T) {
	root := extractedRoot(t)
	units, err := ParseMenaionDay(filepath.Join(root, "mineya/aug/aug/Август/08.hip"), 8, "08")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	for _, u := range units {
		by[strings.TrimPrefix(u.ID, "me.08.08.")] = strings.ToLower(hip.PlainString(u.Body))
	}
	for key, incipit := range map[string]string{
		"matins.k_sedalen.1.1":         "на горе фаворстей преобразился",
		"matins.k_sedalen.2.1":         "неприступною твоею",
		"matins.canon.1.ode.1.irmos":   "воду прошед",
		"matins.aposticha.sticheron.1": "благо же паче ума",
		"matins.aposticha.verse.2":     "твоя суть нбса",
		"matins.aposticha.sticheron.3": "приидите, земнороднии",
		"matins.aposticha.glory_now":   "неодержимое",
		"matins.exapostilarion.1":      "свете неизменный",
	} {
		if got := by[key]; !strings.HasPrefix(strings.TrimSpace(got), incipit) {
			t.Errorf("%s: %q, want %q…", key, got, incipit)
		}
	}
}

// The aposticha after the praises («на хвалитех стихиры…», 16 September):
// the praises do not end the matins of the day.
func TestApostichaAfterPraisesImport(t *testing.T) {
	root := extractedRoot(t)
	units, err := ParseMenaionDay(filepath.Join(root, "mineya/sep/sep/Сентябрь/16.hip"), 9, "16")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, u := range units {
		if strings.HasPrefix(u.ID, "me.09.16.matins.aposticha.sticheron.") {
			n++
			if u.ID == "me.09.16.matins.aposticha.sticheron.1" &&
				!strings.HasPrefix(strings.ToLower(hip.PlainString(u.Body)), "песньми таинственными") {
				t.Errorf("%s: %q", u.ID, u.Body)
			}
		}
	}
	if n != 3 {
		t.Errorf("%d stichera at the aposticha, want 3", n)
	}
	found := false
	for _, u := range units {
		if u.ID == "me.09.16.matins.aposticha.now" {
			found = strings.HasPrefix(strings.ToLower(hip.PlainString(u.Body)), "днесь яко воистинну")
		}
	}
	if !found {
		t.Error("no «И ныне» of the feast at the aposticha")
	}
}

// A canon of compline printed before matins («возследование прпдбнаго
// маркиана поется на повечерии. канон.», 10 January) is not matins.
func TestComplineCanonBeforeMatinsImport(t *testing.T) {
	root := extractedRoot(t)
	units, err := ParseMenaionDay(filepath.Join(root, "mineya/jan/jan/Январь/10.hip"), 1, "10")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	for _, u := range units {
		by[strings.TrimPrefix(u.ID, "me.01.10.")] = strings.ToLower(hip.PlainString(u.Body))
	}
	if !strings.HasPrefix(by["matins.aposticha.sticheron.1"], "ныне странно спс") {
		t.Errorf("aposticha: %q", by["matins.aposticha.sticheron.1"])
	}
	if strings.HasPrefix(by["matins.canon.1.ode.1.irmos"], "песнь победную") {
		t.Error("the compline canon taken for matins")
	}
}
