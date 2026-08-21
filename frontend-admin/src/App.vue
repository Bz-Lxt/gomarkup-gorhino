<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppToast from './components/AppToast.vue'
import { api } from './api'

const route = useRoute()
const alive = ref(0)
const lost = ref(0)
let timer

const links = [
  { to: '/', label: '任务配置' },
  { to: '/live', label: '实时监控' },
  { to: '/reports', label: '历史报告' },
  { to: '/nodes', label: '节点' },
]

const active = computed(() => route.path)

async function refreshNodes() {
  try {
    const data = await api.nodes()
    const list = data.nodes || []
    alive.value = list.filter((n) => n.state === 'alive').length
    lost.value = list.filter((n) => n.state !== 'alive').length
  } catch {
    /* toast reserved for user actions */
  }
}

onMounted(() => {
  refreshNodes()
  timer = window.setInterval(refreshNodes, 3000)
})

onUnmounted(() => window.clearInterval(timer))
</script>

<template>
  <div class="relative min-h-screen w-full">
    <header class="sticky top-0 z-30 border-b border-line bg-[#0c1016]/90 backdrop-blur">
      <div class="flex w-full flex-wrap items-center gap-4 px-4 py-3 md:px-6">
        <router-link to="/" class="flex items-center gap-3">
          <span class="grid h-9 w-9 place-items-center border border-amber text-amber">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M3 16 L10 7 L14 11 L21 4" stroke="currentColor" stroke-width="2" />
              <path d="M16 16 H21 V11" stroke="currentColor" stroke-width="2" />
            </svg>
          </span>
          <span>
            <span class="block font-display text-lg tracking-[0.22em] text-amber">GORHINO</span>
            <span class="block font-mono text-[10px] tracking-[0.28em] text-muted">RANGE TELEMETRY</span>
          </span>
        </router-link>
        <nav class="flex flex-1 flex-wrap gap-1">
          <router-link
            v-for="l in links"
            :key="l.to"
            :to="l.to"
            class="px-3 py-1.5 text-sm"
            :class="active === l.to || (l.to !== '/' && active.startsWith(l.to)) ? 'text-amber border-b-2 border-amber' : 'text-muted hover:text-ink'"
          >
            {{ l.label }}
          </router-link>
        </nav>
        <div class="flex items-center gap-3 font-mono text-xs">
          <span class="flex items-center gap-2 text-phosphor">
            <i class="is-live inline-block h-2 w-2 bg-phosphor" />
            WORKERS {{ alive }}
          </span>
          <span v-if="lost" class="text-danger">LOST {{ lost }}</span>
        </div>
      </div>
    </header>
    <main class="w-full px-4 py-5 md:px-6 md:py-6">
      <router-view />
    </main>
    <AppToast />
  </div>
</template>
