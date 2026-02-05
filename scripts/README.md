# Scripts

## build_catalog_from_asset.py

Builds catalog JSON from Spine asset JSON (skins + animations). Uses **exact names from asset** (e.g. Coustume, eye_boll, Hair).

**Source:** `avatars/<slug>/<slug>.json` (Spine format)

**Top categories:** body, fashion, animation.

- **body:** eyebrow, eye_boll, face, Hair, hair, lip (body-related)
- **fashion:** Coustume, costume, coustume, shoes, watch, fan (not directly body)
- **animation:** animation names from `animations` object

**Rules:**
- **Skins:** Each skin `name` in format `Subcategory/OptionId` (e.g. `Coustume/coustume_1`, `eye_boll/Eye_ball_1`) → subcategory key and label = exact left part from asset; options = right part. `default` excluded.
- **Animations:** Keys of `animations` → subcategory `animation` under top category Animation.
- **Output:** Three categories (body, fashion, animation); subcategory keys/labels match asset; each option has `optionId`, `label`, `skinName` (full `Subcategory/OptionId`), `previewUrl`.

**Usage:**
```bash
python3 scripts/build_catalog_from_asset.py
```
Writes `catalog/avatar1.json` and `catalog/avatar2.json`.

**Backend:** API builds catalog from asset JSON (CDN) with same mapping; fallback is static `catalog/<slug>.json`.
