package kotatsu

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Minimal Kotatsu models used for conversion
type KotatsuBackup struct {
	Favourites []KotatsuFavouriteEntry `json:"favourites"`
	Categories []KotatsuCategory       `json:"categories"`
	History    []KotatsuHistory        `json:"history"`
	Bookmarks  []KotatsuBookmark       `json:"bookmarks"`
	Index      []KotatsuIndexEntry     `json:"index"`
	// Raw sections (for passthrough)
	RawSettings   json.RawMessage `json:"-"`
	RawReaderGrid json.RawMessage `json:"-"`
	RawSources    json.RawMessage `json:"-"`
}

type KotatsuFavouriteEntry struct {
	MangaId    int64        `json:"manga_id"`
	CategoryId int64        `json:"category_id"`
	SortKey    int          `json:"sort_key"`
	Pinned     bool         `json:"pinned"`
	CreatedAt  int64        `json:"created_at"`
	Manga      KotatsuManga `json:"manga"`
}

type KotatsuManga struct {
	Id            int64   `json:"id"`
	Title         string  `json:"title"`
	AltTitle      string  `json:"alt_title"`
	Url           string  `json:"url"`
	PublicUrl     string  `json:"public_url"`
	Rating        float32 `json:"rating"`
	Nsfw          bool    `json:"nsfw"`
	ContentRating string  `json:"content_rating"`
	CoverUrl      string  `json:"cover_url"`
	LargeCover    string  `json:"large_cover_url"`
	State         string  `json:"state"`
	Author        string  `json:"author"`
	Source        string  `json:"source"`
	Tags          []any   `json:"tags"`
}

type KotatsuCategory struct {
	CategoryId int64  `json:"category_id"`
	CreatedAt  int64  `json:"created_at"`
	SortKey    int    `json:"sort_key"`
	Title      string `json:"title"`
}

type KotatsuHistory struct {
	MangaId   int64   `json:"manga_id"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
	ChapterId int64   `json:"chapter_id"`
	Page      int     `json:"page"`
	Scroll    float64 `json:"scroll"`
	Percent   float32 `json:"percent"`
}

type KotatsuBookmark struct {
	MangaId   int64   `json:"manga_id"`
	PageId    int64   `json:"page_id"`
	ChapterId int64   `json:"chapter_id"`
	Page      int     `json:"page"`
	Scroll    float64 `json:"scroll"`
	ImageUrl  string  `json:"image_url"`
	CreatedAt int64   `json:"created_at"`
	Percent   float32 `json:"percent"`
}

type KotatsuIndexEntry struct {
	MangaId  int64            `json:"manga_id"`
	Chapters []KotatsuChapter `json:"chapters"`
}

type KotatsuChapter struct {
	Id         int64   `json:"id"`
	Name       string  `json:"name"`
	Number     float32 `json:"number"`
	Url        string  `json:"url"`
	Scanlator  string  `json:"scanlator"`
	UploadDate int64   `json:"upload_date"`
	Branch     string  `json:"branch"`
}

// LoadKotatsuZip reads a Kotatsu zip and returns parsed backup data.
func LoadKotatsuZip(path string) (*KotatsuBackup, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	kb := &KotatsuBackup{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		switch f.Name {
		case "favourites":
			var arr []KotatsuFavouriteEntry
			if err := json.NewDecoder(rc).Decode(&arr); err != nil {
				rc.Close()
				return nil, fmt.Errorf("decode favourites: %w", err)
			}
			kb.Favourites = arr
		case "categories":
			var arr []KotatsuCategory
			if err := json.NewDecoder(rc).Decode(&arr); err != nil {
				rc.Close()
				return nil, fmt.Errorf("decode categories: %w", err)
			}
			kb.Categories = arr
		case "history":
			var arr []KotatsuHistory
			if err := json.NewDecoder(rc).Decode(&arr); err != nil {
				rc.Close()
				return nil, fmt.Errorf("decode history: %w", err)
			}
			kb.History = arr
		case "bookmarks":
			var arr []KotatsuBookmark
			if err := json.NewDecoder(rc).Decode(&arr); err != nil {
				rc.Close()
				return nil, fmt.Errorf("decode bookmarks: %w", err)
			}
			kb.Bookmarks = arr
		case "index":
			var arr []KotatsuIndexEntry
			if err := json.NewDecoder(rc).Decode(&arr); err != nil {
				rc.Close()
				return nil, fmt.Errorf("decode index: %w", err)
			}
			kb.Index = arr
		case "settings", "reader_grid", "sources":
			// Read raw bytes for passthrough
			buf, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, fmt.Errorf("read %s: %w", f.Name, err)
			}
			switch f.Name {
			case "settings":
				kb.RawSettings = buf
			case "reader_grid":
				kb.RawReaderGrid = buf
			case "sources":
				kb.RawSources = buf
			}
		}
		rc.Close()
	}
	return kb, nil
}

// WriteKotatsuZip writes a minimal Kotatsu zip containing favourites and categories JSON arrays.
// It also passes through any raw sections (settings, reader_grid, sources) that were read from
// the source backup, preserving custom configuration.
func WriteKotatsuZip(path string, kb *KotatsuBackup) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	add := func(name string, v interface{}) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}

	if err := add("favourites", kb.Favourites); err != nil {
		return fmt.Errorf("write favourites: %w", err)
	}
	if err := add("categories", kb.Categories); err != nil {
		return fmt.Errorf("write categories: %w", err)
	}

	// Passthrough raw sections — these are raw JSON bytes, so write them directly
	// to avoid double-encoding through json.Encoder.
	writeRaw := func(name string, data []byte) error {
		if data == nil {
			return nil
		}
		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("create %s entry: %w", name, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
		return nil
	}

	if err := writeRaw("settings", kb.RawSettings); err != nil {
		return err
	}
	if err := writeRaw("reader_grid", kb.RawReaderGrid); err != nil {
		return err
	}
	if err := writeRaw("sources", kb.RawSources); err != nil {
		return err
	}

	return nil
}
