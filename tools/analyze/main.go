package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/galpt/mk-bkconv/pkg/mihon"
	pb "github.com/galpt/mk-bkconv/proto/mihon"
)

func main() {
	in := flag.String("in", "", "input mihon backup file (.tachibk)")
	flag.Parse()
	if *in == "" {
		log.Fatal("-in required")
	}

	// Load the backup
	backup, err := mihon.LoadBackup(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading backup: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("=== BACKUP ANALYSIS ===\n\n")
	fmt.Printf("Manga count: %d\n", len(backup.BackupManga))
	fmt.Printf("Category count: %d\n", len(backup.BackupCategories))
	fmt.Printf("Source count: %d\n", len(backup.BackupSources))
	fmt.Printf("Preferences count: %d\n", len(backup.BackupPreferences))
	fmt.Printf("Source Preferences count: %d\n", len(backup.BackupSourcePreferences))
	fmt.Printf("Extension Repos count: %d\n\n", len(backup.BackupExtensionRepo))

	if len(backup.BackupManga) > 0 {
		fmt.Printf("=== FIRST MANGA DETAILS ===\n")
		m := backup.BackupManga[0]
		analyzeBackupManga(m)
	}

	fmt.Printf("\n=== CHECKING FOR COMMON ISSUES ===\n")
	checkForIssues(backup)
}

func analyzeBackupManga(m *pb.BackupManga) {
	data, _ := json.MarshalIndent(map[string]interface{}{
		"source":             m.GetSource(),
		"url":                m.GetUrl(),
		"title":              m.GetTitle(),
		"artist":             m.GetArtist(),
		"author":             m.GetAuthor(),
		"description":        m.GetDescription(),
		"genre_count":        len(m.GetGenre()),
		"status":             m.GetStatus(),
		"thumbnailUrl":       m.GetThumbnailUrl(),
		"dateAdded":          m.GetDateAdded(),
		"viewer":             m.GetViewer(),
		"chapters_count":     len(m.GetChapters()),
		"categories_count":   len(m.GetCategories()),
		"tracking_count":     len(m.GetTracking()),
		"favorite":           m.GetFavorite(),
		"chapterFlags":       m.GetChapterFlags(),
		"viewer_flags":       m.GetViewerFlags(),
		"history_count":      len(m.GetHistory()),
		"updateStrategy":     m.GetUpdateStrategy(),
		"lastModifiedAt":     m.GetLastModifiedAt(),
		"favoriteModifiedAt": m.GetFavoriteModifiedAt(),
		"excludedScanlators": m.GetExcludedScanlators(),
		"version":            m.GetVersion(),
		"notes":              m.GetNotes(),
		"initialized":        m.GetInitialized(),
	}, "", "  ")
	fmt.Println(string(data))
}

func checkForIssues(backup *pb.Backup) {
	issues := []string{}

	// Check for zero source IDs
	zeroSources := 0
	for _, m := range backup.BackupManga {
		if m.GetSource() == 0 {
			zeroSources++
		}
	}
	if zeroSources > 0 {
		issues = append(issues, fmt.Sprintf("⚠️  %d manga have source = 0 (likely invalid)", zeroSources))
	}

	// Check for uninitialized manga
	uninitialized := 0
	for _, m := range backup.BackupManga {
		if !m.GetInitialized() {
			uninitialized++
		}
	}
	if uninitialized > 0 {
		issues = append(issues, fmt.Sprintf("⚠️  %d manga have initialized = false", uninitialized))
	}

	// Check for missing timestamps
	noDateAdded := 0
	for _, m := range backup.BackupManga {
		if m.GetDateAdded() == 0 {
			noDateAdded++
		}
	}
	if noDateAdded > 0 {
		issues = append(issues, fmt.Sprintf("⚠️  %d manga have dateAdded = 0", noDateAdded))
	}

	// Check for empty sources list
	if len(backup.BackupSources) == 0 {
		issues = append(issues, "⚠️  No sources defined (backupSources is empty)")
	}

	// Check categories without proper IDs
	for i, cat := range backup.BackupCategories {
		if cat.GetId() == 0 {
			issues = append(issues, fmt.Sprintf("⚠️  Category #%d '%s' has id = 0", i, cat.GetName()))
		}
	}

	if len(issues) == 0 {
		fmt.Println("✅ No obvious issues found")
	} else {
		for _, issue := range issues {
			fmt.Println(issue)
		}
	}
}
