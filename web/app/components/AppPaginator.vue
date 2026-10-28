<script setup lang="ts">
import type { Meta } from '~/types/api'

defineProps<{ meta: Meta }>()
const emit = defineEmits<{ change: [page: number] }>()
</script>

<template>
  <div class="mt-4 flex items-center justify-between text-sm text-muted">
    <span>
      Total {{ meta.total }} · Halaman {{ meta.current_page }}/{{ meta.total_pages || 1 }}
    </span>
    <div class="flex gap-2">
      <UButton
        size="xs"
        color="neutral"
        variant="outline"
        :disabled="meta.current_page <= 1"
        @click="emit('change', meta.current_page - 1)"
      >
        Prev
      </UButton>
      <UButton
        size="xs"
        color="neutral"
        variant="outline"
        :disabled="meta.current_page >= (meta.total_pages || 1)"
        @click="emit('change', meta.current_page + 1)"
      >
        Next
      </UButton>
    </div>
  </div>
</template>
