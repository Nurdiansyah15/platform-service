<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { CompanyResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const toast = useToast()
const status = ref<string | undefined>(undefined)
const list = useListQuery<CompanyResponse>('/api/v1/companies', { filters: { status }, syncUrl: true })
const showCreate = ref(false)

const UAvatar = resolveComponent('UAvatar')
const StatusBadge = resolveComponent('StatusBadge')
const UIcon = resolveComponent('UIcon')

const columns: TableColumn<CompanyResponse>[] = [
  {
    accessorKey: 'name',
    header: 'Company',
    cell: ({ row }) => h('div', { class: 'flex items-center gap-3' }, [
      h(UAvatar, { text: initials(row.original.name), size: 'sm', class: 'rounded-md' }),
      h('div', { class: 'min-w-0' }, [
        h('p', { class: 'truncate font-medium text-highlighted' }, row.original.name),
        h('p', { class: 'truncate text-xs text-muted' }, row.original.slug),
      ]),
    ]),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
  { accessorKey: 'member_count', header: 'Anggota aktif', cell: ({ row }) => h('span', { class: 'text-highlighted' }, String(row.original.member_count ?? 0)) },
  { accessorKey: 'created_at', header: 'Dibuat', cell: ({ row }) => h('span', { class: 'text-muted' }, formatDate(row.original.created_at)) },
  { id: 'go', cell: () => h('div', { class: 'text-right' }, h(UIcon, { name: 'i-lucide-chevron-right', class: 'size-4 text-dimmed' })) },
]

function onCreated(c: CompanyResponse) {
  toast.add({ title: `${c.name} dibuat`, description: 'Tambahkan anggota untuk company ini.', color: 'success' })
  navigateTo(`/companies/${c.id}/members?add=1`)
}
</script>

<template>
  <AppPage title="Companies" description="Semua company di platform. Admin bisa membuka company mana pun tanpa menjadi anggota.">
    <template #actions>
      <UButton icon="i-lucide-plus" @click="showCreate = true">Buat company</UButton>
    </template>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari nama atau slug" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'suspended', 'closed'])" placeholder="Semua status" class="w-44" />
    </div>

    <UAlert v-if="list.error.value" color="error" variant="subtle" :title="list.error.value"
      :actions="[{ label: 'Coba lagi', color: 'neutral', variant: 'outline', onClick: list.reload }]" />

    <UTable
      :data="list.items.value"
      :columns="columns"
      :loading="list.loading.value"
      class="rounded-lg border border-default"
      :ui="{ tr: 'cursor-pointer' }"
      @select="(_e: Event, row: { original: CompanyResponse }) => navigateTo(`/companies/${row.original.id}`)"
    >
      <template #empty>
        <UEmpty icon="i-lucide-building-2" title="Tidak ada company" description="Coba ubah pencarian atau buat company baru." variant="naked" />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <CompanyCreateModal v-model:open="showCreate" @created="onCreated" />
  </AppPage>
</template>
