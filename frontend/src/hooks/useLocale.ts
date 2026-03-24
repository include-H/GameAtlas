import { computed } from 'vue'
import zhCN from '@/locale/zh-CN'

type Locale = 'zh-CN' | 'en-US'
interface LocaleMessages {
  [key: string]: string | LocaleMessages
}

const locales: Record<Locale, LocaleMessages> = {
  'zh-CN': zhCN,
  'en-US': {
    menu: {
      login: 'Login',
      register: 'Register',
      dashboard: 'Dashboard',
      games: 'Games',
      pending: {
        center: 'Pending Center',
      },
      notFound: 'Not Found',
      'game.detail': 'Game Detail',
      'wiki.edit': 'Edit Wiki',
      'games.all': 'All Games',
      'games.timeline': 'Timeline',
    },
    common: {
      logout: 'Logout',
      profile: 'Profile',
      settings: 'Settings',
      search: 'Search',
      filter: 'Filter',
      sort: 'Sort',
      loading: 'Loading...',
      noData: 'No Data',
      error: 'Error',
      success: 'Success',
      cancel: 'Cancel',
      confirm: 'Confirm',
      delete: 'Delete',
      edit: 'Edit',
      save: 'Save',
      create: 'Create',
      update: 'Update',
      back: 'Back',
      submit: 'Submit',
      reset: 'Reset',
    },
  },
}

/**
 * Locale hook for internationalization
 */
export default function useLocale() {
  // Default locale can be stored in localStorage or detected from browser
  const currentLocale = computed<Locale>(() => {
    return (localStorage.getItem('locale') as Locale) || 'zh-CN'
  })

  /**
   * Get translation for a key
   * Supports nested keys like 'menu.dashboard'
   */
  const t = (key: string): string => {
    const keys = key.split('.')
    let value: string | LocaleMessages = locales[currentLocale.value]

    for (let index = 0; index < keys.length; index += 1) {
      const segment = keys[index]
      if (!value || typeof value !== 'object') {
        return key // Return key if translation not found
      }

      if (segment in value) {
        const next: string | LocaleMessages = value[segment]
        const remaining = keys.slice(index).join('.')
        // Support flat keys like "games.timeline" under menu while keeping "menu.games" usable.
        if (
          index < keys.length - 1 &&
          (next === null || typeof next !== 'object') &&
          remaining in value
        ) {
          value = value[remaining]
          return typeof value === 'string' ? value : key
        }
        value = next
        continue
      }

      const remaining = keys.slice(index).join('.')
      if (remaining in value) {
        value = value[remaining]
        return typeof value === 'string' ? value : key
      }

      return key
    }

    return typeof value === 'string' ? value : key
  }

  /**
   * Set current locale
   */
  const setLocale = (locale: Locale) => {
    localStorage.setItem('locale', locale)
    window.location.reload() // Reload to apply new locale
  }

  return {
    currentLocale,
    t,
    setLocale,
  }
}
