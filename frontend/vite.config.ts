import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', 'VITE_')
  const apiTarget = env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080'
  // 构建信息：版本号和运行平台由 CI/Docker 通过 VITE_APP_VERSION、VITE_BUILD_PLATFORM 注入，
  // 打包时间取构建时刻；本地开发没有注入时显示 dev 作为标识
  const buildInfo = {
    version: env.VITE_APP_VERSION || 'dev',
    buildTime: new Date().toISOString(),
    platform: env.VITE_BUILD_PLATFORM || 'local',
  }

  return {
    plugins: [react()],
    define: {
      __APP_VERSION__: JSON.stringify(buildInfo.version),
      __BUILD_TIME__: JSON.stringify(buildInfo.buildTime),
      __BUILD_PLATFORM__: JSON.stringify(buildInfo.platform),
    },
    server: {
      port: 5173,
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
        },
      },
    },
    preview: {
      port: 4173,
    },
  }
})
