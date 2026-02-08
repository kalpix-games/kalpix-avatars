#!/usr/bin/env python3
"""
Build catalog JSON from Spine asset JSON (skins + animations).
- Uses exact names from asset (e.g. dress, eyes, Hair). No normalizing.
- Top categories: body, fashion, animation.
  - body: eyebrow, eyes, face, Hair, hair, lips
  - fashion: dress, costume, dress, shoes, watch, fan
  - animation: animation names from animations object
- Excludes "default". Output: catalog/<slug>.json
"""

import json
import re
import sys
from pathlib import Path


def humanize(s: str) -> str:
    """Title-case for labels (e.g. dress_1 -> Dress 1). Does not change subcategory keys."""
    if not s:
        return s
    s = s.replace("_", " ")
    return " ".join(w.capitalize() for w in s.split())


def top_category(sub_key: str) -> str:
    """Map subcategory name (exact from asset) to top category: body, fashion, animation."""
    lower = sub_key.lower()
    if lower in ("dress", "costume", "shoes", "watch", "fan"):
        return "fashion"
    if lower in ("eyebrow", "eyes", "face", "hair", "lips"):
        return "body"
    if lower == "animation":
        return "animation"
    return "body"


def build_catalog(asset_path: Path, slug: str, avatar_name: str, cdn_base: str) -> dict:
    try:
        with open(asset_path, encoding="utf-8") as f:
            data = json.load(f)
    except Exception as e:
        print(f"Failed to load {asset_path}: {e}", file=sys.stderr)
        raise

    # Subcategory key exact from asset (e.g. dress, eyes) -> list of optionIds
    subcategory_options: dict[str, list[str]] = {}

    for skin in data.get("skins", []):
        name = skin.get("name", "")
        if "/" not in name or name.strip().lower() == "default":
            continue
        left, right = name.split("/", 1)
        left = left.strip()
        right = right.strip()
        if not left or not right:
            continue
        if left not in subcategory_options:
            subcategory_options[left] = []
        if right not in subcategory_options[left]:
            subcategory_options[left].append(right)

    animations = data.get("animations", {})
    if animations:
        anim_opts = [k for k in animations if k.strip().lower() != "default"]
        if anim_opts:
            subcategory_options["animation"] = sorted(anim_opts)

    # Group by top category (exact subcategory key from asset)
    body_order = ["eyebrow", "eyes", "face", "Hair", "hair", "lips"]
    fashion_order = ["dress", "costume", "dress", "shoes", "watch", "fan"]

    def build_subcategory_list(sub_keys_order, all_subs, is_animation=False):
        seen = set()
        out = []
        for k in sub_keys_order:
            if k in all_subs and k not in seen:
                seen.add(k)
                opts = sorted(all_subs[k])
                options = [
                    {
                        "optionId": oid,
                        "label": humanize(oid),
                        "skinName": "" if is_animation else f"{k}/{oid}",
                        "previewUrl": "" if is_animation else (f"{cdn_base}/{slug}/previews/{oid}.png" if cdn_base else ""),
                    }
                    for oid in opts
                ]
                out.append({"key": k, "label": k, "options": options})
        for k in all_subs:
            if k in seen:
                continue
            opts_sorted = sorted(all_subs[k])
            options = [
                {
                    "optionId": oid,
                    "label": humanize(oid),
                    "skinName": "" if is_animation else f"{k}/{oid}",
                    "previewUrl": "" if is_animation else (f"{cdn_base}/{slug}/previews/{oid}.png" if cdn_base else ""),
                }
                for oid in opts_sorted
            ]
            out.append({"key": k, "label": k, "options": options})
        return out

    body_subs = {k: subcategory_options[k] for k in subcategory_options if top_category(k) == "body"}
    fashion_subs = {k: subcategory_options[k] for k in subcategory_options if top_category(k) == "fashion"}
    animation_subs = {k: subcategory_options[k] for k in subcategory_options if top_category(k) == "animation"}

    categories = []
    if body_subs:
        categories.append({"key": "body", "label": "Body", "subcategories": build_subcategory_list(body_order, body_subs, False)})
    if fashion_subs:
        categories.append({"key": "fashion", "label": "Fashion", "subcategories": build_subcategory_list(fashion_order, fashion_subs, False)})
    if animation_subs:
        categories.append({"key": "animation", "label": "Animation", "subcategories": build_subcategory_list(["animation"], animation_subs, True)})

    return {"slug": slug, "avatarName": avatar_name, "categories": categories}


def main():
    repo_root = Path(__file__).resolve().parent.parent
    cdn_base = "https://cdn.jsdelivr.net/gh/kalpix-games/kalpix-avatars@main/avatars"
    avatars = [
        ("avatar1", "Genz Boy"),
        ("avatar2", "Japanese Girl"),
    ]
    for slug, avatar_name in avatars:
        asset_path = repo_root / "avatars" / slug / f"{slug}.json"
        if not asset_path.exists():
            print(f"Skip {slug}: {asset_path} not found", file=sys.stderr)
            continue
        catalog = build_catalog(asset_path, slug, avatar_name, cdn_base)
        out_path = repo_root / "catalog" / f"{slug}.json"
        out_path.parent.mkdir(parents=True, exist_ok=True)
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump(catalog, f, indent=2)
        print(f"Wrote {out_path}")


if __name__ == "__main__":
    main()
