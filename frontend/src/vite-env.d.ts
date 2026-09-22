/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// 构建信息，由 vite.config.ts 在构建时通过 define 注入
declare const __APP_VERSION__: string
declare const __BUILD_TIME__: string
declare const __BUILD_PLATFORM__: string
