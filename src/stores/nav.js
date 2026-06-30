import { defineStore } from 'pinia'
import api from '@/api'
import { getToken, setToken } from '@/api/token'

// Shared in-flight /me promise so concurrent initAuth() callers (App.onMounted +
// router guard) don't fire duplicate validation requests.
let authInFlight = null
// Monotonic load id — the last-issued loadAll wins, so out-of-order responses
// (e.g. rapid scope switches) can't leave stale/cross-tenant data on screen.
let loadSeq = 0

export const useNavStore = defineStore('nav', {
  state: () => ({
    categories: [],
    links: [],
    activeCategoryId: 'all',
    keyword: '',
    loading: false,
    loaded: false,
    loadError: null,
    // scope: 'global' (admin-curated, default for everyone) or 'mine' (this user's)
    scope: 'global',
    loadedScope: null,
    // auth
    token: getToken(),
    user: null,
    // site settings (admin-configurable)
    settings: { siteName: '' },
  }),
  getters: {
    sortedCategories: (state) => [...state.categories].sort((a, b) => a.weight - b.weight),

    linkCountByCategory: (state) => {
      const map = {}
      for (const l of state.links) map[l.categoryId] = (map[l.categoryId] || 0) + 1
      return map
    },

    totalClicks: (state) => state.links.reduce((sum, l) => sum + (l.clicks || 0), 0),

    isSearching: (state) => state.keyword.trim().length > 0,
    isAuthenticated: (state) => !!state.token,
    isAdmin: (state) => state.user?.role === 'admin',

    visibleLinks: (state) => {
      let list = state.links
      if (state.activeCategoryId !== 'all') {
        list = list.filter((l) => l.categoryId === state.activeCategoryId)
      }
      const kw = state.keyword.trim().toLowerCase()
      if (kw) {
        list = list.filter(
          (l) =>
            l.title.toLowerCase().includes(kw) ||
            (l.description || '').toLowerCase().includes(kw) ||
            (l.url || '').toLowerCase().includes(kw),
        )
      }
      return [...list].sort((a, b) => a.weight - b.weight)
    },

    canSortLinks: (state) => state.activeCategoryId !== 'all' && state.keyword.trim().length === 0,

    linksByCategory: (state) => (categoryId) =>
      state.links.filter((l) => l.categoryId === categoryId).sort((a, b) => a.weight - b.weight),
  },
  actions: {
    async loadAll() {
      const target = this.scope
      const reqId = ++loadSeq
      this.loading = true
      this.loadError = null
      try {
        const data = await api.fetchAll(target)
        if (reqId !== loadSeq) return // superseded by a newer load
        this.categories = data.categories
        this.links = data.links
        this.loaded = true
        this.loadedScope = target // the scope actually fetched, not this.scope
        this.activeCategoryId = 'all'
      } catch (e) {
        if (reqId !== loadSeq) return
        // Surface instead of leaving an unhandled rejection + a misleading "empty" view.
        this.loadError = e?.message || '数据加载失败'
      } finally {
        if (reqId === loadSeq) this.loading = false
      }
    },
    async refresh() {
      await this.loadAll()
    },

    // ---------- site settings ----------
    async fetchSettings() {
      try {
        this.settings = await api.getSettings()
      } catch {
        /* keep defaults on failure */
      }
      this.applySiteName()
    },
    async updateSettings(data) {
      this.settings = await api.updateSettings(data)
      this.applySiteName()
      return this.settings
    },
    applySiteName() {
      if (typeof document !== 'undefined' && this.settings?.siteName) {
        document.title = this.settings.siteName
      }
    },

    async setScope(scope) {
      if (scope === 'mine' && !this.isAuthenticated) scope = 'global'
      if (scope === this.scope && this.loadedScope === scope) return
      this.scope = scope
      await this.loadAll()
    },
    // Pick the scope a freshly-authenticated user lands on: admins curate global;
    // a normal user sees their own nav if they have one, else the global set.
    async applyDefaultScope() {
      if (!this.user) {
        this.scope = 'global'
      } else if (this.user.role === 'admin') {
        this.scope = 'global'
      } else {
        try {
          const mine = await api.fetchAll('mine')
          if (mine.categories && mine.categories.length) {
            // Reuse the probe payload instead of re-fetching the same scope.
            this.scope = 'mine'
            this.categories = mine.categories
            this.links = mine.links
            this.loaded = true
            this.loadedScope = 'mine'
            this.activeCategoryId = 'all'
            loadSeq++ // invalidate any older in-flight loadAll
            return
          }
          this.scope = 'global'
        } catch {
          this.scope = 'global'
        }
      }
      await this.loadAll()
    },

    setCategory(id) {
      this.activeCategoryId = id
    },
    setKeyword(kw) {
      this.keyword = kw
    },

    async incrementClick(id) {
      const link = this.links.find((l) => l.id === id)
      if (!link) return
      link.clicks = (link.clicks || 0) + 1 // optimistic
      try {
        const res = await api.clickLink(id)
        if (res && typeof res.clicks === 'number') link.clicks = res.clicks
      } catch {
        /* best-effort */
      }
    },

    // ---------- auth ----------
    async login(username, password) {
      const { token, user } = await api.login(username, password)
      this.token = token
      this.user = user
      setToken(token)
      await this.applyDefaultScope()
      return user
    },
    async register(username, password, code) {
      const { token, user } = await api.register(username, password, code)
      this.token = token
      this.user = user
      setToken(token)
      await this.applyDefaultScope()
      return user
    },
    async logout() {
      try {
        await api.logout()
      } catch {
        /* ignore network/logout errors */
      }
      this.token = ''
      this.user = null
      setToken('')
      this.scope = 'global'
      await this.loadAll()
    },
    async initAuth() {
      if (!this.token || this.user) return
      if (authInFlight) return authInFlight
      authInFlight = (async () => {
        try {
          const { user } = await api.me()
          this.user = user
        } catch {
          // token invalid/expired
          this.token = ''
          this.user = null
          setToken('')
        } finally {
          authInFlight = null
        }
        await this.applyDefaultScope()
      })()
      return authInFlight
    },

    // ---------- category CRUD ----------
    async createCategory(data) {
      const cat = await api.createCategory(data, this.scope)
      this.categories.push(cat)
      return cat
    },
    async updateCategory(id, data) {
      const cat = await api.updateCategory(id, data)
      const i = this.categories.findIndex((c) => c.id === id)
      if (i > -1) this.categories[i] = { ...this.categories[i], ...cat }
      return cat
    },
    async deleteCategory(id) {
      await api.deleteCategory(id)
      this.categories = this.categories.filter((c) => c.id !== id)
      this.links = this.links.filter((l) => l.categoryId !== id)
      if (this.activeCategoryId === id) this.activeCategoryId = 'all'
    },
    async reorderCategories(orderedIds) {
      const prev = new Map()
      orderedIds.forEach((id, i) => {
        const c = this.categories.find((c) => c.id === id)
        if (c) {
          prev.set(c, c.weight)
          c.weight = i
        }
      })
      try {
        await api.reorderCategories(orderedIds)
        return true
      } catch {
        prev.forEach((w, c) => (c.weight = w)) // roll back optimistic order
        return false
      }
    },

    // ---------- link CRUD ----------
    async createLink(data) {
      const link = await api.createLink(data, this.scope)
      this.links.push(link)
      return link
    },
    async updateLink(id, data) {
      const link = await api.updateLink(id, data)
      const i = this.links.findIndex((l) => l.id === id)
      if (i > -1) this.links[i] = { ...this.links[i], ...link }
      return link
    },
    async deleteLink(id) {
      await api.deleteLink(id)
      this.links = this.links.filter((l) => l.id !== id)
    },
    async reorderLinks(orderedIds) {
      const prev = new Map()
      orderedIds.forEach((id, i) => {
        const l = this.links.find((l) => l.id === id)
        if (l) {
          prev.set(l, l.weight)
          l.weight = i
        }
      })
      try {
        await api.reorderLinks(orderedIds)
        return true
      } catch {
        prev.forEach((w, l) => (l.weight = w)) // roll back optimistic order
        return false
      }
    },

    // ---------- import ----------
    async importData(payload) {
      const res = await api.importData(payload, this.scope)
      await this.loadAll()
      return res
    },

    // ---------- admin: user management ----------
    fetchUsers() {
      return api.listUsers()
    },
    deleteUser(id) {
      return api.deleteUser(id)
    },

    // ---------- admin: invite codes ----------
    fetchInvites() {
      return api.listInvites()
    },
    createInvite(data) {
      return api.createInvite(data)
    },
    toggleInvite(id, disabled) {
      return api.toggleInvite(id, disabled)
    },
    deleteInvite(id) {
      return api.deleteInvite(id)
    },
    fetchInviteUses(id) {
      return api.inviteUses(id)
    },
  },
})
