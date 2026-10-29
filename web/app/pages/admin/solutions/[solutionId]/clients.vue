<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { DropdownMenuItem, FormError, FormSubmitEvent, TableColumn } from '@nuxt/ui'
import { SERVICE_SCOPES, type SolutionClientResponse, type SolutionClientWithSecretResponse } from '~/types/api'
import SecretModal from '~/components/SecretModal.vue'

definePageMeta({ middleware: 'admin' })

const ctx = useSolutionContext()
const api = useApi()
const toast = useToast()
const overlay = useOverlay()
const secretModal = overlay.create(SecretModal)
const { toastError, fieldErrors } = useApiError()
const { confirm } = useConfirm()
const formRef = useTemplateRef('formRef')

const status = ref<string | undefined>('active')
const list = useListQuery<SolutionClientResponse>(
  () => `/api/v1/solutions/${ctx.solutionId.value}/clients`,
  { filters: { status }, syncUrl: true },
)

const SCOPE_HELP: Record<string, string> = {
  'users:read': 'Baca company & anggota',
  'users:write': 'Kelola anggota company',
  'org:read': 'Baca cabang',
  'org:write': 'Kelola company & cabang',
  'authz:check': 'Cek permission user',
  'roles:write': 'Kelola custom role & assignment',
  'entitlements:read': 'Baca entitlement',
  'usage:write': 'Laporkan pemakaian',
  'catalog:sync': 'Sinkron katalog permission',
  'events:read': 'Baca event',
  'license:read': 'Baca license (on-prem)',
}
const scopeItems = SERVICE_SCOPES.map(s => ({ label: s, value: s, description: SCOPE_HELP[s] }))

// ── Buat / ubah ──
const showForm = ref(false)
const editing = ref<SolutionClientResponse | null>(null)
const saving = ref(false)
const state = reactive<{ name: string; scopes: string[] }>({ name: '', scopes: [] })

function openCreate() {
  editing.value = null
  Object.assign(state, { name: '', scopes: [] })
  showForm.value = true
}
function openEdit(c: SolutionClientResponse) {
  editing.value = c
  Object.assign(state, { name: c.name, scopes: [...c.scopes] })
  showForm.value = true
}

function validate(s: typeof state): FormError[] {
  return s.name.trim() ? [] : [{ name: 'name', message: 'Nama wajib diisi' }]
}

function revealSecret(c: SolutionClientWithSecretResponse, title: string) {
  secretModal.open({ title, clientId: c.client_id, secret: c.client_secret })
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const body = { name: e.data.name.trim(), scopes: e.data.scopes }
    if (editing.value) {
      await api.patch(`/api/v1/solution-clients/${editing.value.id}`, body)
      toast.add({ title: 'Client diperbarui', description: 'Scope baru langsung berlaku.', color: 'success' })
    } else {
      const res = await api.post<SolutionClientWithSecretResponse>(`/api/v1/solutions/${ctx.solutionId.value}/clients`, body)
      revealSecret(res.data, 'Client dibuat')
    }
    showForm.value = false
    list.reload()
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal menyimpan client')
  } finally {
    saving.value = false
  }
}

async function rotate(c: SolutionClientResponse) {
  const ok = await confirm({
    title: `Rotate secret ${c.name}?`,
    description: 'Secret lama langsung tidak berlaku. Backend yang memakainya harus diperbarui.',
    confirmLabel: 'Rotate secret',
  })
  if (!ok) return
  try {
    const res = await api.post<SolutionClientWithSecretResponse>(`/api/v1/solution-clients/${c.id}/rotate-secret`)
    revealSecret(res.data, 'Secret baru')
  } catch (err) {
    toastError(err)
  }
}

async function revoke(c: SolutionClientResponse) {
  const ok = await confirm({
    title: `Revoke ${c.name}?`,
    description: 'Client tidak bisa dipakai lagi dan token yang sudah terbit langsung ditolak. Tindakan ini permanen.',
    confirmLabel: 'Revoke',
    danger: true,
    typeToConfirm: c.name,
  })
  if (!ok) return
  try {
    await api.post(`/api/v1/solution-clients/${c.id}/revoke`)
    toast.add({ title: 'Client di-revoke', color: 'success' })
    list.reload()
  } catch (err) {
    toastError(err)
  }
}

function rowActions(c: SolutionClientResponse): DropdownMenuItem[][] {
  if (c.status !== 'active') return [[{ label: 'Sudah di-revoke', disabled: true }]]
  return [
    [
      { label: 'Ubah nama & scope', icon: 'i-lucide-pencil', onSelect: () => openEdit(c) },
      { label: 'Rotate secret', icon: 'i-lucide-rotate-cw', onSelect: () => rotate(c) },
    ],
    [{ label: 'Revoke', icon: 'i-lucide-ban', color: 'error', onSelect: () => revoke(c) }],
  ]
}

const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const StatusBadge = resolveComponent('StatusBadge')
const CopyText = resolveComponent('CopyText')

const columns: TableColumn<SolutionClientResponse>[] = [
  {
    accessorKey: 'name',
    header: 'Client',
    cell: ({ row }) => h('div', { class: 'min-w-0' }, [
      h('p', { class: 'font-medium text-highlighted' }, row.original.name),
      h(CopyText, { value: row.original.client_id, mono: true, class: 'text-muted' }),
    ]),
  },
  {
    accessorKey: 'scopes',
    header: 'Scope',
    cell: ({ row }) => h('div', { class: 'flex max-w-md flex-wrap gap-1' }, row.original.scopes.length
      ? row.original.scopes.map(s => h(UBadge, { size: 'xs', color: 'neutral', variant: 'outline' }, () => s))
      : [h('span', { class: 'text-xs text-dimmed' }, 'Tanpa scope')]),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
  {
    accessorKey: 'last_used_at',
    header: 'Terakhir dipakai',
    cell: ({ row }) => h('span', { class: 'text-muted' }, row.original.last_used_at ? formatRelative(row.original.last_used_at) : 'Belum pernah'),
  },
  {
    id: 'actions',
    cell: ({ row }) => h('div', { class: 'text-right' }, h(UDropdownMenu, { items: rowActions(row.original), content: { align: 'end' } },
      () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', 'aria-label': 'Aksi' }))),
  },
]
</script>

<template>
  <SolutionPage title="Client" :trail="[{ label: 'Client' }]">
    <template #actions>
      <UButton icon="i-lucide-plus" @click="openCreate">Buat client</UButton>
    </template>

    <p class="text-sm text-muted">
      Kredensial <span class="font-mono">client_credentials</span> untuk backend
      {{ ctx.solution.value?.name ?? 'solution' }}. Tukar menjadi token lewat <span class="font-mono">POST /api/v1/oauth/token</span>.
    </p>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari nama atau client_id" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'revoked'])" placeholder="Semua status" class="w-40" />
    </div>

    <UAlert v-if="list.error.value" color="error" variant="subtle" :title="list.error.value" />

    <UTable :data="list.items.value" :columns="columns" :loading="list.loading.value" class="rounded-lg border border-default">
      <template #empty>
        <UEmpty
          icon="i-lucide-plug"
          title="Belum ada client"
          description="Buat client agar backend solution bisa memanggil API platform."
          :actions="[{ label: 'Buat client', icon: 'i-lucide-plus', onClick: openCreate }]"
          variant="naked"
        />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <UModal v-model:open="showForm" :title="editing ? 'Ubah client' : 'Buat client'">
      <template #body>
        <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
          <UFormField label="Nama" name="name" required help="Mis. lingkungan & fungsi: “Easy HR Backend Prod”.">
            <UInput v-model="state.name" class="w-full" autofocus />
          </UFormField>
          <UFormField label="Scope" name="scopes" help="Berikan hanya scope yang benar-benar dibutuhkan.">
            <USelectMenu v-model="state.scopes" :items="scopeItems" value-key="value" multiple placeholder="Pilih scope" class="w-full" />
          </UFormField>
          <div class="flex justify-end gap-2 pt-2">
            <UButton color="neutral" variant="ghost" @click="showForm = false">Batal</UButton>
            <UButton type="submit" :loading="saving">{{ editing ? 'Simpan' : 'Buat client' }}</UButton>
          </div>
        </UForm>
      </template>
    </UModal>
  </SolutionPage>
</template>
