// create_avatars_list scans the avatars/ folder and writes avatars_list.json
// for upload to Cloudflare. The backend fetches this file when AVATAR_MANIFEST_URL
// points to it and serves it on avatar/list_avatars.
//
// Usage (from kalpix-avatars repo root):
//
//	go run ./scripts/create_avatars_list
//
// Optional: -base-url=https://your-cdn.com/kalpix-avatars to emit full URLs in the manifest.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestEntry matches the backend's expected manifest array element.
// Backend fills missing URL fields from its CDN config when not set.
type ManifestEntry struct {
	Slug         string `json:"slug"`
	AvatarName   string `json:"avatarName"`
	PreviewURL   string `json:"previewUrl,omitempty"`
	BaseAtlasURL string `json:"baseAtlasUrl,omitempty"`
	BaseJSONURL  string `json:"baseJsonUrl,omitempty"`
	BasePNGURL   string `json:"basePngUrl,omitempty"`
	CatalogURL   string `json:"catalogUrl,omitempty"`
	IsActive     bool   `json:"isActive"`
	SortOrder    int    `json:"sortOrder,omitempty"`
}

func humanize(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "_", " ")
	parts := strings.Fields(s)
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

func main() {
	baseURL := flag.String("base-url", "", "Optional CDN base URL (e.g. https://pub-xxx.r2.dev/kalpix-avatars). If set, full URLs are written.")
	avatarsDir := flag.String("avatars", "avatars", "Path to avatars folder (relative to cwd)")
	output := flag.String("out", "avatars_list.json", "Output file path")
	flag.Parse()

	entries, err := scanAvatars(*avatarsDir, *baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "No avatars found under %s\n", *avatarsDir)
		os.Exit(1)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].SortOrder != entries[j].SortOrder {
			return entries[i].SortOrder < entries[j].SortOrder
		}
		return entries[i].AvatarName < entries[j].AvatarName
	})

	out, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Create output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(entries); err != nil {
		fmt.Fprintf(os.Stderr, "Encode JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s (%d avatars)\n", *output, len(entries))
}

func scanAvatars(avatarsDir, baseURL string) ([]ManifestEntry, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	assetsBase := baseURL + "/avatars"
	catalogBase := baseURL + "/catalog"

	dirs, err := os.ReadDir(avatarsDir)
	if err != nil {
		return nil, fmt.Errorf("read avatars dir: %w", err)
	}

	var entries []ManifestEntry
	sortOrder := 1
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		slug := d.Name()
		dirPath := filepath.Join(avatarsDir, slug)

		// Expect <slug>.json, <slug>.txt (atlas), <slug>.png
		baseName := slug
		jsonPath := filepath.Join(dirPath, baseName+".json")
		txtPath := filepath.Join(dirPath, baseName+".txt")

		if _, err := os.Stat(jsonPath); err != nil {
			// Try .atlas.txt if .txt missing
			atlasPath := filepath.Join(dirPath, baseName+".atlas.txt")
			if _, err2 := os.Stat(atlasPath); err2 != nil {
				fmt.Fprintf(os.Stderr, "Skip %s: no %s.json (or atlas) found\n", slug, baseName)
				continue
			}
			txtPath = atlasPath
		}

		e := ManifestEntry{
			Slug:       slug,
			AvatarName: humanize(slug),
			IsActive:   true,
			SortOrder:  sortOrder,
		}
		sortOrder++

		if baseURL != "" {
			e.PreviewURL = assetsBase + "/" + slug + "/" + baseName + ".png"
			e.BaseAtlasURL = assetsBase + "/" + slug + "/" + baseName + ".txt"
			e.BaseJSONURL = assetsBase + "/" + slug + "/" + baseName + ".json"
			e.BasePNGURL = assetsBase + "/" + slug + "/" + baseName + ".png"
			e.CatalogURL = catalogBase + "/" + slug + ".json"
			// If atlas is .atlas.txt, fix
			if _, err := os.Stat(txtPath); err != nil {
				atlasPath := filepath.Join(dirPath, baseName+".atlas.txt")
				if _, err2 := os.Stat(atlasPath); err2 == nil {
					e.BaseAtlasURL = assetsBase + "/" + slug + "/" + baseName + ".atlas.txt"
				}
			}
		}

		entries = append(entries, e)
	}

	return entries, nil
}
