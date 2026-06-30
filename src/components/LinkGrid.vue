<script setup>
import { ref, watch, computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'
import LinkCard from './LinkCard.vue'

const store = useNavStore()
const { t } = useI18n()
const { show } = useToast()

// Local mirror of the store's visible links so vue-draggable-plus can reorder in place.
const list = ref([])
watch(
  () => store.visibleLinks,
  (v) => {
    list.value = [...v]
  },
  { immediate: true },
)

const canSort = computed(() => store.canSortLinks)
const gridClass =
  'grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5'

async function onSorted() {
  const ok = await store.reorderLinks(list.value.map((l) => l.id))
  show(ok ? t('sort.saved') : t('sort.failed'), ok ? 'success' : 'error')
}
</script>

<template>
  <!-- empty state -->
  <div
    v-if="list.length === 0"
    class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-white/60 py-20 text-center"
  >
    <Icon
      :icon="store.isSearching ? 'ri:search-eye-line' : 'ri:inbox-line'"
      width="40"
      height="40"
      class="text-gray-300"
    />
    <p class="mt-3 text-sm text-gray-400">
      {{ store.isSearching ? t('search.empty') : t('sidebar.placeholder') }}
    </p>
  </div>

  <!-- sortable grid -->
  <VueDraggable
    v-else-if="canSort"
    v-model="list"
    :animation="180"
    handle=".drag-handle"
    ghost-class="category-drag-ghost"
    chosen-class="category-drag-chosen"
    :class="gridClass"
    @end="onSorted"
  >
    <LinkCard v-for="link in list" :key="link.id" :link="link" sortable />
  </VueDraggable>

  <!-- static grid -->
  <div v-else :class="gridClass">
    <LinkCard v-for="link in list" :key="link.id" :link="link" />
  </div>
</template>
