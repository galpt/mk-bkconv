package kotatsu

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
)

// Minimal Kotatsu models used for conversion
type KotatsuBackup struct {
	Favourites []KotatsuManga    `json:"favourites"`
	Categories []KotatsuCategory `json:"categories"`
}

type KotatsuManga struct {
	Id         int64         `json:"id"`
	Title      string        `json:"title"`
	Url        string        `json:"url"`
	PublicUrl  string        `json:"public_url"`
	CoverUrl   string        `json:"cover_url"`
	LargeCover string        `json:"large_cover_url"`
	Authors    string        `json:"author"`
	Source     string        `json:"source"`
	Tags       []interface{} `json:"tags"`
}

type KotatsuCategory struct {
	CategoryId int64  `json:"category_id"`
	CreatedAt  int64  `json:"created_at"`
	SortKey    int    `json:"sort_key"`
	Title      string `json:"title"`
}

// LoadKotatsuZip reads a Kotatsu zip and returns parsed favourites and categories.
func LoadKotatsuZip(path string) (*KotatsuBackup, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	kb := &KotatsuBackup{}
	for _, f := range r.File {
		switch f.Name {
		case "favourites":
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			dec := json.NewDecoder(rc)
			// JSON is an array
			var arr []KotatsuManga
			if err := dec.Decode(&arr); err != nil {
				rc.Close()
				return nil, err
			}
			rc.Close()
			kb.Favourites = arr
		case "categories":
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			var arr []KotatsuCategory
			if err := json.NewDecoder(rc).Decode(&arr); err != nil {
				rc.Close()
				return nil, err
			}
			rc.Close()
			kb.Categories = arr
		default:
			// skip
		}
	}
	return kb, nil
}

// WriteKotatsuZip writes a minimal Kotatsu zip containing favourites and categories JSON arrays.
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
		enc.SetIndent("", "")
		return enc.Encode(v)
	}

	if err := add("favourites", kb.Favourites); err != nil {
		return fmt.Errorf("write favourites: %w", err)
	}
	if err := add("categories", kb.Categories); err != nil {
		return fmt.Errorf("write categories: %w", err)
	}
	return nil
}
