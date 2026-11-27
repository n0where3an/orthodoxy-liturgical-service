package orthlib

import (
	"strings"
	"testing"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func TestTriodionLent(t *testing.T) {
	units, warnings := ImportTriodionLent(extractedRoot(t))
	if len(warnings) > 0 {
		t.Fatal(warnings)
	}
	byID := map[string]string{}
	days := map[string]bool{}
	for _, u := range units {
		byID[u.ID] = strings.ToLower(hip.PlainString(u.Body))
		if u.Meta["copy"] != nil {
			byID[u.ID] = "copy:" + u.Meta["copy"].(string)
		}
		days[strings.Split(u.ID, ".")[1]] = true
	}
	// Six weeks of five days, but the Cross's Wednesday and Friday and the
	// Great Canon's Thursday.
	if len(days) != 27 {
		t.Errorf("%d days: %v", len(days), days)
	}
	want := map[string]string{
		// «слава, тойже»; the theotokion after «и ныне, бгородичен. подобен:».
		"tp.w2_mon.matins.sedalen.k2.glory":      "copy:tp.w2_mon.matins.sedalen.k2.1",
		"tp.w2_mon.matins.sedalen.k2.theotokion": "треволненьми страстей",
		// «слава иный, глас тойже. подобен: …:» — its own hymn.
		"tp.w1_tue.matins.sedalen.k2.glory": "покаяния наста время",
		// The 2nd ode on Tuesday from the canon's heading («песнь в.»).
		"tp.w1_tue.canon.1.ode.2.1.troparion": "",
		// The places: Theodore's triode — troparia, triadikon, theotokion,
		// «Слава Тебе, Боже наш», a troparion, then its irmos in full.
		"tp.w2_mon.canon.2.ode.1.3.triadikon":  "триипостасная единице",
		"tp.w2_mon.canon.2.ode.1.4.theotokion": "кто роди сна",
		"tp.w2_mon.canon.2.ode.1.5.refrain":    "слава тебе бже наш",
		"tp.w2_mon.canon.2.ode.1.katavasia":    "помощник и покровитель бысть мне во спсение",
		"tp.w2_mon.canon.2.ode.8.3.refrain":    "блгословим оца",
		// «песнь ф,»; «ирмос тойже.» opens the second triode.
		"tp.w1_thu.canon.2.ode.9.irmos": "величаем хрсте",
		"tp.w2_tue.canon.2.ode.9.irmos": "copy:tp.w2_tue.canon.1.ode.9.irmos",
		// «катавасиа. ирмос: …»
		"tp.w2_wed.canon.2.ode.3.katavasia": "утверди гдси сердца наша",
		// The aposticha: the idiomelon without «дважды», «мчничен», the
		// theotokion «слава, и ныне, глас тойже: …».
		"tp.w2_mon.matins.aposticha.idiomelon":  "дивное оружие",
		"tp.w3_fri.matins.aposticha.martyricon": "гдси, в память стых твоих",
		"tp.w4_mon.matins.aposticha.theotokion": "архаггльское слово",
		// The 6th week's Friday: «Слава» an idiomelon, «И ныне, паки тойже».
		"tp.w6_fri.matins.aposticha.glory":      "душеполезную совершивше",
		"tp.w6_fri.matins.aposticha.theotokion": "copy:tp.w6_fri.matins.aposticha.glory",
	}
	for id, w := range want {
		got, ok := byID[id]
		if !ok || !strings.HasPrefix(got, w) {
			t.Errorf("%s: %q, want %q…", id, got, w)
		}
	}
	if strings.HasSuffix(byID["tp.w2_mon.matins.aposticha.idiomelon"], "дважды.") {
		t.Errorf("«дважды» left in the idiomelon")
	}
}

func TestOctoechosLent(t *testing.T) {
	units, err := ImportOctoechosLent(extractedRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]string{}
	for _, u := range units {
		byID[u.ID] = strings.ToLower(hip.PlainString(u.Body))
	}
	want := map[string]string{
		// The first Trinity hymn with the day's ending.
		"oc.2.trinity.1.tue": "ст, ст, ст еси бже наш, млтвами прдтечи твоего помилуй нас.",
		"oc.2.trinity.1.fri": "ст, ст, ст еси бже наш, силою крста твоего сохрани нас гдси.",
		"oc.8.trinity.now":   "сляцаеми множеством прегрешений",
		// The photagogika: Monday's intercessors replaced by the day's.
		"oc.1.photagogikon.mon": "предстательствы безплотных твоих, и спси мя.",
		"oc.1.photagogikon.thu": "млтвами гдси, апсл твоих, и стителя николаа, и спси мя.",
		"oc.6.photagogikon.wed": "силою гдси, крста твоего, низпосли",
	}
	for id, w := range want {
		if got := byID[id]; !strings.Contains(got, w) {
			t.Errorf("%s: %q, want …%q…", id, got, w)
		}
	}
	if n := len(units); n != 8*3+8*5+8*5 {
		t.Errorf("%d units", n)
	}
}
