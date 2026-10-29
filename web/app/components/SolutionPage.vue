<script setup lang="ts">
import type { BreadcrumbItem } from '@nuxt/ui'

// Kerangka halaman di dalam solution (admin): tab navigasi + breadcrumb.
const props = defineProps<{ title: string; trail?: BreadcrumbItem[]; narrow?: boolean }>()

const ctx = useSolutionContext()
const base = computed(() => `/admin/solutions/${ctx.solutionId.value}`)
const navItems = computed(() => [
  { label: 'Ringkasan', icon: 'i-lucide-info', to: base.value, exact: true },
  { label: 'Permission', icon: 'i-lucide-key-round', to: `${base.value}/permissions` },
  { label: 'Role global', icon: 'i-lucide-shield', to: `${base.value}/roles` },
  { label: 'Client', icon: 'i-lucide-plug', to: `${base.value}/clients` },
])
const breadcrumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Solutions', icon: 'i-lucide-boxes', to: '/admin/solutions' },
  { label: ctx.solution.value?.name ?? '…', to: base.value },
  ...(props.trail ?? []),
])

watch(ctx.solutionId, () => ctx.load(), { immediate: true })
</script>

<template>
  <AppPage :title="title" :breadcrumbs="breadcrumbs" :narrow="narrow">
    <template #actions>
      <slot name="actions" />
    </template>
    <template #toolbar>
      <UNavigationMenu :items="navItems" highlight class="-mx-1 flex-1" />
    </template>
    <UAlert v-if="ctx.error.value" color="error" variant="subtle" :title="ctx.error.value" />
    <slot v-else />
  </AppPage>
</template>
