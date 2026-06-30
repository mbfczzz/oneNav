// Generates src/ri-icons.json — a trimmed offline subset of the Remix Icon set
// containing exactly the `ri:*` icons referenced anywhere under src/, so icons
// render without the Iconify network API. Auto-discovers usages by scanning the
// source, so adding a new icon to a component "just works" after re-running.
//
// Run: npm run gen:icons
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const srcDir = path.join(root, 'src')

// Recursively collect source files.
function walk(dir) {
  const out = []
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(vue|js|ts|jsx|tsx)$/.test(entry.name)) out.push(full)
  }
  return out
}

// Scan for ri:<name> usages.
const names = new Set()
for (const file of walk(srcDir)) {
  const text = fs.readFileSync(file, 'utf8')
  for (const m of text.matchAll(/ri:([a-z0-9-]+)/g)) names.add(m[1])
}

const src = JSON.parse(
  fs.readFileSync(path.join(root, 'node_modules/@iconify-json/ri/icons.json'), 'utf8'),
)

function resolve(name, depth = 0) {
  if (src.icons[name]) return src.icons[name]
  if (src.aliases && src.aliases[name] && depth < 10) {
    const merged = { ...resolve(src.aliases[name].parent, depth + 1), ...src.aliases[name] }
    delete merged.parent
    return merged
  }
  return null
}

const out = { prefix: src.prefix, width: src.width, height: src.height, icons: {} }
const missing = []
for (const n of [...names].sort()) {
  const data = resolve(n)
  if (data) out.icons[n] = data
  else missing.push(n)
}

const dest = path.join(root, 'src/ri-icons.json')
fs.writeFileSync(dest, JSON.stringify(out))
console.log(
  `wrote src/ri-icons.json — ${Object.keys(out.icons).length} icons (scanned ${names.size}), bytes: ${fs.statSync(dest).size}`,
)
if (missing.length) {
  console.error('MISSING (not found in ri set):', missing)
  process.exit(1)
}
