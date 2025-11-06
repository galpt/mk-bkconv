package mihon

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

// Minimal subset of Mihon Backup model to support core fields.
type Backup struct {
	Mangas     []BackupManga
	Categories []BackupCategory
}

type BackupManga struct {
	Source       int64
	Url          string
	Title        string
	Author       string
	Artist       string
	Description  string
	Genres       []string
	Status       int32
	ThumbnailUrl string
	DateAdded    int64
	Chapters     []BackupChapter
	Categories   []int64
}

type BackupChapter struct {
	Url           string
	Name          string
	Scanlator     string
	Read          bool
	Bookmark      bool
	LastPageRead  int64
	ChapterNumber float32
}

type BackupCategory struct {
	Name  string
	Order int64
	Id    int64
}

// LoadBackup reads a Mihon backup file (.tachibk), auto-detects gzip, and parses protobuf.
func LoadBackup(path string) (*Backup, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read first two bytes to detect gzip like Mihon does
	hdr := make([]byte, 2)
	if _, err := f.Read(hdr); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var data []byte
	// 0x1f8b is gzip
	if int(hdr[0])|int(hdr[1])<<8 == 0x1f8b {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		data, err = io.ReadAll(gr)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = io.ReadAll(f)
		if err != nil {
			return nil, err
		}
	}

	p := &protoReader{data: data}
	return parseBackup(p)
}

// WriteBackup writes Mihon protobuf-encoded backup bytes and gzips them to path.
// This is a minimal encoder for core fields and may be extended.
func WriteBackup(path string, b *Backup) error {
	buf := &bytes.Buffer{}
	pw := &protoWriter{buf: buf}

	// field 1: repeated BackupManga
	for _, m := range b.Mangas {
		mBuf := &bytes.Buffer{}
		mw := &protoWriter{buf: mBuf}
		// write source (field 1, varint)
		mw.WriteVarintField(1, uint64(m.Source))
		mw.WriteStringField(2, m.Url)
		mw.WriteStringField(3, m.Title)
		mw.WriteStringField(5, m.Author)
		mw.WriteStringField(4, m.Artist)
		mw.WriteStringField(6, m.Description)
		for _, g := range m.Genres {
			mw.WriteStringField(7, g)
		}
		mw.WriteVarintField(8, uint64(m.Status))
		mw.WriteStringField(9, m.ThumbnailUrl)
		mw.WriteVarintField(13, uint64(m.DateAdded))
		// chapters (field 16)
		for _, ch := range m.Chapters {
			cb := &bytes.Buffer{}
			cw := &protoWriter{buf: cb}
			cw.WriteStringField(1, ch.Url)
			cw.WriteStringField(2, ch.Name)
			cw.WriteStringField(3, ch.Scanlator)
			if ch.Read {
				cw.WriteVarintField(4, 1)
			}
			if ch.Bookmark {
				cw.WriteVarintField(5, 1)
			}
			cw.WriteVarintField(6, uint64(ch.LastPageRead))
			// chapterNumber as 32-bit float (field 9, wire type 5)
			cw.WriteFloat32Field(9, ch.ChapterNumber)
			mw.WriteBytesField(16, cb.Bytes())
		}
		for _, cid := range m.Categories {
			mw.WriteVarintField(17, uint64(cid))
		}
		pw.WriteBytesField(1, mBuf.Bytes())
	}

	// field 2: categories
	for _, c := range b.Categories {
		cb := &bytes.Buffer{}
		cw := &protoWriter{buf: cb}
		cw.WriteStringField(1, c.Name)
		cw.WriteVarintField(2, uint64(c.Order))
		cw.WriteVarintField(3, uint64(c.Id))
		pw.WriteBytesField(2, cb.Bytes())
	}

	// gzip and write
	outf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outf.Close()
	gw := gzip.NewWriter(outf)
	defer gw.Close()
	_, err = gw.Write(buf.Bytes())
	return err
}

// ---- Minimal Protobuf reader/writer ----

type protoReader struct {
	data []byte
	i    int
}

// readByte removed — not used. We operate directly on r.data and r.i in readers.

func (r *protoReader) ReadVarint() (uint64, error) {
	var x uint64
	var s uint
	for i := 0; ; i++ {
		if r.i >= len(r.data) {
			return 0, io.EOF
		}
		b := r.data[r.i]
		r.i++
		if b < 0x80 {
			return x | uint64(b)<<s, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
		if s >= 64 {
			return 0, errors.New("varint overflow")
		}
	}
}

func (r *protoReader) ReadBytes(n int) ([]byte, error) {
	if r.i+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	bs := r.data[r.i : r.i+n]
	r.i += n
	return bs, nil
}

func (r *protoReader) ReadTag() (field int, wire int, err error) {
	v, err := r.ReadVarint()
	if err != nil {
		return 0, 0, err
	}
	field = int(v >> 3)
	wire = int(v & 0x7)
	return
}

// parseBackup parses the minimal subset used by the tool.
func parseBackup(r *protoReader) (*Backup, error) {
	b := &Backup{}
	for r.i < len(r.data) {
		field, wire, err := r.ReadTag()
		if err != nil {
			return nil, err
		}
		switch field {
		case 1: // backupManga repeated message
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for backupManga: %d", wire)
			}
			l, err := r.ReadVarint()
			if err != nil {
				return nil, err
			}
			bs, err := r.ReadBytes(int(l))
			if err != nil {
				return nil, err
			}
			m, err := parseManga(&protoReader{data: bs})
			if err != nil {
				return nil, err
			}
			b.Mangas = append(b.Mangas, *m)
		case 2: // categories
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for categories: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			c, err := parseCategory(&protoReader{data: bs})
			if err != nil {
				return nil, err
			}
			b.Categories = append(b.Categories, *c)
		default:
			// skip unknown
			switch wire {
			case 0:
				_, _ = r.ReadVarint()
			case 1:
				_, _ = r.ReadBytes(8)
			case 2:
				l, _ := r.ReadVarint()
				_, _ = r.ReadBytes(int(l))
			case 5:
				_, _ = r.ReadBytes(4)
			}
		}
	}
	return b, nil
}

func parseManga(r *protoReader) (*BackupManga, error) {
	m := &BackupManga{}
	for r.i < len(r.data) {
		field, wire, err := r.ReadTag()
		if err != nil {
			return nil, err
		}
		switch field {
		case 1:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for source: %d", wire)
			}
			v, _ := r.ReadVarint()
			m.Source = int64(v)
		case 2:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for url: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.Url = string(bs)
		case 3:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for title: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.Title = string(bs)
		case 4:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for artist: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.Artist = string(bs)
		case 5:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for author: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.Author = string(bs)
		case 6:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for description: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.Description = string(bs)
		case 7:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for genre: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.Genres = append(m.Genres, string(bs))
		case 8:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for status: %d", wire)
			}
			v, _ := r.ReadVarint()
			m.Status = int32(v)
		case 9:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for thumbnail: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			m.ThumbnailUrl = string(bs)
		case 13:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for dateAdded: %d", wire)
			}
			v, _ := r.ReadVarint()
			m.DateAdded = int64(v)
		case 16:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for chapters: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			ch, err := parseChapter(&protoReader{data: bs})
			if err != nil {
				return nil, err
			}
			m.Chapters = append(m.Chapters, *ch)
		case 17:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for categories: %d", wire)
			}
			v, _ := r.ReadVarint()
			m.Categories = append(m.Categories, int64(v))
		default:
			// skip
			switch wire {
			case 0:
				_, _ = r.ReadVarint()
			case 1:
				_, _ = r.ReadBytes(8)
			case 2:
				l, _ := r.ReadVarint()
				_, _ = r.ReadBytes(int(l))
			case 5:
				_, _ = r.ReadBytes(4)
			}
		}
	}
	return m, nil
}

func parseChapter(r *protoReader) (*BackupChapter, error) {
	ch := &BackupChapter{}
	for r.i < len(r.data) {
		field, wire, err := r.ReadTag()
		if err != nil {
			return nil, err
		}
		switch field {
		case 1:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for url: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			ch.Url = string(bs)
		case 2:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for name: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			ch.Name = string(bs)
		case 3:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for scanlator: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			ch.Scanlator = string(bs)
		case 4:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for read: %d", wire)
			}
			v, _ := r.ReadVarint()
			ch.Read = v != 0
		case 5:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for bookmark: %d", wire)
			}
			v, _ := r.ReadVarint()
			ch.Bookmark = v != 0
		case 6:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for lastPageRead: %d", wire)
			}
			v, _ := r.ReadVarint()
			ch.LastPageRead = int64(v)
		case 9:
			if wire != 5 {
				return nil, fmt.Errorf("unexpected wire for chapterNumber: %d", wire)
			}
			bs, _ := r.ReadBytes(4)
			ch.ChapterNumber = mathFromBits32(binary.LittleEndian.Uint32(bs))
		default:
			switch wire {
			case 0:
				_, _ = r.ReadVarint()
			case 1:
				_, _ = r.ReadBytes(8)
			case 2:
				l, _ := r.ReadVarint()
				_, _ = r.ReadBytes(int(l))
			case 5:
				_, _ = r.ReadBytes(4)
			}
		}
	}
	return ch, nil
}

func parseCategory(r *protoReader) (*BackupCategory, error) {
	c := &BackupCategory{}
	for r.i < len(r.data) {
		field, wire, err := r.ReadTag()
		if err != nil {
			return nil, err
		}
		switch field {
		case 1:
			if wire != 2 {
				return nil, fmt.Errorf("unexpected wire for name: %d", wire)
			}
			l, _ := r.ReadVarint()
			bs, _ := r.ReadBytes(int(l))
			c.Name = string(bs)
		case 2:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for order: %d", wire)
			}
			v, _ := r.ReadVarint()
			c.Order = int64(v)
		case 3:
			if wire != 0 {
				return nil, fmt.Errorf("unexpected wire for id: %d", wire)
			}
			v, _ := r.ReadVarint()
			c.Id = int64(v)
		default:
			switch wire {
			case 0:
				_, _ = r.ReadVarint()
			case 1:
				_, _ = r.ReadBytes(8)
			case 2:
				l, _ := r.ReadVarint()
				_, _ = r.ReadBytes(int(l))
			case 5:
				_, _ = r.ReadBytes(4)
			}
		}
	}
	return c, nil
}

// ---- protoWriter ----

type protoWriter struct {
	buf *bytes.Buffer
}

func (w *protoWriter) WriteVarintField(field int, v uint64) {
	w.WriteTag(field, 0)
	w.WriteVarint(v)
}

func (w *protoWriter) WriteStringField(field int, s string) {
	if s == "" {
		return
	}
	w.WriteTag(field, 2)
	w.WriteVarint(uint64(len(s)))
	w.buf.WriteString(s)
}

func (w *protoWriter) WriteBytesField(field int, b []byte) {
	w.WriteTag(field, 2)
	w.WriteVarint(uint64(len(b)))
	w.buf.Write(b)
}

func (w *protoWriter) WriteFloat32Field(field int, v float32) {
	w.WriteTag(field, 5)
	// little endian
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, mathToBits32(v))
	w.buf.Write(b)
}

func (w *protoWriter) WriteTag(field int, wire int) {
	w.WriteVarint(uint64(field<<3 | wire))
}

func (w *protoWriter) WriteVarint(v uint64) {
	for v >= 0x80 {
		w.buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	w.buf.WriteByte(byte(v))
}

// helpers for float32 bits
func mathToBits32(f float32) uint32 {
	return math.Float32bits(f)
}

func mathFromBits32(b uint32) float32 {
	return math.Float32frombits(b)
}

// errors package is used in ReadVarint; no additional blank references required.
