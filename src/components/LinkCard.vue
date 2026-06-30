<script setup>
import { ref, computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { useNavStore } from '@/stores/nav'
import { useToast } from '@/composables/toast'
import { isHttpUrl } from '@/utils/url'
import LinkFavicon from './LinkFavicon.vue'
import QrDialog from './QrDialog.vue'

const props = defineProps({
  link: { type: Object, required: true },
  sortable: { type: Boolean, default: false },
})

const { t } = useI18n()
const store = useNavStore()
const { show } = useToast()

const showQr = ref(false)

const domain = computed(() => {
  try {
    return new URL(props.link.url).hostname.replace(/^www\./, '')
  } catch {
    return props.link.url
  }
})

function visit() {
  if (!isHttpUrl(props.link.url)) {
    show('链接地址无效(仅支持 http/https)', 'error')
    return
  }
  store.incrementClick(props.link.id)
  window.open(props.link.url, '_blank', 'noopener,noreferrer')
}

async function copy() {
  try {
    await navigator.clipboard.writeText(props.link.url)
    show(t('common.copied'), 'success')
  } catch {
    show(t('common.copyFailed'), 'error')
  }
}
</script>

<template>
  <div
    class="group relative flex cursor-pointer items-start gap-3 rounded-xl border border-gray-100 bg-white p-3.5 transition-all duration-200 hover:-translate-y-0.5 hover:border-transparent hover:shadow-card"
    @click="visit"
  >
    <!-- drag handle (only when reorderable) -->
    <button
      v-if="sortable"
      class="drag-handle absolute right-2 top-2 cursor-grab text-gray-300 opacity-0 transition group-hover:opacity-100 active:cursor-grabbing"
      title="拖拽排序"
      @click.stop
    >
      <Icon icon="ri:draggable" width="16" height="16" />
    </button>

    <LinkFavicon :url="link.url" :title="link.title" :icon="link.icon" :size="40" />

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-1.5">
        <h3 class="truncate text-sm font-semibold text-gray-800 group-hover:text-primary">
          {{ link.title }}
        </h3>
        <Icon
          icon="ri:external-link-line"
          width="13"
          height="13"
          class="shrink-0 text-gray-300 opacity-0 transition group-hover:opacity-100"
        />
      </div>
      <p class="mt-0.5 line-clamp-2 text-xs leading-relaxed text-gray-500">
        {{ link.description || t('card.noDescription') }}
      </p>
      <div class="mt-1.5 flex items-center gap-2 text-[11px] text-gray-500">
        <span class="inline-flex items-center gap-0.5">
          <Icon icon="ri:fire-line" width="12" height="12" />
          {{ link.clicks }} {{ t('card.clicks') }}
        </span>
        <span class="truncate">{{ domain }}</span>
      </div>
    </div>

    <!-- hover action bar -->
    <div
      class="absolute bottom-2 right-2 flex items-center gap-1 opacity-0 transition group-hover:opacity-100"
    >
      <button
        class="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-primary"
        :title="t('card.copy')"
        @click.stop="copy"
      >
        <Icon icon="ri:file-copy-line" width="15" height="15" />
      </button>
      <button
        class="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-primary"
        :title="t('card.qrcode')"
        @click.stop="showQr = true"
      >
        <Icon icon="ri:qr-code-line" width="15" height="15" />
      </button>
    </div>

    <QrDialog v-model="showQr" :url="link.url" :title="link.title" />
  </div>
</template>
