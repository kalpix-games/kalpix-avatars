#!/usr/bin/env node
/**
 * Builds a sync_avatars payload from catalog/*.json files.
 * Usage: node scripts/build_sync_avatars_payload.js [catalog/avatar1.json catalog/avatar2.json ...]
 * Output: sync_avatars_payload.json (or stdout with -)
 */

const fs = require('fs');
const path = require('path');

const catalogDir = path.join(__dirname, '..', 'catalog');
const defaultCoins = 0;
const defaultGems = 0;

function addPriceToOption(opt) {
  return { ...opt, price: { coins: defaultCoins, gems: defaultGems } };
}

function addPricesToCategories(categories) {
  return categories.map((cat) => ({
    ...cat,
    subcategories: (cat.subcategories || []).map((sub) => ({
      ...sub,
      options: (sub.options || []).map(addPriceToOption),
    })),
  }));
}

function buildDefaultSelection(categories) {
  const sel = {};
  for (const cat of categories || []) {
    for (const sub of cat.subcategories || []) {
      const first = sub.options && sub.options[0];
      if (first && sub.key) sel[sub.key] = first.optionId;
    }
  }
  return sel;
}

function catalogToSyncEntry(filePath, index) {
  const raw = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const categories = raw.categories || [];
  const catalog = {
    defaultSelection: buildDefaultSelection(categories),
    categories: addPricesToCategories(categories),
  };
  return {
    slug: raw.slug,
    avatarName: raw.avatarName || raw.slug,
    sortOrder: index + 1,
    isActive: true,
    catalog,
  };
}

const args = process.argv.slice(2);
let files = args.filter((a) => a !== '-');
if (files.length === 0) {
  const dir = fs.existsSync(catalogDir) ? catalogDir : path.join(__dirname, '..', 'catalog');
  if (fs.existsSync(dir)) {
    files = fs.readdirSync(dir)
      .filter((f) => f.endsWith('.json'))
      .map((f) => path.join(dir, f))
      .sort();
  }
}
if (files.length === 0) {
  console.error('Usage: node build_sync_avatars_payload.js [catalog/avatar1.json ...]');
  process.exit(1);
}

const avatars = files.map((f, i) => catalogToSyncEntry(f, i));
const payload = { avatars };
const toFile = !args.includes('-');
const out = toFile ? fs.createWriteStream(path.join(__dirname, '..', 'sync_avatars_payload.json')) : process.stdout;
out.write(JSON.stringify(payload, null, 2));
if (toFile) {
  out.end();
  console.error('Wrote', avatars.length, 'avatars to sync_avatars_payload.json');
}
