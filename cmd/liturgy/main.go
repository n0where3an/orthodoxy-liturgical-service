// Command liturgy manages the corpus and generates services.
//
//	liturgy migrate
//	liturgy import [-orthlib var/orthlib]
//	liturgy plan   -date 2026-11-12 [-service first_hour|third_hour|sixth_hour|ninth_hour] [-reader] [-standalone] [-typikon]
//	liturgy render -date 2026-11-12 [-mode laity|kliros] [-lang cu] [-o out.html]
//	liturgy serve  [-addr 127.0.0.1:8080]
//
// Dates are civil (Gregorian) unless -julian is given. Configuration comes
// from the environment, with .env in the current directory as a fallback.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/db"
)

func main() {
	if err := db.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: .env:", err)
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	cmds := map[string]func(context.Context, []string) error{
		"migrate": cmdMigrate,
		"import":  cmdImport,
		"plan":    cmdPlan,
		"render":  cmdRender,
		"serve":   cmdServe,
	}
	f, ok := cmds[os.Args[1]]
	if !ok {
		usage()
		os.Exit(2)
	}
	if err := f(ctx, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: liturgy migrate | import | plan | render | serve [flags]")
}
