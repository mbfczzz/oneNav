// Mock adapter — a full in-browser backend backed by localStorage, mirroring
// http.js (incl. multi-tenancy: global vs per-user data, roles, registration).
// Used when VITE_API_BASE is unset so the SPA runs standalone.
import { seedCategories, seedLinks } from '@/data/seed'
import { isHttpUrl } from '@/utils/url'
import { getToken } from './token'

const LS_KEY = 'zmark.db.v2'
const SESS_KEY = 'zmark.mock.sessions'
const GLOBAL = 'global'
const delay = (ms = 100) => new Promise((r) => setTimeout(r, ms))
const uid = () =>
  typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : 'id-' + Math.random().toString(36).slice(2) + Date.now().toString(36)

const nextWeight = (arr) => arr.reduce((m, x) => Math.max(m, x.weight ?? -1), -1) + 1

function clone(v) {
  return typeof structuredClone === 'function' ? structuredClone(v) : JSON.parse(JSON.stringify(v))
}
function load() {
  let db = null
  try {
    const raw = localStorage.getItem(LS_KEY)
    if (raw) db = JSON.parse(raw)
  } catch {
    db = null
  }
  if (!db) {
    db = {
      categories: clone(seedCategories).map((c) => ({ ...c, owner_id: GLOBAL })),
      links: clone(seedLinks).map((l) => ({ ...l, owner_id: GLOBAL })),
      users: [{ id: 'admin', username: 'admin', password: 'admin123', role: 'admin' }],
      invites: [],
    }
    save(db)
    return db
  }
  if (!db.users) db.users = [{ id: 'admin', username: 'admin', password: 'admin123', role: 'admin' }]
  if (!db.invites) db.invites = []
  if (!db.settings) db.settings = {}
  return db
}

const inviteGen = () => {
  const a = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  let s = ''
  for (let i = 0; i < 10; i++) s += a[Math.floor(Math.random() * a.length)]
  return s
}
const mockInviteStatus = (i) => {
  if (i.disabled) return 'disabled'
  if (i.expiresAt && new Date(i.expiresAt).getTime() < Date.now()) return 'expired'
  if (i.maxUses > 0 && i.usedCount >= i.maxUses) return 'used'
  return 'active'
}
const fmtTime = (iso) => (iso ? iso.slice(0, 16).replace('T', ' ') : '')
function save(db) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(db))
  } catch {
    /* storage unavailable */
  }
}

// ---------- sessions (token -> {userId, username, role}) ----------
function sessions() {
  try {
    return JSON.parse(localStorage.getItem(SESS_KEY) || '{}')
  } catch {
    return {}
  }
}
function saveSessions(s) {
  try {
    localStorage.setItem(SESS_KEY, JSON.stringify(s))
  } catch {
    /* ignore */
  }
}
function currentSession() {
  const tok = getToken()
  return tok ? sessions()[tok] || null : null
}
function ownerFor(scope) {
  if (scope === 'mine') {
    const s = currentSession()
    return s ? s.userId : '__none__'
  }
  return GLOBAL
}

export async function fetchAll(scope) {
  await delay()
  const db = load()
  const owner = ownerFor(scope)
  return {
    categories: db.categories.filter((c) => c.owner_id === owner),
    links: db.links.filter((l) => l.owner_id === owner),
  }
}

// ---------- auth ----------
export async function register(username, password, code) {
  await delay(80)
  username = String(username || '').trim()
  if (username.length < 3 || username.length > 32) throw new Error('用户名需 3-32 个字符')
  if (String(password || '').length < 6) throw new Error('密码至少 6 位')
  code = String(code || '').trim().toUpperCase()
  if (!code) throw new Error('请输入邀请码')
  const db = load()
  if (db.users.some((u) => u.username === username)) throw new Error('用户名已存在')
  const inv = db.invites.find((i) => i.code === code)
  if (!inv) throw new Error('邀请码无效')
  if (inv.disabled) throw new Error('邀请码已停用')
  if (inv.expiresAt && new Date(inv.expiresAt).getTime() < Date.now()) throw new Error('邀请码已过期')
  if (inv.maxUses > 0 && inv.usedCount >= inv.maxUses) throw new Error('邀请码已用尽')
  const role = inv.grantRole === 'admin' ? 'admin' : 'user'
  const user = {
    id: uid(),
    username,
    password,
    role,
    inviteCode: code,
    createdAt: fmtTime(new Date().toISOString()),
  }
  db.users.push(user)
  inv.usedCount++
  save(db)
  const token = 'mock-' + uid()
  const s = sessions()
  s[token] = { userId: user.id, username, role }
  saveSessions(s)
  return { token, user: { username, role } }
}
export async function login(username, password) {
  await delay(80)
  const db = load()
  const user = db.users.find((u) => u.username === username && u.password === password)
  if (!user) throw new Error('用户名或密码错误')
  const token = 'mock-' + uid()
  const s = sessions()
  s[token] = { userId: user.id, username: user.username, role: user.role }
  saveSessions(s)
  return { token, user: { username: user.username, role: user.role } }
}
export async function logout() {
  await delay(30)
  const tok = getToken()
  if (tok) {
    const s = sessions()
    delete s[tok]
    saveSessions(s)
  }
  return { ok: true }
}
export async function me() {
  await delay(30)
  const s = currentSession()
  if (!s) throw new Error('unauthorized')
  return { user: { username: s.username, role: s.role } }
}

export async function getSettings() {
  await delay(30)
  const db = load()
  return { siteName: (db.settings && db.settings.siteName) || '启点导航' }
}
export async function updateSettings(data) {
  await delay(50)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const name = String(data.siteName || '').trim()
  if (!name) throw new Error('站点名称不能为空')
  const db = load()
  db.settings = db.settings || {}
  db.settings.siteName = name
  save(db)
  return { siteName: name }
}

export async function listUsers() {
  await delay(40)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const db = load()
  return db.users.map((u) => ({ id: u.id, username: u.username, role: u.role, createdAt: '' }))
}
export async function deleteUser(id) {
  await delay(60)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  if (id === sess.userId) throw new Error('不能删除自己')
  const db = load()
  const u = db.users.find((x) => x.id === id)
  if (!u) throw new Error('not found')
  // Mirror the backend: protect only the bootstrap admin, not invite-minted ones.
  if (u.username === 'admin') throw new Error('不能删除主管理员账号')
  db.users = db.users.filter((x) => x.id !== id)
  db.links = db.links.filter((l) => l.owner_id !== id)
  db.categories = db.categories.filter((c) => c.owner_id !== id)
  save(db)
  const ss = sessions()
  for (const [tok, se] of Object.entries(ss)) if (se.userId === id) delete ss[tok]
  saveSessions(ss)
  return { ok: true }
}

// ---------- invites (admin) ----------
export async function listInvites() {
  await delay(40)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const db = load()
  return db.invites
    .slice()
    .reverse()
    .map((i) => ({
      id: i.id,
      code: i.code,
      note: i.note || '',
      maxUses: i.maxUses,
      usedCount: i.usedCount,
      expiresAt: fmtTime(i.expiresAt),
      disabled: !!i.disabled,
      grantRole: i.grantRole || 'user',
      status: mockInviteStatus(i),
      createdAt: i.createdAt || '',
    }))
}
export async function createInvite(data) {
  await delay(60)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const db = load()
  let code
  do {
    code = inviteGen()
  } while (db.invites.some((i) => i.code === code))
  let expiresAt = null
  if (data.expiresInDays > 0) {
    expiresAt = new Date(Date.now() + data.expiresInDays * 86400000).toISOString()
  }
  const grantRole = data.role === 'admin' ? 'admin' : 'user'
  const inv = {
    id: uid(),
    code,
    note: data.note || '',
    maxUses: data.maxUses < 0 ? 0 : data.maxUses,
    usedCount: 0,
    expiresAt,
    disabled: false,
    grantRole,
    createdBy: sess.userId,
    createdAt: fmtTime(new Date().toISOString()),
  }
  db.invites.push(inv)
  save(db)
  return {
    id: inv.id,
    code,
    note: inv.note,
    maxUses: inv.maxUses,
    usedCount: 0,
    expiresAt: fmtTime(expiresAt),
    disabled: false,
    grantRole,
    status: 'active',
    createdAt: inv.createdAt,
  }
}
export async function inviteUses(id) {
  await delay(40)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const db = load()
  const inv = db.invites.find((i) => i.id === id)
  if (!inv) throw new Error('not found')
  return db.users
    .filter((u) => u.inviteCode === inv.code)
    .map((u) => ({ username: u.username, usedAt: u.createdAt || '' }))
}
export async function toggleInvite(id, disabled) {
  await delay(40)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const db = load()
  const inv = db.invites.find((i) => i.id === id)
  if (!inv) throw new Error('not found')
  inv.disabled = !!disabled
  save(db)
  return { ok: true }
}
export async function deleteInvite(id) {
  await delay(40)
  const sess = currentSession()
  if (!sess || sess.role !== 'admin') throw new Error('仅管理员可访问')
  const db = load()
  db.invites = db.invites.filter((i) => i.id !== id)
  save(db)
  return { ok: true }
}

function requireOwner(scope) {
  const sess = currentSession()
  if (scope === 'global' || !scope) {
    if (!sess || sess.role !== 'admin') throw new Error('只有管理员能配置全局导航')
    return GLOBAL
  }
  if (!sess) throw new Error('unauthorized')
  return sess.userId
}

// ---------- categories ----------
export async function createCategory(data, scope = 'global') {
  await delay(80)
  const owner = requireOwner(scope)
  const db = load()
  const cat = {
    id: uid(),
    owner_id: owner,
    name: data.name,
    icon: data.icon || 'ri:folder-line',
    weight: nextWeight(db.categories.filter((c) => c.owner_id === owner)),
  }
  db.categories.push(cat)
  save(db)
  return cat
}
export async function updateCategory(id, data) {
  await delay(80)
  const db = load()
  const c = db.categories.find((c) => c.id === id)
  if (!c) throw new Error('not found')
  if (data.name != null) c.name = data.name
  if (data.icon != null) c.icon = data.icon
  save(db)
  return c
}
export async function deleteCategory(id) {
  await delay(80)
  const db = load()
  db.categories = db.categories.filter((c) => c.id !== id)
  db.links = db.links.filter((l) => l.categoryId !== id)
  save(db)
  return { ok: true }
}
export async function reorderCategories(orderedIds) {
  await delay(80)
  const db = load()
  orderedIds.forEach((id, i) => {
    const c = db.categories.find((c) => c.id === id)
    if (c) c.weight = i
  })
  save(db)
  return { ok: true }
}

// ---------- links ----------
export async function createLink(data, scope = 'global') {
  await delay(80)
  const owner = requireOwner(scope)
  const db = load()
  const cat = db.categories.find((c) => c.id === data.categoryId)
  if (!cat || cat.owner_id !== owner) throw new Error('invalid category')
  const link = {
    id: uid(),
    owner_id: owner,
    categoryId: data.categoryId,
    title: data.title,
    url: data.url,
    description: data.description || '',
    icon: data.icon || '',
    clicks: 0,
    weight: nextWeight(db.links.filter((l) => l.categoryId === data.categoryId)),
  }
  db.links.push(link)
  save(db)
  return link
}
export async function updateLink(id, data) {
  await delay(80)
  const db = load()
  const l = db.links.find((l) => l.id === id)
  if (!l) throw new Error('not found')
  for (const k of ['categoryId', 'title', 'url', 'description', 'icon', 'clicks']) {
    if (data[k] != null) l[k] = data[k]
  }
  save(db)
  return l
}
export async function deleteLink(id) {
  await delay(80)
  const db = load()
  db.links = db.links.filter((l) => l.id !== id)
  save(db)
  return { ok: true }
}
export async function reorderLinks(orderedIds) {
  await delay(80)
  const db = load()
  orderedIds.forEach((id, i) => {
    const l = db.links.find((l) => l.id === id)
    if (l) l.weight = i
  })
  save(db)
  return { ok: true }
}
export async function clickLink(id) {
  await delay(30)
  const db = load()
  const l = db.links.find((l) => l.id === id)
  if (!l) throw new Error('not found')
  l.clicks = (l.clicks || 0) + 1
  save(db)
  return { clicks: l.clicks }
}

// ---------- import (tolerant ZMark/OneNav-ish JSON, scoped to owner) ----------
export async function importData(payload, scope = 'global') {
  await delay(120)
  const owner = requireOwner(scope)
  const db = load()
  const cats = Array.isArray(payload?.categories) ? payload.categories : []
  const links = Array.isArray(payload?.links)
    ? payload.links
    : Array.isArray(payload)
      ? payload
      : []

  const nameToId = {}
  db.categories.filter((c) => c.owner_id === owner).forEach((c) => (nameToId[c.name] = c.id))
  const seen = new Set(
    db.links.filter((l) => l.owner_id === owner).map((l) => l.categoryId + '\n' + l.url),
  )
  let addedCategories = 0
  let addedLinks = 0
  let skipped = 0
  let truncated = false

  const ensureCat = (name, icon) => {
    if (!name) name = '导入'
    if (!nameToId[name]) {
      const id = uid()
      db.categories.push({
        id,
        owner_id: owner,
        name,
        icon: icon || 'ri:folder-line',
        weight: nextWeight(db.categories.filter((c) => c.owner_id === owner)),
      })
      nameToId[name] = id
      addedCategories++
    }
    return nameToId[name]
  }

  for (const c of cats) ensureCat(c.name || c.title, c.icon)

  for (const l of links) {
    if (addedLinks >= 2000) {
      truncated = true
      break
    }
    const title = l.title || l.name
    const url = l.url || l.link
    if (!title || !url || !isHttpUrl(url)) {
      skipped++
      continue
    }
    let catId =
      l.categoryId && db.categories.find((c) => c.id === l.categoryId && c.owner_id === owner)
        ? l.categoryId
        : null
    if (!catId) catId = ensureCat(l.categoryName || l.category || l.catName)
    const key = catId + '\n' + url
    if (seen.has(key)) {
      skipped++
      continue
    }
    seen.add(key)
    db.links.push({
      id: uid(),
      owner_id: owner,
      categoryId: catId,
      title,
      url,
      description: l.description || l.desc || '',
      icon: l.icon || '',
      clicks: Number(l.clicks) || 0,
      weight: nextWeight(db.links.filter((x) => x.categoryId === catId)),
    })
    addedLinks++
  }
  save(db)
  return { addedCategories, addedLinks, skipped, truncated }
}

export async function resetDb() {
  await delay(40)
  try {
    localStorage.removeItem(LS_KEY)
  } catch {
    /* ignore */
  }
  return load()
}
