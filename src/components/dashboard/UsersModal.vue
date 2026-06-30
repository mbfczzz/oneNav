<script setup>
import { ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import Modal from '@/components/ui/Modal.vue'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
defineEmits(['update:modelValue'])

const store = useNavStore()
const { show } = useToast()

const users = ref([])
const loading = ref(false)
const error = ref('')

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    users.value = await store.fetchUsers()
  } catch (e) {
    error.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}
watch(
  () => props.modelValue,
  (v) => {
    if (v) refresh()
  },
)

async function remove(u) {
  if (!window.confirm(`确定删除用户「${u.username}」?其个人导航数据将一并删除,且不可恢复。`)) return
  try {
    await store.deleteUser(u.id)
    show('用户已删除', 'success')
    await refresh()
  } catch (e) {
    show(e?.message || '删除失败', 'error')
  }
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    title="用户管理"
    description="管理注册用户及其个人导航"
    icon="ri:team-line"
    max-width="max-w-lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>
    <div v-else-if="error" class="py-8 text-center text-sm text-rose-500">{{ error }}</div>
    <div v-else class="space-y-1">
      <div
        v-for="u in users"
        :key="u.id"
        class="group flex items-center gap-3 rounded-lg px-2 py-2 transition hover:bg-gray-50"
      >
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg"
          :class="u.role === 'admin' ? 'bg-primary-50 text-primary' : 'bg-gray-100 text-gray-500'"
        >
          <Icon
            :icon="u.role === 'admin' ? 'ri:shield-user-line' : 'ri:user-3-line'"
            width="17"
            height="17"
          />
        </span>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-gray-800">
            {{ u.username }}
            <span
              v-if="u.role === 'admin'"
              class="ml-1 rounded bg-primary-50 px-1.5 py-0.5 text-[10px] text-primary"
            >
              管理员
            </span>
          </p>
          <p v-if="u.createdAt" class="text-[11px] text-gray-400">注册于 {{ u.createdAt }}</p>
        </div>
        <button
          v-if="u.username !== store.user?.username"
          class="rounded-md p-1 text-gray-400 transition hover:bg-rose-50 hover:text-rose-500"
          title="删除用户"
          @click="remove(u)"
        >
          <Icon icon="ri:delete-bin-line" width="16" height="16" />
        </button>
        <span v-else class="text-[11px] text-gray-300">本人</span>
      </div>
      <div v-if="users.length === 0" class="py-8 text-center text-sm text-gray-400">暂无用户</div>
    </div>
  </Modal>
</template>
