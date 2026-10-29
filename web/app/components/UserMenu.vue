<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

defineProps<{ collapsed?: boolean }>()

const auth = useAuthStore()
const colorMode = useColorMode()

async function logout() {
  await auth.logout()
  await navigateTo('/login')
}

const items = computed<DropdownMenuItem[][]>(() => [
  [{ type: 'label', label: auth.user?.name ?? '', description: auth.user?.email ?? '' }],
  [
    { label: 'Profil', icon: 'i-lucide-user-round', to: '/profile' },
    {
      label: 'Tema',
      icon: 'i-lucide-sun-moon',
      children: [
        { label: 'Terang', icon: 'i-lucide-sun', type: 'checkbox', checked: colorMode.preference === 'light', onSelect: () => (colorMode.preference = 'light') },
        { label: 'Gelap', icon: 'i-lucide-moon', type: 'checkbox', checked: colorMode.preference === 'dark', onSelect: () => (colorMode.preference = 'dark') },
        { label: 'Ikuti sistem', icon: 'i-lucide-monitor', type: 'checkbox', checked: colorMode.preference === 'system', onSelect: () => (colorMode.preference = 'system') },
      ],
    },
  ],
  [{ label: 'Keluar', icon: 'i-lucide-log-out', color: 'error', onSelect: logout }],
])
</script>

<template>
  <UDropdownMenu :items="items" :content="{ align: 'center', collisionPadding: 12 }" :ui="{ content: collapsed ? 'w-48' : 'w-(--reka-dropdown-menu-trigger-width)' }">
    <UButton
      color="neutral"
      variant="ghost"
      block
      :square="collapsed"
      class="data-[state=open]:bg-elevated"
      :class="collapsed ? '' : 'justify-start'"
    >
      <UAvatar :text="initials(auth.user?.name)" size="xs" />
      <template v-if="!collapsed">
        <span class="min-w-0 flex-1 text-left">
          <span class="block truncate text-sm font-medium text-highlighted">{{ auth.user?.name }}</span>
          <span class="block truncate text-xs text-muted">{{ auth.isPlatformAdmin ? 'Platform admin' : auth.user?.email }}</span>
        </span>
        <UIcon name="i-lucide-chevrons-up-down" class="size-4 text-dimmed" />
      </template>
    </UButton>
  </UDropdownMenu>
</template>
