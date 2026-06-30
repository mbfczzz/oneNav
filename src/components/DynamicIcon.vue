<script setup>
// Thin wrapper over Iconify (mirrors the original DynamicIcon). Defaults to Remix Icon set.
import { Icon } from '@iconify/vue'

defineProps({
  name: { type: String, default: 'ri:link' },
  size: { type: [Number, String], default: 20 },
})
</script>

<template>
  <Icon :icon="name" :width="size" :height="size" />
</template>
