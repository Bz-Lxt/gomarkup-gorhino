import { test, expect } from '@playwright/test'

const base = process.env.GORHINO_BASE || 'http://127.0.0.1:37101'

test.describe('GoRhino critical path (Target / mock, cost ¥0)', () => {
  test('health via console proxy', async ({ request }) => {
    const res = await request.get(`${base}/api/v1/health`)
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.ok).toBeTruthy()
    expect(body.data.service).toBe('master')
  })

  test('create start live report against builtin target', async ({ page, request }) => {
    await page.goto(base + '/')
    await expect(page.getByRole('heading', { name: '任务配置' })).toBeVisible()
    await page.locator('input[placeholder="http://target:8088/echo"]').fill('http://target:8088/echo')
    await page.getByRole('button', { name: '创建任务' }).click()
    await page.getByRole('button', { name: '写入任务' }).click()
    await expect(page.getByText(/已就绪 tsk_/)).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: '下发起压' }).click()
    await expect(page).toHaveURL(/\/live/)
    await expect(page.getByRole('heading', { name: '实时监控' })).toBeVisible()

    const health = await request.get(`${base}/api/v1/tasks`)
    expect(health.ok()).toBeTruthy()
    const listed = await health.json()
    const running = (listed.data.items || []).find((t: { status: string }) => t.status === 'running' || t.status === 'completed' || t.status === 'stopped')
    expect(running).toBeTruthy()
  })

  test('nodes page renders', async ({ page }) => {
    await page.goto(base + '/nodes')
    await expect(page.getByRole('heading', { name: 'Worker 节点' })).toBeVisible()
  })

  test('reports page renders', async ({ page }) => {
    await page.goto(base + '/reports')
    await expect(page.getByRole('heading', { name: '历史报告' })).toBeVisible()
  })
})
