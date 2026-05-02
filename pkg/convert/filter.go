package convert

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/galpt/mk-bkconv/proto/mihon"
)

// resolveReferencesRoot returns the path to the references directory.
// It checks the REFERENCES_ROOT env var first, then falls back to
// ../references relative to the working directory.
func resolveReferencesRoot() string {
	refRoot := os.Getenv("REFERENCES_ROOT")
	if refRoot != "" {
		return refRoot
	}
	cwd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(cwd, "..", "..", "references")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// FilterBackupToCommon removes mangas and sources from the Mihon backup
// that don't have matching sources available in both Kotatsu and Mihon.
// It attempts to discover Mihon extension names from a references folder
// (ENV "REFERENCES_ROOT" or ../references by default). If discovery fails
// it falls back to KnownSourceMapping as a conservative whitelist.
func FilterBackupToCommon(b *pb.Backup, kotatsuRawSources []byte) {
	refRoot := resolveReferencesRoot()

	mihonNames := make(map[string]struct{})
	// Seed from KnownSourceMapping values (guaranteed known mappings)
	for _, m := range KnownSourceMapping {
		mihonNames[strings.ToLower(m.MihonName)] = struct{}{}
	}

	// If we have a references root, try to walk and discover extension directories
	if refRoot != "" {
		if err := filepath.Walk(refRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			// look for Kotlin/Java extension files path that include the tachiyomi extension package
			p := filepath.ToSlash(path)
			if strings.Contains(p, "/eu/kanade/tachiyomi/extension/") && strings.HasSuffix(p, ".kt") {
				// attempt to extract extension short name (the parent folder under extension)
				parts := strings.Split(p, "/eu/kanade/tachiyomi/extension/")
				if len(parts) > 1 {
					rest := parts[1]
					segs := strings.Split(rest, "/")
					if len(segs) > 0 {
						mihonNames[strings.ToLower(segs[0])] = struct{}{}
					}
				}
			}
			return nil
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: error walking references directory %s: %v\n", refRoot, err)
		}
	}

	// Build allowed ID set from mihonNames — this includes both KnownSourceMapping
	// entries and any names discovered from the references walk.
	allowedIDs := make(map[int64]struct{})
	for name := range mihonNames {
		// Check if it's a known mapping first
		found := false
		for _, m := range KnownSourceMapping {
			if strings.ToLower(m.MihonName) == name {
				id := GenerateMihonSourceID(m.MihonName, m.MihonLang, m.MihonVersionID)
				allowedIDs[id] = struct{}{}
				found = true
				break
			}
		}
		if !found {
			// Discovered from references — use FNV hash as fallback
			h := fnv.New64a()
			_, _ = h.Write([]byte(name))
			id := int64(h.Sum64())
			allowedIDs[id] = struct{}{}
		}
	}

	// parse kotatsuRawSources if available to extract kotatsu source names (best-effort)
	kotatsuNames := make(map[string]struct{})
	if len(kotatsuRawSources) > 0 {
		var arr []map[string]any
		if err := json.Unmarshal(kotatsuRawSources, &arr); err == nil {
			for _, el := range arr {
				if s, ok := el["name"].(string); ok {
					kotatsuNames[strings.ToLower(s)] = struct{}{}
				}
			}
		}
	}

	// If allowedIDs is empty, fall back to allowing all KnownSourceMapping IDs
	if len(allowedIDs) == 0 {
		for k := range KnownSourceMapping {
			m := KnownSourceMapping[k]
			id := GenerateMihonSourceID(m.MihonName, m.MihonLang, m.MihonVersionID)
			allowedIDs[id] = struct{}{}
		}
	}

	// Filter BackupManga entries: keep only those whose Source ID is in allowedIDs
	var kept []*pb.BackupManga
	for _, m := range b.BackupManga {
		if _, ok := allowedIDs[m.GetSource()]; ok {
			kept = append(kept, m)
			continue
		}
		// If source not by id, try to match by name from BackupSources
		foundByName := false
		for _, s := range b.BackupSources {
			if s.GetSourceId() == m.GetSource() {
				if s.GetName() != "" {
					if _, ok := mihonNames[strings.ToLower(s.GetName())]; ok {
						foundByName = true
						break
					}
				}
			}
		}
		if foundByName {
			kept = append(kept, m)
		}
	}
	b.BackupManga = kept

	// Filter BackupSources similarly
	var keptSources []*pb.BackupSource
	for _, s := range b.BackupSources {
		if _, ok := allowedIDs[s.GetSourceId()]; ok {
			keptSources = append(keptSources, s)
			continue
		}
		if s.GetName() != "" {
			if _, ok := mihonNames[strings.ToLower(s.GetName())]; ok {
				keptSources = append(keptSources, s)
			}
		}
	}
	b.BackupSources = keptSources
}

// FilterMihonForKotatsu removes Mihon backup entries that don't have a corresponding
// Kotatsu source available. It attempts to discover Kotatsu parser names from
// references (ENV "REFERENCES_ROOT" or ../references by default) and falls back
// to KnownSourceMapping keys if discovery fails.
func FilterMihonForKotatsu(b *pb.Backup) {
	refRoot := resolveReferencesRoot()

	kotatsuNames := make(map[string]struct{})
	// Seed from KnownSourceMapping keys
	for k := range KnownSourceMapping {
		kotatsuNames[strings.ToLower(k)] = struct{}{}
	}

	if refRoot != "" {
		// look for kotatsu-parsers-master repo
		if err := filepath.Walk(refRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}
			p := filepath.ToSlash(path)
			// match paths like .../kotatsu-parsers-master/src/main/kotlin/.../site/<sourcename>
			if strings.Contains(p, "/kotatsu-parsers-master/src/main/kotlin/") && strings.Contains(p, "/site/") {
				// extract after /site/
				parts := strings.Split(p, "/site/")
				if len(parts) > 1 {
					segs := strings.Split(parts[1], "/")
					if len(segs) > 0 && segs[0] != "" {
						kotatsuNames[strings.ToLower(segs[0])] = struct{}{}
					}
				}
			}
			return nil
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: error walking references directory %s: %v\n", refRoot, err)
		}
	}

	// Build allowed Mihon IDs for kotatsu-supported sources.
	// Iterate over kotatsuNames (which includes both seeded KnownSourceMapping keys
	// and any names discovered from the references walk) so the walk actually influences
	// which sources are allowed through.
	allowedIDs := make(map[int64]struct{})
	for name := range kotatsuNames {
		if m, exists := KnownSourceMapping[strings.ToUpper(name)]; exists {
			id := GenerateMihonSourceID(m.MihonName, m.MihonLang, m.MihonVersionID)
			allowedIDs[id] = struct{}{}
		}
	}

	// If allowedIDs empty, keep existing backup untouched (conservative)
	if len(allowedIDs) == 0 {
		return
	}

	// Filter BackupManga and BackupSources
	var kept []*pb.BackupManga
	for _, m := range b.BackupManga {
		if _, ok := allowedIDs[m.GetSource()]; ok {
			kept = append(kept, m)
		}
	}
	b.BackupManga = kept

	var keptSources []*pb.BackupSource
	for _, s := range b.BackupSources {
		if _, ok := allowedIDs[s.GetSourceId()]; ok {
			keptSources = append(keptSources, s)
		}
	}
	b.BackupSources = keptSources
}
