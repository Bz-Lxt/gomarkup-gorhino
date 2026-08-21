async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(path, { ...options, headers })
  let payload = null
  try {
    payload = await res.json()
  } catch {
    payload = null
  }
  if (!payload) {
    throw { code: 'INTERNAL', message: `空响应 (${res.status})` }
  }
  if (!payload.ok) {
    const err = payload.error || {}
    throw { code: err.code || 'INTERNAL', message: err.message || '请求失败' }
  }
  return payload.data
}

export const api = {
  health: () => request('/api/v1/health'),
  nodes: () => request('/api/v1/nodes'),
  whitelist: () => request('/api/v1/whitelist'),
  putWhitelist: (patterns) =>
    request('/api/v1/whitelist', { method: 'PUT', body: JSON.stringify({ patterns }) }),
  createTask: (body) => request('/api/v1/tasks', { method: 'POST', body: JSON.stringify(body) }),
  listTasks: () => request('/api/v1/tasks'),
  getTask: (id) => request(`/api/v1/tasks/${encodeURIComponent(id)}`),
  startTask: (id) => request(`/api/v1/tasks/${encodeURIComponent(id)}/start`, { method: 'POST' }),
  stopTask: (id) => request(`/api/v1/tasks/${encodeURIComponent(id)}/stop`, { method: 'POST' }),
  listReports: () => request('/api/v1/reports'),
  getReport: (id) => request(`/api/v1/reports/${encodeURIComponent(id)}`),
}

export function liveSocket() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  return new WebSocket(`${proto}://${location.host}/api/v1/ws/live`)
}
