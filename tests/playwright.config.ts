import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: '.',
  timeout: 60_000,
  use: {
    baseURL: process.env.GORHINO_BASE || 'http://127.0.0.1:37101',
    locale: 'zh-CN',
  },
  reporter: [['list']],
})
