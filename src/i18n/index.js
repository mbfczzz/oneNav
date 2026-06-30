import { createI18n } from 'vue-i18n'
import zh from './locales/zh'
import en from './locales/en'

const saved = localStorage.getItem('zmark.lang')
const browserZh = (navigator.language || 'zh').toLowerCase().startsWith('zh')

const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: saved || (browserZh ? 'zh' : 'en'),
  fallbackLocale: 'en',
  messages: { zh, en },
})

export function setLocale(lang) {
  i18n.global.locale.value = lang
  localStorage.setItem('zmark.lang', lang)
  document.documentElement.setAttribute('lang', lang === 'zh' ? 'zh' : 'en')
}

export default i18n
