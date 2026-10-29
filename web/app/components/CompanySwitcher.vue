<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

defineProps<{ collapsed?: boolean }>()

const auth = useAuthStore()
const route = useRoute()
const toast = useToast()
const { toastError } = useApiError()

const active = computed(() => auth.activeCompany)

async function select(id: string) {
  if (id === auth.activeCompanyId) return
  const previous = auth.activeCompanyId
  try {
    await auth.switchCompany(id)
    toast.add({ title: 'Company aktif diganti', color: 'success' })
    // Tetap di halaman yang sama untuk company baru bila sedang di dalam company lama.
    if (previous && route.path.startsWith(`/companies/${previous}`)) {
      await navigateTo(route.path.replace(`/companies/${previous}`, `/companies/${id}`).replace(/\/(members|roles)\/[^/]+$/, '/$1'))
    } else if (!auth.isPlatformAdmin) {
      await navigateTo(`/companies/${id}`)
    }
  } catch (err) {
    toastError(err, 'Gagal mengganti company')
  }
}

const items = computed<DropdownMenuItem[][]>(() => [
  auth.companies.map(c => ({
    label: c.name,
    icon: c.id === auth.activeCompanyId ? 'i-lucide-check' : 'i-lucide-building-2',
    onSelect: () => select(c.id),
  })),
  [
    { label: 'Buat company baru', icon: 'i-lucide-plus', to: '/onboarding?create=1' },
    ...(auth.isPlatformAdmin ? [{ label: 'Semua company', icon: 'i-lucide-list', to: '/admin/companies' }] : []),
  ],
])
</script>

<template>
  <UDropdownMenu :items="items" :content="{ align: 'start' }" :ui="{ content: 'w-64' }">
    <UButton
      color="neutral"
      variant="ghost"
      block
      :square="collapsed"
      class="data-[state=open]:bg-elevated"
      :class="collapsed ? '' : 'justify-start py-2'"
    >
      <UAvatar :text="initials(active?.name ?? 'P')" size="xs" class="rounded-md bg-primary text-inverted" />
      <template v-if="!collapsed">
        <span class="min-w-0 flex-1 text-left">
          <span class="block truncate text-sm font-semibold text-highlighted">{{ active?.name ?? 'Platform Console' }}</span>
          <span class="block truncate text-xs text-muted">{{ active ? 'Company aktif' : auth.isPlatformAdmin ? 'Platform admin' : 'Pilih company' }}</span>
        </span>
        <UIcon name="i-lucide-chevrons-up-down" class="size-4 text-dimmed" />
      </template>
    </UButton>
  </UDropdownMenu>
</template>
