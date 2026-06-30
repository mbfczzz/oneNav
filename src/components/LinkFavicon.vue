<script setup>
import { computed, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'

const props = defineProps({
  url: { type: String, default: '' },
  title: { type: String, default: '' },
  icon: { type: String, default: '' }, // optional explicit iconify name
  size: { type: Number, default: 38 },
})

const failed = ref(false)

const domain = computed(() => {
  try {
    return new URL(props.url).hostname
  } catch {
    return ''
  }
})

const faviconUrl = computed(() =>
  domain.value ? `https://www.google.com/s2/favicons?domain=${domain.value}&sz=64` : '',
)

// Reset the error flag when the source changes (the card is reused across edits).
watch(faviconUrl, () => {
  failed.value = false
})

const initial = computed(() => (props.title || '?').trim().charAt(0).toUpperCase() || '?')

// Deterministic pastel color from the title.
const hue = computed(() => {
  let h = 0
  const s = props.title || ''
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  return h % 360
})
const bg = computed(() => `hsl(${hue.value} 70% 94%)`)
const fg = computed(() => `hsl(${hue.value} 55% 45%)`)
</script>

<template>
  <span
    class="inline-flex shrink-0 items-center justify-center overflow-hidden rounded-lg"
    :style="{ width: size + 'px', height: size + 'px', background: icon ? '#eff6ff' : bg }"
  >
    <Icon v-if="icon" :icon="icon" :width="size * 0.58" :height="size * 0.58" class="text-primary" />
    <img
      v-else-if="faviconUrl && !failed"
      :src="faviconUrl"
      :alt="title"
      :width="size * 0.62"
      :height="size * 0.62"
      loading="lazy"
      referrerpolicy="no-referrer"
      @error="failed = true"
    />
    <span v-else class="text-sm font-semibold" :style="{ color: fg }">{{ initial }}</span>
  </span>
</template>
