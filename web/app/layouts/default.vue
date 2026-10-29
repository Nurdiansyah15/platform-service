<script setup lang="ts">
const auth = useAuthStore()
const route = useRoute()

interface NavItem {
  label: string
  to: string
  icon: string
  adminOnly?: boolean
}

const nav: NavItem[] = [
  { label: 'Dashboard', to: '/', icon: 'i-lucide-layout-dashboard' },
  { label: 'Companies', to: '/companies', icon: 'i-lucide-building-2' },
  { label: 'Solutions', to: '/solutions', icon: 'i-lucide-boxes' },
  { label: 'Users', to: '/admin/users', icon: 'i-lucide-users', adminOnly: true },
  { label: 'Profil', to: '/profile', icon: 'i-lucide-user-round' },
]

const visibleNav = computed(() => nav.filter(i => !i.adminOnly || auth.isPlatformAdmin))

function isActive(to: string) {
  if (to === '/') return route.path === '/'
  return route.path.startsWith(to)
}

async function onLogout() {
  await auth.logout()
  await navigateTo('/login')
}
</script>

<template>
  <div class="min-h-screen bg-elevated/40">
    <div class="flex min-h-screen">
      <aside class="hidden w-64 flex-col border-r border-default bg-default md:flex">
        <div class="flex h-14 items-center gap-2 border-b border-default px-4">
          <UIcon name="i-lucide-hexagon" class="size-5 text-primary" />
          <span class="font-semibold text-highlighted">Platform Console</span>
        </div>
        <nav class="flex-1 space-y-1 p-3">
          <NuxtLink
            v-for="item in visibleNav"
            :key="item.to"
            :to="item.to"
            class="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition"
            :class="isActive(item.to)
              ? 'bg-primary/10 text-primary'
              : 'text-muted hover:bg-elevated hover:text-highlighted'"
          >
            <UIcon :name="item.icon" class="size-4" />
            {{ item.label }}
          </NuxtLink>
        </nav>
        <div class="border-t border-default p-3">
          <p class="truncate text-xs text-dimmed">{{ auth.user?.email }}</p>
        </div>
      </aside>

      <div class="flex min-w-0 flex-1 flex-col">
        <header
          class="sticky top-0 z-10 flex h-14 items-center justify-between gap-4 border-b border-default bg-default px-4 md:px-6"
        >
          <div class="flex items-center gap-3">
            <NuxtLink to="/" class="flex items-center gap-2 md:hidden">
              <UIcon name="i-lucide-hexagon" class="size-5 text-primary" />
            </NuxtLink>
            <CompanySwitcher />
            <UBadge v-if="auth.isPlatformAdmin" color="primary" variant="subtle" size="sm">
              Platform Admin
            </UBadge>
          </div>
          <div class="flex items-center gap-2">
            <span class="hidden text-sm text-muted sm:inline">{{ auth.user?.name }}</span>
            <UButton color="neutral" variant="ghost" icon="i-lucide-log-out" @click="onLogout">
              Logout
            </UButton>
          </div>
        </header>

        <main class="flex-1 p-4 md:p-6">
          <slot />
        </main>
      </div>
    </div>
  </div>
</template>
