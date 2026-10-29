<script setup lang="ts">
import type { BreadcrumbItem } from '@nuxt/ui'

defineProps<{
  title: string
  description?: string
  breadcrumbs?: BreadcrumbItem[]
  /** Lebar konten dibatasi (form / detail) agar nyaman dibaca. */
  narrow?: boolean
}>()
</script>

<template>
  <UDashboardPanel>
    <template #header>
      <UDashboardNavbar :title="title">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <slot name="actions" />
        </template>
      </UDashboardNavbar>
      <UDashboardToolbar v-if="$slots.toolbar">
        <slot name="toolbar" />
      </UDashboardToolbar>
    </template>

    <template #body>
      <div :class="narrow ? 'mx-auto w-full max-w-3xl' : 'w-full'" class="space-y-6">
        <div v-if="breadcrumbs?.length || description" class="space-y-1">
          <UBreadcrumb v-if="breadcrumbs?.length" :items="breadcrumbs" />
          <p v-if="description" class="text-sm text-muted">{{ description }}</p>
        </div>
        <slot />
      </div>
    </template>
  </UDashboardPanel>
</template>
