<script setup>
import { useToast } from '../composables/useToast'

const toast = useToast()
</script>

<template>
  <div class="fixed top-4 right-4 z-50 flex w-full max-w-sm flex-col gap-2 px-3 sm:px-0">
    <div
      v-for="item in toast.state.items"
      :key="item.id"
      class="panel flex items-start gap-3 px-3 py-2"
      :class="{
        'border-phosphor': item.type === 'ok',
        'border-danger': item.type === 'err',
        'border-cyan': item.type === 'info',
      }"
    >
      <span
        class="mt-1 h-2 w-2 shrink-0"
        :class="{
          'bg-phosphor': item.type === 'ok',
          'bg-danger': item.type === 'err',
          'bg-cyan': item.type === 'info',
        }"
      />
      <p class="flex-1 text-sm leading-6">{{ item.message }}</p>
      <button class="text-muted hover:text-ink" type="button" @click="toast.dismiss(item.id)">×</button>
    </div>
  </div>
</template>
