<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { FormError, FormSubmitEvent, TableColumn } from '@nuxt/ui'
import type { BranchResponse } from '~/types/api'

const ctx = useCompanyContext()
const route = useRoute()
const api = useApi()
const toast = useToast()
const { toastError, fieldErrors } = useApiError()
const { confirm } = useConfirm()
const formRef = useTemplateRef('formRef')
const perms = useCompanyPermissions()

const status = ref<string | undefined>(undefined)
const list = useListQuery<BranchResponse>(
  () => (ctx.canAccess.value ? `/api/v1/companies/${ctx.companyId.value}/branches` : null),
  { filters: { status }, syncUrl: true },
)
const base = computed(() => `/companies/${ctx.companyId.value}/branches`)

// ── Form buat / ubah (modal yang sama) ──
const showForm = ref(route.query.add === '1')
const editing = ref<BranchResponse | null>(null)
const saving = ref(false)
const state = reactive({ name: '', code: '', active: true })

function openCreate() {
  editing.value = null
  Object.assign(state, { name: '', code: '', active: true })
  showForm.value = true
}

function openEdit(b: BranchResponse) {
  editing.value = b
  Object.assign(state, { name: b.name, code: b.code, active: b.status === 'active' })
  showForm.value = true
}

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!s.code.trim()) errors.push({ name: 'code', message: 'Kode wajib diisi' })
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  const body = { name: e.data.name.trim(), code: e.data.code.trim(), status: e.data.active ? 'active' : 'inactive' }
  try {
    if (editing.value) await api.patch(`/api/v1/branches/${editing.value.id}`, body)
    else await api.post(`/api/v1/companies/${ctx.companyId.value}/branches`, body)
    toast.add({ title: editing.value ? 'Cabang diperbarui' : 'Cabang dibuat', color: 'success' })
    showForm.value = false
    list.reload()
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal menyimpan cabang')
  } finally {
    saving.value = false
  }
}

async function remove(b: BranchResponse) {
  const ok = await confirm({
    title: `Hapus cabang ${b.name}?`,
    description: 'Role anggota yang berlaku khusus di cabang ini ikut dicabut. Tindakan ini tidak bisa dibatalkan.',
    confirmLabel: 'Hapus cabang',
    danger: true,
    typeToConfirm: b.code,
  })
  if (!ok) return
  try {
    await api.del(`/api/v1/branches/${b.id}`)
    toast.add({ title: 'Cabang dihapus', color: 'success' })
    list.reload()
  } catch (err) {
    toastError(err)
  }
}

const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const StatusBadge = resolveComponent('StatusBadge')

const columns: TableColumn<BranchResponse>[] = [
  {
    accessorKey: 'name',
    header: 'Cabang',
    cell: ({ row }) => h('div', [
      h('p', { class: 'font-medium text-highlighted' }, row.original.name),
      h('p', { class: 'font-mono text-xs text-muted' }, row.original.code),
    ]),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
  { accessorKey: 'created_at', header: 'Dibuat', cell: ({ row }) => h('span', { class: 'text-muted' }, formatDate(row.original.created_at)) },
  {
    id: 'actions',
    cell: ({ row }) => perms.canInBranch('branch.manage', row.original.id) && h('div', { class: 'text-right' }, h(UDropdownMenu, {
      content: { align: 'end' },
      items: [
        [{ label: 'Ubah', icon: 'i-lucide-pencil', onSelect: () => openEdit(row.original) }],
        [{ label: 'Hapus', icon: 'i-lucide-trash-2', color: 'error', onSelect: () => remove(row.original) }],
      ],
    }, () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', 'aria-label': 'Aksi' }))),
  },
]
</script>

<template>
  <CompanyPage title="Cabang" :section="{ label: 'Cabang', to: base }">
    <template #actions>
      <UButton v-if="perms.can('branch.manage')" icon="i-lucide-plus" @click="openCreate">Tambah cabang</UButton>
    </template>

    <p class="text-sm text-muted">Cabang dipakai untuk membatasi role anggota ke lokasi atau unit tertentu.</p>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari nama atau kode" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'inactive'])" placeholder="Semua status" class="w-40" />
    </div>

    <UAlert v-if="list.error.value" color="error" variant="subtle" :title="list.error.value"
      :actions="[{ label: 'Coba lagi', color: 'neutral', variant: 'outline', onClick: list.reload }]" />

    <UTable :data="list.items.value" :columns="columns" :loading="list.loading.value" class="rounded-lg border border-default">
      <template #empty>
        <UEmpty
          icon="i-lucide-git-branch"
          :title="list.search.value || status ? 'Tidak ada cabang yang cocok' : 'Belum ada cabang'"
          :description="list.search.value || status ? 'Coba ubah pencarian atau filter.' : 'Tambahkan cabang pertama company ini.'"
          :actions="list.search.value || status || !perms.can('branch.manage') ? [] : [{ label: 'Tambah cabang', icon: 'i-lucide-plus', onClick: openCreate }]"
          variant="naked"
        />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <UModal v-model:open="showForm" :title="editing ? 'Ubah cabang' : 'Tambah cabang'">
      <template #body>
        <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
          <UFormField label="Nama" name="name" required>
            <UInput v-model="state.name" placeholder="Cabang Jakarta Selatan" class="w-full" autofocus />
          </UFormField>
          <UFormField label="Kode" name="code" required help="Unik di company ini; otomatis huruf besar.">
            <UInput v-model="state.code" placeholder="JKT-S" class="w-full" />
          </UFormField>
          <USwitch v-model="state.active" label="Aktif" description="Cabang nonaktif tetap tersimpan." />
          <div class="flex justify-end gap-2 pt-2">
            <UButton color="neutral" variant="ghost" @click="showForm = false">Batal</UButton>
            <UButton type="submit" :loading="saving">{{ editing ? 'Simpan' : 'Tambah' }}</UButton>
          </div>
        </UForm>
      </template>
    </UModal>
  </CompanyPage>
</template>
