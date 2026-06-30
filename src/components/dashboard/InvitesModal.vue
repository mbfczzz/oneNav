<script setup>
import { ref, watch, reactive } from 'vue'
import { Icon } from '@iconify/vue'
import Modal from '@/components/ui/Modal.vue'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
defineEmits(['update:modelValue'])

const store = useNavStore()
const { show } = useToast()

const invites = ref([])
const loading = ref(false)
const error = ref('')
const creating = ref(false)
const form = reactive({ note: '', maxUses: 1, expiresInDays: 0, role: 'user' })

// usage-records expand state
const expandedId = ref('')
const uses = reactive({})
const usesLoading = ref(false)

const STATUS = {
  active: { label: '有效', cls: 'bg-emerald-50 text-emerald-600' },
  used: { label: '已用尽', cls: 'bg-gray-100 text-gray-500' },
  expired: { label: '已过期', cls: 'bg-amber-50 text-amber-600' },
  disabled: { label: '已停用', cls: 'bg-rose-50 text-rose-500' },
}
const EXPIRY = [
  { v: 0, t: '永久' },
  { v: 7, t: '7 天' },
  { v: 30, t: '30 天' },
  { v: 90, t: '90 天' },
]

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    invites.value = await store.fetchInvites()
  } catch (e) {
    error.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}
watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      form.note = ''
      form.maxUses = 1
      form.expiresInDays = 0
      form.role = 'user'
      expandedId.value = ''
      refresh()
    }
  },
)

async function create() {
  creating.value = true
  try {
    const inv = await store.createInvite({
      note: form.note.trim(),
      maxUses: Number(form.maxUses) || 0,
      expiresInDays: Number(form.expiresInDays) || 0,
      role: form.role,
    })
    show('邀请码已生成:' + inv.code, 'success', 3000)
    form.note = ''
    form.role = 'user'
    await refresh()
  } catch (e) {
    show(e?.message || '生成失败', 'error')
  } finally {
    creating.value = false
  }
}
async function copy(code) {
  try {
    await navigator.clipboard.writeText(code)
    show('邀请码已复制', 'success')
  } catch {
    show('复制失败', 'error')
  }
}
async function toggle(inv) {
  try {
    await store.toggleInvite(inv.id, !inv.disabled)
    await refresh()
  } catch (e) {
    show(e?.message || '操作失败', 'error')
  }
}
async function remove(inv) {
  if (!window.confirm(`确定删除邀请码「${inv.code}」?`)) return
  try {
    await store.deleteInvite(inv.id)
    show('已删除', 'success')
    await refresh()
  } catch (e) {
    show(e?.message || '删除失败', 'error')
  }
}
async function toggleUses(inv) {
  if (expandedId.value === inv.id) {
    expandedId.value = ''
    return
  }
  expandedId.value = inv.id
  if (!uses[inv.id]) {
    usesLoading.value = true
    try {
      uses[inv.id] = await store.fetchInviteUses(inv.id)
    } catch (e) {
      uses[inv.id] = []
      show(e?.message || '加载记录失败', 'error')
    } finally {
      usesLoading.value = false
    }
  }
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    title="邀请码管理"
    description="生成并管理注册邀请码"
    icon="ri:ticket-2-line"
    max-width="max-w-2xl"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <!-- create -->
    <div class="mb-4 rounded-xl border border-gray-100 bg-gray-50/70 p-3">
      <div class="flex flex-wrap items-center gap-2">
        <input v-model="form.note" class="input min-w-0 flex-1" placeholder="备注(可选,如:给朋友)" />
        <input
          v-model.number="form.maxUses"
          type="number"
          min="0"
          class="input w-20"
          title="可用次数(0 = 不限)"
          placeholder="次数"
        />
        <select v-model.number="form.expiresInDays" class="input w-24" title="有效期">
          <option v-for="e in EXPIRY" :key="e.v" :value="e.v">{{ e.t }}</option>
        </select>
        <select v-model="form.role" class="input w-28" title="注册后角色">
          <option value="user">普通用户</option>
          <option value="admin">管理员</option>
        </select>
        <button class="btn-primary" :disabled="creating" @click="create">
          <Icon
            :icon="creating ? 'ri:loader-4-line' : 'ri:add-line'"
            width="15"
            height="15"
            :class="creating ? 'animate-spin' : ''"
          />
          生成
        </button>
      </div>
      <p class="mt-1.5 text-[11px] text-gray-400">
        次数填 0 表示不限;有效期「永久」长期有效;角色「管理员」将授予用此码注册者管理员权限,请谨慎使用。
      </p>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-400">加载中…</div>
    <div v-else-if="error" class="py-8 text-center text-sm text-rose-500">{{ error }}</div>
    <div v-else-if="invites.length === 0" class="py-8 text-center text-sm text-gray-400">
      还没有邀请码,生成一个吧
    </div>
    <div v-else class="space-y-1.5">
      <div v-for="inv in invites" :key="inv.id" class="rounded-lg border border-gray-100">
        <div class="flex items-center gap-3 px-3 py-2">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <code
                class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-sm font-semibold tracking-wider text-gray-800"
              >
                {{ inv.code }}
              </code>
              <span class="rounded-full px-2 py-0.5 text-[10px]" :class="STATUS[inv.status]?.cls">
                {{ STATUS[inv.status]?.label }}
              </span>
              <span
                v-if="inv.grantRole === 'admin'"
                class="inline-flex items-center gap-0.5 rounded-full bg-amber-50 px-2 py-0.5 text-[10px] text-amber-600"
              >
                <Icon icon="ri:shield-user-line" width="11" height="11" />管理员邀请
              </span>
              <button class="text-gray-400 transition hover:text-primary" title="复制" @click="copy(inv.code)">
                <Icon icon="ri:file-copy-line" width="14" height="14" />
              </button>
            </div>
            <p class="mt-0.5 text-[11px] text-gray-400">
              <span v-if="inv.note">{{ inv.note }} · </span>
              用量 {{ inv.usedCount }}/{{ inv.maxUses === 0 ? '∞' : inv.maxUses }}
              <span v-if="inv.expiresAt"> · 至 {{ inv.expiresAt }}</span>
              <span v-else> · 永久</span>
            </p>
          </div>
          <button
            class="rounded-md p-1 text-gray-400 transition hover:bg-gray-100 hover:text-primary"
            title="注册记录"
            @click="toggleUses(inv)"
          >
            <Icon icon="ri:eye-line" width="16" height="16" />
          </button>
          <button
            class="rounded-md p-1 text-gray-400 transition hover:bg-gray-100"
            :title="inv.disabled ? '启用' : '停用'"
            @click="toggle(inv)"
          >
            <Icon :icon="inv.disabled ? 'ri:play-circle-line' : 'ri:pause-circle-line'" width="16" height="16" />
          </button>
          <button
            class="rounded-md p-1 text-gray-400 transition hover:bg-rose-50 hover:text-rose-500"
            title="删除"
            @click="remove(inv)"
          >
            <Icon icon="ri:delete-bin-line" width="16" height="16" />
          </button>
        </div>

        <!-- usage records -->
        <div v-if="expandedId === inv.id" class="border-t border-gray-100 bg-gray-50/50 px-3 py-2">
          <p v-if="usesLoading && !uses[inv.id]" class="text-[11px] text-gray-400">加载记录…</p>
          <p v-else-if="!uses[inv.id] || uses[inv.id].length === 0" class="text-[11px] text-gray-400">
            暂无注册记录
          </p>
          <ul v-else class="space-y-0.5">
            <li
              v-for="(u, i) in uses[inv.id]"
              :key="i"
              class="flex items-center justify-between text-[11px] text-gray-500"
            >
              <span class="inline-flex items-center gap-1">
                <Icon icon="ri:user-3-line" width="12" height="12" />{{ u.username }}
              </span>
              <span class="text-gray-400">{{ u.usedAt || '—' }}</span>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </Modal>
</template>
