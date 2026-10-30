<script setup lang="ts">
// Menu apps (grid) di navbar: pintasan ke solution lain untuk company aktif.
const { apps, launchHref } = useApps()
</script>

<template>
  <UPopover v-if="apps.length" :content="{ align: 'end' }">
    <UButton icon="i-lucide-grid-3x3" color="neutral" variant="ghost" aria-label="Aplikasi" />

    <template #content>
      <div class="w-80 p-3">
        <p class="mb-2 px-1 text-xs font-medium text-muted">Aplikasi</p>
        <div class="grid grid-cols-3 gap-1">
          <component
            :is="app.is_current ? 'div' : 'a'"
            v-for="app in apps"
            :key="app.id"
            :href="app.is_current ? undefined : launchHref(app)"
            class="flex flex-col items-center gap-1.5 rounded-lg p-2 text-center transition"
            :class="app.is_current ? 'bg-primary/10' : 'hover:bg-elevated'"
            :title="app.description ?? app.name"
          >
            <AppIcon :app="app" />
            <span class="line-clamp-2 text-xs text-highlighted">{{ app.name }}</span>
          </component>
        </div>
      </div>
    </template>
  </UPopover>
</template>
