import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:5173',
    headless: true,
  },
  webServer: [
    { command: 'cd ../backend && go run ./cmd/server', port: 8080, reuseExistingServer: !process.env.CI },
  ],
})
