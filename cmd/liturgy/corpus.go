package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/corpus"
	"github.com/n0where3an/orthodoxy-liturgical-service/internal/db"
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

	pool, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	st := &corpus.Store{Pool: pool}
	for _, s := range m.SourceRecords() {
		if err := st.UpsertSource(ctx, s); err != nil {
			return err
		}
	}
	if _, err := st.DeleteBook(ctx, "menaion"); err != nil {
		return err
	}
	all := append(fixed, men...)
	const batch = 500
	for i := 0; i < len(all); i += batch {
		j := min(i+batch, len(all))
		if err := st.UpsertUnits(ctx, all[i:j]); err != nil {
			return err
		}
	}
	kinds := map[string]int{}
	for _, u := range men {
		kinds[u.Kind]++
	}
	fmt.Printf("imported %d fixed units, %d Menaion units (%d troparia, %d kontakia); %d warnings\n",
		len(fixed), len(men), kinds["troparion"], kinds["kontakion"], len(warnings))
	if *verbose {
		for _, w := range warnings {
			fmt.Println("warning:", w)
		}
	}
	return nil
}
