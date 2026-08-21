<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

const props = defineProps({
  title: { type: String, required: true },
  x: { type: Array, default: () => [] },
  series: { type: Array, default: () => [] },
  empty: { type: String, default: 'WAITING FOR FIRE' },
})

const el = ref(null)
let chart

function option() {
  return {
    backgroundColor: 'transparent',
    animationDuration: 250,
    title: {
      text: props.x.length ? '' : props.empty,
      left: 'center',
      top: 'middle',
      textStyle: { color: '#8b9bb4', fontFamily: 'Oxanium', fontSize: 14, letterSpacing: 3 },
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#10141b',
      borderColor: '#243044',
      textStyle: { color: '#e8edf5', fontFamily: 'Noto Sans SC' },
    },
    legend: {
      top: 4,
      right: 8,
      textStyle: { color: '#8b9bb4', fontFamily: 'IBM Plex Mono', fontSize: 11 },
    },
    grid: { left: 48, right: 16, top: 36, bottom: 28 },
    xAxis: {
      type: 'category',
      data: props.x,
      boundaryGap: false,
      axisLine: { lineStyle: { color: '#243044' } },
      axisLabel: { color: '#8b9bb4', fontSize: 10, fontFamily: 'IBM Plex Mono' },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: '#1a2230' } },
      axisLabel: { color: '#8b9bb4', fontSize: 10, fontFamily: 'Oxanium' },
    },
    series: props.series.map((s) => ({
      name: s.name,
      type: 'line',
      showSymbol: false,
      smooth: true,
      data: s.data,
      lineStyle: { width: 2, color: s.color },
      itemStyle: { color: s.color },
      areaStyle: s.area
        ? { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: s.area },
            { offset: 1, color: 'rgba(0,0,0,0)' },
          ]) }
        : undefined,
    })),
  }
}

function render() {
  if (!chart) return
  chart.setOption(option(), true)
}

onMounted(() => {
  chart = echarts.init(el.value, null, { renderer: 'canvas' })
  render()
  window.addEventListener('resize', resize)
})

function resize() {
  chart && chart.resize()
}

watch(() => [props.x, props.series], render, { deep: true })

onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  chart && chart.dispose()
})
</script>

<template>
  <div class="panel scanline relative p-3">
    <div class="mb-1 font-mono text-[11px] uppercase tracking-[0.2em] text-muted">{{ title }}</div>
    <div ref="el" class="h-64 w-full sm:h-72"></div>
  </div>
</template>
