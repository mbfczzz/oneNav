<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { useNavStore } from '@/stores/nav'
import TopBar from '@/components/TopBar.vue'
import Sidebar from '@/components/Sidebar.vue'
import LinkGrid from '@/components/LinkGrid.vue'

const store = useNavStore()
const { t } = useI18n()
const drawerOpen = ref(false)
const showTop = ref(false)

function onScroll() {
  showTop.value = window.scrollY > 320
}
function backToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

onMounted(async () => {
  // Let the auth flow own the first load (applyDefaultScope) to avoid a
  // redundant global fetch + scope race; only load directly when anonymous.
  if (store.token && !store.loaded) await store.initAuth()
  if (!store.loaded) store.loadAll()
  window.addEventListener('scroll', onScroll, { passive: true })
})
onBeforeUnmount(() => window.removeEventListener('scroll', onScroll))

const currentTitle = computed(() => {
  if (store.isSearching) return t('search.title')
  if (store.activeCategoryId === 'all') return t('sidebar.all')
  const c = store.categories.find((c) => c.id === store.activeCategoryId)
  return c ? c.name : t('sidebar.all')
})
const currentCount = computed(() => store.visibleLinks.length)
</script>

<template>
  <div class="min-h-screen">
    <TopBar @toggle-sidebar="drawerOpen = true" />

    <div class="mx-auto flex max-w-[1600px]">
      <!-- desktop sidebar -->
      <aside
        class="sticky top-14 hidden h-[calc(100vh-3.5rem)] w-60 shrink-0 border-r border-gray-100 bg-white lg:block"
      >
        <Sidebar />
      </aside>

      <!-- main content -->
      <main class="min-w-0 flex-1 px-4 py-5 sm:px-6">
        <div class="mb-4 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <h2 class="text-lg font-semibold text-gray-800">{{ currentTitle }}</h2>
            <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
              {{ currentCount }}
            </span>
          </div>
          <p v-if="store.isSearching" class="text-xs text-gray-400">
            {{ t('search.count', { n: currentCount }) }}
          </p>
        </div>

        <!-- load error -->
        <div
          v-if="store.loadError"
          class="mb-4 flex items-center justify-between rounded-xl border border-rose-100 bg-rose-50 px-4 py-3 text-sm text-rose-600"
        >
          <span class="flex items-center gap-2">
            <Icon icon="ri:error-warning-line" width="16" height="16" />
            {{ store.loadError }}
          </span>
          <button
            class="rounded-md px-2 py-1 text-xs font-medium hover:bg-rose-100"
            @click="store.loadAll()"
          >
            重试
          </button>
        </div>

        <!-- skeleton while first load -->
        <div
          v-if="store.loading && !store.loaded"
          class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5"
        >
          <div v-for="i in 10" :key="i" class="h-24 animate-pulse rounded-xl bg-gray-100" />
        </div>
        <template v-else>
          <!-- "我的" scope but nothing created yet -->
          <div
            v-if="store.scope === 'mine' && store.categories.length === 0 && !store.loadError"
            class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-white/60 py-20 text-center"
          >
            <Icon icon="ri:folder-add-line" width="40" height="40" class="text-gray-300" />
            <p class="mt-3 text-sm font-medium text-gray-600">{{ t('mine.emptyTitle') }}</p>
            <p class="mt-1 max-w-xs text-xs text-gray-400">{{ t('mine.emptyHint') }}</p>
            <router-link to="/dashboard" class="btn-primary mt-4">
              <Icon icon="ri:add-line" width="16" height="16" />
              {{ t('mine.add') }}
            </router-link>
          </div>
          <LinkGrid v-else />
        </template>

        <footer class="mt-10 border-t border-gray-100 pt-5 text-center text-xs text-gray-500">
          <p>{{ t('app.name') }} · 面向自由职业者与独立开发者的网址导航</p>
          <p class="mt-1">© 2026 ZMark · Powered by Vue 3 + Vite</p>
        </footer>
      </main>
    </div>

    <!-- mobile drawer -->
    <transition name="mobile-sidebar-fade">
      <div
        v-if="drawerOpen"
        class="fixed inset-0 z-50 bg-black/40 lg:hidden"
        @click="drawerOpen = false"
      />
    </transition>
    <transition name="mobile-sidebar-slide">
      <aside
        v-if="drawerOpen"
        class="fixed inset-y-0 left-0 z-50 w-64 bg-white shadow-card lg:hidden"
      >
        <Sidebar @navigate="drawerOpen = false" />
      </aside>
    </transition>

    <!-- back to top -->
    <transition name="fade">
      <button
        v-if="showTop"
        class="fixed bottom-6 right-6 z-40 flex h-11 w-11 items-center justify-center rounded-full bg-white text-gray-500 shadow-card ring-1 ring-black/5 hover:text-primary"
        :title="t('common.backToTop')"
        @click="backToTop"
      >
        <Icon icon="ri:arrow-up-line" width="20" height="20" />
      </button>
    </transition>
  </div>
</template>
