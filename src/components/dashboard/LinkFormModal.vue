<script setup>
import { ref, watch, computed } from 'vue'
import { Icon } from '@iconify/vue'
import Modal from '@/components/ui/Modal.vue'
import LinkFavicon from '@/components/LinkFavicon.vue'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  link: { type: Object, default: null },
  defaultCategoryId: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useNavStore()
const { show } = useToast()

const categoryId = ref('')
const title = ref('')
const url = ref('')
const description = ref('')
const icon = ref('')
const saving = ref(false)
const error = ref('')
const isEdit = computed(() => !!props.link)

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      categoryId.value =
        props.link?.categoryId || props.defaultCategoryId || store.sortedCategories[0]?.id || ''
      title.value = props.link?.title || ''
      url.value = props.link?.url || ''
      description.value = props.link?.description || ''
      icon.value = props.link?.icon || ''
      error.value = ''
    }
  },
)

async function save() {
  if (!categoryId.value) {
    error.value = '请选择分类'
    return
  }
  if (!title.value.trim() || !url.value.trim()) {
    error.value = '请填写标题与链接地址'
    return
  }
  saving.value = true
  error.value = ''
  try {
    const payload = {
      categoryId: categoryId.value,
      title: title.value.trim(),
      url: url.value.trim(),
      description: description.value.trim(),
      icon: icon.value.trim(),
    }
    if (isEdit.value) await store.updateLink(props.link.id, payload)
    else await store.createLink(payload)
    show(isEdit.value ? '链接已更新' : '链接已创建', 'success')
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
    :title="isEdit ? '编辑链接' : '新建链接'"
    :description="isEdit ? '修改链接信息' : '添加一个网址到当前范围'"
    icon="ri:links-line"
    max-width="max-w-lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="space-y-4">
      <!-- live preview -->
      <div class="flex items-center gap-3 rounded-xl border border-gray-100 bg-gray-50/70 p-3">
        <LinkFavicon :url="url" :title="title || '?'" :icon="icon" :size="40" />
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-gray-800">{{ title || '链接标题' }}</p>
          <p class="truncate text-xs text-gray-400">{{ url || 'https://...' }}</p>
        </div>
      </div>

      <label class="block">
        <span class="mb-1.5 block text-xs font-medium text-gray-500">所属分类 <span class="text-rose-400">*</span></span>
        <select v-model="categoryId" class="input">
          <option v-for="c in store.sortedCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
      </label>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block">
          <span class="mb-1.5 block text-xs font-medium text-gray-500">标题 <span class="text-rose-400">*</span></span>
          <div class="relative">
            <Icon icon="ri:text" width="16" height="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="title" class="input pl-9" placeholder="如:GitHub" />
          </div>
        </label>
        <label class="block">
          <span class="mb-1.5 block text-xs font-medium text-gray-500">图标(可选)</span>
          <div class="relative">
            <Icon icon="ri:image-line" width="16" height="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="icon" class="input pl-9" placeholder="留空则用站点 favicon" />
          </div>
        </label>
      </div>

      <label class="block">
        <span class="mb-1.5 block text-xs font-medium text-gray-500">链接地址 <span class="text-rose-400">*</span></span>
        <div class="relative">
          <Icon icon="ri:link" width="16" height="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input v-model="url" class="input pl-9" placeholder="https://..." @keyup.enter="save" />
        </div>
      </label>

      <label class="block">
        <span class="mb-1.5 block text-xs font-medium text-gray-500">描述(可选)</span>
        <textarea v-model="description" rows="2" class="input resize-none" placeholder="一句话描述" />
      </label>

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
