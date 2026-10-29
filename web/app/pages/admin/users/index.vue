<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { UserResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const route = useRoute()
const toast = useToast()

const status = ref<string | undefined>(undefined)
const isAdmin = ref<boolean | undefined>(undefined)

const list = useListQuery<UserResponse>('/api/v1/users', {
  filters: { status, is_platform_admin: isAdmin },
  syncUrl: true,
})
const showCreate = ref(route.query.create === '1')

const roleItems = [
  { label: 'Semua peran', value: undefined },
  { label: 'Platform admin', value: true },
  { label: 'User biasa', value: false },
]

const UAvatar = resolveComponent('UAvatar')
const UBadge = resolveComponent('UBadge')
const UIcon = resolveComponent('UIcon')
const StatusBadge = resolveComponent('StatusBadge')

const columns: TableColumn<UserResponse>[] = [
  {
    accessorKey: 'name',
    header: 'User',
    cell: ({ row }) => h('div', { class: 'flex items-center gap-3' }, [
      h(UAvatar, { text: initials(row.original.name), size: 'sm' }),
      h('div', { class: 'min-w-0' }, [
        h('p', { class: 'truncate font-medium text-highlighted' }, row.original.name),
        h('p', { class: 'truncate text-xs text-muted' }, row.original.email),
      ]),
    ]),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
  {
    id: 'role',
    header: 'Peran',
    cell: ({ row }) => h('div', { class: 'flex flex-wrap gap-1' }, [
      row.original.is_platform_admin
        ? h(UBadge, { size: 'sm', variant: 'subtle', color: 'primary' }, () => 'Platform admin')
        : h('span', { class: 'text-sm text-muted' }, 'User'),
      !row.original.has_password ? h(UBadge, { size: 'sm', variant: 'subtle', color: 'warning' }, () => 'Tanpa password') : null,
    ]),
  },
  { accessorKey: 'created_at', header: 'Dibuat', cell: ({ row }) => h('span', { class: 'text-muted' }, formatDate(row.original.created_at)) },
  { id: 'go', cell: () => h('div', { class: 'text-right' }, h(UIcon, { name: 'i-lucide-chevron-right', class: 'size-4 text-dimmed' })) },
]

function onCreated(u: UserResponse) {
  toast.add({ title: `${u.name} dibuat`, description: 'Tambahkan ke company dari halaman detail.', color: 'success' })
  navigateTo(`/admin/users/${u.id}`)
}
</script>

<template>
  <AppPage title="Users" description="Semua akun di platform.">
    <template #actions>
      <UButton icon="i-lucide-user-plus" @click="showCreate = true">Buat user</UButton>
    </template>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari nama atau email" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'inactive', 'suspended'])" placeholder="Semua status" class="w-44" />
      <USelect v-model="isAdmin" :items="roleItems" placeholder="Semua peran" class="w-44" />
    </div>

    <UAlert v-if="list.error.value" color="error" variant="subtle" :title="list.error.value"
      :actions="[{ label: 'Coba lagi', color: 'neutral', variant: 'outline', onClick: list.reload }]" />

    <UTable
      :data="list.items.value"
      :columns="columns"
      :loading="list.loading.value"
      class="rounded-lg border border-default"
      :ui="{ tr: 'cursor-pointer' }"
      @select="(_e: Event, row: { original: UserResponse }) => navigateTo(`/admin/users/${row.original.id}`)"
    >
      <template #empty>
        <UEmpty icon="i-lucide-users" title="Tidak ada user yang cocok" description="Coba ubah pencarian atau filter." variant="naked" />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <UserCreateModal v-model:open="showCreate" @created="onCreated" />
  </AppPage>
</template>
