package convert

import (
	"github.com/galpt/mk-bkconv/pkg/kotatsu"
	"github.com/galpt/mk-bkconv/pkg/mihon"
)

func MihonToKotatsu(b *mihon.Backup) *kotatsu.KotatsuBackup {
	kb := &kotatsu.KotatsuBackup{}
	for i, m := range b.Mangas {
		km := kotatsu.KotatsuManga{
			Id:         int64(i + 1),
			Title:      m.Title,
			Url:        m.Url,
			PublicUrl:  m.Url,
			CoverUrl:   m.ThumbnailUrl,
			LargeCover: m.ThumbnailUrl,
			Authors:    m.Author,
			Source:     "",
			Tags:       []interface{}{},
		}
		kb.Favourites = append(kb.Favourites, km)
	}
	for _, c := range b.Categories {
		kb.Categories = append(kb.Categories, kotatsu.KotatsuCategory{
			CategoryId: c.Id,
			CreatedAt:  c.Order,
			SortKey:    0,
			Title:      c.Name,
		})
	}
	return kb
}

func KotatsuToMihon(kb *kotatsu.KotatsuBackup) *mihon.Backup {
	b := &mihon.Backup{}
	for _, km := range kb.Favourites {
		m := mihon.BackupManga{
			Source:       0,
			Url:          km.Url,
			Title:        km.Title,
			Author:       km.Authors,
			Artist:       "",
			Description:  "",
			Genres:       []string{},
			Status:       0,
			ThumbnailUrl: km.CoverUrl,
			DateAdded:    0,
			Chapters:     []mihon.BackupChapter{},
			Categories:   []int64{},
		}
		b.Mangas = append(b.Mangas, m)
	}
	for _, c := range kb.Categories {
		b.Categories = append(b.Categories, mihon.BackupCategory{
			Name:  c.Title,
			Order: c.CreatedAt,
			Id:    c.CategoryId,
		})
	}
	return b
}
