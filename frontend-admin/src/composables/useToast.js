import { reactive } from 'vue'

const state = reactive({
  items: [],
})

let seq = 0

function push(type, message) {
  const id = ++seq
  state.items.push({ id, type, message })
  window.setTimeout(() => dismiss(id), 5000)
}

function dismiss(id) {
  const i = state.items.findIndex((t) => t.id === id)
  if (i >= 0) state.items.splice(i, 1)
}

export function useToast() {
  return {
    state,
    ok: (m) => push('ok', m),
    err: (m) => push('err', m),
    info: (m) => push('info', m),
    dismiss,
  }
}
