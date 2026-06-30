<script setup>
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useNavStore } from '@/stores/nav'
import ToastHost from '@/components/ToastHost.vue'

// Validate/refresh any persisted session on startup.
onMounted(() => useNavStore().initAuth())
</script>

<template>
  <RouterView v-slot="{ Component }">
    <transition name="fade" mode="out-in">
      <component :is="Component" />
    </transition>
  </RouterView>
  <ToastHost />
</template>
