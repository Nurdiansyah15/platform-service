<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { PermissionGroupResponse, PermissionResponse } from '~/types/api'

// Modal buat / ubah permission manual (platform admin). Permission hasil sync tidak lewat sini.
const props = defineProps<{
  solutionId: string
  permission?: PermissionResponse | null
  groups: PermissionGroupResponse[]
}>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [permission: PermissionResponse] }>()

const api = useApi()
const { toastError, fieldErrors } = useApiError()
const formRef = useTemplateRef('formRef')

const state = reactive({ code: '', name: '', description: '', group: undefined as string | undefined })
const saving = ref(false)
const isEdit = computed(() => !!props.permission)

watch(open, (v) => {
  if (!v) return
  Object.assign(state, {
    code: props.permission?.code ?? '',
    name: props.permission?.name ?? '',
    description: props.permission?.description ?? '',
    group: props.permission?.group?.code,
  })
})

const groupItems = computed(() => [
  { label: 'Tanpa grup', value: undefined as string | undefined },
  ...props.groups
    .filter(g => g.status === 'active' || g.code === props.permission?.group?.code)
    .map(g => ({ label: g.name, value: g.code as string | undefined, description: g.description ?? undefined })),
])

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!isEdit.value && !/^[a-z][a-z0-9_-]*([.:][a-z0-9_-]+)*$/.test(s.code)) {
    errors.push({ name: 'code', message: 'Huruf kecil, segmen dipisah titik, mis. report.export' })
  }
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const body = {
      name: e.data.name.trim(),
      description: e.data.description.trim(),
      group: e.data.group ?? '',
    }
    const saved = props.permission
      ? (await api.patch<PermissionResponse>(`/api/v1/permissions/${props.permission.id}`, body)).data
      : (await api.post<PermissionResponse>(`/api/v1/solutions/${props.solutionId}/permissions`, {
          ...body, code: e.data.code, description: body.description || null, group: e.data.group ?? null,
        })).data
    open.value = false
    emit('saved', saved)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal menyimpan permission')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="isEdit ? 'Ubah permission' : 'Tambah permission'" description="Permission manual dikelola dari platform dan tidak disentuh sync katalog.">
    <template #body>
      <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama" name="name" required>
          <UInput v-model="state.name" placeholder="Export laporan" class="w-full" autofocus />
        </UFormField>
        <UFormField
          label="Kode"
          name="code"
          :required="!isEdit"
          :help="isEdit ? 'Kode tidak bisa diubah.' : 'Dipakai aplikasi saat authz check; tidak bisa diubah setelah dibuat.'"
        >
          <UInput v-model="state.code" :disabled="isEdit" placeholder="report.export" class="w-full font-mono" />
        </UFormField>
        <UFormField label="Deskripsi" name="description">
          <UTextarea v-model="state.description" :rows="2" placeholder="Opsional" class="w-full" />
        </UFormField>
        <UFormField label="Grup" name="group">
          <USelectMenu v-model="state.group" :items="groupItems" value-key="value" placeholder="Tanpa grup" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Batal</UButton>
          <UButton type="submit" :loading="saving">{{ isEdit ? 'Simpan' : 'Tambah permission' }}</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
