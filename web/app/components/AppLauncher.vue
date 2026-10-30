<script setup lang="ts">
// Launcher: kartu solution yang bisa dibuka untuk company aktif.
const { apps, loading, launchHref } = useApps()
const others = computed(() => apps.value.filter(a => !a.is_current))
</script>

<template>
  <div v-if="loading || others.length">
    <h2 class="mb-3 text-sm font-semibold text-highlighted">Aplikasi</h2>
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <template v-if="loading && !others.length">
        <USkeleton v-for="i in 3" :key="i" class="h-20 w-full" />
      </template>
      <a
        v-for="app in others"
        :key="app.id"
        :href="launchHref(app)"
        class="flex items-center gap-3 rounded-lg border border-default p-4 transition hover:border-primary hover:bg-primary/5"
      >
        <AppIcon :app="app" size="xl" />
        <span class="min-w-0 flex-1">
          <span class="block truncate font-medium text-highlighted">{{ app.name }}</span>
          <span class="line-clamp-2 text-sm text-muted">{{ app.description ?? app.code }}</span>
        </span>
        <UIcon name="i-lucide-arrow-up-right" class="size-4 shrink-0 text-dimmed" />
      </a>
    </div>
  </div>
</template>
