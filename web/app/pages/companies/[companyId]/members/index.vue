<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui'
import type { CompanyMemberResponse, MemberCandidate, MemberStatus } from '~/types/api'

const ctx = useCompanyContext()
const route = useRoute()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()
const { confirm } = useConfirm()

const status = ref<string | undefined>(undefined)
const list = useListQuery<CompanyMemberResponse>(
  () => (ctx.canAccess.value ? `/api/v1/companies/${ctx.companyId.value}/members` : null),
  { filters: { status }, syncUrl: true },
)
const base = computed(() => `/companies/${ctx.companyId.value}/members`)

// ── Tambah anggota ──
const showAdd = ref(route.query.add === '1')
const candidate = ref<MemberCandidate | undefined>()
const addStatus = ref<'active' | 'invited'>('active')
const adding = ref(false)

watch(showAdd, (v) => {
  if (v) {
    candidate.value = undefined
    addStatus.value = 'active'
  }
})

async function addMember() {
  if (!candidate.value) return
  adding.value = true
  try {
    await api.post(`/api/v1/companies/${ctx.companyId.value}/members`, { user_id: candidate.value.id, status: addStatus.value })
    toast.add({ title: `${candidate.value.name} ditambahkan`, color: 'success' })
    showAdd.value = false
    list.reload()
  } catch (err) {
    toastError(err, 'Gagal menambah anggota')
  } finally {
    adding.value = false
  }
}

// ── Aksi baris ──
async function setStatus(m: CompanyMemberResponse, next: MemberStatus) {
  try {
    await api.patch(`/api/v1/companies/${ctx.companyId.value}/members/${m.user_id}`, { status: next })
    toast.add({ title: next === 'active' ? 'Anggota diaktifkan' : 'Status diperbarui', color: 'success' })
    list.reload()
  } catch (err) {
    toastError(err)
  }
}

async function remove(m: CompanyMemberResponse) {
  const ok = await confirm({
    title: `Keluarkan ${m.name}?`,
    description: 'Semua role-nya di company ini ikut dicabut. Kamu bisa menambahkannya lagi nanti.',
    confirmLabel: 'Keluarkan',
    danger: true,
  })
  if (!ok) return
  try {
    await api.del(`/api/v1/companies/${ctx.companyId.value}/members/${m.user_id}`)
    toast.add({ title: `${m.name} dikeluarkan`, color: 'success' })
    list.reload()
  } catch (err) {
    toastError(err)
  }
}

function rowActions(m: CompanyMemberResponse): DropdownMenuItem[][] {
  const items: DropdownMenuItem[][] = [[{ label: 'Lihat role & akses', icon: 'i-lucide-shield', to: `${base.value}/${m.user_id}` }]]
  const second: DropdownMenuItem[] = []
  if (m.status !== 'active') second.push({ label: 'Aktifkan', icon: 'i-lucide-check', onSelect: () => setStatus(m, 'active') })
  if (m.status !== 'inactive') second.push({ label: 'Keluarkan', icon: 'i-lucide-user-minus', color: 'error', onSelect: () => remove(m) })
  return second.length ? [...items, second] : items
}

const UAvatar = resolveComponent('UAvatar')
const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const StatusBadge = resolveComponent('StatusBadge')

const columns: TableColumn<CompanyMemberResponse>[] = [
  {
    accessorKey: 'name',
    header: 'Anggota',
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
    accessorKey: 'role_count',
    header: 'Role',
    cell: ({ row }) => h('span', { class: row.original.role_count ? 'text-highlighted' : 'text-dimmed' },
      row.original.role_count ? `${row.original.role_count} role` : 'Belum ada'),
  },
  { accessorKey: 'joined_at', header: 'Bergabung', cell: ({ row }) => h('span', { class: 'text-muted' }, formatDate(row.original.joined_at)) },
  {
    id: 'actions',
    cell: ({ row }) => h('div', { class: 'text-right' }, h(UDropdownMenu, { items: rowActions(row.original), content: { align: 'end' } },
      () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', 'aria-label': 'Aksi', onClick: (e: Event) => e.stopPropagation() }))),
  },
]

function onSelect(_e: Event, row: { original: CompanyMemberResponse }) {
  navigateTo(`${base.value}/${row.original.user_id}`)
}
</script>

<template>
  <CompanyPage title="Anggota" :section="{ label: 'Anggota', to: base }">
    <template #actions>
      <UButton icon="i-lucide-user-plus" @click="showAdd = true">Tambah anggota</UButton>
    </template>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari nama atau email" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'invited', 'inactive'])" placeholder="Semua status" class="w-40" />
    </div>

    <UAlert v-if="list.error.value" color="error" variant="subtle" :title="list.error.value"
      :actions="[{ label: 'Coba lagi', color: 'neutral', variant: 'outline', onClick: list.reload }]" />

    <UTable
      :data="list.items.value"
      :columns="columns"
      :loading="list.loading.value"
      class="rounded-lg border border-default"
      :ui="{ tr: 'cursor-pointer' }"
      @select="onSelect"
    >
      <template #empty>
        <UEmpty
          icon="i-lucide-users"
          :title="list.search.value || status ? 'Tidak ada anggota yang cocok' : 'Belum ada anggota'"
          :description="list.search.value || status ? 'Coba ubah pencarian atau filter.' : 'Tambahkan user terdaftar sebagai anggota company.'"
          variant="naked"
        />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <UModal v-model:open="showAdd" title="Tambah anggota" description="Cari user yang sudah terdaftar di platform.">
      <template #body>
        <div class="space-y-4">
          <UFormField label="User" required>
            <UserPicker v-model="candidate" :company-id="ctx.companyId.value" />
          </UFormField>
          <UFormField label="Status" help="Diundang = belum aktif; bisa diaktifkan nanti.">
            <URadioGroup
              v-model="addStatus"
              orientation="horizontal"
              :items="[{ label: 'Aktif', value: 'active' }, { label: 'Diundang', value: 'invited' }]"
            />
          </UFormField>
          <p class="text-xs text-muted">User belum terdaftar? Minta admin platform membuatkan akunnya.</p>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="showAdd = false">Batal</UButton>
          <UButton :disabled="!candidate" :loading="adding" @click="addMember">Tambahkan</UButton>
        </div>
      </template>
    </UModal>
  </CompanyPage>
</template>
