// Adapter switch: use the real HTTP backend when VITE_API_BASE is set,
// otherwise fall back to the in-browser mock (localStorage). Both modules
// expose the identical function surface, so the store is agnostic.
import * as mock from './mock'
import * as http from './http'

const useHttp = !!import.meta.env.VITE_API_BASE

const api = useHttp ? http : mock

export const API_MODE = useHttp ? 'http' : 'mock'
export default api
