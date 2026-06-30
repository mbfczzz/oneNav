// HTTP adapter — talks to the Go backend (backend-go). Selected when
// VITE_API_BASE is set. Mirrors the exact surface of mock.js.
import { getToken, setToken } from './token'

const BASE = (import.meta.env.VITE_API_BASE || '').replace(/\/$/, '')

async function req(method, path, body) {
  const headers = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body != null ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const data = await res.json()
      if (data && data.error) msg = data.error
    } catch {
      /* non-JSON error body */
    }
    // Stale/expired session: a 401 while holding a token means it's no longer
    // valid (e.g. the backend restarted, wiping its in-memory token store).
    // Clear it and bounce protected pages to login. Login sends no token, so
    // a wrong-password 401 is unaffected.
    if (res.status === 401 && getToken() && path !== '/logout') {
      setToken('')
      if (typeof window !== 'undefined' && window.location.pathname.startsWith('/dashboard')) {
        const back = encodeURIComponent(window.location.pathname + window.location.search)
        window.location.assign(`/user?redirect=${back}`)
      }
    }
    throw new Error(msg)
  }
  if (res.status === 204) return null
  return res.json()
}

export const fetchAll = (scope) => req('GET', scope ? `/all?scope=${scope}` : '/all')

export const getSettings = () => req('GET', '/settings')
export const updateSettings = (data) => req('PUT', '/settings', data)

export const register = (username, password, code) =>
  req('POST', '/register', { username, password, code })
export const login = (username, password) => req('POST', '/login', { username, password })
export const logout = () => req('POST', '/logout')
export const me = () => req('GET', '/me')

export const listUsers = () => req('GET', '/users')
export const deleteUser = (id) => req('DELETE', `/users/${id}`)

export const listInvites = () => req('GET', '/invites')
export const createInvite = (data) => req('POST', '/invites', data)
export const toggleInvite = (id, disabled) => req('PUT', `/invites/${id}`, { disabled })
export const deleteInvite = (id) => req('DELETE', `/invites/${id}`)
export const inviteUses = (id) => req('GET', `/invites/${id}/uses`)

export const createCategory = (data, scope = 'global') =>
  req('POST', `/categories?scope=${scope}`, data)
export const updateCategory = (id, data) => req('PUT', `/categories/${id}`, data)
export const deleteCategory = (id) => req('DELETE', `/categories/${id}`)
export const reorderCategories = (orderedIds) => req('PUT', '/categories/order', { orderedIds })

export const createLink = (data, scope = 'global') => req('POST', `/links?scope=${scope}`, data)
export const updateLink = (id, data) => req('PUT', `/links/${id}`, data)
export const deleteLink = (id) => req('DELETE', `/links/${id}`)
export const reorderLinks = (orderedIds) => req('PUT', '/links/order', { orderedIds })
export const clickLink = (id) => req('POST', `/links/${id}/click`)

export const importData = (payload, scope = 'global') => req('POST', `/import?scope=${scope}`, payload)
