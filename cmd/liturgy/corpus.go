package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/db"
	englishpkg "github.com/n0where3an/orthodoxy-liturgical-service/internal/english"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/orthlib"
)

func cmdMigrate(ctx context.Context, args []string) error {
	pool, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	applied, err := db.Migrate(ctx, pool)
	for _, a := range applied {
		fmt.Println("applied", a)
	}
	if err == nil && len(applied) == 0 {
		fmt.Println("schema is up to date")
	}
	return err
}

func cmdImport(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	root := fs.String("orthlib", "var/orthlib", "extracted orthlib tree (scripts/extract-orthlib.sh)")
	manifest := fs.String("manifest", "data/orthlib/units.yaml", "manifest of fixed units")
	textsDir := fs.String("texts", "data/texts", "texts typed in for the project (HIP)")
	englishMD := fs.String("english-md", os.Getenv("ENGLISH_MD_DIR"), "english-md checkout (Lambertsen Menaion); empty to skip")
	coverdale := fs.String("coverdale", "var/english/coverdale.txt", "Coverdale Psalter as text (scripts/extract-english.sh); missing file is skipped")
	ponomar := fs.String("ponomar", os.Getenv("PONOMAR_DIR"), "Ponomar checkout (lectionary table); empty to skip the readings")
	verbose := fs.Bool("v", false, "print warnings")
	fs.Parse(args)

	m, err := orthlib.LoadManifest(*manifest)
	if err != nil {
		return err
	}
	fixed, err := m.Extract(*root)
	if err != nil {
		return err
	}
	men, warnings, err := orthlib.ImportMenaion(*root)
	if err != nil {
		return err
	}
	oct, w := orthlib.ImportOctoechosVespers(*root)
	warnings = append(warnings, w...)
	bogor, err := orthlib.ImportResurrectionalTheotokia(*root)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	men = append(men, bogor...)
	gc, w := orthlib.ImportFeastCompline(*root)
	warnings = append(warnings, w...)
	men = append(men, gc...)
	eve, w := orthlib.ImportEveVespers(*root)
	warnings = append(warnings, w...)
	men = append(men, eve...)
	royal, w := orthlib.ImportRoyalHours(*root)
	warnings = append(warnings, w...)
	men = append(men, royal...)
	special, w := orthlib.ImportSpecialDays(*root)
	warnings = append(warnings, w...)
	men = append(men, special...)
	om, w := orthlib.ImportGeneralMenaion(*root)
	warnings = append(warnings, w...)
	men = append(men, om...)
	octMatins, w := orthlib.ImportOctoechosMatins(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octMatins...)
	octSatMatins, w := orthlib.ImportOctoechosSaturdayMatins(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octSatMatins...)
	octCompline, w := orthlib.ImportOctoechosCompline(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octCompline...)
	octMidnight, w := orthlib.ImportOctoechosMidnight(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octMidnight...)
	octSun, w := orthlib.ImportOctoechosSundayVespers(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octSun...)
	octSunLit, w := orthlib.ImportOctoechosSundayLiturgy(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octSunLit...)
	octSunMatins, w := orthlib.ImportOctoechosSundayMatins(*root)
	warnings = append(warnings, w...)
	oct = append(oct, octSunMatins...)
	if signs, err := orthlib.ImportTypikonSigns(*root); err == nil {
		oct = append(oct, signs...)
	} else {
		warnings = append(warnings, err.Error())
	}
	irmologion, w := orthlib.ImportIrmologionOdes(*root)
	warnings = append(warnings, w...)
	oct = append(oct, irmologion...)
	if refrains, err := orthlib.ImportOdeNineRefrains(*root); err == nil {
		oct = append(oct, refrains...)
	} else {
		warnings = append(warnings, err.Error())
	}
	triodion, w := orthlib.ImportTriodionPrep(*root)
	warnings = append(warnings, w...)
	oct = append(oct, triodion...)
	if mag, err := orthlib.ImportMagnifications(*root); err == nil {
		oct = append(oct, mag...)
	} else {
		warnings = append(warnings, err.Error())
	}
	if app, err := orthlib.ImportOctoechosSundayAppendix(*root); err == nil {
		oct = append(oct, app...)
	} else {
		warnings = append(warnings, err.Error())
	}
	psalter, w := orthlib.ImportPsalter(*root)
	warnings = append(warnings, w...)
	theot, w := orthlib.ImportDismissalTheotokia(*root)
	warnings = append(warnings, w...)
	if ps, err := orthlib.ImportPsalms(*root, 134, 135); err == nil {
		psalter = append(psalter, ps...)
	} else {
		warnings = append(warnings, err.Error())
	}
	if th, err := orthlib.ImportResurrectionalDismissalTheotokia(*root); err == nil {
		theot = append(theot, th...)
	} else {
		warnings = append(warnings, err.Error())
	}
	manualSrc, manual, manualTr, err := orthlib.ImportManual(*textsDir)
	if err != nil {
		return err
	}
	var readings []corpus.Unit
	if *ponomar != "" {
		readings, w, err = orthlib.ImportReadings(*root, filepath.Join(*ponomar, "scripts/Perl/data/lectionary.tsv"))
		if err != nil {
			return err
		}
		warnings = append(warnings, w...)
		manualSrc = append(manualSrc, orthlib.LectionarySource)
	}

	pool, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	st := &corpus.Store{Pool: pool}
	for _, s := range append(m.SourceRecords(), manualSrc...) {
		if err := st.UpsertSource(ctx, s); err != nil {
			return err
		}
	}
	zach, w, err := orthlib.ImportZachala(*root)
	if err != nil {
		return err
	}
	warnings = append(warnings, w...)
	all := append(append(append(append(append(append(append(fixed, men...), oct...), psalter...), theot...), manual...), readings...), zach...)
	// The Menaion's liturgy rubrics and cited texts.
	all, w = orthlib.ResolveLiturgy(*root, all)
	warnings = append(warnings, w...)
	// The common parts of the General Menaion for the days that lack them.
	all, w = orthlib.SupplementFromGeneralMenaion(all)
	warnings = append(warnings, w...)
	// The magnifications and selected psalms of the Irmologion.
	all = orthlib.SupplementMagnifications(all)
	if dup := duplicateIDs(all); len(dup) > 0 {
		return fmt.Errorf("duplicate unit keys: %v", dup)
	}
	// Re-import replaces whole books so that renamed or vanished keys go away.
	books := map[string]bool{}
	for _, u := range all {
		books[u.Book] = true
	}
	for b := range books {
		if _, err := st.DeleteBook(ctx, b); err != nil {
			return err
		}
	}
	const batch = 500
	for i := 0; i < len(all); i += batch {
		j := min(i+batch, len(all))
		if err := st.UpsertUnits(ctx, all[i:j]); err != nil {
			return err
		}
	}
	// English translations of the units above.
	english := manualTr
	var enSources []corpus.Source
	if _, err := os.Stat(*coverdale); err == nil {
		cov, w, err := englishpkg.ImportCoverdale(*coverdale)
		if err != nil {
			return err
		}
		warnings = append(warnings, w...)
		english = append(english, cov...)
		enSources = append(enSources, englishpkg.CoverdaleSource)
	}
	if *englishMD != "" {
		services, err := englishpkg.LoadServices(*englishMD)
		if err != nil {
			return err
		}
		men, stats := englishpkg.Align(services, all)
		english = append(english, men...)
		enSources = append(enSources, englishpkg.LambertsenSource)
		fmt.Printf("English Menaion aligned: troparia %d/%d, kontakia %d/%d, stichera at «Lord, I have cried» %d/%d\n",
			stats["troparion.en"], stats["troparion.cu"], stats["kontakion.en"], stats["kontakion.cu"],
			stats["vespers.lic.sticheron.en"], stats["vespers.lic.sticheron.cu"])
	}
	for _, s := range enSources {
		if err := st.UpsertSource(ctx, s); err != nil {
			return err
		}
	}
	if len(english) > 0 {
		skipped, err := st.UpsertTranslations(ctx, english)
		if err != nil {
			return err
		}
		fmt.Printf("imported %d English translations (%d skipped: no such unit)\n", len(english)-len(skipped), len(skipped))
	}

	kinds := map[string]int{}
	for _, u := range men {
		kinds[u.Kind]++
	}
	fmt.Printf("imported %d fixed units, %d Menaion units (%d troparia, %d kontakia), %d Octoechos, %d Psalter, %d theotokia, %d typed-in, %d readings; %d warnings\n",
		len(fixed), len(men), kinds["troparion"], kinds["kontakion"], len(oct), len(psalter), len(theot), len(manual), len(readings)/2, len(warnings))
	if *verbose {
		for _, w := range warnings {
			fmt.Println("warning:", w)
		}
	}
	return nil
}

func duplicateIDs(us []corpus.Unit) []string {
	seen := map[string]bool{}
	var dup []string
	for _, u := range us {
		if seen[u.ID] {
			dup = append(dup, u.ID)
		}
		seen[u.ID] = true
	}
	return dup
}
