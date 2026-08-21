<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api, liveSocket } from '../api'
import { useToast } from '../composables/useToast'
import AppModal from '../components/AppModal.vue'
import LineChart from '../components/LineChart.vue'

const toast = useToast()
const frame = ref(null)
const xs = ref([])
const rps = ref([])
const p95 = ref([])
const p99 = ref([])
const confirmStop = ref(false)
const stopping = ref(false)
let ws
let pingTimer

function pushPoint(f) {
  frame.value = f
  const label = (f.ts || '').slice(11) || String(xs.value.length)
  xs.value = [...xs.value, label].slice(-120)
  rps.value = [...rps.value, Number(f.rps || 0)].slice(-120)
  p95.value = [...p95.value, Number(f.p95_ms || 0)].slice(-120)
  p99.value = [...p99.value, Number(f.p99_ms || 0)].slice(-120)
}

function connect() {
  ws = liveSocket()
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      if (msg.type === 'hello') return
      if (msg.type === 'frame' && msg.data) pushPoint(msg.data)
    } catch {
      /* ignore malformed */
    }
  }
  ws.onerror = () => toast.err('实时通道异常')
  ws.onclose = () => {
    window.setTimeout(() => {
      if (!ws || ws.readyState === WebSocket.CLOSED) connect()
    }, 1500)
  }
}

async function doStop() {
  const id = frame.value && frame.value.task_id
  if (!id) return
  stopping.value = true
  try {
    await api.stopTask(id)
    toast.ok('已发送优雅停止')
    confirmStop.value = false
  } catch (e) {
    toast.err(e.message || '停止失败')
  } finally {
    stopping.value = false
  }
}

const tiles = computed(() => {
  const f = frame.value || {}
  return [
    { k: 'RPS', v: fmt(f.rps, 0), sub: 'completed / s', color: 'text-phosphor' },
    { k: 'P95 ≈', v: fmt(f.p95_ms, 2), sub: 'ms · HDR', color: 'text-cyan' },
    { k: 'P99 ≈', v: fmt(f.p99_ms, 2), sub: 'ms · HDR', color: 'text-amber' },
    { k: 'ERROR', v: pct(f.error_rate), sub: 'non-2xx + transport', color: 'text-danger' },
  ]
})

function fmt(n, d) {
  if (n === undefined || n === null || Number.isNaN(Number(n))) return '—'
  return Number(n).toFixed(d)
}
function pct(n) {
  if (n === undefined || n === null) return '—'
  return `${(Number(n) * 100).toFixed(2)}%`
}

onMounted(() => {
  connect()
  pingTimer = window.setInterval(() => {
    if (ws && ws.readyState === WebSocket.OPEN) ws.send('{"type":"ping"}')
  }, 10000)
})

onUnmounted(() => {
  window.clearInterval(pingTimer)
  if (ws) {
    ws.onclose = null
    ws.close()
  }
})
</script>

<template>
  <div class="w-full space-y-5">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="font-mono text-[11px] tracking-[0.24em] text-phosphor">LIVE WALL</p>
        <h1 class="font-display text-2xl">实时监控</h1>
        <p class="mt-1 font-mono text-xs text-muted">
          百分位为 HDR 近似（≤ 1%）。时间 {{ frame && frame.ts ? frame.ts : '—' }}
        </p>
      </div>
      <div class="flex items-center gap-3">
        <span class="pill border-line text-muted">TASK {{ frame && frame.task_id ? frame.task_id : 'IDLE' }}</span>
        <span class="pill border-line text-muted">{{ frame && frame.status ? frame.status : 'idle' }}</span>
        <button class="btn btn-danger" type="button" :disabled="!frame || frame.status !== 'running'" @click="confirmStop = true">
          停止
        </button>
      </div>
    </div>

    <div v-if="!frame" class="panel p-8 text-center">
      <p class="font-display text-xl text-muted">WAITING FOR FIRE</p>
      <p class="mt-2 text-sm text-muted">还没有运行中的任务。<router-link to="/" class="text-amber">去配置并下发</router-link></p>
    </div>

    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <article v-for="t in tiles" :key="t.k" class="panel p-4">
        <p class="font-mono text-[11px] tracking-[0.2em] text-muted">{{ t.k }}</p>
        <p class="font-display text-4xl" :class="t.color">{{ t.v }}</p>
        <p class="mt-1 text-xs text-muted">{{ t.sub }}</p>
      </article>
    </div>

    <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
      <LineChart
        title="RPS"
        :x="xs"
        :series="[{ name: 'RPS', data: rps, color: '#7cffb2', area: 'rgba(124,255,178,0.25)' }]"
      />
      <LineChart
        title="LATENCY ≈ ms"
        :x="xs"
        :series="[
          { name: 'P95', data: p95, color: '#5ec8ff' },
          { name: 'P99', data: p99, color: '#f6a53b' },
        ]"
      />
    </div>

    <section v-if="frame" class="panel p-4">
      <p class="mb-3 font-mono text-[11px] tracking-[0.2em] text-muted">STATUS MIX / WORKERS {{ frame.workers }} · 剩余 {{ frame.remaining_sec }}s</p>
      <div class="flex flex-wrap gap-2 font-mono text-xs">
        <span v-for="(n, k) in frame.codes || {}" :key="k" class="pill border-line">{{ k }} {{ n }}</span>
      </div>
    </section>
  </div>

  <AppModal
    v-if="confirmStop"
    title="停止当前压测"
    confirm-text="优雅停止"
    danger
    :busy="stopping"
    @cancel="confirmStop = false"
    @confirm="doStop"
  >
    Worker 将在 5 秒内排空在途请求并上报最后一帧直方图。
  </AppModal>
</template>
