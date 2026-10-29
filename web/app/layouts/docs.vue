<script setup lang="ts">
const auth = useAuthStore()

const { data: navigation } = await useAsyncData('docs-navigation', () => queryCollectionNavigation('docs'))
const links = computed(() => navigation.value?.find(item => item.path === '/docs')?.children ?? navigation.value ?? [])
const open = ref(false)
</script>

<template>
  <div class="min-h-dvh bg-default">
    <header class="sticky top-0 z-40 border-b border-default bg-default/90 backdrop-blur">
      <div class="mx-auto flex h-14 max-w-7xl items-center justify-between gap-3 px-4 sm:px-6">
        <div class="flex items-center gap-2">
          <UButton
            class="lg:hidden"
            color="neutral"
            variant="ghost"
            icon="i-lucide-menu"
            aria-label="Buka navigasi"
            @click="open = !open"
          />
          <NuxtLink to="/docs" class="flex items-center gap-2">
            <div class="flex size-7 items-center justify-center rounded-md bg-primary text-inverted">
              <UIcon name="i-lucide-hexagon" class="size-4" />
            </div>
            <span class="font-semibold text-highlighted">Platform Docs</span>
          </NuxtLink>
        </div>
        <div class="flex items-center gap-2">
          <UButton to="/docs/api-reference" color="neutral" variant="ghost" icon="i-lucide-braces" class="hidden sm:inline-flex">
            API Reference
          </UButton>
          <UButton :to="auth.isAuthenticated ? '/' : '/login'" color="neutral" variant="outline" size="sm">
            {{ auth.isAuthenticated ? 'Buka console' : 'Masuk' }}
          </UButton>
        </div>
      </div>
    </header>

    <div class="mx-auto flex max-w-7xl gap-8 px-4 sm:px-6">
      <aside
        class="shrink-0 lg:sticky lg:top-14 lg:block lg:h-[calc(100dvh-3.5rem)] lg:w-64 lg:overflow-x-hidden lg:overflow-y-auto lg:py-8"
        :class="open ? 'fixed inset-x-0 top-14 bottom-0 z-30 block overflow-y-auto bg-default px-4 py-6' : 'hidden'"
        @click="open = false"
      >
        <UContentNavigation :navigation="links" highlight />
        <UNavigationMenu
          class="mt-4"
          orientation="vertical"
          :items="[{ label: 'API Reference', icon: 'i-lucide-braces', to: '/docs/api-reference' }]"
        />
      </aside>
      <main class="min-w-0 flex-1 py-8">
        <slot />
      </main>
    </div>
  </div>
</template>
