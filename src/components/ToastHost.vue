<script setup>
import { Icon } from '@iconify/vue'
import { useToast } from '@/composables/toast'

const { state } = useToast()

const iconFor = (type) =>
  type === 'success'
    ? 'ri:checkbox-circle-fill'
    : type === 'error'
      ? 'ri:error-warning-fill'
      : 'ri:information-fill'

const colorFor = (type) =>
  type === 'success'
    ? 'text-emerald-500'
    : type === 'error'
      ? 'text-rose-500'
      : 'text-primary'
</script>

<template>
  <div
    role="status"
    aria-live="polite"
    class="pointer-events-none fixed inset-x-0 top-4 z-[1000] flex flex-col items-center gap-2"
  >
    <transition-group name="search-panel-fade">
      <div
        v-for="item in state.items"
        :key="item.id"
        class="pointer-events-auto flex items-center gap-2 rounded-xl bg-white px-4 py-2.5 text-sm text-gray-700 shadow-card ring-1 ring-black/5"
      >
        <Icon :icon="iconFor(item.type)" :class="colorFor(item.type)" width="18" height="18" />
        <span>{{ item.message }}</span>
      </div>
    </transition-group>
  </div>
</template>
