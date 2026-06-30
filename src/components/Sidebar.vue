<script setup>
import { ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'

const emit = defineEmits(['navigate'])

const store = useNavStore()
const { t } = useI18n()
const { show } = useToast()

const cats = ref([])
watch(
  () => store.sortedCategories,
  (v) => {
    cats.value = [...v]
  },
  { immediate: true },
)

function pick(id) {
  store.setCategory(id)
  emit('navigate')
}

async function onSorted() {
  const ok = await store.reorderCategories(cats.value.map((c) => c.id))
  show(ok ? t('sort.saved') : t('sort.failed'), ok ? 'success' : 'error')
}
</script>

<template>
  <nav class="flex h-full flex-col">
    <div class="px-4 pb-2 pt-4">
      <p class="text-xs font-semibold uppercase tracking-wide text-gray-400">
        {{ t('sidebar.title') }}
      </p>
    </div>

    <div class="flex-1 overflow-y-auto px-2 pb-4">
      <!-- All -->
      <button
        class="mb-1 flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition"
        :class="
          store.activeCategoryId === 'all'
            ? 'bg-primary-50 font-medium text-primary'
            : 'text-gray-600 hover:bg-gray-50'
        "
        @click="pick('all')"
      >
        <Icon icon="ri:function-line" width="18" height="18" />
        <span class="flex-1 text-left">{{ t('sidebar.all') }}</span>
        <span class="text-xs text-gray-400">{{ store.links.length }}</span>
      </button>

      <!-- empty -->
      <div v-if="cats.length === 0" class="px-3 py-6 text-center">
        <Icon icon="ri:folder-add-line" width="28" height="28" class="mx-auto text-gray-300" />
        <p class="mt-2 text-xs text-gray-400">{{ t('sidebar.empty') }}</p>
        <p class="text-[11px] text-gray-300">{{ t('sidebar.emptyHint1') }}</p>
        <p class="text-[11px] text-gray-300">{{ t('sidebar.emptyHint2') }}</p>
      </div>

      <!-- categories (reorderable) -->
      <VueDraggable
        v-else
        v-model="cats"
        :animation="180"
        handle=".cat-handle"
        ghost-class="category-drag-ghost"
        chosen-class="category-drag-chosen"
        class="space-y-1"
        @end="onSorted"
      >
        <div
          v-for="cat in cats"
          :key="cat.id"
          class="group flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition"
          :class="
            store.activeCategoryId === cat.id
              ? 'bg-primary-50 font-medium text-primary'
              : 'text-gray-600 hover:bg-gray-50'
          "
        >
          <Icon
            :icon="cat.icon || 'ri:folder-line'"
            width="18"
            height="18"
            class="shrink-0 cursor-pointer"
            @click="pick(cat.id)"
          />
          <span class="flex-1 cursor-pointer truncate text-left" @click="pick(cat.id)">
            {{ cat.name }}
          </span>
          <span class="text-xs text-gray-400">{{ store.linkCountByCategory[cat.id] || 0 }}</span>
          <button
            class="cat-handle cursor-grab text-gray-300 opacity-0 transition group-hover:opacity-100 active:cursor-grabbing"
            title="拖拽排序"
          >
            <Icon icon="ri:draggable" width="15" height="15" />
          </button>
        </div>
      </VueDraggable>
    </div>
  </nav>
</template>
