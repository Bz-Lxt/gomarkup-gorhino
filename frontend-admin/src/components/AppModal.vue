<script setup>
defineProps({
  title: { type: String, required: true },
  confirmText: { type: String, default: '确认' },
  danger: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
})

const emit = defineEmits(['cancel', 'confirm'])
</script>

<template>
  <div class="fixed inset-0 z-40 flex items-center justify-center bg-black/70 px-4">
    <div class="panel w-full max-w-md p-5">
      <h3 class="font-display text-lg tracking-wide text-amber">{{ title }}</h3>
      <div class="mt-3 text-sm leading-7 text-ink">
        <slot />
      </div>
      <div class="mt-5 flex justify-end gap-2">
        <button class="btn btn-ghost" type="button" :disabled="busy" @click="emit('cancel')">取消</button>
        <button
          class="btn"
          :class="danger ? 'btn-danger' : 'btn-primary'"
          type="button"
          :disabled="busy"
          @click="emit('confirm')"
        >
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>
