<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { api } from '../api'
import { useToast } from '../composables/useToast'

const toast = useToast()
const nodes = ref([])
let timer

async function load() {
  try {
    const data = await api.nodes()
    nodes.value = data.nodes || []
  } catch (e) {
    toast.err(e.message || '读取节点失败')
  }
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 3000)
})
onUnmounted(() => window.clearInterval(timer))
</script>

<template>
  <div class="w-full">
    <div class="mb-4">
      <p class="font-mono text-[11px] tracking-[0.24em] text-phosphor">CLUSTER</p>
      <h1 class="font-display text-2xl">Worker 节点</h1>
      <p class="mt-1 text-sm text-muted">Worker 主动向 Master 建立 gRPC 双向流。心跳 3s，10s 无心跳判失联。</p>
    </div>
    <div class="panel overflow-x-auto">
      <table class="w-full min-w-[720px] text-left text-sm">
        <thead class="bg-[#161c26] font-mono text-[11px] uppercase tracking-wider text-muted">
          <tr>
            <th class="px-4 py-3">Node</th>
            <th class="px-4 py-3">Hostname</th>
            <th class="px-4 py-3">CPU</th>
            <th class="px-4 py-3">State</th>
            <th class="px-4 py-3">Last heartbeat</th>
            <th class="px-4 py-3">Assigned VU</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!nodes.length">
            <td colspan="6" class="px-4 py-8 text-center text-muted">尚无 Worker 注册</td>
          </tr>
          <tr v-for="n in nodes" :key="n.id" class="border-t border-line">
            <td class="px-4 py-3 font-mono text-xs text-amber">{{ n.id }}</td>
            <td class="px-4 py-3 font-mono text-xs">{{ n.hostname }}</td>
            <td class="px-4 py-3">{{ n.cpu_count }}</td>
            <td class="px-4 py-3">
              <span class="pill" :class="n.state === 'alive' ? 'border-phosphor text-phosphor' : 'border-danger text-danger'">
                {{ n.state }}
              </span>
            </td>
            <td class="px-4 py-3 font-mono text-xs">{{ n.last_heartbeat }}</td>
            <td class="px-4 py-3">{{ n.assigned_vu || 0 }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
