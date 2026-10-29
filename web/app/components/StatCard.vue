<script setup lang="ts">
defineProps<{ label: string; value?: number | null; icon: string; to?: string; hint?: string }>()
</script>

<template>
  <UPageCard :to="to" variant="outline" :ui="{ container: 'gap-y-1.5' }">
    <div class="flex items-center justify-between">
      <span class="text-sm text-muted">{{ label }}</span>
      <UIcon :name="icon" class="size-5 text-dimmed" />
    </div>
    <USkeleton v-if="value === undefined || value === null" class="h-8 w-16" />
    <span v-else class="text-2xl font-semibold text-highlighted">{{ value }}</span>
    <span v-if="hint" class="text-xs text-muted">{{ hint }}</span>
  </UPageCard>
</template>
