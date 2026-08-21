<script setup>
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../composables/useToast'

const toast = useToast()
const rows = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const data = await api.listReports()
    rows.value = data.items || []
  } catch (e) {
    toast.err(e.message || '读取报告失败')
  } finally {
    loading.value = false
  }
})

function pillClass(s) {
  if (s === 'completed') return 'border-phosphor text-phosphor'
  if (s === 'stopped') return 'border-warn text-warn'
  if (s === 'failed') return 'border-danger text-danger'
  return 'border-line text-muted'
}
</script>

<template>
  <div class="w-full">
    <div class="mb-4">
      <p class="font-mono text-[11px] tracking-[0.24em] text-cyan">AFTER ACTION</p>
      <h1 class="font-display text-2xl">历史报告</h1>
      <p class="mt-1 text-sm text-muted">按版本标签归档。多报告对比属于 V1，本页仅列表与详情。</p>
    </div>
    <div class="panel overflow-x-auto">
      <table class="w-full min-w-[880px] text-left text-sm">
        <thead class="sticky top-0 bg-[#161c26] font-mono text-[11px] uppercase tracking-wider text-muted">
          <tr>
            <th class="px-4 py-3">任务</th>
            <th class="px-4 py-3">版本</th>
            <th class="px-4 py-3">URL</th>
            <th class="px-4 py-3">VU</th>
            <th class="px-4 py-3">开始</th>
            <th class="px-4 py-3">RPS</th>
            <th class="px-4 py-3">P99 ≈</th>
            <th class="px-4 py-3">错误率</th>
            <th class="px-4 py-3">状态</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="9" class="px-4 py-8 text-center text-muted">读取中…</td>
          </tr>
          <tr v-else-if="!rows.length">
            <td colspan="9" class="px-4 py-8 text-center text-muted">还没有落库的报告</td>
          </tr>
          <tr
            v-for="r in rows"
            :key="r.id"
            class="cursor-pointer border-t border-line hover:bg-[#161c26]"
            @click="$router.push(`/reports/${r.id}`)"
          >
            <td class="px-4 py-3 font-mono text-xs text-amber">{{ r.id }}</td>
            <td class="px-4 py-3 font-mono text-xs">{{ r.version_tag }}</td>
            <td class="max-w-xs truncate px-4 py-3 font-mono text-xs">{{ r.url }}</td>
            <td class="px-4 py-3">{{ r.vu }}</td>
            <td class="px-4 py-3 font-mono text-xs">{{ r.started_at || '—' }}</td>
            <td class="px-4 py-3 font-display text-phosphor">{{ Number(r.final_rps || 0).toFixed(0) }}</td>
            <td class="px-4 py-3 font-display text-amber">{{ Number(r.p99_ms || 0).toFixed(2) }}ms</td>
            <td class="px-4 py-3">{{ ((r.error_rate || 0) * 100).toFixed(2) }}%</td>
            <td class="px-4 py-3"><span class="pill" :class="pillClass(r.status)">{{ r.status }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
