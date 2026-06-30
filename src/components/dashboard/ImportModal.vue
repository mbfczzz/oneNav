<script setup>
import { ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import Modal from '@/components/ui/Modal.vue'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue', 'imported'])

const store = useNavStore()
const { show } = useToast()

const text = ref('')
const importing = ref(false)
const error = ref('')
const dragging = ref(false)
const fileName = ref('')

const SAMPLE = `{
  "categories": [{ "name": "我的收藏", "icon": "ri:star-line" }],
  "links": [
    { "category": "我的收藏", "title": "GitHub", "url": "https://github.com", "description": "代码托管" }
  ]
}`

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      error.value = ''
      text.value = ''
      fileName.value = ''
    }
  },
)

function readFile(file) {
  if (!file) return
  fileName.value = file.name
  const reader = new FileReader()
  reader.onload = () => {
    text.value = String(reader.result || '')
  }
  reader.readAsText(file)
}
function onFile(e) {
  readFile(e.target.files?.[0])
}
function onDrop(e) {
  dragging.value = false
  readFile(e.dataTransfer?.files?.[0])
}

async function runImport() {
  error.value = ''
  let parsed
  try {
    parsed = JSON.parse(text.value)
  } catch {
    error.value = '仅支持 ZMark/OneNav 导出的 JSON 文件'
    return
  }
  if (parsed == null || typeof parsed !== 'object') {
    error.value = '仅支持 ZMark/OneNav 导出的 JSON 文件'
    return
  }
  importing.value = true
  try {
    const res = await store.importData(parsed)
    let msg = `导入完成:新增 ${res.addedCategories} 个分类、${res.addedLinks} 条链接`
    if (res.skipped) msg += `,跳过 ${res.skipped} 条(重复或无效)`
    if (res.truncated) msg += `;已达 2000 条上限,其余未导入`
    show(msg, 'success', 4000)
    emit('imported')
    emit('update:modelValue', false)
    text.value = ''
  } catch (e) {
    error.value = e?.message || '导入失败'
  } finally {
    importing.value = false
  }
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    title="批量导入"
    :description="`导入到「${store.scope === 'mine' ? '我的导航' : '通用导航'}」`"
    icon="ri:upload-cloud-2-line"
    max-width="max-w-lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="space-y-3">
      <!-- drop zone -->
      <label
        class="flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-4 py-6 text-center transition"
        :class="dragging ? 'border-primary bg-primary-50/60' : 'border-gray-200 hover:border-primary/50 hover:bg-gray-50'"
        @dragover.prevent="dragging = true"
        @dragleave.prevent="dragging = false"
        @drop.prevent="onDrop"
      >
        <Icon icon="ri:upload-cloud-2-line" width="28" height="28" class="text-primary/70" />
        <p class="mt-2 text-sm text-gray-600">
          <span class="font-medium text-primary">点击选择</span> 或拖拽 JSON 文件到这里
        </p>
        <p v-if="fileName" class="mt-1 text-xs text-gray-400">{{ fileName }}</p>
        <input type="file" accept=".json,application/json" class="hidden" @change="onFile" />
      </label>

      <div class="flex items-center justify-between">
        <p class="text-[11px] text-gray-400">支持 ZMark / OneNav JSON;单次最多 2000 条,同名分类合并、重复链接自动跳过。</p>
        <button class="shrink-0 text-[11px] text-primary hover:underline" @click="text = SAMPLE">填入示例</button>
      </div>

      <textarea
        v-model="text"
        rows="8"
        class="input resize-none bg-gray-50 font-mono text-xs leading-relaxed"
        placeholder="也可直接在此粘贴 JSON…"
      />
      <p v-if="error" class="flex items-center gap-1 text-xs text-rose-500">
        <Icon icon="ri:error-warning-line" width="14" height="14" />
        {{ error }}
      </p>
    </div>

    <template #footer>
      <button class="btn-ghost" @click="$emit('update:modelValue', false)">取消</button>
      <button class="btn-primary" :disabled="importing || !text.trim()" @click="runImport">
        <Icon
          :icon="importing ? 'ri:loader-4-line' : 'ri:upload-2-line'"
          width="15"
          height="15"
          :class="importing ? 'animate-spin' : ''"
        />
        导入
      </button>
    </template>
  </Modal>
</template>
