// Command hipcat converts orthlib .hip files (cp1251) to Unicode text, one
// paragraph per line; red (cinnabar) text is wrapped in [[...]].
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

func main() {
	plain := flag.Bool("plain", false, "print civil-script plain form")
	flag.Parse()
	for _, f := range flag.Args() {
		b, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		ps, err := hip.ParseFile(b)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for i, p := range ps {
			if *plain {
				fmt.Printf("%d\t%s\n", i, hip.PlainString(p.Text()))
				continue
			}
			fmt.Printf("%d\t", i)
			for _, s := range p {
				if s.Red {
					fmt.Printf("[[%s]]", s.Text)
				} else {
					fmt.Print(s.Text)
				}
			}
			fmt.Println()
		}
	}
}
