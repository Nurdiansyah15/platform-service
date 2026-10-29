<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { RolePermissionResponse, RoleResponse } from '~/types/api'

// Modal buat / ubah role. Saat membuat, permission bisa disalin dari role lain (templates).
const props = defineProps<{
  /** Endpoint POST pembuatan role. */
  createPath: string
  /** Endpoint PATCH untuk role yang diubah. */
  updatePath?: (roleId: string) => string
  /** Endpoint PUT permission untuk role baru (dipakai saat menyalin permission). */
  permissionsPath?: (roleId: string) => string
  role?: RoleResponse | null
  templates?: RoleResponse[]
}>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [role: RoleResponse] }>()

const api = useApi()
const { toastError, fieldErrors } = useApiError()
const formRef = useTemplateRef('formRef')

const state = reactive({ name: '', code: '', description: '', template: undefined as string | undefined })
const codeTouched = ref(false)
const saving = ref(false)
const isEdit = computed(() => !!props.role)

watch(open, (v) => {
  if (!v) return
  codeTouched.value = false
  Object.assign(state, {
    name: props.role?.name ?? '',
    code: props.role?.code ?? '',
    description: props.role?.description ?? '',
    template: undefined,
  })
})
watch(() => state.name, (name) => {
  if (!isEdit.value && !codeTouched.value) state.code = codify(name)
})

const templateItems = computed(() => (props.templates ?? []).map(r => ({
  label: r.name, value: r.id, description: `${r.permission_count} permission`,
})))

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!isEdit.value && !/^[a-z][a-z0-9_]*$/.test(s.code)) {
    errors.push({ name: 'code', message: 'Huruf kecil, angka, dan garis bawah; diawali huruf' })
  }
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    let saved: RoleResponse
    if (props.role && props.updatePath) {
      saved = (await api.patch<RoleResponse>(props.updatePath(props.role.id), {
        name: e.data.name.trim(), description: e.data.description.trim(),
      })).data
    } else {
      saved = (await api.post<RoleResponse>(props.createPath, {
        code: e.data.code, name: e.data.name.trim(), description: e.data.description.trim() || null,
      })).data
      if (e.data.template && props.permissionsPath) {
        const perms = (await api.get<RolePermissionResponse[]>(`/api/v1/roles/${e.data.template}/permissions`)).data
        if (perms.length) {
          await api.put(props.permissionsPath(saved.id), { permission_codes: perms.map(p => p.code) })
        }
      }
    }
    open.value = false
    emit('saved', saved)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal menyimpan role')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="isEdit ? 'Ubah role' : 'Buat role'">
    <template #body>
      <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama" name="name" required>
          <UInput v-model="state.name" placeholder="Kepala Cabang" class="w-full" autofocus />
        </UFormField>
        <UFormField label="Kode" name="code" :required="!isEdit" :help="isEdit ? 'Kode tidak bisa diubah.' : 'Dipakai aplikasi untuk mengenali role.'">
          <UInput v-model="state.code" :disabled="isEdit" placeholder="kepala_cabang" class="w-full font-mono" @update:model-value="codeTouched = true" />
        </UFormField>
        <UFormField label="Deskripsi" name="description">
          <UTextarea v-model="state.description" :rows="2" placeholder="Opsional" class="w-full" />
        </UFormField>
        <UFormField v-if="!isEdit && templateItems.length" label="Salin permission dari" help="Opsional — mulai dari permission role yang sudah ada.">
          <USelectMenu v-model="state.template" :items="templateItems" value-key="value" placeholder="Mulai kosong" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Batal</UButton>
          <UButton type="submit" :loading="saving">{{ isEdit ? 'Simpan' : 'Buat role' }}</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
