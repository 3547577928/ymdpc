import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:5173',
    headless: true,
  },
  // 干净环境可直接跑：自动拉起后端 (8080) 与 Vite dev server (5173)。
  // 后端冷编译（go run 首次拉依赖）可能超过默认 60s，放宽到 180s
  webServer: [
    { command: 'cd ../backend && go run ./cmd/server', port: 8080, reuseExistingServer: !process.env.CI, timeout: 180_000 },
    { command: 'npx vite --port 5173', port: 5173, reuseExistingServer: !process.env.CI },
  ],
})
