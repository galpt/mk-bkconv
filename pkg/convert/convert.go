package convert

import (
	"hash/fnv"

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

func stringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// generateSourceID creates a deterministic numeric source ID from a Kotatsu source name
// First attempts to use known source mappings (for sources that exist in both ecosystems)
// Falls back to FNV hash for unknown sources
func generateSourceID(sourceName string) int64 {
	if sourceName == "" {
		// Use MangaDex as fallback
		return GenerateMihonSourceID("MangaDex", "all", 1)
	}

	// Try known mapping first
	if id, _, found := LookupKnownSource(sourceName); found {
		return id
	}

	// Fallback to FNV hash for unknown sources
	h := fnv.New64a()
	h.Write([]byte(sourceName))
	return int64(h.Sum64())
}

// MihonToKotatsu converts from protobuf-based Mihon backup to Kotatsu backup
func MihonToKotatsu(b *pb.Backup) *kotatsu.KotatsuBackup {
	kb := &kotatsu.KotatsuBackup{}

	for i, m := range b.BackupManga {
		fav := kotatsu.KotatsuFavouriteEntry{
			MangaId:    int64(i + 1),
			CategoryId: 0, // Will be updated if manga has categories
			SortKey:    i,
			Pinned:     false,
			CreatedAt:  m.DateAdded,
			Manga: kotatsu.KotatsuManga{
				Id:         int64(i + 1),
				Title:      m.Title,
				Url:        m.Url,
				PublicUrl:  m.Url,
				CoverUrl:   stringVal(m.ThumbnailUrl),
				LargeCover: stringVal(m.ThumbnailUrl),
				Author:     stringVal(m.Author),
				Source:     "",
				Tags:       []interface{}{},
			},
		}

		// Assign first category if exists
		if len(m.Categories) > 0 {
			fav.CategoryId = m.Categories[0]
		}

		kb.Favourites = append(kb.Favourites, fav)
	}

	// Convert categories
	for _, c := range b.BackupCategories {
		kb.Categories = append(kb.Categories, kotatsu.KotatsuCategory{
			CategoryId: c.Id,
			CreatedAt:  c.Order,
			SortKey:    0,
			Title:      c.Name,
		})
	}

	return kb
}

// KotatsuToMihon converts from Kotatsu backup to protobuf-based Mihon backup
func KotatsuToMihon(kb *kotatsu.KotatsuBackup) *pb.Backup {
	b := &pb.Backup{}

	// Build a map of manga ID -> chapters from the index
	chaptersByManga := make(map[int64][]*pb.BackupChapter)
	for _, idx := range kb.Index {
		var chapters []*pb.BackupChapter
		for _, kc := range idx.Chapters {
			chapters = append(chapters, &pb.BackupChapter{
				Url:            kc.Url,
				Name:           kc.Name,
				Scanlator:      stringPtr(kc.Scanlator),
				Read:           false,
				Bookmark:       false,
				LastPageRead:   0,
				ChapterNumber:  kc.Number,
				DateFetch:      0,
				DateUpload:     kc.UploadDate,
				SourceOrder:    0,
				LastModifiedAt: 0,
				Version:        1,
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
		sourceID := generateSourceID(km.Source)
		if _, exists := sourceMap[km.Source]; !exists {
			sourceMap[km.Source] = sourceID
			// Try to get the Mihon source name, fall back to Kotatsu name
			sourceName := km.Source
			if id, name, found := LookupKnownSource(km.Source); found {
				sourceName = name
				sourceID = id
			}
			backupSources = append(backupSources, &pb.BackupSource{
				Name:     sourceName,
				SourceId: sourceID,
			})
		}

		m := &pb.BackupManga{
			Source:         sourceID, // Now using generated source ID
			Url:            km.Url,
			Title:          km.Title,
			Author:         stringPtr(km.Author),
			Artist:         stringPtr(""),
			Description:    stringPtr(""),
			Genre:          []string{},
			Status:         0,
			ThumbnailUrl:   stringPtr(km.CoverUrl),
			DateAdded:      fav.CreatedAt,
			Viewer:         0,
			Chapters:       chaptersByManga[km.Id],
			Categories:     []int64{fav.CategoryId},
			Favorite:       true,
			ChapterFlags:   0,
			ViewerFlags:    nil,
			UpdateStrategy: 0, // ALWAYS_UPDATE
			LastModifiedAt: fav.CreatedAt,
			Version:        1,
			Initialized:    true, // Mark as initialized
		}
		b.BackupManga = append(b.BackupManga, m)
	}

	// Convert categories
	for _, c := range kb.Categories {
		b.BackupCategories = append(b.BackupCategories, &pb.BackupCategory{
			Name:  c.Title,
			Order: c.CreatedAt,
			Id:    c.CategoryId,
			Flags: 0,
		})
	}

	// Add the source mappings
	b.BackupSources = backupSources

	return b
}
