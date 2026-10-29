<script setup lang="ts">
// Kartu bagian halaman: judul + deskripsi + aksi di kanan, lalu konten.
defineProps<{ title: string; description?: string }>()
</script>

<template>
  <UCard variant="outline" :ui="{ header: 'p-4 sm:px-5', body: 'p-4 sm:p-5 space-y-4' }">
    <template #header>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="font-semibold text-highlighted">{{ title }}</h2>
          <p v-if="description" class="mt-0.5 text-sm text-muted">{{ description }}</p>
        </div>
        <div v-if="$slots.actions" class="flex shrink-0 items-center gap-2">
          <slot name="actions" />
        </div>
      </div>
    </template>
    <slot />
  </UCard>
</template>
