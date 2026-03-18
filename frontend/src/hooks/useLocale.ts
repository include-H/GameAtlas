import { computed } from 'vue'
import zhCN from '@/locale/zh-CN'

export type Locale = 'zh-CN' | 'en-US'

const locales: Record<Locale, any> = {
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
    let value: any = locales[currentLocale.value]

    for (const k of keys) {
      if (value && typeof value === 'object') {
        value = value[k]
      } else {
        return key // Return key if translation not found
      }
    }

    return value || key
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
