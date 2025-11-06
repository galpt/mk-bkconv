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

	if len(backup.BackupManga) > 0 {
		fmt.Printf("\n=== CHECKING FOR COMMON ISSUES ===\n")
		checkForIssues(backup)
	}
}

func analyzeBackupManga(m *pb.BackupManga) {
	data, _ := json.MarshalIndent(map[string]interface{}{
		"source":             m.Source,
		"url":                m.Url,
		"title":              m.Title,
		"artist":             m.Artist,
		"author":             m.Author,
		"description":        m.Description,
		"genre_count":        len(m.Genre),
		"status":             m.Status,
		"thumbnailUrl":       m.ThumbnailUrl,
		"dateAdded":          m.DateAdded,
		"viewer":             m.Viewer,
		"chapters_count":     len(m.Chapters),
		"categories_count":   len(m.Categories),
		"tracking_count":     len(m.Tracking),
		"favorite":           m.Favorite,
		"chapterFlags":       m.ChapterFlags,
		"viewer_flags":       m.ViewerFlags,
		"history_count":      len(m.History),
		"updateStrategy":     m.UpdateStrategy,
		"lastModifiedAt":     m.LastModifiedAt,
		"favoriteModifiedAt": m.FavoriteModifiedAt,
		"excludedScanlators": m.ExcludedScanlators,
		"version":            m.Version,
		"notes":              m.Notes,
		"initialized":        m.Initialized,
	}, "", "  ")
	fmt.Println(string(data))
}

func checkForIssues(backup *pb.Backup) {
	issues := []string{}

	// Check for zero source IDs
	zeroSources := 0
	for _, m := range backup.BackupManga {
		if m.Source == 0 {
			zeroSources++
		}
	}
	if zeroSources > 0 {
		issues = append(issues, fmt.Sprintf("⚠️  %d manga have source = 0 (likely invalid)", zeroSources))
	}

	// Check for uninitialized manga
	uninitialized := 0
	for _, m := range backup.BackupManga {
		if !m.Initialized {
			uninitialized++
		}
	}
	if uninitialized > 0 {
		issues = append(issues, fmt.Sprintf("⚠️  %d manga have initialized = false", uninitialized))
	}

	// Check for missing timestamps
	noDateAdded := 0
	for _, m := range backup.BackupManga {
		if m.DateAdded == 0 {
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
		if cat.Id == 0 {
			issues = append(issues, fmt.Sprintf("⚠️  Category #%d '%s' has id = 0", i, cat.Name))
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
