# Spine Assets and Flutter

This doc describes how the **avatar Spine assets** (`.json`, `.txt`, `.png`) in `avatars/<slug>/` are structured and how the Flutter app loads them.

---

## 1. Asset layout (correct for spine_flutter)

Each avatar folder must contain:

| File | Purpose |
|------|--------|
| `<slug>.json` | Spine skeleton/animations (Spine 4.x format). Required. |
| `<slug>.txt` | Spine **atlas** (text format). First line is the image filename (e.g. `avatar1.png`). Required. |
| `<slug>.png` | Texture atlas image. Must match the name in the first line of the `.txt` file. Required. |

- The backend serves **baseJsonUrl**, **baseAtlasUrl**, and **basePngUrl** (full CDN URLs) in `avatar/list_avatars` and in user avatar config.
- Flutter uses **spine_flutter** and loads the character with `SpineWidget.fromHttp(atlasUrl, jsonUrl, ...)`. The atlas URL points to the `.txt` file; the runtime resolves the PNG from the same path (e.g. `.../avatar1/avatar1.txt` → `.../avatar1/avatar1.png`). So the current layout is **correct** for Flutter to load and run the Spine assets.

---

## 2. Atlas format (`.txt`)

The `.txt` file is Spine’s **text atlas** format:

- First line: **page name** (e.g. `avatar1.png`) — same directory as the atlas file.
- Then: `size:W,H`, `filter:...`, then one block per region: **region name** (e.g. `hair_1`), then `bounds:x,y,w,h` and optional `rotate:90`, `offsets:...`.

Your `avatar1.txt` / `avatar2.txt` follow this format. Minor typos in region names (e.g. `blone_left_eyebrow`, `black_4_eft_eyebrow`) only matter if the Spine JSON references those exact names; if skins use the same names, they will work.

---

## 3. Skeleton JSON (`.json`)

The `<slug>.json` file is standard **Spine 4.x** skeleton JSON:

- `skeleton` (spine version, hash, size, paths)
- `bones`, `slots`, `skins`, `animations`, etc.

The Flutter runtime loads this via the **jsonUrl**; it does not rely on the `"images": "./Images/"` path inside the JSON when loading from HTTP. So the current exports are **compatible** with Flutter.

---

## 4. Option previewUrl: full URL in API

Catalog option images (e.g. `catalog/avatar1/eyebrow/2.webp`) can be stored in the catalog JSON as **relative** paths (e.g. `catalog/avatar1/eyebrow/2.webp`).

- When the backend serves the catalog via `avatar/get_character_catalog`, it **rewrites** each option’s `previewUrl` to a **full URL** using the configured CDN base (e.g. `https://your-cdn/catalog/avatar1/eyebrow/2.webp`). So the client always receives a full URL and can load preview images without extra logic.
- Optionally, when generating catalogs you can pass **`-cdn-base=https://your-cdn`** to `create_avatars_catalog` so the generated `catalog/<slug>.json` files already contain full `previewUrl` values; the backend still accepts and rewrites relative URLs.

---

## 5. Summary

| Check | Status |
|-------|--------|
| Spine `.json` format | Valid Spine 4.x; Flutter loads via `jsonUrl`. |
| Atlas `.txt` format | Valid text atlas; first line = PNG name; Flutter resolves PNG from same path as atlas. |
| PNG next to atlas | Required; name must match first line of `.txt`. |
| previewUrl in catalog | Can be relative; backend returns **full URL** in `get_character_catalog` response. |

The current **kalpix-avatars** layout and file formats are correct for the Flutter app to load and run the Spine assets.
