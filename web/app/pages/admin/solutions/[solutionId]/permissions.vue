<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { PermissionGroupResponse, PermissionResponse, ScopeTypeResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const api = useApi()
const ctx = useSolutionContext()
const toast = useToast()
const { toastError } = useApiError()
const { confirm } = useConfirm()
const status = ref<string | undefined>(undefined)
const group = ref<string | undefined>(undefined)
const source = ref<string | undefined>(undefined)
const list = useListQuery<PermissionResponse>(
  () => `/api/v1/solutions/${ctx.solutionId.value}/permissions`,
  { filters: { status, group, source }, limit: 50, syncUrl: true },
)

const sourceOptions = [
  { label: 'Semua sumber', value: undefined as string | undefined },
  { label: 'Sync katalog', value: 'sync' as string | undefined },
  { label: 'Manual', value: 'manual' as string | undefined },
]

// ── Permission manual: tambah / ubah / deprecate ──
const showForm = ref(false)
const editing = ref<PermissionResponse | null>(null)
function openCreate() {
  editing.value = null
  showForm.value = true
}
function openEdit(p: PermissionResponse) {
  editing.value = p
  showForm.value = true
}
function onSaved(p: PermissionResponse) {
  toast.add({ title: editing.value ? 'Permission diperbarui' : `Permission ${p.code} ditambahkan`, color: 'success' })
  list.reload()
}
async function setStatus(p: PermissionResponse, next: 'active' | 'deprecated') {
  if (next === 'deprecated') {
    const ok = await confirm({
      title: `Hapus permission ${p.name}?`,
      description: 'Permission menjadi deprecated: tidak bisa ditambahkan ke role lagi, tapi role yang sudah memakainya tetap berlaku. Bisa diaktifkan kembali.',
      confirmLabel: 'Hapus',
      danger: true,
    })
    if (!ok) return
  }
  try {
    await api.patch(`/api/v1/permissions/${p.id}`, { status: next })
    toast.add({ title: next === 'active' ? 'Permission diaktifkan' : 'Permission dihapus (deprecated)', color: 'success' })
    list.reload()
  } catch (err) {
    toastError(err)
  }
}
// ── Scope type manual: tambah / ubah / deprecate ──
const showScopeForm = ref(false)
const editingScope = ref<ScopeTypeResponse | null>(null)
function openScopeForm(st: ScopeTypeResponse | null) {
  editingScope.value = st
  showScopeForm.value = true
}
function onScopeSaved(st: ScopeTypeResponse) {
  toast.add({ title: editingScope.value ? 'Scope type diperbarui' : `Scope type ${st.name} ditambahkan`, color: 'success' })
  scopeTypes.load(true)
}
async function setScopeStatus(st: ScopeTypeResponse, next: 'active' | 'deprecated') {
  if (next === 'deprecated') {
    const ok = await confirm({
      title: `Hapus scope type ${st.name}?`,
      description: 'Scope type menjadi deprecated: tidak bisa dipakai untuk memberi role lagi, tapi role yang sudah diberikan di scope ini tetap berlaku. Bisa diaktifkan kembali.',
      confirmLabel: 'Hapus',
      danger: true,
    })
    if (!ok) return
  }
  try {
    await api.patch(`/api/v1/scope-types/${st.id}`, { status: next })
    toast.add({ title: next === 'active' ? 'Scope type diaktifkan' : 'Scope type dihapus (deprecated)', color: 'success' })
    scopeTypes.load(true)
  } catch (err) {
    toastError(err)
  }
}
function scopeActions(st: ScopeTypeResponse) {
  return [[
    { label: 'Ubah', icon: 'i-lucide-pencil', onSelect: () => openScopeForm(st) },
    st.status === 'active'
      ? { label: 'Hapus', icon: 'i-lucide-trash-2', color: 'error' as const, onSelect: () => setScopeStatus(st, 'deprecated') }
      : { label: 'Aktifkan', icon: 'i-lucide-rotate-ccw', onSelect: () => setScopeStatus(st, 'active') },
  ]]
}

// ── Grup manual: tambah / ubah / deprecate ──
const showGroupForm = ref(false)
const editingGroup = ref<PermissionGroupResponse | null>(null)
function openGroupForm(g: PermissionGroupResponse | null) {
  editingGroup.value = g
  showGroupForm.value = true
}
function onGroupSaved(g: PermissionGroupResponse) {
  toast.add({ title: editingGroup.value ? 'Grup diperbarui' : `Grup ${g.name} ditambahkan`, color: 'success' })
  loadGroups()
  list.reload()
}
async function setGroupStatus(g: PermissionGroupResponse, next: 'active' | 'deprecated') {
  if (next === 'deprecated') {
    const ok = await confirm({
      title: `Hapus grup ${g.name}?`,
      description: 'Grup menjadi deprecated: tidak bisa dipilih untuk permission lagi, tapi permission yang sudah ada di grup ini tetap tampil di dalamnya. Bisa diaktifkan kembali.',
      confirmLabel: 'Hapus',
      danger: true,
    })
    if (!ok) return
  }
  try {
    await api.patch(`/api/v1/permission-groups/${g.id}`, { status: next })
    toast.add({ title: next === 'active' ? 'Grup diaktifkan' : 'Grup dihapus (deprecated)', color: 'success' })
    loadGroups()
  } catch (err) {
    toastError(err)
  }
}
function groupActions(g: PermissionGroupResponse) {
  return [[
    { label: 'Ubah', icon: 'i-lucide-pencil', onSelect: () => openGroupForm(g) },
    g.status === 'active'
      ? { label: 'Hapus', icon: 'i-lucide-trash-2', color: 'error' as const, onSelect: () => setGroupStatus(g, 'deprecated') }
      : { label: 'Aktifkan', icon: 'i-lucide-rotate-ccw', onSelect: () => setGroupStatus(g, 'active') },
  ]]
}

function rowActions(p: PermissionResponse) {
  return [[
    { label: 'Ubah', icon: 'i-lucide-pencil', onSelect: () => openEdit(p) },
    p.status === 'active'
      ? { label: 'Hapus', icon: 'i-lucide-trash-2', color: 'error' as const, onSelect: () => setStatus(p, 'deprecated') }
      : { label: 'Aktifkan', icon: 'i-lucide-rotate-ccw', onSelect: () => setStatus(p, 'active') },
  ]]
}

const scopeTypes = useScopeTypes(() => ctx.solutionId.value)
const groups = ref<PermissionGroupResponse[]>([])
async function loadGroups() {
  try {
    groups.value = (await api.get<PermissionGroupResponse[]>(`/api/v1/solutions/${ctx.solutionId.value}/permission-groups`)).data
  } catch {
    groups.value = []
  }
}
watch(() => ctx.solutionId.value, loadGroups, { immediate: true })
const orderedGroups = computed(() => orderGroupTree(groups.value))

const groupOptions = computed(() => [
  { label: 'Semua grup', value: undefined as string | undefined },
  ...orderedGroups.value.map(({ group: g, path }) => ({ label: path, value: g.code as string | undefined })),
  { label: 'Tanpa grup', value: '-' as string | undefined },
])
const customScopes = computed(() => scopeTypes.items.value.filter(s => !s.builtin))

const StatusBadge = resolveComponent('StatusBadge')
const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const columns: TableColumn<PermissionResponse>[] = [
  {
    accessorKey: 'name',
    header: 'Permission',
    cell: ({ row }) => h('div', [
      h('p', { class: 'text-highlighted' }, row.original.name),
      h('p', { class: 'font-mono text-xs text-muted' }, row.original.code),
      row.original.description ? h('p', { class: 'text-xs text-muted' }, row.original.description) : null,
    ]),
  },
  {
    id: 'group',
    header: 'Grup',
    cell: ({ row }) => row.original.group
      ? h('span', { class: 'text-highlighted' }, row.original.group.name)
      : h('span', { class: 'text-muted' }, '—'),
  },
  {
    accessorKey: 'source',
    header: 'Sumber',
    cell: ({ row }) => h(UBadge, row.original.source === 'manual'
      ? { label: 'Manual', color: 'primary', variant: 'subtle', size: 'sm' }
      : { label: 'Sync', color: 'neutral', variant: 'subtle', size: 'sm' }),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
  {
    id: 'actions',
    header: '',
    cell: ({ row }) => row.original.source === 'manual'
      ? h('div', { class: 'flex justify-end' }, h(UDropdownMenu, { items: rowActions(row.original), content: { align: 'end' } },
          () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', size: 'sm', 'aria-label': 'Aksi' })))
      : null,
  },
]
</script>

<template>
  <SolutionPage title="Permission" :trail="[{ label: 'Permission' }]">
    <UAlert
      color="neutral"
      variant="subtle"
      icon="i-lucide-refresh-cw"
      title="Dua sumber permission"
      description="Sync: dikirim backend solution lewat PUT /solutions/{id}/permissions/sync (beserta grup & scope type) dan hanya-baca di sini. Manual: ditambahkan dari halaman ini dan tidak disentuh sync — code-nya tidak boleh dipakai lagi di katalog sync."
    />

    <div class="grid gap-4 lg:grid-cols-2">
      <SectionCard title="Scope type" description="Lingkup tempat role bisa diberikan. Company dan cabang selalu tersedia.">
        <template #actions>
          <UButton icon="i-lucide-plus" size="sm" variant="soft" @click="openScopeForm(null)">Tambah scope</UButton>
        </template>
        <ul class="divide-y divide-default text-sm">
          <li v-for="s in scopeTypes.items.value" :key="s.code" class="flex items-start justify-between gap-3 py-2">
            <div class="min-w-0">
              <p class="text-highlighted">{{ scopeTypes.nameOf(s.code) }} <span class="font-mono text-xs text-muted">{{ s.code }}</span></p>
              <p class="text-xs text-muted">
                {{ s.builtin ? 'Bawaan platform' : `Di bawah ${s.parent === 'branch' ? 'cabang' : 'company'}` }}
                <template v-if="s.description"> · {{ s.description }}</template>
              </p>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <UBadge v-if="!s.builtin" :label="s.source === 'manual' ? 'Manual' : 'Sync'" :color="s.source === 'manual' ? 'primary' : 'neutral'" variant="subtle" size="sm" />
              <StatusBadge v-if="s.status !== 'active'" :status="s.status" size="xs" />
              <UDropdownMenu v-if="s.source === 'manual'" :items="scopeActions(s)" :content="{ align: 'end' }">
                <UButton icon="i-lucide-ellipsis-vertical" color="neutral" variant="ghost" size="xs" aria-label="Aksi scope type" />
              </UDropdownMenu>
            </div>
          </li>
        </ul>
        <p v-if="!customScopes.length" class="text-xs text-muted">Solution ini belum mendefinisikan scope custom.</p>
      </SectionCard>

      <SectionCard title="Grup permission" description="Pengelompokan permission saat menyusun role.">
        <template #actions>
          <UButton icon="i-lucide-plus" size="sm" variant="soft" @click="openGroupForm(null)">Tambah grup</UButton>
        </template>
        <ul v-if="groups.length" class="divide-y divide-default text-sm">
          <li v-for="{ group: g, depth } in orderedGroups" :key="g.code" class="flex items-start justify-between gap-3 py-2">
            <div class="flex min-w-0 items-start gap-1.5" :style="{ paddingLeft: `${depth * 1.25}rem` }">
              <UIcon v-if="depth" name="i-lucide-corner-down-right" class="mt-0.5 size-3.5 shrink-0 text-dimmed" />
              <div class="min-w-0">
                <p class="text-highlighted">{{ g.name }} <span class="font-mono text-xs text-muted">{{ g.code }}</span></p>
                <p v-if="g.description" class="text-xs text-muted">{{ g.description }}</p>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <UBadge :label="g.source === 'manual' ? 'Manual' : 'Sync'" :color="g.source === 'manual' ? 'primary' : 'neutral'" variant="subtle" size="sm" />
              <StatusBadge v-if="g.status !== 'active'" :status="g.status" size="xs" />
              <UDropdownMenu v-if="g.source === 'manual'" :items="groupActions(g)" :content="{ align: 'end' }">
                <UButton icon="i-lucide-ellipsis-vertical" color="neutral" variant="ghost" size="xs" aria-label="Aksi grup" />
              </UDropdownMenu>
            </div>
          </li>
        </ul>
        <p v-else class="text-xs text-muted">Belum ada grup; permission dikelompokkan berdasarkan prefix kode.</p>
      </SectionCard>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari kode atau nama" class="w-full sm:w-72" />
      <USelect v-model="group" :items="groupOptions" placeholder="Semua grup" class="w-44" />
      <USelect v-model="source" :items="sourceOptions" placeholder="Semua sumber" class="w-40" />
      <USelect v-model="status" :items="statusOptions(['active', 'deprecated'])" placeholder="Semua status" class="w-40" />
      <UButton icon="i-lucide-plus" class="ml-auto" @click="openCreate">Tambah permission</UButton>
    </div>

    <UTable :data="list.items.value" :columns="columns" :loading="list.loading.value" class="rounded-lg border border-default">
      <template #empty>
        <UEmpty icon="i-lucide-key-round" title="Belum ada permission" description="Katalog belum pernah disinkronkan." variant="naked" />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <PermissionFormModal
      v-model:open="showForm"
      :solution-id="ctx.solutionId.value"
      :permission="editing"
      :groups="groups"
      @saved="onSaved"
    />
    <ScopeTypeFormModal
      v-model:open="showScopeForm"
      :solution-id="ctx.solutionId.value"
      :scope-type="editingScope"
      @saved="onScopeSaved"
    />
    <PermissionGroupFormModal
      v-model:open="showGroupForm"
      :solution-id="ctx.solutionId.value"
      :group="editingGroup"
      :groups="groups"
      @saved="onGroupSaved"
    />
  </SolutionPage>
</template>
