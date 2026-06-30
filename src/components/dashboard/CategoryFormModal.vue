<script setup>
import { ref, watch, computed } from 'vue'
import { Icon } from '@iconify/vue'
import Modal from '@/components/ui/Modal.vue'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  category: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useNavStore()
const { show } = useToast()

const name = ref('')
const icon = ref('ri:folder-line')
const saving = ref(false)
const error = ref('')
const isEdit = computed(() => !!props.category)

// Quick-pick palette (these literals are also picked up by gen:icons for the offline bundle).
const COMMON_ICONS = [
  'ri:apps-2-line', 'ri:code-s-slash-line', 'ri:palette-line', 'ri:global-line',
  'ri:robot-2-line', 'ri:film-line', 'ri:book-2-line', 'ri:tools-line',
  'ri:music-2-line', 'ri:cloud-line', 'ri:heart-line', 'ri:star-line',
  'ri:gamepad-line', 'ri:image-line', 'ri:links-line', 'ri:briefcase-line',
]

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      name.value = props.category?.name || ''
      icon.value = props.category?.icon || 'ri:folder-line'
      error.value = ''
    }
  },
)

async function save() {
  if (!name.value.trim()) {
    error.value = '请输入分类名称'
    return
  }
  saving.value = true
  error.value = ''
  try {
    const payload = { name: name.value.trim(), icon: icon.value.trim() || 'ri:folder-line' }
    if (isEdit.value) await store.updateCategory(props.category.id, payload)
    else await store.createCategory(payload)
    show(isEdit.value ? '分类已更新' : '分类已创建', 'success')
    emit('saved')
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
    :title="isEdit ? '编辑分类' : '新建分类'"
    :description="isEdit ? '修改分类名称与图标' : '为你的导航新增一个分组'"
    icon="ri:price-tag-3-line"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="space-y-4">
      <label class="block">
        <span class="mb-1.5 block text-xs font-medium text-gray-500">
          名称 <span class="text-rose-400">*</span>
        </span>
        <input v-model="name" class="input" placeholder="如:常用工具" @keyup.enter="save" />
      </label>

      <div>
        <span class="mb-1.5 block text-xs font-medium text-gray-500">图标</span>
        <div class="flex items-center gap-2">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary">
            <Icon :icon="icon || 'ri:folder-line'" width="20" height="20" />
          </span>
          <input v-model="icon" class="input flex-1" placeholder="ri:folder-line" />
        </div>
        <div class="mt-2 flex flex-wrap gap-1.5">
          <button
            v-for="ic in COMMON_ICONS"
            :key="ic"
            type="button"
            class="flex h-8 w-8 items-center justify-center rounded-lg border transition"
            :class="
              icon === ic
                ? 'border-primary bg-primary-50 text-primary'
                : 'border-gray-200 text-gray-400 hover:bg-gray-50 hover:text-gray-600'
            "
            @click="icon = ic"
          >
            <Icon :icon="ic" width="16" height="16" />
          </button>
        </div>
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
