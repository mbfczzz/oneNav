<script setup>
import { computed, watch, onBeforeUnmount } from 'vue'
import { Icon } from '@iconify/vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/toast'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  url: { type: String, default: '' },
  title: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()
const { show } = useToast()

const qrSrc = computed(
  () => `https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${encodeURIComponent(props.url)}`,
)

function close() {
  emit('update:modelValue', false)
}
function onKey(e) {
  if (e.key === 'Escape') close()
}
watch(
  () => props.modelValue,
  (v) => {
    if (typeof document === 'undefined') return
    if (v) document.addEventListener('keydown', onKey)
    else document.removeEventListener('keydown', onKey)
  },
)
onBeforeUnmount(() => {
  if (typeof document !== 'undefined') document.removeEventListener('keydown', onKey)
})

async function copy() {
  try {
    await navigator.clipboard.writeText(props.url)
    show(t('common.copied'), 'success')
  } catch {
    show(t('common.copyFailed'), 'error')
  }
}
</script>

<template>
  <Teleport to="body">
    <transition name="modal">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-[900] flex items-center justify-center bg-gray-900/40 px-4 backdrop-blur-sm"
        @click.self="close"
      >
        <div
          class="zmodal-panel w-full max-w-xs overflow-hidden rounded-2xl bg-white shadow-card ring-1 ring-black/5"
          role="dialog"
          aria-modal="true"
          :aria-label="t('card.qrTitle')"
        >
          <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
            <h3 class="flex items-center gap-1.5 text-sm font-semibold text-gray-800">
              <Icon icon="ri:qr-code-line" width="16" height="16" class="text-primary" />
              {{ t('card.qrTitle') }}
            </h3>
            <button
              class="-mr-1 rounded-lg p-1 text-gray-400 transition hover:bg-gray-100 hover:text-gray-600"
              :aria-label="t('common.close')"
              @click="close"
            >
              <Icon icon="ri:close-line" width="20" height="20" />
            </button>
          </div>
          <div class="flex flex-col items-center gap-3 p-5">
            <div class="rounded-xl border border-gray-100 bg-white p-2.5 shadow-sm">
              <img :src="qrSrc" :alt="title" width="200" height="200" class="rounded-md" />
            </div>
            <p class="line-clamp-1 max-w-full text-center text-xs text-gray-500" :title="url">
              {{ title }}
            </p>
            <button class="btn-ghost w-full border border-gray-200" @click="copy">
              <Icon icon="ri:file-copy-line" width="15" height="15" />
              {{ t('card.copy') }}
            </button>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>
