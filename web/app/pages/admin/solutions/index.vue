<script setup lang="ts">
import { h, resolveComponent } from 'vue'
import type { FormError, FormSubmitEvent, TableColumn } from '@nuxt/ui'
import type { SolutionResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const api = useApi()
const toast = useToast()
const { toastError, fieldErrors } = useApiError()
const formRef = useTemplateRef('formRef')

const status = ref<string | undefined>(undefined)
const list = useListQuery<SolutionResponse>('/api/v1/solutions', { filters: { status }, syncUrl: true })

const showCreate = ref(false)
const saving = ref(false)
const state = reactive({ name: '', code: '', description: '' })
const codeTouched = ref(false)

watch(showCreate, (v) => {
  if (v) {
    Object.assign(state, { name: '', code: '', description: '' })
    codeTouched.value = false
  }
})
watch(() => state.name, (n) => {
  if (!codeTouched.value) state.code = slugify(n)
})

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(s.code)) errors.push({ name: 'code', message: 'Huruf kecil, angka, dan tanda hubung' })
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const res = await api.post<SolutionResponse>('/api/v1/solutions', {
      code: e.data.code, name: e.data.name.trim(), description: e.data.description.trim() || null,
    })
    showCreate.value = false
    toast.add({ title: `${res.data.name} dibuat`, color: 'success' })
    navigateTo(`/admin/solutions/${res.data.id}`)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal membuat solution')
  } finally {
    saving.value = false
  }
}

const StatusBadge = resolveComponent('StatusBadge')
const UIcon = resolveComponent('UIcon')
const columns: TableColumn<SolutionResponse>[] = [
  {
    accessorKey: 'name',
    header: 'Solution',
    cell: ({ row }) => h('div', [
      h('p', { class: 'font-medium text-highlighted' }, row.original.name),
      h('p', { class: 'font-mono text-xs text-muted' }, row.original.code),
    ]),
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }) => h(StatusBadge, { status: row.original.status }) },
  {
    accessorKey: 'permission_catalog_version',
    header: 'Versi katalog',
    cell: ({ row }) => h('span', { class: 'font-mono text-xs text-muted' }, row.original.permission_catalog_version ?? 'belum sync'),
  },
  { id: 'go', cell: () => h('div', { class: 'text-right' }, h(UIcon, { name: 'i-lucide-chevron-right', class: 'size-4 text-dimmed' })) },
]
</script>

<template>
  <AppPage title="Solutions" description="Produk yang dilayani platform, beserta role global, permission, dan client backend-nya.">
    <template #actions>
      <UButton icon="i-lucide-plus" @click="showCreate = true">Buat solution</UButton>
    </template>

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="list.search.value" icon="i-lucide-search" placeholder="Cari nama atau kode" class="w-full sm:w-72" />
      <USelect v-model="status" :items="statusOptions(['active', 'inactive'])" placeholder="Semua status" class="w-40" />
    </div>

    <UAlert v-if="list.error.value" color="error" variant="subtle" :title="list.error.value" />

    <UTable
      :data="list.items.value"
      :columns="columns"
      :loading="list.loading.value"
      class="rounded-lg border border-default"
      :ui="{ tr: 'cursor-pointer' }"
      @select="(_e: Event, row: { original: SolutionResponse }) => navigateTo(`/admin/solutions/${row.original.id}`)"
    >
      <template #empty>
        <UEmpty icon="i-lucide-boxes" title="Tidak ada solution" variant="naked" />
      </template>
    </UTable>

    <ListPagination v-model:page="list.page.value" :meta="list.meta.value" />

    <UModal v-model:open="showCreate" title="Buat solution">
      <template #body>
        <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
          <UFormField label="Nama" name="name" required>
            <UInput v-model="state.name" placeholder="Easy HR" class="w-full" autofocus />
          </UFormField>
          <UFormField label="Kode" name="code" required help="Dipakai sebagai audience token; tidak bisa diubah setelah dibuat.">
            <UInput v-model="state.code" placeholder="easy-hr" class="w-full font-mono" @update:model-value="codeTouched = true" />
          </UFormField>
          <UFormField label="Deskripsi" name="description">
            <UTextarea v-model="state.description" :rows="2" class="w-full" />
          </UFormField>
          <div class="flex justify-end gap-2 pt-2">
            <UButton color="neutral" variant="ghost" @click="showCreate = false">Batal</UButton>
            <UButton type="submit" :loading="saving">Buat solution</UButton>
          </div>
        </UForm>
      </template>
    </UModal>
  </AppPage>
</template>
