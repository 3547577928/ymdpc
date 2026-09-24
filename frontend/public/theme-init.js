// 首屏主题初始化：在 CSS 应用前读出用户选择，避免暗色/亮色闪烁。
// 必须是外部文件——CSP script-src 'self' 会拦截内联脚本。
try {
  document.documentElement.dataset.theme = localStorage.getItem('qs:theme') || (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
} catch { /* localStorage 不可用时保持默认亮色 */ }
