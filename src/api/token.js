// Shared bearer-token storage used by the http adapter and the auth store.
const KEY = 'zmark.token'

export const getToken = () => {
  try {
    return localStorage.getItem(KEY) || ''
  } catch {
    return ''
  }
}

export const setToken = (token) => {
  try {
    if (token) localStorage.setItem(KEY, token)
    else localStorage.removeItem(KEY)
  } catch {
    /* storage unavailable */
  }
}
