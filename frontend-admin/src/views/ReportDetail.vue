<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { useToast } from '../composables/useToast'
import LineChart from '../components/LineChart.vue'

const route = useRoute()
const toast = useToast()
const report = ref(null)

onMounted(async () => {
  try {
    report.value = await api.getReport(route.params.id)
  } catch (e) {
    toast.err(e.message || '报告不存在')
  }
})

const xs = computed(() => (report.value?.series || []).map((s) => (s.ts || '').slice(11)))
const rps = computed(() => (report.value?.series || []).map((s) => s.rps))
const p95 = computed(() => (report.value?.series || []).map((s) => s.p95_ms))
const p99 = computed(() => (report.value?.series || []).map((s) => s.p99_ms))

function pillClass(s) {
  if (s === 'completed') return 'border-phosphor text-phosphor'
  if (s === 'stopped') return 'border-warn text-warn'
  if (s === 'failed') return 'border-danger text-danger'
  return 'border-line text-muted'
}
</script>

<template>
  <div v-if="report" class="w-full space-y-5">
    <div>
      <p class="font-mono text-[11px] tracking-[0.24em] text-amber">REPORT</p>
      <h1 class="font-display text-2xl">{{ report.id }}</h1>
      <p class="mt-1 font-mono text-xs text-muted">
        {{ report.method }} {{ report.url }} · tag {{ report.version_tag }} ·
        {{ report.started_at || '—' }} → {{ report.ended_at || '—' }}
        <span v-if="report.status" class="pill ml-2" :class="pillClass(report.status)">{{ report.status }}</span>
      </p>
    </div>
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <article class="panel p-4">
        <p class="font-mono text-[11px] text-muted">AVG RPS</p>
        <p class="font-display text-3xl text-phosphor">{{ Number(report.final_rps || 0).toFixed(0) }}</p>
      </article>
      <article class="panel p-4">
        <p class="font-mono text-[11px] text-muted">P99 ≈</p>
        <p class="font-display text-3xl text-amber">{{ Number(report.p99_ms || 0).toFixed(2) }}ms</p>
      </article>
      <article class="panel p-4">
        <p class="font-mono text-[11px] text-muted">ERROR</p>
        <p class="font-display text-3xl text-danger">{{ ((report.error_rate || 0) * 100).toFixed(2) }}%</p>
      </article>
      <article class="panel p-4">
        <p class="font-mono text-[11px] text-muted">REQUESTS</p>
        <p class="font-display text-3xl">{{ report.total_requests || 0 }}</p>
      </article>
    </div>
    <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
      <LineChart title="RPS SERIES" :x="xs" :series="[{ name: 'RPS', data: rps, color: '#7cffb2', area: 'rgba(124,255,178,0.25)' }]" />
      <LineChart
        title="LATENCY ≈"
        :x="xs"
        :series="[
          { name: 'P95', data: p95, color: '#5ec8ff' },
          { name: 'P99', data: p99, color: '#f6a53b' },
        ]"
      />
    </div>
    <p class="text-xs text-muted">HDR Histogram 近似百分位，误差 ≤ 1%。对比不同版本请使用任务上的版本标签（V1 将提供并排 diff）。</p>
  </div>
</template>
