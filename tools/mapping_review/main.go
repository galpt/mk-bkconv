package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/galpt/mk-bkconv/pkg/convert"
)

func main() {
	srcPath := "pkg/convert/source_mapping.go"
	outPath := "known_mappings_review.csv"
	data, err := os.ReadFile(srcPath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", srcPath, err)
	}
	text := string(data)

	// Find all mapping entries using a regex that captures key and the body
	re := regexp.MustCompile(`(?s)"([^"]+)"\s*:\s*\{(.*?)\}`)
	matches := re.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		log.Fatalf("no mapping entries found in %s", srcPath)
	}

	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("failed to create output csv: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	w.Write([]string{"KotatsuKey", "MihonName", "MihonLang", "MihonVersionID", "MihonSourceID", "Notes"})

	// regexes for fields
	reName := regexp.MustCompile(`MihonName\s*:\s*"([^"]*)"`)
	reLang := regexp.MustCompile(`MihonLang\s*:\s*"([^"]*)"`)
	reVer := regexp.MustCompile(`MihonVersionID\s*:\s*(\d+)`)
	reNotes := regexp.MustCompile(`Notes\s*:\s*"([^"]*)"`)

	for _, m := range matches {
		kotKey := m[1]
		body := m[2]
		mihon := ""
		lang := "all"
		ver := 1
		notes := ""

		if sm := reName.FindStringSubmatch(body); len(sm) > 1 {
			mihon = sm[1]
		}
		if sm := reLang.FindStringSubmatch(body); len(sm) > 1 {
			lang = sm[1]
		}
		if sm := reVer.FindStringSubmatch(body); len(sm) > 1 {
			if vi, err := strconv.Atoi(sm[1]); err == nil {
				ver = vi
			}
		}
		if sm := reNotes.FindStringSubmatch(body); len(sm) > 1 {
			notes = sm[1]
		}

		id := convert.GenerateMihonSourceID(mihon, lang, ver)
		w.Write([]string{kotKey, mihon, lang, strconv.Itoa(ver), strconv.FormatInt(id, 10), notes})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("csv write error: %v", err)
	}

	fmt.Printf("wrote %s with %d entries\n", outPath, len(matches))
}
