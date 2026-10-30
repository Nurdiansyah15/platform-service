<script setup lang="ts">
import type { AppResponse } from '~/types/api'

/** Ikon solution: icon_url bila ada, selain itu inisial nama. */
defineProps<{ app: AppResponse; size?: 'md' | 'lg' | 'xl' }>()
</script>

<template>
  <UAvatar :src="app.icon_url ?? undefined" :text="initials(app.name)" :alt="app.name" :size="size ?? 'lg'" class="rounded-lg" />
</template>
