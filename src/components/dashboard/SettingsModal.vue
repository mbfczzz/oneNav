<script setup>
import { ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import Modal from '@/components/ui/Modal.vue'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue'])

const store = useNavStore()
const { show } = useToast()

const siteName = ref('')
const saving = ref(false)
const error = ref('')

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      siteName.value = store.settings.siteName || ''
      error.value = ''
    }
  },
)

async function save() {
  if (!siteName.value.trim()) {
    error.value = '请输入站点名称'
    return
  }
  saving.value = true
  error.value = ''
  try {
    await store.updateSettings({ siteName: siteName.value.trim() })
    show('站点设置已保存', 'success')
    emit('update:modelValue', false)
  } catch (e) {
    error.value = e?.message || '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    title="站点设置"
    description="自定义站点名称(浏览器标题 / 顶栏 / 页脚)"
    icon="ri:settings-3-line"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="space-y-3">
      <label class="block">
        <span class="mb-1.5 block text-xs font-medium text-gray-500">
          站点名称 <span class="text-rose-400">*</span>
        </span>
        <input
          v-model="siteName"
          class="input"
          placeholder="如:我的导航"
          maxlength="40"
          @keyup.enter="save"
        />
      </label>
      <div class="rounded-lg bg-gray-50 p-3 text-xs text-gray-500">
        预览:浏览器标题与顶栏将显示
        <span class="font-semibold text-gray-800">{{ siteName || '启点导航' }}</span>
      </div>
      <p v-if="error" class="flex items-center gap-1 text-xs text-rose-500">
        <Icon icon="ri:error-warning-line" width="14" height="14" />
        {{ error }}
      </p>
    </div>
    <template #footer>
      <button class="btn-ghost" @click="$emit('update:modelValue', false)">取消</button>
      <button class="btn-primary" :disabled="saving" @click="save">
        <Icon v-if="saving" icon="ri:loader-4-line" width="15" height="15" class="animate-spin" />
        保存
      </button>
    </template>
  </Modal>
</template>
