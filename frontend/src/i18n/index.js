import { createI18n } from 'vue-i18n'
import enUS from './en-US'
import zhCN from './zh-CN'

export const LOCALE_STORAGE_KEY = 'nats-ui-locale'
export const SUPPORTED_LOCALES = ['zh-CN', 'en-US']

function normalizeLocale(locale) {
  if (locale === 'zh-CN' || locale?.toLowerCase().startsWith('zh')) {
    return 'zh-CN'
  }
  return 'en-US'
}

function resolveInitialLocale() {
  if (typeof window !== 'undefined') {
    const savedLocale = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    if (savedLocale) {
      return normalizeLocale(savedLocale)
    }
    return normalizeLocale(window.navigator.language)
  }
  return 'zh-CN'
}

function syncDocumentLocale(locale) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale
  }
}

const initialLocale = resolveInitialLocale()

const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

syncDocumentLocale(initialLocale)

export function setAppLocale(locale) {
  const nextLocale = normalizeLocale(locale)
  i18n.global.locale.value = nextLocale
  syncDocumentLocale(nextLocale)
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, nextLocale)
  }
}

export default i18n
