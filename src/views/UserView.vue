<script setup>
import { ref, computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const store = useNavStore()
const { show } = useToast()

const mode = ref('login') // 'login' | 'register'
const username = ref('')
const password = ref('')
const inviteCode = ref('')
const loading = ref(false)
const error = ref('')
const isRegister = computed(() => mode.value === 'register')

function toggleMode() {
  mode.value = isRegister.value ? 'login' : 'register'
  error.value = ''
}

async function submit() {
  if (loading.value) return
  error.value = ''
  loading.value = true
  try {
    if (isRegister.value) {
      await store.register(username.value.trim(), password.value, inviteCode.value.trim())
      show(`${t('user.register')} ✓`, 'success')
    } else {
      await store.login(username.value.trim(), password.value)
      show(`${t('user.login')} ✓`, 'success')
    }
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    router.replace(redirect)
  } catch (e) {
    error.value = e?.message || '操作失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm rounded-2xl border border-gray-100 bg-white p-7 shadow-card">
      <div class="mb-6 flex flex-col items-center">
        <span class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-white">
          <Icon icon="ri:bookmark-fill" width="26" height="26" />
        </span>
        <h1 class="mt-3 text-lg font-semibold text-gray-800">
          {{ isRegister ? t('user.registerTitle') : t('user.title') }}
        </h1>
      </div>

      <form class="space-y-3" @submit.prevent="submit">
        <label class="block">
          <span class="mb-1 block text-xs text-gray-500">{{ t('user.username') }}</span>
          <input
            v-model="username"
            type="text"
            autocomplete="username"
            class="input"
          />
        </label>
        <label class="block">
          <span class="mb-1 block text-xs text-gray-500">{{ t('user.password') }}</span>
          <input
            v-model="password"
            type="password"
            :autocomplete="isRegister ? 'new-password' : 'current-password'"
            class="input"
            @keyup.enter="submit"
          />
        </label>

        <label v-if="isRegister" class="block">
          <span class="mb-1 block text-xs text-gray-500">
            {{ t('user.inviteCode') }} <span class="text-rose-400">*</span>
          </span>
          <div class="relative">
            <Icon
              icon="ri:ticket-2-line"
              width="16"
              height="16"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="inviteCode"
              type="text"
              class="input pl-9 uppercase tracking-wider placeholder:normal-case placeholder:tracking-normal"
              :placeholder="t('user.invitePlaceholder')"
              @keyup.enter="submit"
            />
          </div>
        </label>

        <p v-if="error" class="flex items-center gap-1 text-xs text-rose-500">
          <Icon icon="ri:error-warning-line" width="14" height="14" />
          {{ error }}
        </p>

        <button type="submit" :disabled="loading" class="btn-primary w-full">
          <Icon v-if="loading" icon="ri:loader-4-line" width="16" height="16" class="animate-spin" />
          {{ isRegister ? t('user.register') : t('user.login') }}
        </button>
      </form>

      <button class="mt-3 w-full text-center text-xs text-primary hover:underline" @click="toggleMode">
        {{ isRegister ? t('user.toLogin') : t('user.toRegister') }}
      </button>

      <router-link
        to="/"
        class="mt-4 flex items-center justify-center gap-1 text-xs text-gray-400 hover:text-primary"
      >
        <Icon icon="ri:arrow-left-line" width="14" height="14" />
        {{ t('user.backHome') }}
      </router-link>
    </div>
  </div>
</template>
