// create_avatars_catalog scans each avatar's Spine JSON under avatars/ and writes
// catalog/<slug>.json for each. The backend fetches these from the CDN when the
// client calls avatar/get_character_catalog.
//
// Usage (from kalpix-avatars repo root):
//
//	go run ./scripts/create_avatars_catalog
//
// Optional: -cdn-base to set previewUrl base (e.g. https://cdn.example.com/kalpix-avatars).
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

// spineAsset is a minimal view of the Spine JSON for skins and animations.
type spineAsset struct {
	Skins      []struct{ Name string `json:"name"` } `json:"skins"`
	Animations map[string]interface{}               `json:"animations"`
}

// catalogOutput matches backend/catalog shape; backend adds avatarId when serving.
type catalogOutput struct {
	Slug       string        `json:"slug"`
	AvatarName string        `json:"avatarName"`
	Categories []categoryOut `json:"categories"`
}

type categoryOut struct {
	Key           string          `json:"key"`
	Label         string          `json:"label"`
	Subcategories []subcategoryOut `json:"subcategories"`
}

type subcategoryOut struct {
	Key     string      `json:"key"`
	Label   string      `json:"label"`
	Options []optionOut `json:"options"`
}

type optionOut struct {
	OptionID   string `json:"optionId"`
	Label      string `json:"label"`
	SkinName   string `json:"skinName,omitempty"`
	PreviewURL string `json:"previewUrl,omitempty"`
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

func topCategory(subKey string) string {
	lower := strings.ToLower(subKey)
	switch lower {
	case "dress", "shoes", "watch", "fan":
		return "fashion"
	case "eyebrow", "eyes", "face", "hair", "lips":
		return "body"
	case "animation":
		return "animation"
	}
	return "body"
}

var (
	bodyOrder    = []string{"eyebrow", "eyes", "face", "Hair", "hair", "lips"}
	fashionOrder = []string{"dress", "shoes", "watch", "fan"}
)

func main() {
	avatarsDir := flag.String("avatars", "avatars", "Path to avatars folder (relative to cwd)")
	catalogDir := flag.String("catalog", "catalog", "Path to catalog output folder")
	cdnBase := flag.String("cdn-base", "", "Optional CDN base for previewUrl (e.g. https://cdn.example.com/kalpix-avatars)")
	flag.Parse()

	dirs, err := os.ReadDir(*avatarsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*catalogDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	written := 0
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		slug := d.Name()
		assetPath := filepath.Join(*avatarsDir, slug, slug+".json")
		if _, err := os.Stat(assetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Skip %s: %s not found\n", slug, assetPath)
			continue
		}

		catalog, err := buildCatalog(assetPath, slug, humanize(slug), *cdnBase)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Build catalog for %s: %v\n", slug, err)
			continue
		}

		outPath := filepath.Join(*catalogDir, slug+".json")
		out, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Create %s: %v\n", outPath, err)
			continue
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "\t")
		if err := enc.Encode(catalog); err != nil {
			out.Close()
			fmt.Fprintf(os.Stderr, "Encode %s: %v\n", outPath, err)
			continue
		}
		out.Close()
		fmt.Printf("Wrote %s\n", outPath)
		written++
	}

	fmt.Printf("Done: %d catalog(s) written to %s\n", written, *catalogDir)
}

func buildCatalog(assetPath, slug, avatarName, cdnBase string) (*catalogOutput, error) {
	data, err := os.ReadFile(assetPath)
	if err != nil {
		return nil, err
	}

	var asset spineAsset
	if err := json.Unmarshal(data, &asset); err != nil {
		return nil, fmt.Errorf("parse Spine JSON: %w", err)
	}

	// subcategory key -> sorted option IDs (from skin names "sub/optionId")
	subcategoryOptions := make(map[string][]string)
	for _, skin := range asset.Skins {
		name := strings.TrimSpace(skin.Name)
		if name == "" || strings.ToLower(name) == "default" {
			continue
		}
		if !strings.Contains(name, "/") {
			continue
		}
		parts := strings.SplitN(name, "/", 2)
		left, right := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if left == "" || right == "" {
			continue
		}
		opts := subcategoryOptions[left]
		seen := false
		for _, o := range opts {
			if o == right {
				seen = true
				break
			}
		}
		if !seen {
			subcategoryOptions[left] = append(opts, right)
		}
	}

	// Animation names from animations object keys
	if asset.Animations != nil {
		var animOpts []string
		for k := range asset.Animations {
			if strings.TrimSpace(strings.ToLower(k)) != "default" {
				animOpts = append(animOpts, k)
			}
		}
		if len(animOpts) > 0 {
			sort.Strings(animOpts)
			subcategoryOptions["animation"] = animOpts
		}
	}

	for k := range subcategoryOptions {
		sort.Strings(subcategoryOptions[k])
	}

	bodySubs := make(map[string][]string)
	fashionSubs := make(map[string][]string)
	animationSubs := make(map[string][]string)
	for k, opts := range subcategoryOptions {
		switch topCategory(k) {
		case "body":
			bodySubs[k] = opts
		case "fashion":
			fashionSubs[k] = opts
		case "animation":
			animationSubs[k] = opts
		default:
			bodySubs[k] = opts
		}
	}

	previewBase := ""
	if cdnBase != "" {
		previewBase = strings.TrimSuffix(cdnBase, "/") + "/catalog/" + slug
	}

	var categories []categoryOut
	if len(bodySubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "body",
			Label:         "Body",
			Subcategories: buildSubcategoryList(bodyOrder, bodySubs, slug, previewBase, false),
		})
	}
	if len(fashionSubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "fashion",
			Label:         "Fashion",
			Subcategories: buildSubcategoryList(fashionOrder, fashionSubs, slug, previewBase, false),
		})
	}
	if len(animationSubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "animation",
			Label:         "Animation",
			Subcategories: buildSubcategoryList([]string{"animation"}, animationSubs, slug, previewBase, true),
		})
	}

	return &catalogOutput{
		Slug:       slug,
		AvatarName: avatarName,
		Categories: categories,
	}, nil
}

func buildSubcategoryList(order []string, all map[string][]string, slug, previewBase string, isAnimation bool) []subcategoryOut {
	seen := make(map[string]bool)
	var result []subcategoryOut
	for _, k := range order {
		opts, ok := all[k]
		if !ok || seen[k] {
			continue
		}
		seen[k] = true
		var options []optionOut
		for i, oid := range opts {
			opt := optionOut{OptionID: oid, Label: humanize(oid)}
			if !isAnimation {
				opt.SkinName = k + "/" + oid
				if previewBase != "" {
					opt.PreviewURL = previewBase + "/" + k + "/" + oid + ".webp"
				} else {
					opt.PreviewURL = "catalog/" + slug + "/" + k + "/" + fmt.Sprintf("%d", i+1) + ".webp"
				}
			}
			options = append(options, opt)
		}
		result = append(result, subcategoryOut{Key: k, Label: k, Options: options})
	}
	for k, opts := range all {
		if seen[k] {
			continue
		}
		var options []optionOut
		for i, oid := range opts {
			opt := optionOut{OptionID: oid, Label: humanize(oid)}
			if !isAnimation {
				opt.SkinName = k + "/" + oid
				if previewBase != "" {
					opt.PreviewURL = previewBase + "/" + k + "/" + oid + ".webp"
				} else {
					opt.PreviewURL = "catalog/" + slug + "/" + k + "/" + fmt.Sprintf("%d", i+1) + ".webp"
				}
			}
			options = append(options, opt)
		}
		result = append(result, subcategoryOut{Key: k, Label: k, Options: options})
	}
	return result
}
