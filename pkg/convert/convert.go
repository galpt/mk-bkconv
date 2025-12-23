package convert

import (
	"fmt"
	"hash/fnv"
	"strings"
	"errors"

	"github.com/galpt/mk-bkconv/pkg/kotatsu"
	pb "github.com/galpt/mk-bkconv/proto/mihon"
)

// Helper functions to work with optional string pointers
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// pointer helpers for proto2 generated fields
func int64Ptr(i int64) *int64                                  { return &i }
func int32Ptr(i int32) *int32                                  { return &i }
func boolPtr(b bool) *bool                                     { return &b }
func float32Ptr(f float32) *float32                            { return &f }
func updateStrategyPtr(u pb.UpdateStrategy) *pb.UpdateStrategy { return &u }

// generateSourceID creates a deterministic numeric source ID from a Kotatsu source name
// First attempts to use known source mappings (for sources that exist in both ecosystems)
// Falls back to FNV hash for unknown sources
func generateSourceID(sourceName string, allowFallback bool) (int64, error) {
	if sourceName == "" {
		// Use MangaDex as fallback
		return GenerateMihonSourceID("MangaDex", "all", 1), nil
	}

	// Try known mapping first
	if id, _, found := LookupKnownSource(sourceName); found {
		return id, nil
	}

	if !allowFallback {
		return -1, errors.New("no known mapping found for " + sourceName + " and fallback not allowed")
	}
	// Fallback to FNV hash for unknown sources
	h := fnv.New64a()
	h.Write([]byte(sourceName))
	return int64(h.Sum64()), nil
}

// MihonToKotatsu converts from protobuf-based Mihon backup to Kotatsu backup
func MihonToKotatsu(b *pb.Backup) *kotatsu.KotatsuBackup {
	// Ensure the incoming Mihon backup only contains sources that have a corresponding
	// Kotatsu source implementation (best-effort). This drops entries that would
	// otherwise point to missing Kotatsu sources.
	FilterMihonForKotatsu(b)

	kb := &kotatsu.KotatsuBackup{}

	for i, m := range b.BackupManga {
		fav := kotatsu.KotatsuFavouriteEntry{
			MangaId:    int64(i + 1),
			CategoryId: 0, // Will be updated if manga has categories
			SortKey:    i,
			Pinned:     false,
			CreatedAt:  m.GetDateAdded(),
			Manga: kotatsu.KotatsuManga{
				Id:         int64(i + 1),
				Title:      m.GetTitle(),
				Url:        m.GetUrl(),
				PublicUrl:  m.GetUrl(),
				CoverUrl:   m.GetThumbnailUrl(),
				LargeCover: m.GetThumbnailUrl(),
				Author:     m.GetAuthor(),
				Source:     "",
				Tags:       []interface{}{},
			},
		}

		// Assign first category if exists
		if len(m.GetCategories()) > 0 {
			fav.CategoryId = m.GetCategories()[0]
		}

		kb.Favourites = append(kb.Favourites, fav)
	}

	// Convert categories
	for _, c := range b.BackupCategories {
		kb.Categories = append(kb.Categories, kotatsu.KotatsuCategory{
			CategoryId: c.GetId(),
			CreatedAt:  c.GetOrder(),
			SortKey:    0,
			Title:      c.GetName(),
		})
	}

	return kb
}

// KotatsuToMihon converts from Kotatsu backup to protobuf-based Mihon backup
func KotatsuToMihon(kb *kotatsu.KotatsuBackup, allowSourceFallback bool) (*pb.Backup, error) {
	b := &pb.Backup{}

	// Build a map of manga ID -> chapters from the index
	chaptersByManga := make(map[int64][]*pb.BackupChapter)
	for _, idx := range kb.Index {
		var chapters []*pb.BackupChapter
		for _, kc := range idx.Chapters {
			chapters = append(chapters, &pb.BackupChapter{
				Url:            stringPtr(kc.Url),
				Name:           stringPtr(kc.Name),
				Scanlator:      stringPtr(kc.Scanlator),
				Read:           boolPtr(false),
				Bookmark:       boolPtr(false),
				LastPageRead:   int64Ptr(0),
				ChapterNumber:  float32Ptr(kc.Number),
				DateFetch:      int64Ptr(0),
				DateUpload:     int64Ptr(kc.UploadDate),
				SourceOrder:    int64Ptr(0),
				LastModifiedAt: int64Ptr(0),
				Version:        int64Ptr(1),
			})
		}
		chaptersByManga[idx.MangaId] = chapters
	}

	// Track unique sources and build source mapping
	sourceMap := make(map[string]int64)
	var backupSources []*pb.BackupSource

	// Convert favourites to mangas with their chapters
	for _, fav := range kb.Favourites {
		km := fav.Manga

		// Generate or retrieve source ID
		sourceID, err := generateSourceID(km.Source, allowSourceFallback)
		if err != nil { return nil, err }
		if _, exists := sourceMap[km.Source]; !exists {
			sourceMap[km.Source] = sourceID
			// Try to get the Mihon source name, fall back to Kotatsu name
			sourceName := km.Source
			if id, name, found := LookupKnownSource(km.Source); found {
				sourceName = name
				sourceID = id
			}
			backupSources = append(backupSources, &pb.BackupSource{
				Name:     stringPtr(sourceName),
				SourceId: int64Ptr(sourceID),
			})
		}

		m := &pb.BackupManga{
			Source:         int64Ptr(sourceID), // Now using generated source ID
			Url:            stringPtr(km.Url),
			Title:          stringPtr(km.Title),
			Author:         stringPtr(km.Author),
			Artist:         stringPtr(""),
			Description:    stringPtr(""),
			Genre:          []string{},
			Status:         int32Ptr(0),
			ThumbnailUrl:   stringPtr(km.CoverUrl),
			DateAdded:      int64Ptr(fav.CreatedAt),
			Viewer:         int32Ptr(0),
			Chapters:       chaptersByManga[km.Id],
			Categories:     []int64{fav.CategoryId},
			Favorite:       boolPtr(true),
			ChapterFlags:   int32Ptr(0),
			ViewerFlags:    nil,
			UpdateStrategy: updateStrategyPtr(pb.UpdateStrategy_ALWAYS_UPDATE),
			LastModifiedAt: int64Ptr(fav.CreatedAt),
			Version:        int64Ptr(1),
			Initialized:    boolPtr(true), // Mark as initialized
		}
		b.BackupManga = append(b.BackupManga, m)
	}

	// Convert categories
	for _, c := range kb.Categories {
		b.BackupCategories = append(b.BackupCategories, &pb.BackupCategory{
			Name:  stringPtr(c.Title),
			Order: int64Ptr(c.CreatedAt),
			Id:    int64Ptr(c.CategoryId),
			Flags: int64Ptr(0),
		})
	}

	// Add the source mappings
	b.BackupSources = backupSources

	// Populate BackupExtensionRepos if there are any sources
	// This ensures fresh Mihon installs can discover/install required extensions
	if len(b.BackupSources) > 0 {
		keiyoushiRepo := &pb.BackupExtensionRepos{
			BaseUrl:               stringPtr("https://raw.githubusercontent.com/keiyoushi/extensions/repo"),
			Name:                  stringPtr("Keiyoushi"),
			ShortName:             stringPtr("keiyoushi"),
			Website:               stringPtr("https://keiyoushi.github.io"),
			SigningKeyFingerprint: stringPtr("9add655a78e96c4ec7a53ef89dccb557cb5d767489fac5e785d671a5a75d4da2"),
		}
		b.BackupExtensionRepo = []*pb.BackupExtensionRepos{keiyoushiRepo}

		fmt.Printf("\n=== Conversion Summary ===\n")
		fmt.Printf("✅ Added Keiyoushi extension repository to backup\n")
		fmt.Printf("✅ Converted %d manga entries\n", len(b.BackupManga))
		fmt.Printf("✅ Found %d unique sources\n\n", len(b.BackupSources))

		fmt.Printf("📋 Sources in this backup:\n")
		for i, src := range b.BackupSources {
			fmt.Printf("   %d. %s (Source ID: %d)\n", i+1, src.GetName(), src.GetSourceId())
		}

		fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("📱 HOW TO USE THIS BACKUP IN MIHON:\n")
		fmt.Printf(strings.Repeat("=", 60) + "\n\n")
		fmt.Printf("STEP 1: Restore the backup\n")
		fmt.Printf("   • Open Mihon → Settings → Backup and restore\n")
		fmt.Printf("   • Select 'Restore backup' and choose the .tachibk file\n")
		fmt.Printf("   • The Keiyoushi extension repo will be automatically added\n\n")

		fmt.Printf("STEP 2: Install required extensions\n")
		fmt.Printf("   • Open Mihon → Browse → Extensions tab\n")
		fmt.Printf("   • You'll see the sources list above\n")
		fmt.Printf("   • Search for each source name and install its extension\n")
		fmt.Printf("   • Extensions are automatically trusted from Keiyoushi repo\n\n")

		fmt.Printf("STEP 3: Verify your manga\n")
		fmt.Printf("   • Go to Library tab\n")
		fmt.Printf("   • Your manga should now be readable\n")
		fmt.Printf("   • Tap any manga to verify chapters are available\n\n")

		fmt.Printf("💡 TIP: Extension names usually match source names\n")
		fmt.Printf("   Example: 'MangaDex' source → install 'MangaDex' extension\n\n")
		fmt.Printf(strings.Repeat("=", 60) + "\n\n")
	} // Filter out any sources/mangas that are not available in Mihon
	// pass kb.RawSources (may be empty) so the filter can attempt to read kotatsu-provided list
	FilterBackupToCommon(b, kb.RawSources)

	return b, nil
}
