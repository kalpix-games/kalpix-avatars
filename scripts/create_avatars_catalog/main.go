// create_avatars_catalog scans each avatar's Spine JSON under avatars/ and writes
// catalog/<slug>.json for each. The backend fetches these when the
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
	"strconv"
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

// sortOptionsNatural sorts option IDs by numeric suffix so hair_1, hair_2, ..., hair_9, hair_10, hair_11.
// Option IDs with no trailing number keep lexicographic order relative to each other.
func sortOptionsNatural(opts []string) {
	sort.Slice(opts, func(i, j int) bool {
		ni, nj := trailingNumber(opts[i]), trailingNumber(opts[j])
		if ni >= 0 && nj >= 0 {
			return ni < nj
		}
		if ni >= 0 {
			return true
		}
		if nj >= 0 {
			return false
		}
		return opts[i] < opts[j]
	})
}

// trailingNumber returns the number after the last underscore (e.g. "hair_10" -> 10, "eyes_1" -> 1).
// Returns -1 if there is no numeric suffix.
func trailingNumber(optionID string) int {
	idx := strings.LastIndex(optionID, "_")
	if idx < 0 || idx == len(optionID)-1 {
		return -1
	}
	n, err := strconv.Atoi(optionID[idx+1:])
	if err != nil {
		return -1
	}
	return n
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
	return "others"
}

var (
	bodyOrder    = []string{"face","eyes","eyebrow","hair", "lips"}
	fashionOrder = []string{"dress", "shoes", "watch", "fan"}
)

func main() {
	avatarsDir := flag.String("avatars", "avatars", "Path to avatars folder (relative to cwd)")
	catalogDir := flag.String("catalog", "catalog", "Path to catalog output folder")
	cdnBase := flag.String("cdn-base", "", "Optional CDN base for previewUrl (e.g. https://cdn.example.com/kalpix-avatars)")
	previewExt := flag.String("preview-ext", "png", "Preview image extension for option previewUrl: png or webp (must match your uploaded files)")
	flag.Parse()

	ext := strings.ToLower(strings.TrimPrefix(*previewExt, "."))
	if ext != "png" && ext != "webp" {
		fmt.Fprintf(os.Stderr, "Error: preview-ext must be png or webp, got %q\n", *previewExt)
		os.Exit(1)
	}

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

		catalog, err := buildCatalog(assetPath, slug, humanize(slug), *cdnBase, ext)
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

func buildCatalog(assetPath, slug, avatarName, cdnBase, previewExt string) (*catalogOutput, error) {
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
		sortOptionsNatural(subcategoryOptions[k])
	}

	bodySubs := make(map[string][]string)
	fashionSubs := make(map[string][]string)
	animationSubs := make(map[string][]string)
	othersSubs := make(map[string][]string)

	for k, opts := range subcategoryOptions {
		switch topCategory(k) {
		case "body":
			bodySubs[k] = opts
		case "fashion":
			fashionSubs[k] = opts
		case "animation":
			animationSubs[k] = opts
		default:
			othersSubs[k] = opts
		}
	}

	previewBase := ""
	if cdnBase != "" {
		previewBase = strings.TrimSuffix(cdnBase, "/") + "/avatars/" + slug + "/previews"
	}

	var categories []categoryOut
	if len(bodySubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "body",
			Label:         "Body",
			Subcategories: buildSubcategoryList(bodyOrder, bodySubs, slug, previewBase, previewExt, false),
		})
	}
	if len(fashionSubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "fashion",
			Label:         "Fashion",
			Subcategories: buildSubcategoryList(fashionOrder, fashionSubs, slug, previewBase, previewExt, false),
		})
	}
	if len(animationSubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "animation",
			Label:         "Animation",
			Subcategories: buildSubcategoryList([]string{"animation"}, animationSubs, slug, previewBase, previewExt, true),
		})
	}
	if len(othersSubs) > 0 {
		categories = append(categories, categoryOut{
			Key:           "others",
			Label:         "Others",
			Subcategories: buildSubcategoryList([]string{"others"}, othersSubs, slug, previewBase, previewExt, false),
		})
	}
	return &catalogOutput{
		Slug:       slug,
		AvatarName: avatarName,
		Categories: categories,
	}, nil
}

func buildSubcategoryList(order []string, all map[string][]string, slug, previewBase, previewExt string, isAnimation bool) []subcategoryOut {
	seen := make(map[string]bool)
	var result []subcategoryOut

	buildOptions := func(k string, opts []string) []optionOut {
		var options []optionOut
		suffix := "." + previewExt
		for _, oid := range opts {
			opt := optionOut{OptionID: oid, Label: humanize(oid)}
			if !isAnimation {
				opt.SkinName = k + "/" + oid
			}
			// Animations are not Spine skins; omit skinName but still emit previewUrl (R2: previews/animation/<name>.webp).
			if previewBase != "" {
				opt.PreviewURL = previewBase + "/" + k + "/" + oid + suffix
			} else {
				opt.PreviewURL = "avatars/" + slug + "/previews/" + k + "/" + oid + suffix
			}
			options = append(options, opt)
		}
		return options
	}

	for _, k := range order {
		opts, ok := all[k]
		if !ok || seen[k] {
			continue
		}
		seen[k] = true
		result = append(result, subcategoryOut{Key: k, Label: k, Options: buildOptions(k, opts)})
	}
	for k, opts := range all {
		if seen[k] {
			continue
		}
		result = append(result, subcategoryOut{Key: k, Label: k, Options: buildOptions(k, opts)})
	}
	return result
}
