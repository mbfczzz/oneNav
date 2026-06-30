// Guards against dangerous URL schemes (javascript:, data:, file:, …) that would
// otherwise become a stored-XSS / phishing vector when opened or embedded.
export function isHttpUrl(raw) {
  try {
    const u = new URL(String(raw).trim())
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}
