package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/galpt/mk-bkconv/pkg/convert"
	"github.com/galpt/mk-bkconv/pkg/kotatsu"
	"github.com/galpt/mk-bkconv/pkg/mihon"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	sub := os.Args[1]
	allowSourcesFallback := slices.Contains(os.Args, "--allow-fallback")
	switch sub {
	case "mihon-to-kotatsu":
		in := flag.String("in", "", "input mihon backup file (.tachibk)")
		out := flag.String("out", "", "output kotatsu zip file")
		flag.CommandLine.Parse(os.Args[2:])
		if *in == "" || *out == "" {
			usage()
			os.Exit(2)
		}
		b, err := mihon.LoadBackup(*in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading mihon backup: %v\n", err)
			os.Exit(3)
		}
		kb := convert.MihonToKotatsu(b)
		if err := kotatsu.WriteKotatsuZip(*out, kb); err != nil {
			fmt.Fprintf(os.Stderr, "error writing kotatsu zip: %v\n", err)
			os.Exit(4)
		}
		fmt.Println("Conversion complete.")

	case "kotatsu-to-mihon":
		in := flag.String("in", "", "input kotatsu zip file")
		out := flag.String("out", "", "output mihon backup file (.tachibk)")
		flag.CommandLine.Parse(os.Args[2:])
		if *in == "" || *out == "" {
			usage()
			os.Exit(2)
		}
		kb, err := kotatsu.LoadKotatsuZip(*in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading kotatsu zip: %v\n", err)
			os.Exit(3)
		}
		b, err := convert.KotatsuToMihon(kb, allowSourcesFallback)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error converting kotatsu to mihon: %v\n", err)
			os.Exit(5)
		}
		if err := mihon.WriteBackup(*out, b); err != nil {
			fmt.Fprintf(os.Stderr, "error writing mihon backup: %v\n", err)
			os.Exit(4)
		}
		fmt.Println("Conversion complete.")

	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("mk-bkconv: convert between Mihon and Kotatsu backups")
	fmt.Println("USAGE:")
	fmt.Println("  mk-bkconv <mihon-to-kotatsu|kotatsu-to-mihon> -in <input> -out <output> --allow-fallback")
	fmt.Println("    --allow-fallback   this flag allows you to fallback to hashing when there was no mapping for a source found")

}
