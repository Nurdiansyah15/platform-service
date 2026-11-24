<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { PermissionGroupResponse } from '~/types/api'

// Modal buat / ubah permission group manual (platform admin). Group hasil sync tidak lewat sini.
const props = defineProps<{ solutionId: string; group?: PermissionGroupResponse | null }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [group: PermissionGroupResponse] }>()

const api = useApi()
const { toastError, fieldErrors } = useApiError()
const formRef = useTemplateRef('formRef')

const state = reactive({ code: '', name: '', description: '', sort_order: undefined as number | undefined })
const codeTouched = ref(false)
const saving = ref(false)
const isEdit = computed(() => !!props.group)

watch(open, (v) => {
  if (!v) return
  codeTouched.value = false
  Object.assign(state, {
    code: props.group?.code ?? '',
    name: props.group?.name ?? '',
    description: props.group?.description ?? '',
    sort_order: props.group?.sort_order,
  })
})
watch(() => state.name, (name) => {
  if (!isEdit.value && !codeTouched.value) state.code = codify(name)
})

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!isEdit.value && !/^[a-z][a-z0-9_-]*$/.test(s.code)) {
    errors.push({ name: 'code', message: 'Huruf kecil, angka, - atau _; diawali huruf' })
  }
  if (s.sort_order !== undefined && s.sort_order !== null && s.sort_order < 0) {
    errors.push({ name: 'sort_order', message: 'Tidak boleh negatif' })
  }
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const sortOrder = typeof e.data.sort_order === 'number' ? e.data.sort_order : undefined
    const saved = props.group
      ? (await api.patch<PermissionGroupResponse>(`/api/v1/permission-groups/${props.group.id}`, {
          name: e.data.name.trim(), description: e.data.description.trim(), sort_order: sortOrder,
        })).data
      : (await api.post<PermissionGroupResponse>(`/api/v1/solutions/${props.solutionId}/permission-groups`, {
          code: e.data.code, name: e.data.name.trim(), description: e.data.description.trim() || null, sort_order: sortOrder,
        })).data
    open.value = false
    emit('saved', saved)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal menyimpan grup')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="isEdit ? 'Ubah grup' : 'Tambah grup'" description="Grup manual dikelola dari platform dan tidak disentuh sync katalog.">
    <template #body>
      <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama" name="name" required>
          <UInput v-model="state.name" placeholder="Laporan" class="w-full" autofocus />
        </UFormField>
        <UFormField label="Kode" name="code" :required="!isEdit" :help="isEdit ? 'Kode tidak bisa diubah.' : 'Tidak bisa diubah setelah dibuat.'">
          <UInput v-model="state.code" :disabled="isEdit" placeholder="report" class="w-full font-mono" @update:model-value="codeTouched = true" />
        </UFormField>
        <UFormField label="Deskripsi" name="description">
          <UTextarea v-model="state.description" :rows="2" placeholder="Opsional" class="w-full" />
        </UFormField>
        <UFormField label="Urutan" name="sort_order" :help="isEdit ? 'Angka kecil tampil lebih dulu.' : 'Kosongkan agar tampil setelah grup terakhir.'">
          <UInputNumber v-model="state.sort_order" :min="0" class="w-40" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Batal</UButton>
          <UButton type="submit" :loading="saving">{{ isEdit ? 'Simpan' : 'Tambah grup' }}</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
