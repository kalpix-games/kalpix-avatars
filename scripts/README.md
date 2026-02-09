# Scripts

- **What exactly each script does:** [docs/SCRIPTS_WHAT_THEY_DO.md](../docs/SCRIPTS_WHAT_THEY_DO.md)
- **Spine naming rules** (skins, animations, folder layout): [docs/SPINE_NAMING_CONVENTIONS.md](../docs/SPINE_NAMING_CONVENTIONS.md)

Each script lives in its own directory (`create_avatars_list/`, `create_avatars_catalog/`) so the Go tool treats them as separate programs (no “redeclared” errors). Run from **kalpix-avatars repo root**.

---

## create_avatars_list

Scans `avatars/` (each subdirectory = one avatar) and writes **`avatars_list.json`** at the repo root. Upload to Cloudflare; backend fetches it when `AVATAR_MANIFEST_URL` is set and serves it on `avatar/list_avatars`.

**Usage (from repo root):**
```bash
go run ./scripts/create_avatars_list
```

**Options:** `-out=avatars_list.json`, `-avatars=avatars`, `-base-url=https://...` (optional full URLs in manifest).

---

## create_avatars_catalog

Reads each `avatars/<slug>/<slug>.json` (Spine), extracts skins (`Subcategory/OptionId`) and animations, and writes **`catalog/<slug>.json`** for each avatar. Backend fetches these from CDN for `avatar/get_character_catalog`.

**Usage (from repo root):**
```bash
go run ./scripts/create_avatars_catalog
```

**Options:** `-avatars=avatars`, `-catalog=catalog`, `-cdn-base=https://...` (optional full preview URLs).

---

## build_catalog_from_asset.py

Builds catalog JSON from Spine asset JSON (skins + animations). Uses **exact names from asset** (e.g. dress, eyes, Hair).

**Source:** `avatars/<slug>/<slug>.json` (Spine format)

**Top categories:** body, fashion, animation.

- **body:** eyebrow, eyes, face, Hair, hair, lips (body-related)
- **fashion:** dress, dress, dress, shoes, watch, fan (not directly body)
- **animation:** animation names from `animations` object

**Rules:**
- **Skins:** Each skin `name` in format `Subcategory/OptionId` (e.g. `dress/dress_1`, `eyes/eyes_1`) → subcategory key and label = exact left part from asset; options = right part. `default` excluded.
- **Animations:** Keys of `animations` → subcategory `animation` under top category Animation.
- **Output:** Three categories (body, fashion, animation); subcategory keys/labels match asset; each option has `optionId`, `label`, `skinName` (full `Subcategory/OptionId`), `previewUrl`.

**Usage:**
```bash
python3 scripts/build_catalog_from_asset.py
```
Writes `catalog/avatar1.json` and `catalog/avatar2.json`.

**Backend:** API builds catalog from asset JSON (CDN) with same mapping; fallback is static `catalog/<slug>.json`.
