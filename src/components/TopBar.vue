<script setup>
import { computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'
import { setLocale } from '@/i18n'

const emit = defineEmits(['toggle-sidebar'])
const { t, locale } = useI18n()
const store = useNavStore()
const router = useRouter()
const route = useRoute()
const { show } = useToast()

const keyword = computed({
  get: () => store.keyword,
  set: (v) => store.setKeyword(v),
})

function toggleLang() {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}
function refresh() {
  store.refresh()
  show(t('topbar.refresh'), 'success')
}
function ai() {
  show(t('topbar.aiAssistant'), 'info')
}
async function logout() {
  await store.logout()
  show('已退出登录', 'success')
  if (route.name === 'dashboard') router.push('/')
}
</script>

<template>
  <header class="sticky top-0 z-40 border-b border-gray-100 bg-white/85 backdrop-blur">
    <div class="flex h-14 items-center gap-3 px-3 sm:px-5">
      <button
        class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 lg:hidden"
        aria-label="打开菜单"
        @click="emit('toggle-sidebar')"
      >
        <Icon icon="ri:menu-line" width="22" height="22" />
      </button>

      <router-link to="/" class="flex items-center gap-2">
        <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-white">
          <Icon icon="ri:bookmark-fill" width="18" height="18" />
        </span>
        <span class="hidden text-base font-semibold text-gray-800 sm:block">{{ t('app.name') }}</span>
      </router-link>

      <div class="relative ml-2 max-w-md flex-1">
        <Icon
          icon="ri:search-line"
          width="17"
          height="17"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
        />
        <input
          v-model="keyword"
          type="text"
          :placeholder="t('topbar.search')"
          class="w-full rounded-full border border-gray-200 bg-gray-50 py-1.5 pl-9 pr-8 text-sm outline-none transition focus:border-primary focus:bg-white focus:ring-2 focus:ring-primary/15"
        />
        <button
          v-if="keyword"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
          aria-label="清空搜索"
          @click="keyword = ''"
        >
          <Icon icon="ri:close-circle-fill" width="16" height="16" />
        </button>
      </div>

      <div class="flex items-center gap-1">
        <button
          class="hidden items-center gap-1 rounded-lg px-2.5 py-1.5 text-sm text-gray-600 hover:bg-gray-100 sm:flex"
          :aria-label="t('topbar.aiAssistant')"
          @click="ai"
        >
          <Icon icon="ri:sparkling-2-line" width="17" height="17" class="text-primary" />
          <span class="hidden md:block">{{ t('topbar.aiAssistant') }}</span>
        </button>
        <button
          class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100"
          :title="t('topbar.refresh')"
          @click="refresh"
        >
          <Icon icon="ri:refresh-line" width="19" height="19" />
        </button>
        <!-- scope switch (通用 / 我的) — only when logged in -->
        <div
          v-if="store.isAuthenticated"
          class="hidden items-center gap-0.5 rounded-lg bg-gray-100 p-0.5 text-xs sm:flex"
        >
          <button
            class="rounded-md px-2 py-1 transition"
            :class="store.scope === 'global' ? 'bg-white text-primary shadow-sm' : 'text-gray-500'"
            @click="store.setScope('global')"
          >
            {{ t('topbar.scopeGlobal') }}
          </button>
          <button
            class="rounded-md px-2 py-1 transition"
            :class="store.scope === 'mine' ? 'bg-white text-primary shadow-sm' : 'text-gray-500'"
            @click="store.setScope('mine')"
          >
            {{ t('topbar.scopeMine') }}
          </button>
        </div>

        <button
          class="rounded-lg px-2 py-1.5 text-xs font-medium text-gray-500 hover:bg-gray-100"
          :aria-label="t('topbar.language')"
          @click="toggleLang"
        >
          {{ locale === 'zh' ? 'EN' : '中' }}
        </button>

        <router-link
          to="/dashboard"
          class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100"
          :title="t('topbar.toDashboard')"
        >
          <Icon icon="ri:dashboard-3-line" width="19" height="19" />
        </router-link>

        <!-- auth state -->
        <template v-if="store.isAuthenticated">
          <span class="hidden items-center gap-1 rounded-lg px-2 py-1.5 text-xs text-gray-500 sm:flex">
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
        </template>
        <router-link
          v-else
          to="/user"
          class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100"
          title="登录"
        >
          <Icon icon="ri:login-box-line" width="19" height="19" />
        </router-link>
      </div>
    </div>
  </header>
</template>
