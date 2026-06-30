import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { addCollection } from '@iconify/vue'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import riIcons from './ri-icons.json'
import './style.css'

// Register the Remix Icon subset offline so every icon renders without the
// Iconify network API (regenerate src/ri-icons.json via scripts when icons change).
addCollection(riIcons)

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(i18n)
app.mount('#app')

// Remove the static loading shell (index.html #app-loading) once Vue has mounted.
const shell = document.getElementById('app-loading')
if (shell) {
  shell.classList.add('is-hidden')
  setTimeout(() => shell.remove(), 320)
}
