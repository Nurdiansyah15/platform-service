<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { PermissionResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const ctx = useSolutionContext()
const status = ref<string | undefined>(undefined)
const list = useListQuery<PermissionResponse>(
  () => `/api/v1/solutions/${ctx.solutionId.value}/permissions`,
  { filters: { status }, limit: 50, syncUrl: true },
)

const StatusBadge = resolveComponent('StatusBadge')
const columns: TableColumn<PermissionResponse>[] = [
  { accessorKey: 'code', header: 'Kode', cell: ({ row }) => h('span', { class: 'font-mono text-xs text-highlighted' }, row.original.code) },
  {
    accessorKey: 'name',
    header: 'Nama',
    cell: ({ row }) => h('div', [
      h('p', { class: 'text-highlighted' }, row.original.name),
      row.original.description ? h('p', { class: 'text-xs text-muted' }, row.original.description) : null,
    ]),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
]
</script>

<template>
  <SolutionPage title="Permission" :trail="[{ label: 'Permission' }]">
    <UAlert
      color="neutral"
      variant="subtle"
      icon="i-lucide-refresh-cw"
      title="Katalog dikirim oleh backend solution"
      description="Permission disinkronkan lewat PUT /solutions/{id}/permissions/sync (token service, scope catalog:sync). Halaman ini hanya-baca."
    />

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari kode atau nama" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'deprecated'])" placeholder="Semua status" class="w-40" />
    </div>

    <UTable :data="list.items.value" :columns="columns" :loading="list.loading.value" class="rounded-lg border border-default">
      <template #empty>
        <UEmpty icon="i-lucide-key-round" title="Belum ada permission" description="Katalog belum pernah disinkronkan." variant="naked" />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />
  </SolutionPage>
</template>
