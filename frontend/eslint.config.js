import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'

export default tseslint.config(
  { ignores: ['dist', 'node_modules'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    // public 下的原生脚本运行在浏览器环境
    files: ['public/**/*.js'],
    languageOptions: { globals: { document: 'readonly', localStorage: 'readonly', matchMedia: 'readonly', window: 'readonly', self: 'readonly', caches: 'readonly', fetch: 'readonly' } },
  },
  {
    files: ['e2e/**/*.ts'],
    // Playwright 测试使用全局 fixture（test, expect 等），由 playwright.config.ts 注入
  },
  {
    files: ['src/**/*.{ts,tsx}'],
    plugins: { 'react-hooks': reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      // 渐进引入：历史代码里存在的模式先降级，新增代码按推荐约束
      'react-hooks/exhaustive-deps': 'warn',
      // v7 新规则，把「effect 里发起异步取数」这类惯用模式也判错，对本项目噪音过大
      'react-hooks/set-state-in-effect': 'off',
    },
  },
)
