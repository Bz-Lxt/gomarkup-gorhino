<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useToast } from '../composables/useToast'
import AppModal from '../components/AppModal.vue'

const router = useRouter()
const toast = useToast()

const form = reactive({
  method: 'GET',
  url: 'http://target:8088/echo',
  headersText: '{\n  "User-Agent": "GoRhino/0.1"\n}',
  body: '',
  vu: 50,
  duration_sec: 30,
  qps: 0,
  version_tag: 'v0.1.0',
})

const errors = reactive({})
const whitelist = ref([])
const submitting = ref(false)
const confirmOpen = ref(false)
const createdId = ref('')

const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD']

function validate() {
  Object.keys(errors).forEach((k) => delete errors[k])
  if (!methods.includes(form.method)) errors.method = '不支持的 HTTP 方法'
  if (!form.url || !/^https?:\/\/.+/i.test(form.url.trim())) errors.url = '必须是 http(s) URL'
  if (!form.version_tag.trim()) errors.version_tag = '版本标签必填'
  const vu = Number(form.vu)
  if (!Number.isInteger(vu) || vu < 1 || vu > 100000) errors.vu = '并发用户数须为 1–100000 的整数'
  const dur = Number(form.duration_sec)
  if (!Number.isInteger(dur) || dur < 1 || dur > 86400) errors.duration_sec = '持续时长须为 1–86400 秒'
  const qps = Number(form.qps)
  if (!Number.isInteger(qps) || qps < 0 || qps > 1000000) errors.qps = 'QPS 须为 0–1000000 的整数，0 表示不限'
  let headers = {}
  try {
    const parsed = JSON.parse(form.headersText || '{}')
    if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
      errors.headersText = 'Headers 必须是 JSON 对象'
    } else {
      for (const [k, v] of Object.entries(parsed)) {
        if (typeof k !== 'string' || typeof v !== 'string') {
          errors.headersText = 'Headers 的键和值都必须是字符串'
          break
        }
      }
      headers = parsed
    }
  } catch {
    errors.headersText = 'Headers 不是合法 JSON'
  }
  if (form.body.length > 1_000_000) errors.body = 'Body 超过 1MB'
  return { ok: Object.keys(errors).length === 0, headers }
}

async function loadWhitelist() {
  try {
    const data = await api.whitelist()
    whitelist.value = data.patterns || []
  } catch (e) {
    toast.err(e.message || '读取白名单失败')
  }
}

function onSubmit() {
  const v = validate()
  if (!v.ok) {
    toast.err('表单未通过校验，请检查标红字段')
    return
  }
  confirmOpen.value = true
}

async function confirmCreate() {
  const v = validate()
  if (!v.ok) return
  submitting.value = true
  try {
    const data = await api.createTask({
      method: form.method,
      url: form.url.trim(),
      headers: v.headers,
      body: form.body,
      vu: Number(form.vu),
      duration_sec: Number(form.duration_sec),
      qps: Number(form.qps),
      version_tag: form.version_tag.trim(),
    })
    createdId.value = data.id
    toast.ok(`任务已创建 ${data.id}`)
    confirmOpen.value = false
  } catch (e) {
    toast.err(e.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function startNow() {
  if (!createdId.value) return
  submitting.value = true
  try {
    await api.startTask(createdId.value)
    toast.ok('已下发，转入实时监控')
    router.push('/live')
  } catch (e) {
    toast.err(e.message || '启动失败')
  } finally {
    submitting.value = false
  }
}

onMounted(loadWhitelist)
</script>

<template>
  <div class="grid w-full grid-cols-12 gap-5">
    <section class="panel col-span-12 p-5 lg:col-span-5">
      <div class="mb-4 flex items-end justify-between">
        <div>
          <p class="font-mono text-[11px] tracking-[0.24em] text-amber">FIRE CONTROL</p>
          <h1 class="font-display text-2xl">任务配置</h1>
        </div>
        <span class="pill border-line text-muted">MVP · 单任务</span>
      </div>

      <div class="space-y-4">
        <div class="grid grid-cols-3 gap-3">
          <label class="col-span-1">
            <span class="lbl">Method *</span>
            <select v-model="form.method" class="field">
              <option v-for="m in methods" :key="m" :value="m">{{ m }}</option>
            </select>
            <p v-if="errors.method" class="err">{{ errors.method }}</p>
          </label>
          <label class="col-span-2">
            <span class="lbl">URL *</span>
            <input v-model="form.url" class="field font-mono" placeholder="http://target:8088/echo" />
            <p v-if="errors.url" class="err">{{ errors.url }}</p>
          </label>
        </div>

        <label class="block">
          <span class="lbl">Headers (JSON 对象)</span>
          <textarea v-model="form.headersText" rows="5" class="field font-mono" />
          <p v-if="errors.headersText" class="err">{{ errors.headersText }}</p>
        </label>

        <label class="block">
          <span class="lbl">Body</span>
          <textarea v-model="form.body" rows="3" class="field font-mono" placeholder="仅 POST/PUT/PATCH 常用" />
          <p v-if="errors.body" class="err">{{ errors.body }}</p>
        </label>

        <div class="grid grid-cols-3 gap-3">
          <label>
            <span class="lbl">并发 VU *</span>
            <input v-model.number="form.vu" type="number" min="1" max="100000" class="field" />
            <p v-if="errors.vu" class="err">{{ errors.vu }}</p>
          </label>
          <label>
            <span class="lbl">持续秒 *</span>
            <input v-model.number="form.duration_sec" type="number" min="1" max="86400" class="field" />
            <p v-if="errors.duration_sec" class="err">{{ errors.duration_sec }}</p>
          </label>
          <label>
            <span class="lbl">QPS 上限</span>
            <input v-model.number="form.qps" type="number" min="0" class="field" />
            <p v-if="errors.qps" class="err">{{ errors.qps }}</p>
          </label>
        </div>

        <label class="block">
          <span class="lbl">版本标签 *（用于后续对比，非自动 Git）</span>
          <input v-model="form.version_tag" class="field font-mono" placeholder="v0.1.0 或 git SHA" />
          <p v-if="errors.version_tag" class="err">{{ errors.version_tag }}</p>
        </label>

        <div class="flex flex-wrap gap-2 pt-2">
          <button class="btn btn-primary" type="button" :disabled="submitting" @click="onSubmit">创建任务</button>
          <button class="btn btn-ghost" type="button" :disabled="!createdId || submitting" @click="startNow">
            下发起压
          </button>
        </div>
        <p v-if="createdId" class="font-mono text-xs text-phosphor">已就绪 {{ createdId }}</p>
      </div>
    </section>

    <aside class="col-span-12 space-y-5 lg:col-span-7">
      <section class="panel p-5">
        <p class="font-mono text-[11px] tracking-[0.24em] text-cyan">TARGET WHITELIST</p>
        <h2 class="mb-3 font-display text-xl">允许压测的地址</h2>
        <p class="mb-3 text-sm leading-7 text-muted">
          默认拒绝白名单外的 URL。解析后的链路本地 / metadata 地址会被 SSRF 护栏拦截。
          内置靶子 <span class="font-mono text-phosphor">http://target:8088/echo</span> 已预置。
        </p>
        <ul class="space-y-1 font-mono text-xs text-ink">
          <li v-for="p in whitelist" :key="p" class="border border-line bg-inset px-3 py-2">{{ p }}</li>
          <li v-if="!whitelist.length" class="text-muted">白名单为空或尚未连上 Master</li>
        </ul>
      </section>
      <section class="panel p-5">
        <p class="font-mono text-[11px] tracking-[0.24em] text-muted">HINT</p>
        <ul class="mt-2 space-y-2 text-sm leading-7 text-muted">
          <li>百分位 P50 / P95 / P99 为 HDR Histogram 近似值，误差 ≤ 1%。</li>
          <li>同一时刻只能运行一条任务。Worker 通过 gRPC 双向流自注册。</li>
          <li>扩容：<span class="font-mono text-ink">docker compose up --scale worker=N</span></li>
        </ul>
      </section>
    </aside>
  </div>

  <AppModal
    v-if="confirmOpen"
    title="确认创建压测任务"
    confirm-text="写入任务"
    :busy="submitting"
    @cancel="confirmOpen = false"
    @confirm="confirmCreate"
  >
    将对 <span class="font-mono text-amber">{{ form.method }} {{ form.url }}</span>
    以 {{ form.vu }} VU 持续 {{ form.duration_sec }}s。版本标签
    <span class="font-mono">{{ form.version_tag }}</span>。
  </AppModal>
</template>
