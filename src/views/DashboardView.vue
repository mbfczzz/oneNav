<script setup>
import { ref, computed, watch, onMounted, reactive } from 'vue'
import { Icon } from '@iconify/vue'
import { useRouter, useRoute } from 'vue-router'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'
import { API_MODE } from '@/api'
import { VueDraggable } from 'vue-draggable-plus'
import LinkFavicon from '@/components/LinkFavicon.vue'
import Modal from '@/components/ui/Modal.vue'
import CategoryFormModal from '@/components/dashboard/CategoryFormModal.vue'
import LinkFormModal from '@/components/dashboard/LinkFormModal.vue'
import ImportModal from '@/components/dashboard/ImportModal.vue'
import UsersModal from '@/components/dashboard/UsersModal.vue'
import InvitesModal from '@/components/dashboard/InvitesModal.vue'
import SettingsModal from '@/components/dashboard/SettingsModal.vue'

const store = useNavStore()
const router = useRouter()
const route = useRoute()
const { show } = useToast()

const selectedCatId = ref('')

// modal state
const showCat = ref(false)
const editingCat = ref(null)
const showLink = ref(false)
const editingLink = ref(null)
const showImport = ref(false)
const showUsers = ref(false)
const showInvites = ref(false)
const showSettings = ref(false)
const confirm = reactive({ open: false, title: '', message: '', onConfirm: null })

// reorderable mirrors
const cats = ref([])
const links = ref([])
watch(
  () => store.sortedCategories,
  (v) => {
    cats.value = [...v]
    if (!selectedCatId.value && v.length) selectedCatId.value = v[0].id
    if (selectedCatId.value && !v.find((c) => c.id === selectedCatId.value)) {
      selectedCatId.value = v[0]?.id || ''
    }
  },
  { immediate: true, deep: true },
)
watch(
  [selectedCatId, () => store.links],
  () => {
    links.value = selectedCatId.value ? store.linksByCategory(selectedCatId.value) : []
  },
  { immediate: true, deep: true },
)

const selectedCat = computed(() => store.categories.find((c) => c.id === selectedCatId.value))

onMounted(async () => {
  // Normal users manage only their own nav; admins keep whatever scope they're in.
  if (!store.isAdmin && store.scope !== 'mine') await store.setScope('mine')
  else if (!store.loadedScope) await store.loadAll()
})

// ---- categories ----
function newCategory() {
  editingCat.value = null
  showCat.value = true
}
function editCategory(cat) {
  editingCat.value = cat
  showCat.value = true
}
function askDeleteCategory(cat) {
  confirm.title = '删除分类'
  confirm.message = `确定删除分类「${cat.name}」?其下所有链接也会一并删除,且不可恢复。`
  confirm.onConfirm = async () => {
    await store.deleteCategory(cat.id)
    show('分类已删除', 'success')
  }
  confirm.open = true
}
async function onCatsSorted() {
  const ok = await store.reorderCategories(cats.value.map((c) => c.id))
  show(ok ? '排序已保存' : '排序保存失败', ok ? 'success' : 'error')
}

// ---- links ----
function newLink() {
  editingLink.value = null
  showLink.value = true
}
function editLink(link) {
  editingLink.value = link
  showLink.value = true
}
function askDeleteLink(link) {
  confirm.title = '删除链接'
  confirm.message = `确定删除链接「${link.title}」?`
  confirm.onConfirm = async () => {
    await store.deleteLink(link.id)
    show('链接已删除', 'success')
  }
  confirm.open = true
}
async function onLinksSorted() {
  const ok = await store.reorderLinks(links.value.map((l) => l.id))
  show(ok ? '排序已保存' : '排序保存失败', ok ? 'success' : 'error')
}

async function doConfirm() {
  const fn = confirm.onConfirm
  confirm.open = false
  if (fn) {
    try {
      await fn()
    } catch (e) {
      show(e?.message || '操作失败', 'error')
    }
  }
}

async function logout() {
  await store.logout()
  router.push('/')
}
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <header class="sticky top-0 z-40 border-b border-gray-100 bg-white/85 backdrop-blur">
      <div class="mx-auto flex h-14 max-w-6xl items-center gap-3 px-4">
        <router-link to="/" class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100" title="返回前台">
          <Icon icon="ri:arrow-left-line" width="20" height="20" />
        </router-link>
        <h1 class="text-base font-semibold text-gray-800">后台数据管理</h1>
        <span
          class="hidden items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-400 sm:inline-flex"
        >
          <Icon icon="ri:database-2-line" width="12" height="12" />
          {{ API_MODE === 'http' ? 'MySQL 后端' : '本地 Mock' }}
        </span>

        <!-- 管理范围:管理员可在「通用 / 我的」间切换;普通用户只管理自己的 -->
        <div
          v-if="store.isAdmin"
          class="ml-1 flex items-center gap-0.5 rounded-lg bg-gray-100 p-0.5 text-xs"
        >
          <button
            class="rounded-md px-2 py-1 transition"
            :class="store.scope === 'global' ? 'bg-white text-primary shadow-sm' : 'text-gray-500'"
            @click="store.setScope('global')"
          >
            通用
          </button>
          <button
            class="rounded-md px-2 py-1 transition"
            :class="store.scope === 'mine' ? 'bg-white text-primary shadow-sm' : 'text-gray-500'"
            @click="store.setScope('mine')"
          >
            我的
          </button>
        </div>
        <span
          v-else
          class="ml-1 rounded-full bg-primary-50 px-2 py-0.5 text-[11px] text-primary"
        >
          我的导航
        </span>

        <div class="flex-1" />
        <span class="hidden items-center gap-1 text-xs text-gray-500 sm:flex">
          <Icon icon="ri:user-3-line" width="15" height="15" />
          {{ store.user?.username || 'admin' }}
        </span>
        <button
          class="rounded-lg p-1.5 text-gray-500 hover:bg-rose-50 hover:text-rose-500"
          title="退出登录"
          @click="logout"
        >
          <Icon icon="ri:logout-box-r-line" width="19" height="19" />
        </button>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-4 py-6">
      <!-- stats -->
      <div class="grid grid-cols-3 gap-3 sm:gap-4">
        <div class="rounded-2xl border border-gray-100 bg-white p-4 shadow-card">
          <p class="text-xs text-gray-400">分类</p>
          <p class="mt-1 text-2xl font-semibold text-gray-800">{{ store.categories.length }}</p>
        </div>
        <div class="rounded-2xl border border-gray-100 bg-white p-4 shadow-card">
          <p class="text-xs text-gray-400">链接</p>
          <p class="mt-1 text-2xl font-semibold text-gray-800">{{ store.links.length }}</p>
        </div>
        <div class="rounded-2xl border border-gray-100 bg-white p-4 shadow-card">
          <p class="text-xs text-gray-400">总点击</p>
          <p class="mt-1 text-2xl font-semibold text-gray-800">{{ store.totalClicks }}</p>
        </div>
      </div>

      <!-- toolbar -->
      <div class="mt-5 flex items-center justify-between gap-2">
        <h2 class="text-sm font-semibold text-gray-700">分类与链接管理</h2>
        <div class="flex items-center gap-2">
          <button
            v-if="store.isAdmin"
            class="btn-ghost border border-gray-200"
            @click="showSettings = true"
          >
            <Icon icon="ri:settings-3-line" width="16" height="16" />
            <span class="hidden sm:inline">站点设置</span>
          </button>
          <button
            v-if="store.isAdmin"
            class="btn-ghost border border-gray-200"
            @click="showInvites = true"
          >
            <Icon icon="ri:ticket-2-line" width="16" height="16" />
            <span class="hidden sm:inline">邀请码</span>
          </button>
          <button
            v-if="store.isAdmin"
            class="btn-ghost border border-gray-200"
            @click="showUsers = true"
          >
            <Icon icon="ri:team-line" width="16" height="16" />
            <span class="hidden sm:inline">用户管理</span>
          </button>
          <button class="btn-ghost border border-gray-200" @click="showImport = true">
            <Icon icon="ri:upload-cloud-2-line" width="16" height="16" />
            <span class="hidden sm:inline">批量导入</span>
          </button>
          <button class="btn-ghost border border-gray-200" @click="newCategory">
            <Icon icon="ri:add-line" width="16" height="16" />
            <span class="hidden sm:inline">新建分类</span>
          </button>
          <button class="btn-primary" @click="newLink">
            <Icon icon="ri:add-line" width="16" height="16" />
            新建链接
          </button>
        </div>
      </div>

      <!-- panels -->
      <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[280px_1fr]">
        <!-- categories -->
        <section class="rounded-2xl border border-gray-100 bg-white p-2 shadow-card">
          <p class="px-2 py-2 text-xs font-semibold uppercase tracking-wide text-gray-400">分类</p>
          <div v-if="cats.length === 0" class="px-3 py-8 text-center text-xs text-gray-400">
            暂无分类,点击「新建分类」创建
          </div>
          <VueDraggable
            v-else
            v-model="cats"
            :animation="180"
            handle=".cat-handle"
            ghost-class="category-drag-ghost"
            chosen-class="category-drag-chosen"
            class="space-y-0.5"
            @end="onCatsSorted"
          >
            <div
              v-for="cat in cats"
              :key="cat.id"
              class="group flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition"
              :class="
                selectedCatId === cat.id
                  ? 'bg-primary-50 text-primary'
                  : 'text-gray-600 hover:bg-gray-50'
              "
            >
              <button class="cat-handle cursor-grab text-gray-300 active:cursor-grabbing">
                <Icon icon="ri:draggable" width="15" height="15" />
              </button>
              <Icon :icon="cat.icon || 'ri:folder-line'" width="17" height="17" class="shrink-0" />
              <button class="flex-1 truncate text-left" @click="selectedCatId = cat.id">
                {{ cat.name }}
              </button>
              <span class="text-xs text-gray-400">{{ store.linkCountByCategory[cat.id] || 0 }}</span>
              <button
                class="text-gray-300 opacity-0 transition hover:text-primary group-hover:opacity-100"
                title="编辑"
                @click="editCategory(cat)"
              >
                <Icon icon="ri:pencil-line" width="15" height="15" />
              </button>
              <button
                class="text-gray-300 opacity-0 transition hover:text-rose-500 group-hover:opacity-100"
                title="删除"
                @click="askDeleteCategory(cat)"
              >
                <Icon icon="ri:delete-bin-line" width="15" height="15" />
              </button>
            </div>
          </VueDraggable>
        </section>

        <!-- links -->
        <section class="rounded-2xl border border-gray-100 bg-white p-3 shadow-card">
          <div class="mb-2 flex items-center justify-between px-1">
            <p class="text-sm font-medium text-gray-700">
              {{ selectedCat ? selectedCat.name : '链接' }}
              <span class="ml-1 text-xs font-normal text-gray-400">{{ links.length }} 条</span>
            </p>
            <button
              v-if="selectedCat"
              class="text-xs text-primary hover:underline"
              @click="newLink"
            >
              + 添加到此分类
            </button>
          </div>

          <div
            v-if="links.length === 0"
            class="flex flex-col items-center justify-center py-16 text-center"
          >
            <Icon icon="ri:links-line" width="34" height="34" class="text-gray-300" />
            <p class="mt-2 text-xs text-gray-400">该分类下暂无链接</p>
          </div>

          <VueDraggable
            v-else
            v-model="links"
            :animation="180"
            handle=".link-handle"
            ghost-class="category-drag-ghost"
            chosen-class="category-drag-chosen"
            class="space-y-1"
            @end="onLinksSorted"
          >
            <div
              v-for="link in links"
              :key="link.id"
              class="group flex items-center gap-3 rounded-lg border border-transparent px-2 py-2 transition hover:border-gray-100 hover:bg-gray-50"
            >
              <button class="link-handle cursor-grab text-gray-300 active:cursor-grabbing">
                <Icon icon="ri:draggable" width="16" height="16" />
              </button>
              <LinkFavicon :url="link.url" :title="link.title" :icon="link.icon" :size="32" />
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-medium text-gray-800">{{ link.title }}</p>
                <p class="truncate text-xs text-gray-500">{{ link.url }}</p>
              </div>
              <span class="hidden items-center gap-0.5 text-[11px] text-gray-400 sm:flex">
                <Icon icon="ri:fire-line" width="12" height="12" />
                {{ link.clicks }}
              </span>
              <button
                class="rounded-md p-1 text-gray-400 opacity-0 transition hover:bg-white hover:text-primary group-hover:opacity-100"
                title="编辑"
                @click="editLink(link)"
              >
                <Icon icon="ri:pencil-line" width="16" height="16" />
              </button>
              <button
                class="rounded-md p-1 text-gray-400 opacity-0 transition hover:bg-white hover:text-rose-500 group-hover:opacity-100"
                title="删除"
                @click="askDeleteLink(link)"
              >
                <Icon icon="ri:delete-bin-line" width="16" height="16" />
              </button>
            </div>
          </VueDraggable>
        </section>
      </div>
    </main>

    <!-- modals -->
    <CategoryFormModal v-model="showCat" :category="editingCat" />
    <LinkFormModal v-model="showLink" :link="editingLink" :default-category-id="selectedCatId" />
    <ImportModal v-model="showImport" />
    <UsersModal v-model="showUsers" />
    <InvitesModal v-model="showInvites" />
    <SettingsModal v-model="showSettings" />

    <Modal
      v-model="confirm.open"
      :title="confirm.title"
      icon="ri:error-warning-line"
      icon-class="bg-rose-50 text-rose-500"
    >
      <p class="text-sm leading-relaxed text-gray-600">{{ confirm.message }}</p>
      <template #footer>
        <button class="btn-ghost" @click="confirm.open = false">取消</button>
        <button class="btn-danger" @click="doConfirm">
          <Icon icon="ri:delete-bin-line" width="15" height="15" />
          删除
        </button>
      </template>
    </Modal>
  </div>
</template>
