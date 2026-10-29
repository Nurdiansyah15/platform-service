<script setup lang="ts">
const props = defineProps<{ status?: string | null; size?: 'xs' | 'sm' | 'md' }>()
const meta = computed(() => statusMeta(props.status))
</script>

<template>
  <UBadge :color="meta.color" variant="subtle" :size="size ?? 'sm'">{{ meta.label }}</UBadge>
</template>
