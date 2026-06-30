<script setup>
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-4 text-center">
    <p class="text-7xl font-bold text-primary/20">404</p>
    <h1 class="mt-2 text-lg font-semibold text-gray-800">{{ t('notFound.title') }}</h1>
    <p class="mt-1 text-sm text-gray-400">{{ t('notFound.desc') }}</p>
    <router-link
      to="/"
      class="mt-5 inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-600"
    >
      <Icon icon="ri:home-4-line" width="16" height="16" />
      {{ t('notFound.back') }}
    </router-link>
  </div>
</template>
