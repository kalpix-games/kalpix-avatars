# What the Avatar Scripts Do

This doc explains **exactly** what each script does, step by step. Run both from the **kalpix-avatars repo root**.

---

## 1. `create_avatars_list` (list of avatars for the backend)

**What it does:**

1. **Scans** the `avatars/` folder. Every **subdirectory** is treated as one avatar (e.g. `avatar1`, `avatar2`). The directory name is the avatar **slug**.

2. **For each subdirectory** it checks that the required Spine files exist:
   - `<slug>.json` (Spine skeleton/data) — **required**
   - Atlas: either `<slug>.txt` or `<slug>.atlas.txt` — **required** (script needs at least one to consider the avatar valid)

   If `<slug>.json` is missing, that folder is **skipped** and not added to the list.

3. **Builds one manifest entry per avatar** with:
   - `slug` = folder name
   - `avatarName` = humanized slug (e.g. `avatar1` → "Avatar1", `genz_boy` → "Genz Boy")
   - `isActive` = true
   - `sortOrder` = 1, 2, 3, … in the order folders are read

   If you pass `-base-url=<your-cdn-base>`, it also fills in full URLs for `previewUrl`, `baseAtlasUrl`, `baseJsonUrl`, `basePngUrl`, `catalogUrl`. Otherwise those fields are left empty and the **backend** fills them from its CDN config when it fetches this file.

4. **Writes** a single file: **`avatars_list.json`** (or the path you set with `-out`) at the **current working directory**. The file is a **JSON array** of those entries.

**Why it exists:**  
The backend’s `avatar/list_avatars` RPC can be configured to fetch the avatar list from the CDN instead of the database. You upload this generated `avatars_list.json` to Cloudflare (or your CDN), set `AVATAR_MANIFEST_URL` to that file’s URL, and the backend fetches it and serves it to the client. So this script **builds the manifest file** that the backend will fetch from the CDN.

**Run:**
```bash
go run ./scripts/create_avatars_list
```
Optional flags: `-avatars=avatars`, `-out=avatars_list.json`, `-base-url=https://...`

---

## 2. `create_avatars_catalog` (catalog JSON per avatar for customization)

**What it does:**

1. **Scans** the `avatars/` folder. For each **subdirectory** (slug), it looks for the Spine JSON at `avatars/<slug>/<slug>.json`. If that file is missing, that avatar is **skipped**.

2. **For each found Spine JSON** it:
   - **Reads** the `skins` array. Each skin has a `name`.
   - **Parses skin names** that contain a **single slash** as `Subcategory/OptionId` (e.g. `eyebrow/eyebrow_1`, `dress/dress_2`). Skins named `default` or with no slash are **ignored**.
   - **Collects** all unique subcategory keys and their option IDs (e.g. subcategory `eyebrow` → options `eyebrow_1`, `eyebrow_2`, …).
   - **Reads** the top-level `animations` object. Each **key** (e.g. `idel_basic`, `walk`) becomes an option under the **Animation** category. The key `default` is excluded.

3. **Groups subcategories** into three top-level categories:
   - **Body:** eyebrow, eyes, face, Hair, hair, lips (and any other unknown key)
   - **Fashion:** dress, dress, shoes, watch, fan
   - **Animation:** the animation keys from the Spine file

4. **Builds** a catalog structure for that avatar: slug, avatarName (humanized slug), and categories with subcategories and options. Each option has:
   - `optionId` (e.g. `eyebrow_1`)
   - `label` (humanized, e.g. "Eyebrow 1")
   - `skinName` (e.g. `eyebrow/eyebrow_1`) for Spine `setSkin`
   - `previewUrl` (relative like `catalog/avatar1/eyebrow/1.webp` or full URL if `-cdn-base` is set)

5. **Writes** one file per avatar: **`catalog/<slug>.json`** (e.g. `catalog/avatar1.json`, `catalog/avatar2.json`). The backend fetches these from the CDN when the client calls `avatar/get_character_catalog` with an `avatarId`; it resolves the slug and requests `catalog/<slug>.json` from the CDN.

**Why it exists:**  
The backend does **not** read the Spine JSON directly. It only serves the **pre-built** catalog JSON. This script **generates** that catalog from the Spine asset (skins + animations) so you can upload `catalog/avatar1.json`, etc., to the CDN. That way the app knows which options exist (e.g. which hairs, dresses) and their `skinName` for the Spine runtime.

**Run:**
```bash
go run ./scripts/create_avatars_catalog
```
Optional flags: `-avatars=avatars`, `-catalog=catalog`, `-cdn-base=https://...`

---

## Summary

| Script | Input | Output | Used by |
|--------|--------|--------|--------|
| **create_avatars_list** | Folder layout under `avatars/` (each subdir = one avatar; must have `<slug>.json` and atlas). | Single file **`avatars_list.json`** (JSON array of avatar entries). | Backend fetches this from CDN when `AVATAR_MANIFEST_URL` is set and serves it on `avatar/list_avatars`. |
| **create_avatars_catalog** | Spine JSON at `avatars/<slug>/<slug>.json` (skins with `Subcategory/OptionId` names + `animations` keys). | One file per avatar: **`catalog/<slug>.json`** (categories, subcategories, options with optionId, skinName, previewUrl). | Backend fetches these from CDN when client calls `avatar/get_character_catalog` and serves the catalog. |

Both scripts run from the **kalpix-avatars** repo root. After running them, upload the generated files to your CDN (e.g. Cloudflare R2) so the backend can fetch them.
