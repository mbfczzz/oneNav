// Emits backend-go/seed.json from the shared JS seed so the Go backend and the
// frontend mock share identical initial data. Run: node scripts/gen-seed.mjs
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { seedCategories, seedLinks } from '../src/data/seed.js'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const dest = path.join(root, 'backend-go', 'seed.json')
fs.mkdirSync(path.dirname(dest), { recursive: true })
fs.writeFileSync(
  dest,
  JSON.stringify({ categories: seedCategories, links: seedLinks }, null, 2),
)
console.log(
  `wrote backend-go/seed.json — ${seedCategories.length} categories, ${seedLinks.length} links`,
)
