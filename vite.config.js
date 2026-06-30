import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Mirrors the original build: ESM output under /static/assets with hashed chunks.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  base: '/',
  build: {
    assetsDir: 'static/assets',
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-vue': ['vue', 'vue-router', 'pinia'],
          'vendor-i18n': ['vue-i18n'],
          'vendor-iconify': ['@iconify/vue'],
        },
      },
    },
  },
  server: {
    host: true,
    port: 5173,
    // When running the real backend (npm run server), set VITE_API_BASE=/api in
    // .env.local and these proxy rules forward API calls to it (no CORS needed).
    proxy: {
      '/api': {
        target: 'http://localhost:8787',
        changeOrigin: true,
      },
    },
  },
})
