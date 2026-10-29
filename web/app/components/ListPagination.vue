<script setup lang="ts">
import type { Meta } from '~/types/api'

defineProps<{ meta: Meta | null }>()
const page = defineModel<number>('page', { required: true })
</script>

<template>
  <div v-if="meta && meta.total > 0" class="flex flex-wrap items-center justify-between gap-3 border-t border-default pt-4">
    <p class="text-sm text-muted">
      {{ (meta.current_page - 1) * meta.per_page + 1 }}–{{ Math.min(meta.current_page * meta.per_page, meta.total) }}
      dari {{ meta.total }}
    </p>
    <UPagination
      v-if="meta.total_pages > 1"
      v-model:page="page"
      :total="meta.total"
      :items-per-page="meta.per_page"
      size="sm"
    />
  </div>
</template>
