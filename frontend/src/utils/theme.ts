export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'qs:theme'

// getTheme 读取用户选择，未选择时跟随系统偏好
export function getTheme(): Theme {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark') return stored
  } catch {
    // localStorage 不可用（隐私模式）时按系统偏好
  }
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme
  try {
    window.localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // 忽略持久化失败，当前会话仍然生效
  }
}
