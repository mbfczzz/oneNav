import { reactive } from 'vue'

const state = reactive({ items: [] })
let seq = 0

export function useToast() {
  function show(message, type = 'info', timeout = 2000) {
    const id = ++seq
    state.items.push({ id, message, type })
    setTimeout(() => {
      const idx = state.items.findIndex((t) => t.id === id)
      if (idx > -1) state.items.splice(idx, 1)
    }, timeout)
  }
  return { state, show }
}
