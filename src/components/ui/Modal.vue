<script setup>
import { watch, onBeforeUnmount, ref, nextTick } from 'vue'
import { Icon } from '@iconify/vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  icon: { type: String, default: '' },
  iconClass: { type: String, default: 'bg-primary-50 text-primary' },
  maxWidth: { type: String, default: 'max-w-md' },
})
const emit = defineEmits(['update:modelValue'])
const panel = ref(null)

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
    document.body.style.overflow = v ? 'hidden' : ''
    if (v) {
      document.addEventListener('keydown', onKey)
      nextTick(() => panel.value?.focus())
    } else {
      document.removeEventListener('keydown', onKey)
    }
  },
)
onBeforeUnmount(() => {
  if (typeof document !== 'undefined') {
    document.body.style.overflow = ''
    document.removeEventListener('keydown', onKey)
  }
})
</script>

<template>
  <Teleport to="body">
    <transition name="modal">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-[900] flex items-start justify-center overflow-y-auto bg-gray-900/40 px-4 py-10 backdrop-blur-sm"
        @click.self="close"
      >
        <div
          ref="panel"
          class="zmodal-panel w-full overflow-hidden rounded-2xl bg-white shadow-card outline-none ring-1 ring-black/5"
          :class="maxWidth"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          tabindex="-1"
        >
          <header class="flex items-start gap-3 border-b border-gray-100 px-5 py-3.5">
            <span
              v-if="icon"
              class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg"
              :class="iconClass"
            >
              <Icon :icon="icon" width="18" height="18" />
            </span>
            <div class="min-w-0 flex-1">
              <h3 class="text-sm font-semibold text-gray-800">{{ title }}</h3>
              <p v-if="description" class="mt-0.5 truncate text-xs text-gray-400">{{ description }}</p>
            </div>
            <button
              class="-mr-1 rounded-lg p-1 text-gray-400 transition hover:bg-gray-100 hover:text-gray-600"
              aria-label="关闭"
              @click="close"
            >
              <Icon icon="ri:close-line" width="20" height="20" />
            </button>
          </header>

          <div class="max-h-[70vh] overflow-y-auto px-5 py-4">
            <slot />
          </div>

          <footer
            v-if="$slots.footer"
            class="flex justify-end gap-2 border-t border-gray-100 bg-gray-50/60 px-5 py-3"
          >
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </transition>
  </Teleport>
</template>
