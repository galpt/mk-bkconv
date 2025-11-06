package main

import (
	"fmt"

	"github.com/galpt/mk-bkconv/pkg/convert"
)

func main() {
	// Test the source ID generation for MangaFire
	id := convert.GenerateMihonSourceID("mangafire", "en", 1)
	fmt.Printf("MangaFire (en, v1) ID: %d (0x%016x)\n", id, id)

	// Test MangaDex too
	id2 := convert.GenerateMihonSourceID("mangadex", "all", 1)
	fmt.Printf("MangaDex (all, v1) ID: %d (0x%016x)\n", id2, id2)

	// Test lookup
	if id, name, found := convert.LookupKnownSource("MANGAFIRE_EN"); found {
		fmt.Printf("\nLookup MANGAFIRE_EN: ID=%d, Name=%s\n", id, name)
	}

	if id, name, found := convert.LookupKnownSource("MANGADEX"); found {
		fmt.Printf("Lookup MANGADEX: ID=%d, Name=%s\n", id, name)
	}
}
