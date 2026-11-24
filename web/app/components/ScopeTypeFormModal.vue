<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { ScopeTypeResponse } from '~/types/api'

// Modal buat / ubah scope type custom manual (platform admin). Scope hasil sync tidak lewat sini.
const props = defineProps<{ solutionId: string; scopeType?: ScopeTypeResponse | null }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [scopeType: ScopeTypeResponse] }>()

const api = useApi()
const { toastError, fieldErrors } = useApiError()
const formRef = useTemplateRef('formRef')

const state = reactive({
  code: '', name: '', description: '', parent: 'company' as 'company' | 'branch', sort_order: undefined as number | undefined,
})
const codeTouched = ref(false)
const saving = ref(false)
const isEdit = computed(() => !!props.scopeType)

const parentItems = [
  { label: 'Company', value: 'company', description: 'Hanya assignment level company yang ikut mencakupnya.' },
  { label: 'Cabang', value: 'branch', description: 'Assignment di cabang induknya juga ikut mencakupnya.' },
]

watch(open, (v) => {
  if (!v) return
  codeTouched.value = false
  Object.assign(state, {
    code: props.scopeType?.code ?? '',
    name: props.scopeType?.name ?? '',
    description: props.scopeType?.description ?? '',
    parent: props.scopeType?.parent === 'branch' ? 'branch' : 'company',
    sort_order: props.scopeType?.sort_order,
  })
})
watch(() => state.name, (name) => {
  if (!isEdit.value && !codeTouched.value) state.code = codify(name).replace(/-/g, '_')
})

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!isEdit.value) {
    if (!/^[a-z][a-z0-9_]*$/.test(s.code) || s.code.length > 30) {
      errors.push({ name: 'code', message: 'Huruf kecil, angka, dan garis bawah; diawali huruf (maks 30)' })
    } else if (s.code === 'company' || s.code === 'branch') {
      errors.push({ name: 'code', message: 'company dan branch sudah bawaan' })
    }
  }
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const sortOrder = typeof e.data.sort_order === 'number' ? e.data.sort_order : undefined
    const saved = props.scopeType
      ? (await api.patch<ScopeTypeResponse>(`/api/v1/scope-types/${props.scopeType.id}`, {
          name: e.data.name.trim(), description: e.data.description.trim(), parent: e.data.parent, sort_order: sortOrder,
        })).data
      : (await api.post<ScopeTypeResponse>(`/api/v1/solutions/${props.solutionId}/scope-types`, {
          code: e.data.code, name: e.data.name.trim(), description: e.data.description.trim() || null,
          parent: e.data.parent, sort_order: sortOrder,
        })).data
    open.value = false
    emit('saved', saved)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal menyimpan scope type')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="isEdit ? 'Ubah scope type' : 'Tambah scope type'" description="Scope custom tempat role bisa diberikan, mis. departemen atau proyek. Data ID-nya milik aplikasi solution.">
    <template #body>
      <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama" name="name" required>
          <UInput v-model="state.name" placeholder="Departemen" class="w-full" autofocus />
        </UFormField>
        <UFormField label="Kode" name="code" :required="!isEdit" :help="isEdit ? 'Kode tidak bisa diubah.' : 'Dipakai sebagai scope_type saat assign & authz check.'">
          <UInput v-model="state.code" :disabled="isEdit" placeholder="department" class="w-full font-mono" @update:model-value="codeTouched = true" />
        </UFormField>
        <UFormField label="Deskripsi" name="description">
          <UTextarea v-model="state.description" :rows="2" placeholder="Opsional" class="w-full" />
        </UFormField>
        <UFormField label="Bagian dari" name="parent">
          <URadioGroup v-model="state.parent" :items="parentItems" />
        </UFormField>
        <UFormField label="Urutan" name="sort_order" :help="isEdit ? 'Angka kecil tampil lebih dulu.' : 'Kosongkan agar tampil setelah scope terakhir.'">
          <UInputNumber v-model="state.sort_order" :min="0" class="w-40" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Batal</UButton>
          <UButton type="submit" :loading="saving">{{ isEdit ? 'Simpan' : 'Tambah scope type' }}</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
