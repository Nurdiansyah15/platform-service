<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const auth = useAuthStore()
const open = ref(false)

const nav = computed<NavigationMenuItem[][]>(() => {
  const groups: NavigationMenuItem[][] = []
  const cid = auth.activeCompanyId
  if (auth.isPlatformAdmin) {
    groups.push([{ label: 'Dashboard', icon: 'i-lucide-layout-dashboard', to: '/', exact: true }])
  }
  if (cid) {
    const base = `/companies/${cid}`
    groups.push([
      { label: auth.activeCompany?.name ?? 'Company', type: 'label' },
      { label: 'Ringkasan', icon: 'i-lucide-gauge', to: base, exact: true },
      { label: 'Anggota', icon: 'i-lucide-users', to: `${base}/members` },
      { label: 'Cabang', icon: 'i-lucide-git-branch', to: `${base}/branches` },
      { label: 'Role', icon: 'i-lucide-shield', to: `${base}/roles` },
      { label: 'Pengaturan', icon: 'i-lucide-settings', to: `${base}/settings` },
    ])
  }
  if (auth.isPlatformAdmin) {
    groups.push([
      { label: 'Platform', type: 'label' },
      { label: 'Companies', icon: 'i-lucide-building-2', to: '/admin/companies' },
      { label: 'Solutions', icon: 'i-lucide-boxes', to: '/admin/solutions' },
      { label: 'Users', icon: 'i-lucide-user-cog', to: '/admin/users' },
    ])
  }
  return groups
})
</script>

<template>
  <UDashboardGroup>
    <UDashboardSidebar
      id="main"
      v-model:open="open"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <CompanySwitcher :collapsed="collapsed" />
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu
          v-for="(group, i) in nav"
          :key="i"
          :collapsed="collapsed"
          :items="group"
          orientation="vertical"
          tooltip
          :class="i > 0 ? 'mt-2' : ''"
        />
        <UNavigationMenu
          :collapsed="collapsed"
          :items="[{ label: 'Dokumentasi', icon: 'i-lucide-book-open', to: '/docs' }]"
          orientation="vertical"
          tooltip
          class="mt-auto"
        />
      </template>

      <template #footer="{ collapsed }">
        <UserMenu :collapsed="collapsed" />
      </template>
    </UDashboardSidebar>

    <slot />
  </UDashboardGroup>
</template>
